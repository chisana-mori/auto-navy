-- 场景1：入池订单生成测试数据
-- 测试目标：验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单

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

-- 重置自增ID
DELETE FROM sqlite_sequence WHERE name IN (
    'strategy_execution_history', 'order_device', 'elastic_scaling_order_details',
    'orders', 'resource_snapshots', 'strategy_cluster_associations',
    'elastic_scaling_strategies', 'query_templates', 'devices', 'k8s_clusters'
);

-- 创建集群
INSERT INTO k8s_clusters (id, cluster_name, created_at, updated_at) VALUES 
(1, 'production-cluster', datetime('now'), datetime('now'));

-- 创建设备（可用于入池）
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
(1, 'DEV001', '192.168.1.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(2, 'DEV002', '192.168.1.11', 'x86_64', 16.0, 32.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now')),
(3, 'DEV003', '192.168.1.12', 'arm64', 12.0, 24.0, 'in_stock', 'worker', 'production-cluster', 1, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找可用设备）
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(1, 'Find Available Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（入池策略）
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

-- 验证数据插入
SELECT 'Clusters:' as table_name, count(*) as count FROM k8s_clusters
UNION ALL
SELECT 'Devices:', count(*) FROM devices
UNION ALL
SELECT 'Query Templates:', count(*) FROM query_templates
UNION ALL
SELECT 'Strategies:', count(*) FROM elastic_scaling_strategies
UNION ALL
SELECT 'Strategy Associations:', count(*) FROM strategy_cluster_associations
UNION ALL
SELECT 'Resource Snapshots:', count(*) FROM resource_snapshots;

-- 显示策略详情
SELECT 
    id, name, threshold_trigger_action, 
    cpu_threshold_value, cpu_threshold_type,
    duration_minutes, status
FROM elastic_scaling_strategies;

-- 显示资源快照趋势
SELECT 
    date(created_at) as snapshot_date,
    max_cpu_usage_ratio,
    max_memory_usage_ratio,
    CASE 
        WHEN max_cpu_usage_ratio > 80 THEN 'BREACH'
        ELSE 'NORMAL'
    END as threshold_status
FROM resource_snapshots 
ORDER BY created_at;
