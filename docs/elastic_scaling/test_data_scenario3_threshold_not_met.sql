-- 场景3：阈值未达到测试数据
-- 测试目标：验证当CPU使用率未连续达到阈值时，系统不会生成订单

-- 清理现有数据
DELETE FROM ng_strategy_execution_history;
DELETE FROM ng_order_device;
DELETE FROM ng_elastic_scaling_order_details;
DELETE FROM ng_orders;
DELETE FROM k8s_cluster_resource_snapshot;
DELETE FROM ng_strategy_cluster_association;
DELETE FROM ng_resource_pool_device_matching_policy;
DELETE FROM ng_elastic_scaling_strategy;
DELETE FROM query_template;
DELETE FROM device;
DELETE FROM k8s_cluster;

-- 重置自增ID
DELETE FROM sqlite_sequence WHERE name IN (
    'ng_strategy_execution_history', 'ng_order_device', 'ng_elastic_scaling_order_details',
'ng_orders', 'k8s_cluster_resource_snapshot', 'ng_strategy_cluster_association',
'ng_resource_pool_device_matching_policy', 'ng_elastic_scaling_strategy', 'query_template', 'device', 'k8s_cluster'
);

-- 创建集群
INSERT INTO k8s_cluster (id, clustername, created_at, updated_at) VALUES
(1, 'production-cluster', datetime('now'), datetime('now'));

-- 创建设备（可用于入池）
INSERT INTO device (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES
(1001, 'DEV001', '192.168.1.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(1002, 'DEV002', '192.168.1.11', 'x86_64', 16.0, 32.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(1003, 'DEV003', '192.168.1.12', 'arm64', 12.0, 24.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找可用设备）
INSERT INTO query_template (id, name, groups, created_at, updated_at) VALUES
(1, 'Find Available Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（入池策略）
INSERT INTO ng_elastic_scaling_strategy (
    id, name, description, threshold_trigger_action,
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes,
    resource_types, status,
    created_by, created_at, updated_at
) VALUES (
    1, 'CPU High Usage Scale Out', 'Scale out when CPU usage is high', 'pool_entry',
    80.0, 'usage', 70.0,
    0, '', 0,
    'AND', 3, 60,
    'total', 'enabled',
    'admin', datetime('now'), datetime('now')
);

-- 创建设备匹配策略（入池）
INSERT INTO ng_resource_pool_device_matching_policy (
    id, name, description, resource_pool_type, action_type,
    query_template_id, status, addition_conds,
    created_by, updated_by, created_at, updated_at
) VALUES (
    1, 'Total Pool Entry Device Matching', 'Device matching policy for total pool entry', 'total', 'pool_entry',
    1, 'enabled', '',
    'admin', 'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO ng_strategy_cluster_association (strategy_id, cluster_id) VALUES
(1, 1);

-- 创建资源快照数据（CPU使用率波动，未连续3天超过80%）
INSERT INTO k8s_cluster_resource_snapshot (
    cluster_id, resource_type, resource_pool,
    max_cpu, max_memory,
    cpu_request, cpu_capacity, mem_request, mem_capacity,
    created_at, updated_at
) VALUES
-- 3天前：CPU 85% (超过阈值)
(1, 'total', 'total', 85.0, 65.0, 850.0, 1000.0, 6500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：CPU 75% (未超过阈值)
(1, 'total', 'total', 75.0, 60.0, 750.0, 1000.0, 6000.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：CPU 88% (超过阈值，但不连续)
(1, 'total', 'total', 88.0, 68.0, 880.0, 1000.0, 6800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));

-- 注意：此场景不应生成订单，因为CPU使用率未连续3天超过80%

-- 验证数据插入
SELECT 'Clusters:' as table_name, count(*) as count FROM k8s_cluster
UNION ALL
SELECT 'Devices:', count(*) FROM device
UNION ALL
SELECT 'Query Templates:', count(*) FROM query_template
UNION ALL
SELECT 'Strategies:', count(*) FROM ng_elastic_scaling_strategy
UNION ALL
SELECT 'Device Matching Policies:', count(*) FROM ng_resource_pool_device_matching_policy
UNION ALL
SELECT 'Strategy Associations:', count(*) FROM ng_strategy_cluster_association
UNION ALL
SELECT 'Resource Snapshots:', count(*) FROM k8s_cluster_resource_snapshot
UNION ALL
SELECT 'Orders:', count(*) FROM ng_orders
UNION ALL
SELECT 'Order Details:', count(*) FROM ng_elastic_scaling_order_details
UNION ALL
SELECT 'Order Devices:', count(*) FROM ng_order_device
UNION ALL
SELECT 'Execution History:', count(*) FROM ng_strategy_execution_history;

-- 显示策略详情
SELECT
    id, name, threshold_trigger_action,
    cpu_threshold_value, cpu_threshold_type,
    duration_minutes, status
FROM ng_elastic_scaling_strategy;

-- 显示设备匹配策略详情
SELECT
    id, name, resource_pool_type, action_type,
    query_template_id, status, addition_conds
FROM ng_resource_pool_device_matching_policy;

-- 显示资源快照趋势（应该显示波动的CPU使用率）
SELECT
    date(created_at) as snapshot_date,
    max_cpu,
    max_memory,
    CASE
        WHEN max_cpu > 80 THEN 'BREACH'
        ELSE 'NORMAL'
    END as threshold_status,
    CASE
        WHEN max_cpu > 80 THEN '超过阈值'
        ELSE '正常范围'
    END as status_description
FROM k8s_cluster_resource_snapshot
ORDER BY created_at;

-- 验证没有生成订单（应该返回0行）
SELECT
    o.id as order_id,
    o.order_number,
    o.name as order_name,
    o.description,
    o.type,
    o.status,
    o.created_by
FROM ng_orders o;

-- 验证没有策略执行历史（应该返回0行）
SELECT
    seh.id,
    seh.strategy_id,
    seh.cluster_id,
    seh.resource_type,
    seh.triggered_value,
    seh.threshold_value,
    seh.result,
    seh.reason
FROM ng_strategy_execution_history seh;