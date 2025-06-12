-- 场景1：入池订单生成测试数据 (MySQL Version)
-- 测试目标：验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单

-- 清理现有数据
SET FOREIGN_KEY_CHECKS = 0;
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
SET FOREIGN_KEY_CHECKS = 1;

-- 重置自增ID (MySQL - 如果需要，可以为每个表单独设置)
-- 注意: 如果表之间有外键约束，删除顺序很重要，或者暂时禁用外键检查
-- ALTER TABLE ng_strategy_execution_history AUTO_INCREMENT = 1;
-- ALTER TABLE ng_order_device AUTO_INCREMENT = 1; -- ng_order_device 通常没有自增主键
-- ALTER TABLE ng_elastic_scaling_order_details AUTO_INCREMENT = 1;
-- ALTER TABLE ng_orders AUTO_INCREMENT = 1;
-- ALTER TABLE k8s_cluster_resource_snapshot AUTO_INCREMENT = 1;
-- ALTER TABLE ng_strategy_cluster_association AUTO_INCREMENT = 1; -- ng_strategy_cluster_association 通常是关联表，没有自增主键
-- ALTER TABLE ng_resource_pool_device_matching_policy AUTO_INCREMENT = 1;
-- ALTER TABLE ng_elastic_scaling_strategy AUTO_INCREMENT = 1;
-- ALTER TABLE query_template AUTO_INCREMENT = 1;
-- ALTER TABLE device AUTO_INCREMENT = 1;
-- ALTER TABLE k8s_cluster AUTO_INCREMENT = 1;

-- 创建集群
INSERT INTO k8s_clusters (id, clustername, created_at, updated_at) VALUES
(1000, 'production-cluster', NOW(), NOW());

-- 创建设备（可用于入池）
INSERT INTO device (ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, created_at, updated_at) VALUES
( 'DEV011', '192.168.1.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'production-cluster', 1000, NOW(), NOW()),
('DEV012', '192.168.1.11', 'x86_64', 16.0, 32.0, 'in_stock', 'worker', 'production-cluster', 1000, NOW(), NOW()),
('DEV013', '192.168.1.12', 'arm64', 12.0, 24.0, 'in_stock', 'worker', 'production-cluster', 1000, NOW(), NOW());

-- 创建查询模板（查找可用设备）
INSERT INTO query_template (id, name, groups, created_at, updated_at) VALUES
(1, 'Find Available Devices', '[{\"id\":\"1\",\"blocks\":[{\"id\":\"2\",\"type\":\"device\",\"key\":\"status\",\"conditionType\":\"equal\",\"value\":\"in_stock\"}],\"operator\":\"AND\"}]', NOW(), NOW());

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
    'admin', NOW(), NOW()
);

-- 创建设备匹配策略（入池）
INSERT INTO ng_resource_pool_device_matching_policy (
    id, name, description, resource_pool_type, action_type,
    query_template_id, status, addition_conds,
    created_by, updated_by, created_at, updated_at
) VALUES (
    1, 'Total Pool Entry Device Matching', 'Device matching policy for total pool entry', 'total', 'pool_entry',
    1, 'enabled', '',
    'admin', 'admin', NOW(), NOW()
);

-- 创建策略集群关联
INSERT INTO ng_strategy_cluster_association (strategy_id, cluster_id) VALUES
(1, 1000);

-- 创建资源快照数据（连续3天CPU使用率超过80%）
INSERT INTO k8s_cluster_resource_snapshot (
    cluster_id, resource_type, resource_pool, node_count,
    max_cpu, max_memory,
    cpu_request, cpu_capacity, mem_request, mem_capacity,
    created_at, updated_at
) VALUES
-- 3天前：CPU 85%
(1000, 'total', 'total', 10, 85.0, 65.0, 850.0, 1000.0, 6500.0, 10000.0, DATE_SUB(NOW(), INTERVAL 3 DAY), DATE_SUB(NOW(), INTERVAL 3 DAY)),
-- 2天前：CPU 90%
(1000, 'total', 'total', 10, 90.0, 70.0, 900.0, 1000.0, 7000.0, 10000.0, DATE_SUB(NOW(), INTERVAL 2 DAY), DATE_SUB(NOW(), INTERVAL 2 DAY)),
-- 1天前：CPU 88%
(1000, 'total', 'total', 10, 88.0, 68.0, 880.0, 1000.0, 6800.0, 10000.0, DATE_SUB(NOW(), INTERVAL 1 DAY), DATE_SUB(NOW(), INTERVAL 1 DAY)),
-- 今天：CPU 86%
(1000, 'total', 'total', 10, 86.0, 66.0, 860.0, 1000.0, 6600.0, 10000.0, NOW(), NOW());

-- 模拟系统自动生成的订单数据
-- 创建基础订单
INSERT INTO ng_orders (id, order_number, name, description, type, status, created_by, created_at, updated_at) VALUES
(1, 'ESO20241201123456', '策略触发-入池-production-cluster-total', '策略 \'CPU High Usage Scale Out\' 触发入池操作。集群：production-cluster，资源类型：total，涉及设备：2台。', 'elastic_scaling', 'pending', 'system/auto', NOW(), NOW());

-- 创建弹性伸缩订单详情
INSERT INTO ng_elastic_scaling_order_details (id, order_id, cluster_id, strategy_id, action_type, resource_pool_type, device_count, strategy_triggered_value, strategy_threshold_value, created_at, updated_at) VALUES
(1, 1, 1000, 1, 'pool_entry', 'total', 2, 'CPU使用率: 88.0%', 'CPU阈值: 80.0%', NOW(), NOW());

-- 创建订单设备关联（选择前2台可用设备）
INSERT INTO ng_order_device (order_id, device_id, status, created_at, updated_at) VALUES
(1, 1001, 'pending', NOW(), NOW()),
(1, 1002, 'pending', NOW(), NOW());

-- 创建策略执行历史记录
INSERT INTO ng_strategy_execution_history (id, strategy_id, cluster_id, resource_type, execution_time, triggered_value, threshold_value, result, order_id, reason, created_at, updated_at) VALUES
(1, 1, 1000, 'total', NOW(), 'CPU使用率: 88.0%', 'CPU阈值: 80.0%', 'order_created', 1, '连续3天CPU使用率超过80%，成功生成入池订单', NOW(), NOW());

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
    DATE(created_at) as snapshot_date,
    max_cpu,
    max_memory,
    CASE
        WHEN max_cpu > 80 THEN 'BREACH'
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