# 弹性扩缩容测试场景概览

本文档描述了7个弹性扩缩容测试场景，每个场景都有对应的SQL测试数据文件。

## 测试场景列表

### 场景1：入池订单生成测试 ✅
**文件**: `test_data_scenario1_pool_entry.sql`
**测试目标**: 验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单
**关键特征**:
- 连续3天CPU使用率: 85%, 90%, 88% (均超过80%阈值)
- 有2台可用设备 (status='in_stock')
- 策略要求2台设备
- **预期结果**: 生成订单，匹配2台设备

### 场景2：退池订单生成测试 ✅
**文件**: `test_data_scenario2_pool_exit.sql`
**测试目标**: 验证当CPU使用率连续3天低于30%时，系统能够正确生成退池订单
**关键特征**:
- 连续3天CPU使用率: 25%, 20%, 28% (均低于30%阈值)
- 有2台运行中设备 (status='running')
- 策略要求2台设备
- **预期结果**: 生成订单，匹配2台设备

### 场景3：阈值未达到测试 ✅
**文件**: `test_data_scenario3_threshold_not_met.sql`
**测试目标**: 验证当CPU使用率未连续达到阈值时，系统不会生成订单
**关键特征**:
- 3天CPU使用率: 85%, 75%, 88% (中间一天未超过80%阈值)
- 有可用设备
- **预期结果**: 不生成订单，不触发策略

### 场景4：入池无可用设备测试 ✅
**文件**: `test_data_scenario4_pool_entry_no_devices.sql`
**测试目标**: 验证当触发入池条件但无可用设备时，系统仍会生成订单作为告警
**关键特征**:
- 连续3天CPU使用率超过80%
- 无可用设备 (全部为running/maintenance状态)
- **预期结果**: 生成告警订单，设备数量为0，作为值班告警

### 场景5：退池无可用设备测试 ✅
**文件**: `test_data_scenario5_pool_exit_no_devices.sql`
**测试目标**: 验证当触发退池条件但无可退池设备时，系统仍会生成订单作为告警
**关键特征**:
- 连续3天CPU使用率低于30%
- 无运行中设备 (全部为in_stock/maintenance状态)
- **预期结果**: 生成告警订单，设备数量为0

### 场景6：入池部分设备匹配测试 ✅
**文件**: `test_data_scenario6_pool_entry_partial_devices.sql`
**测试目标**: 验证当触发入池条件但只有部分设备可用时，系统生成订单并匹配可用设备
**关键特征**:
- 连续3天CPU使用率超过80%
- 策略要求3台设备，但只有2台可用 (2台in_stock, 1台running, 1台maintenance)
- **预期结果**: 生成订单，匹配2台可用设备

### 场景7：退池部分设备匹配测试 ✅
**文件**: `test_data_scenario7_pool_exit_partial_devices.sql`
**测试目标**: 验证当触发退池条件但只有部分设备可退池时，系统生成订单并匹配可用设备
**关键特征**:
- 连续3天CPU使用率低于30%
- 策略要求3台设备，但只有2台运行中 (2台running, 1台in_stock, 1台maintenance)
- **预期结果**: 生成订单，匹配2台运行中设备

## 测试数据结构

每个测试场景的SQL文件都包含以下数据结构：

### 基础数据
- **k8s_cluster**: 集群信息
- **device**: 设备信息（不同状态）
- **query_template**: 查询模板（设备匹配条件）
- **elastic_scaling_strategy**: 弹性扩缩容策略
- **strategy_cluster_association**: 策略集群关联

### 触发数据
- **k8s_cluster_resource_snapshot**: 资源快照（连续3天的CPU/内存使用率）

### 结果数据
- **orders**: 基础订单信息
- **elastic_scaling_order_details**: 弹性扩缩容订单详情
- **order_device**: 订单设备关联（如果有匹配设备）
- **strategy_execution_history**: 策略执行历史

### 验证查询
每个文件都包含验证查询，用于：
- 统计各表数据量
- 显示策略详情
- 显示资源快照趋势
- 显示生成的订单信息
- 显示设备匹配情况
- 显示策略执行历史

## 使用方法

### 运行单个场景测试
```bash
# 进入项目根目录
cd /Users/heningyu/software/goapp/auto-navy

# 运行场景1测试
sqlite3 navy.db < docs/elastic_scaling/test_data_scenario1_pool_entry.sql

# 运行场景2测试
sqlite3 navy.db < docs/elastic_scaling/test_data_scenario2_pool_exit.sql

# 以此类推...
```

### 验证API响应
```bash
# 启动服务器
go run server/portal/internal/main.go

# 测试订单API
curl "http://localhost:8081/fe-v1/elastic-scaling/orders/1" | jq .

# 测试订单设备API
curl "http://localhost:8081/fe-v1/elastic-scaling/orders/1/devices" | jq .
```

## 重要修复

所有测试文件都已修复以下问题：
1. ✅ 使用正确的设备ID (1001, 1002, 1003, 1004)
2. ✅ 使用正确的 `order_id` 字段查询 `order_device` 表
3. ✅ 包含完整的验证查询
4. ✅ 提供详细的测试场景说明

## 测试覆盖范围

这7个场景覆盖了弹性扩缩容系统的所有主要功能：
- ✅ 正常入池/退池流程
- ✅ 阈值检测逻辑
- ✅ 设备匹配逻辑
- ✅ 无设备告警机制
- ✅ 部分设备匹配处理
- ✅ 订单生成和设备关联
- ✅ 策略执行历史记录
