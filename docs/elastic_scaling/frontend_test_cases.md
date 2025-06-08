# 弹性伸缩策略评估前端测试用例

## 概述

基于现有的单元测试 `elastic_scaling_evaluation_test.go`，生成3个前端页面测试场景，用于验证弹性伸缩策略的评估逻辑和订单生成功能。

## 测试场景

### 场景1：入池订单生成 - CPU使用率持续超过阈值

**测试目标**：验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单

**前置条件**：
- 策略状态为启用
- 设置CPU阈值为80%，触发动作为入池
- 有可用的设备用于匹配

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问策略管理页面，验证策略状态
3. 访问资源监控页面，查看资源快照数据
4. 等待或手动触发策略评估
5. 访问订单管理页面，验证入池订单生成
6. 查看订单详情，验证设备分配和邮件通知

**预期结果**：
- 策略评估成功，生成入池订单
- 订单状态为"待处理"
- 订单包含匹配的设备信息
- 策略执行历史记录为"order_created"

### 场景2：退池订单生成 - 内存分配率持续低于阈值

**测试目标**：验证当内存分配率连续2天低于20%时，系统能够正确生成退池订单

**前置条件**：
- 策略状态为启用
- 设置内存分配率阈值为20%，触发动作为退池
- 有在池设备可用于退池

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问策略管理页面，验证策略配置
3. 访问资源监控页面，查看内存分配趋势
4. 等待或手动触发策略评估
5. 访问订单管理页面，验证退池订单生成
6. 查看订单详情和设备信息

**预期结果**：
- 策略评估成功，生成退池订单
- 订单类型为"退池"
- 包含需要退池的设备列表
- 策略执行历史记录为"order_created"

### 场景3：不满足条件 - 阈值未持续达到要求

**测试目标**：验证当资源使用率未连续满足阈值要求时，系统不生成订单

**前置条件**：
- 策略状态为启用
- 设置CPU阈值为80%，要求连续3天
- 资源快照数据中断（第2天低于阈值）

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问策略管理页面，验证策略配置
3. 访问资源监控页面，查看资源使用趋势
4. 等待或手动触发策略评估
5. 访问订单管理页面，验证无新订单生成
6. 查看策略执行历史，验证失败原因

**预期结果**：
- 策略评估完成，但不生成订单
- 策略执行历史记录为"failure_threshold_not_met"
- 历史记录包含详细的失败原因
- 订单列表无新增订单

### 场景4：入池无法匹配到设备

**测试目标**：验证当满足入池条件但无可用设备时，系统生成提醒订单的处理逻辑

**前置条件**：
- 策略状态为启用，要求2台设备
- CPU阈值满足触发条件
- 所有设备均为非可用状态（in_pool、maintenance、offline、reserved）

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问设备管理页面，确认无可用设备
3. 访问策略管理页面，验证策略配置
4. 手动触发策略评估
5. 访问订单管理页面，验证生成提醒订单
6. 查看订单详情，确认设备不足提醒信息
7. 查看邮件通知内容，确认设备申请提醒

**预期结果**：
- 策略评估成功，生成提醒订单
- 订单状态为"待处理"，设备数量为0
- 订单详情显示"找不到要处理的设备，请自行协调处理"
- 邮件通知包含设备申请提醒内容
- 策略执行历史记录为"order_created_no_devices"

### 场景5：退池无法匹配到设备

**测试目标**：验证当满足退池条件但无在池设备时，系统生成提醒订单的处理逻辑

**前置条件**：
- 策略状态为启用，要求1台设备
- 内存分配率满足触发条件
- 所有设备均为非在池状态（in_stock、maintenance、offline、reserved）

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问设备管理页面，确认无在池设备
3. 访问策略管理页面，验证策略配置
4. 手动触发策略评估
5. 访问订单管理页面，验证生成提醒订单
6. 查看订单详情，确认设备不足提醒信息
7. 查看邮件通知内容，确认协调处理提醒

**预期结果**：
- 策略评估成功，生成提醒订单
- 订单状态为"待处理"，设备数量为0
- 订单详情显示"找不到要处理的设备，请自行协调处理"
- 邮件通知包含协调处理提醒内容
- 策略执行历史记录为"order_created_no_devices"

### 场景6：入池只能匹配部分设备

**测试目标**：验证当满足入池条件但只能匹配到部分设备时，系统的处理逻辑

**前置条件**：
- 策略状态为启用，要求5台设备
- CPU阈值满足触发条件
- 只有2台可用设备，其余为不可用状态

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问设备管理页面，确认只有2台可用设备
3. 访问策略管理页面，验证策略配置
4. 手动触发策略评估
5. 访问订单管理页面，验证部分订单生成
6. 查看订单详情，确认只包含2台设备

**预期结果**：
- 策略评估成功，生成部分订单
- 订单包含2台可用设备
- 策略执行历史记录为"order_created_partial"
- 历史记录说明部分匹配情况

### 场景7：退池只能匹配部分设备

**测试目标**：验证当满足退池条件但只能匹配到部分设备时，系统的处理逻辑

**前置条件**：
- 策略状态为启用，要求4台设备
- 内存分配率满足触发条件
- 只有2台在池设备，其余为非在池状态

**测试步骤**：
1. 执行Mock数据初始化SQL
2. 访问设备管理页面，确认只有2台在池设备
3. 访问策略管理页面，验证策略配置
4. 手动触发策略评估
5. 访问订单管理页面，验证部分订单生成
6. 查看订单详情，确认只包含2台设备

**预期结果**：
- 策略评估成功，生成部分订单
- 订单包含2台在池设备
- 策略执行历史记录为"order_created_partial"
- 历史记录说明部分匹配情况

## Mock数据SQL脚本

### 场景1：入池订单生成数据

```sql
-- 清理现有数据
DELETE FROM strategy_execution_history;
DELETE FROM order_device;
DELETE FROM elastic_scaling_order_details;
DELETE FROM orders;
DELETE FROM resource_snapshots;
DELETE FROM strategy_cluster_associations;
DELETE FROM elastic_scaling_strategies;
DELETE FROM query_templates;
DELETE FROM devices;
DELETE FROM k8s_clusters;

-- 创建集群
INSERT INTO k8s_clusters (id, cluster_name, created_at, updated_at) VALUES 
(1, 'production-cluster', datetime('now'), datetime('now'));

-- 创建设备
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
(1, 'DEV001', '192.168.1.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(2, 'DEV002', '192.168.1.11', 'x86_64', 16.0, 32.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(3, 'DEV003', '192.168.1.12', 'arm64', 12.0, 24.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(1, 'Find Available Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略
INSERT INTO elastic_scaling_strategies (
    id, name, description, threshold_trigger_action, 
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes, device_count,
    node_selector, resource_types, status, entry_query_template_id,
    created_by, created_at, updated_at
) VALUES (
    1, 'CPU High Usage Scale Out', 'Scale out when CPU usage is high', 'pool_entry',
    80.0, 'usage', 70.0,
    0, '', 0,
    'AND', 3, 60, 2,
    '', 'total', 'enabled', 1,
    'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO strategy_cluster_associations (strategy_id, cluster_id, created_at, updated_at) VALUES 
(1, 1, datetime('now'), datetime('now'));

-- 创建资源快照数据（连续3天CPU使用率超过80%）
INSERT INTO resource_snapshots (
    cluster_id, resource_type, resource_pool,
    max_cpu_usage_ratio, max_memory_usage_ratio,
    cpu_request, cpu_capacity, mem_request, memory_capacity,
    created_at, updated_at
) VALUES 
-- 3天前：CPU 85%
(1, 'total', 'total', 85.0, 65.0, 850.0, 1000.0, 6500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：CPU 90%
(1, 'total', 'total', 90.0, 70.0, 900.0, 1000.0, 7000.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：CPU 88%
(1, 'total', 'total', 88.0, 68.0, 880.0, 1000.0, 6800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));
```

### 场景2：退池订单生成数据

```sql
-- 清理现有数据
DELETE FROM strategy_execution_history;
DELETE FROM order_device;
DELETE FROM elastic_scaling_order_details;
DELETE FROM orders;
DELETE FROM resource_snapshots;
DELETE FROM strategy_cluster_associations;
DELETE FROM elastic_scaling_strategies;
DELETE FROM query_templates;
DELETE FROM devices;
DELETE FROM k8s_clusters;

-- 创建集群
INSERT INTO k8s_clusters (id, cluster_name, created_at, updated_at) VALUES 
(2, 'staging-cluster', datetime('now'), datetime('now'));

-- 创建在池设备
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
(4, 'DEV004', '192.168.2.10', 'x86_64', 8.0, 16.0, 'in_pool', 'worker', 'staging-cluster', 2, 0, 0, datetime('now'), datetime('now')),
(5, 'DEV005', '192.168.2.11', 'x86_64', 16.0, 32.0, 'in_pool', 'worker', 'staging-cluster', 2, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(2, 'Find Pool Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_pool"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（退池）
INSERT INTO elastic_scaling_strategies (
    id, name, description, threshold_trigger_action, 
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes, device_count,
    node_selector, resource_types, status, exit_query_template_id,
    created_by, created_at, updated_at
) VALUES (
    2, 'Memory Low Usage Scale In', 'Scale in when memory allocation is low', 'pool_exit',
    0, '', 0,
    20.0, 'allocated', 30.0,
    'AND', 2, 60, 1,
    '', 'total', 'enabled', 2,
    'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO strategy_cluster_associations (strategy_id, cluster_id, created_at, updated_at) VALUES 
(2, 2, datetime('now'), datetime('now'));

-- 创建资源快照数据（连续2天内存分配率低于20%）
INSERT INTO resource_snapshots (
    cluster_id, resource_type, resource_pool,
    max_cpu_usage_ratio, max_memory_usage_ratio,
    cpu_request, cpu_capacity, mem_request, memory_capacity,
    created_at, updated_at
) VALUES 
-- 2天前：内存分配率 15%
(2, 'total', 'total', 45.0, 35.0, 450.0, 1000.0, 1500.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：内存分配率 18%
(2, 'total', 'total', 40.0, 32.0, 400.0, 1000.0, 1800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));
```

### 场景3：不满足条件数据

```sql
-- 清理现有数据
DELETE FROM strategy_execution_history;
DELETE FROM order_device;
DELETE FROM elastic_scaling_order_details;
DELETE FROM orders;
DELETE FROM resource_snapshots;
DELETE FROM strategy_cluster_associations;
DELETE FROM elastic_scaling_strategies;
DELETE FROM query_templates;
DELETE FROM devices;
DELETE FROM k8s_clusters;

-- 创建集群
INSERT INTO k8s_clusters (id, cluster_name, created_at, updated_at) VALUES
(3, 'test-cluster', datetime('now'), datetime('now'));

-- 创建设备
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES
(6, 'DEV006', '192.168.3.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'test-cluster', 3, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES
(3, 'Find Test Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略
INSERT INTO elastic_scaling_strategies (
    id, name, description, threshold_trigger_action,
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes, device_count,
    node_selector, resource_types, status, entry_query_template_id,
    created_by, created_at, updated_at
) VALUES (
    3, 'CPU Threshold Not Met Test', 'Test strategy for threshold not met scenario', 'pool_entry',
    80.0, 'usage', 70.0,
    0, '', 0,
    'AND', 3, 60, 1,
    '', 'total', 'enabled', 3,
    'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO strategy_cluster_associations (strategy_id, cluster_id, created_at, updated_at) VALUES
(3, 3, datetime('now'), datetime('now'));

-- 创建资源快照数据（第2天中断，不满足连续3天要求）
INSERT INTO resource_snapshots (
    cluster_id, resource_type, resource_pool,
    max_cpu_usage_ratio, max_memory_usage_ratio,
    cpu_request, cpu_capacity, mem_request, memory_capacity,
    created_at, updated_at
) VALUES
-- 4天前：CPU 85%（满足）
(3, 'total', 'total', 85.0, 65.0, 850.0, 1000.0, 6500.0, 10000.0, datetime('now', '-4 days'), datetime('now', '-4 days')),
-- 3天前：CPU 90%（满足）
(3, 'total', 'total', 90.0, 70.0, 900.0, 1000.0, 7000.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：CPU 70%（不满足，中断连续性）
(3, 'total', 'total', 70.0, 60.0, 700.0, 1000.0, 6000.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：CPU 88%（满足，但连续性已中断）
(3, 'total', 'total', 88.0, 68.0, 880.0, 1000.0, 6800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));
```

## 测试执行指南

### 环境准备

1. **数据库准备**
   ```bash
   # 连接到SQLite数据库
   sqlite3 /path/to/your/database.db

   # 执行对应场景的SQL脚本
   .read scenario1_data.sql
   ```

2. **前端环境启动**
   ```bash
   cd web/navy-fe
   npm install
   npm run dev
   ```

3. **后端服务启动**
   ```bash
   cd server/portal
   go run main.go
   ```

### 测试页面路径

1. **策略管理页面**：`/elastic-scaling/strategies`
2. **资源监控页面**：`/elastic-scaling/dashboard`
3. **订单管理页面**：`/elastic-scaling/orders`
4. **策略执行历史**：`/elastic-scaling/strategies/{id}/history`

### 手动触发策略评估

如果需要手动触发策略评估（用于测试），可以通过以下API：

```bash
# 触发所有策略评估
curl -X POST http://localhost:8080/api/v1/elastic-scaling/evaluate

# 触发特定策略评估
curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/{id}/evaluate
```

### 验证要点

#### 场景1验证要点
- [ ] 策略状态显示为"启用"
- [ ] 资源监控图表显示CPU使用率连续3天超过80%
- [ ] 订单列表中出现新的入池订单
- [ ] 订单详情显示正确的设备分配
- [ ] 策略执行历史显示"order_created"记录
- [ ] 邮件通知内容生成正确

#### 场景2验证要点
- [ ] 策略配置显示退池动作和内存阈值
- [ ] 资源监控显示内存分配率持续低于20%
- [ ] 生成退池订单，包含在池设备
- [ ] 订单类型正确标识为"退池"
- [ ] 策略执行历史记录正确

#### 场景3验证要点
- [ ] 策略配置正确，但资源数据不满足连续性要求
- [ ] 资源监控显示第2天CPU使用率低于阈值
- [ ] 订单列表无新增订单
- [ ] 策略执行历史显示"failure_threshold_not_met"
- [ ] 失败原因描述清晰准确

### 测试数据清理

测试完成后，可以使用以下SQL清理测试数据：

```sql
-- 清理所有测试数据
DELETE FROM strategy_execution_history;
DELETE FROM order_device;
DELETE FROM elastic_scaling_order_details;
DELETE FROM orders;
DELETE FROM resource_snapshots;
DELETE FROM strategy_cluster_associations;
DELETE FROM elastic_scaling_strategies;
DELETE FROM query_templates;
DELETE FROM devices;
DELETE FROM k8s_clusters;

-- 重置自增ID（如果需要）
DELETE FROM sqlite_sequence WHERE name IN (
    'strategy_execution_history', 'order_device', 'elastic_scaling_order_details',
    'orders', 'resource_snapshots', 'strategy_cluster_associations',
    'elastic_scaling_strategies', 'query_templates', 'devices', 'k8s_clusters'
);
```

## 注意事项

1. **时间依赖**：测试数据使用相对时间（如`datetime('now', '-1 days')`），确保测试时间的准确性
2. **数据一致性**：确保集群ID、设备ID、策略ID等外键关系正确
3. **策略评估周期**：了解系统的策略评估周期，或使用手动触发进行测试
4. **邮件功能**：邮件通知功能需要配置邮件服务器或查看日志中的邮件内容
5. **权限验证**：确保测试用户具有查看和操作相关页面的权限

## 扩展测试场景

可以基于这些基础场景扩展更多测试用例：

- **冷却期测试**：验证策略在冷却期内不会重复触发
- **多策略并发**：测试多个策略同时满足条件的情况
- **设备不足**：测试可用设备不足时的处理逻辑
- **网络异常**：模拟网络异常情况下的系统行为
- **并发评估**：测试多个策略评估实例的并发处理
