-- 场景2：退池订单生成测试数据
-- 测试目标：验证当内存分配率连续2天低于20%时，系统能够正确生成退池订单

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
(2, 'staging-cluster', datetime('now'), datetime('now'));

-- 创建在池设备（可用于退池）
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
(4, 'DEV004', '192.168.2.10', 'x86_64', 8.0, 16.0, 'in_pool', 'worker', 'staging-cluster', 2, 0, 0, datetime('now'), datetime('now')),
(5, 'DEV005', '192.168.2.11', 'x86_64', 16.0, 32.0, 'in_pool', 'worker', 'staging-cluster', 2, 0, 0, datetime('now'), datetime('now')),
(6, 'DEV006', '192.168.2.12', 'arm64', 12.0, 24.0, 'in_pool', 'worker', 'staging-cluster', 2, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找在池设备）
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(2, 'Find Pool Devices', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_pool"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（退池策略）
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
-- 3天前：内存分配率 25%（不满足）
(2, 'total', 'total', 45.0, 35.0, 450.0, 1000.0, 2500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：内存分配率 15%（满足）
(2, 'total', 'total', 45.0, 35.0, 450.0, 1000.0, 1500.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：内存分配率 18%（满足）
(2, 'total', 'total', 40.0, 32.0, 400.0, 1000.0, 1800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));

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
    memory_threshold_value, memory_threshold_type,
    duration_minutes, status
FROM elastic_scaling_strategies;

-- 显示资源快照趋势（计算内存分配率）
SELECT 
    date(created_at) as snapshot_date,
    max_cpu_usage_ratio,
    max_memory_usage_ratio,
    mem_request,
    memory_capacity,
    ROUND((mem_request * 100.0 / memory_capacity), 2) as memory_allocation_ratio,
    CASE 
        WHEN (mem_request * 100.0 / memory_capacity) < 20 THEN 'BREACH'
        ELSE 'NORMAL'
    END as threshold_status
FROM resource_snapshots 
ORDER BY created_at;

-- 显示在池设备信息
SELECT 
    ci_code, ip, arch_type, 
    cpu, memory, status, cluster
FROM devices 
WHERE status = 'in_pool';
