# 集群资源未入池设备内存计算实现

## 概述

本文档描述了将未入池设备的内存总和计算到 `SecurityZoneResourceDTO` 的 `Pending` 字段中的实现。

## 修改内容

### 1. DTO 结构修改

在 `SecurityZoneResourceDTO` 中添加了 `Pending` 字段：

```go
type SecurityZoneResourceDTO struct {
    SecurityZone   string `json:"security_zone"`
    AvailableMem   string `json:"available_mem"`
    AvailableCount string `json:"available_count"`
    Pending        string `json:"pending"`  // 新增字段
}
```

### 2. 服务逻辑修改

#### 主要计算方法
- 修改了 `CalculateRemainingResources` 方法，使用新的构建方法
- 创建了 `buildResponseStructureWithPendingField` 方法来处理新的数据结构

#### 数据处理流程
1. **集群资源计算**: 计算现有集群的剩余内存容量
2. **设备资源计算**: 计算未入池设备的总内存
3. **数据合并**: 将设备内存总和放入对应zone的 `Pending` 字段

### 3. 新增方法

#### `buildResponseStructureWithPendingField`
- 构建包含 `Pending` 字段的响应结构
- 将集群资源放入 `available_mem` 和 `available_count`
- 将设备资源放入 `pending` 字段

#### `buildOrganizationResourceMapWithPendingField`
- 按组织和IDC分组资源数据
- 处理集群和设备资源的映射关系
- 确保每个zone都有完整的数据

#### `convertOrgMapToListWithPendingField`
- 将内部映射结构转换为最终的DTO列表
- 保持IDC级别的 `pending` 数组为空（不使用）

## 数据来源和计算逻辑

### 集群资源 (available_mem)
- **数据源**: `k8s_cluster_resource_snapshot` + `k8s_cluster`
- **过滤条件**: `resource_type = 'hg_common'`
- **计算公式**: `(capacity * 0.75) - request`
- **结果**: 集群的可用内存容量

### 设备资源 (pending)
- **数据源**: `devices` 表
- **过滤条件**: 
  - `cluster = ''` (未入池)
  - `ci_code LIKE '%EQUHST%'`
  - `appid IN ('85004', '85494')`
  - `arch_type = 'X86'`
  - `is_localization = true`
- **计算公式**: `SUM(total_memory)` 按IDC和Zone分组
- **结果**: 未入池设备的总内存

## 响应结构示例

```json
{
    "code": 0,
    "message": "success",
    "list": [
        {
            "organization": "总行",
            "idcs": [
                {
                    "idc_name": "gl",
                    "zones": [
                        {
                            "security_zone": "APP",
                            "available_mem": "256.5GiB",
                            "available_count": "32",
                            "pending": "128.0GiB"
                        }
                    ],
                    "pending": []
                }
            ]
        }
    ]
}
```

## 业务价值

### 1. 区域级别可见性
- 在每个安全区域级别同时显示集群容量和设备库存
- 便于进行精确的容量规划

### 2. 清晰的数据分离
- `available_mem`: 立即可用的集群容量
- `pending`: 需要配置的设备内存
- 两个值清楚地区分了不同类型的资源

### 3. 决策支持
- **高 available_mem**: 使用现有集群容量
- **高 pending**: 考虑用新设备扩展集群
- **两者都高**: 容量情况良好，有扩展选项
- **两者都低**: 可能需要采购新硬件或优化使用

## 使用场景

1. **容量规划**: 查看每个区域的即时和未来容量
2. **资源分配决策**: 在使用现有集群容量和分配新设备之间做决定
3. **容量趋势分析**: 跟踪集群利用率和设备库存随时间的变化
4. **组织资源摘要**: 按组织查看具有区域级别详细信息的总资源

## 测试

创建了 `cluster_resource_pending_field_test.go` 来验证新实现的正确性，包括：
- 数据结构测试
- 计算逻辑验证
- 业务场景覆盖
- API使用示例

## 兼容性

- 保持了现有的API接口不变
- 只是在响应结构中添加了新字段
- 向后兼容，不影响现有客户端