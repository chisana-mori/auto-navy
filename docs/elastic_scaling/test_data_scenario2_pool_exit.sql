-- 场景2：退池订单生成测试数据
-- 测试目标：验证当CPU使用率连续3天低于30%时，系统能够正确生成退池订单

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

-- 创建设备（已在池中，可用于退池）
INSERT INTO device (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES
(1001, 'DEV001', '192.168.1.10', 'x86_64', 8.0, 16.0, 'running', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(1002, 'DEV002', '192.168.1.11', 'x86_64', 16.0, 32.0, 'running', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(1003, 'DEV003', '192.168.1.12', 'arm64', 12.0, 24.0, 'running', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找运行中的设备）
INSERT INTO query_template (id, name, groups, created_at, updated_at) VALUES
(1, 'Find Running Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"running"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（退池策略）
INSERT INTO ng_elastic_scaling_strategy (
    id, name, description, threshold_trigger_action,
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes,
    resource_types, status,
    created_by, created_at, updated_at
) VALUES (
    1, 'CPU Low Usage Scale In', 'Scale in when CPU usage is low', 'pool_exit',
    30.0, 'usage', 40.0,
    0, '', 0,
    'AND', 3, 60,
    'total', 'enabled',
    'admin', datetime('now'), datetime('now')
);

-- 创建设备匹配策略（退池）
INSERT INTO ng_resource_pool_device_matching_policy (
    id, name, description, resource_pool_type, action_type,
    query_template_id, status, addition_conds,
    created_by, updated_by, created_at, updated_at
) VALUES (
    1, 'Total Pool Exit Device Matching', 'Device matching policy for total pool exit', 'total', 'pool_exit',
    1, 'enabled', '',
    'admin', 'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO ng_strategy_cluster_association (strategy_id, cluster_id) VALUES
(1, 1);

-- 创建资源快照数据（连续3天CPU使用率低于30%）
INSERT INTO k8s_cluster_resource_snapshot (
    cluster_id, resource_type, resource_pool,
    max_cpu, max_memory,
    cpu_request, cpu_capacity, mem_request, mem_capacity,
    created_at, updated_at
) VALUES
-- 3天前：CPU 25%
(1, 'total', 'total', 25.0, 45.0, 250.0, 1000.0, 4500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：CPU 20%
(1, 'total', 'total', 20.0, 40.0, 200.0, 1000.0, 4000.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：CPU 28%
(1, 'total', 'total', 28.0, 42.0, 280.0, 1000.0, 4200.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days')),
-- 1天前：CPU 28%
(1, 'total', 'total', 10, 86.0, 280.0, 1000.0, 6600.0, 10000.0, datetime('now'), datetime('now'));
-- 模拟系统自动生成的订单数据
-- 创建基础订单
INSERT INTO ng_orders (id, order_number, name, description, type, status, created_by, created_at, updated_at) VALUES
(1, 'ESO20241201123457', '策略触发-退池-production-cluster-total', '策略 ''CPU Low Usage Scale In'' 触发退池操作。集群：production-cluster，资源类型：total，涉及设备：2台。', 'elastic_scaling', 'pending', 'system/auto', datetime('now'), datetime('now'));

-- 创建弹性伸缩订单详情
INSERT INTO ng_elastic_scaling_order_details (id, order_id, cluster_id, strategy_id, action_type, resource_pool_type, device_count, strategy_triggered_value, strategy_threshold_value, created_at, updated_at) VALUES
(1, 1, 1, 1, 'pool_exit', 'total', 2, 'CPU使用率: 28.0%', 'CPU阈值: 30.0%', datetime('now'), datetime('now'));

-- 创建订单设备关联（选择前2台运行中设备）
INSERT INTO ng_order_device (order_id, device_id, status, created_at, updated_at) VALUES
(1, 1001, 'pending', datetime('now'), datetime('now')),
(1, 1002, 'pending', datetime('now'), datetime('now'));

-- 创建策略执行历史记录
INSERT INTO ng_strategy_execution_history (id, strategy_id, cluster_id, resource_type, execution_time, triggered_value, threshold_value, result, order_id, reason, created_at, updated_at) VALUES
(1, 1, 1, 'total', datetime('now'), 'CPU使用率: 28.0%', 'CPU阈值: 30.0%', 'order_created', 1, '连续3天CPU使用率低于30%，成功生成退池订单', datetime('now'), datetime('now'));

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

-- 显示资源快照趋势
SELECT
    date(created_at) as snapshot_date,
    max_cpu,
    max_memory,
    CASE
        WHEN max_cpu < 30 THEN 'BREACH'
        ELSE 'NORMAL'
    END as threshold_status
FROM k8s_cluster_resource_snapshot
ORDER BY created_at;

-- 显示生成的订单信息
SELECT
    o.id as order_id,
    o.order_number,
    o.name as order_name,
    o.description,
    o.type,
    o.status,
    o.created_by,
    esd.action_type,
    esd.device_count,
    esd.strategy_triggered_value,
    esd.strategy_threshold_value
FROM ng_orders o
JOIN ng_elastic_scaling_order_details esd ON o.id = esd.order_id;

-- 显示订单关联的设备信息
SELECT
    od.order_id,
    od.device_id,
    od.status as device_status,
    d.ci_code,
    d.ip,
    d.arch_type,
    d.cpu,
    d.memory,
    d.status as device_current_status
FROM ng_order_device od
JOIN device d ON od.device_id = d.id
ORDER BY od.order_id, od.device_id;

-- 显示策略执行历史
SELECT
    seh.id,
    seh.strategy_id,
    seh.cluster_id,
    seh.resource_type,
    seh.triggered_value,
    seh.threshold_value,
    seh.result,
    seh.order_id,
    seh.reason
FROM ng_strategy_execution_history seh;