-- 场景6：入池只能匹配部分设备测试数据
-- 测试目标：验证当满足入池条件但只能匹配到部分设备时，系统的处理逻辑

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
(6, 'partial-devices-cluster', datetime('now'), datetime('now'));

-- 创建设备（只有部分可用设备，数量少于策略要求）
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
-- 可用设备（只有2台，但策略需要5台）
(18, 'DEV018', '192.168.6.10', 'x86_64', 8.0, 16.0, 'in_stock', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now')),
(19, 'DEV019', '192.168.6.11', 'x86_64', 16.0, 32.0, 'in_stock', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now')),
-- 不可用设备
(20, 'DEV020', '192.168.6.12', 'arm64', 12.0, 24.0, 'in_pool', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now')),
(21, 'DEV021', '192.168.6.13', 'x86_64', 8.0, 16.0, 'maintenance', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now')),
(22, 'DEV022', '192.168.6.14', 'x86_64', 16.0, 32.0, 'offline', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now')),
(23, 'DEV023', '192.168.6.15', 'arm64', 12.0, 24.0, 'reserved', 'worker', 'partial-devices-cluster', 6, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找可用设备）
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(6, 'Find Available Devices - Partial Match', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_stock"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（入池策略，要求5台设备但只能匹配到2台）
INSERT INTO elastic_scaling_strategies (
    id, name, description, threshold_trigger_action, 
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes, device_count,
    node_selector, resource_types, status, entry_query_template_id,
    created_by, created_at, updated_at
) VALUES (
    6, 'CPU High Partial Devices', 'Test pool entry with partial device match', 'pool_entry',
    80.0, 'usage', 70.0,
    0, '', 0,
    'AND', 3, 60, 5,  -- 要求5台设备
    '', 'total', 'enabled', 6,
    'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO strategy_cluster_associations (strategy_id, cluster_id, created_at, updated_at) VALUES 
(6, 6, datetime('now'), datetime('now'));

-- 创建资源快照数据（连续3天CPU使用率超过80%，满足触发条件）
INSERT INTO resource_snapshots (
    cluster_id, resource_type, resource_pool,
    max_cpu_usage_ratio, max_memory_usage_ratio,
    cpu_request, cpu_capacity, mem_request, memory_capacity,
    created_at, updated_at
) VALUES 
-- 3天前：CPU 85%
(6, 'total', 'total', 85.0, 65.0, 850.0, 1000.0, 6500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：CPU 90%
(6, 'total', 'total', 90.0, 70.0, 900.0, 1000.0, 7000.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：CPU 88%
(6, 'total', 'total', 88.0, 68.0, 880.0, 1000.0, 6800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));

-- 验证数据插入
SELECT 'Clusters:' as table_name, count(*) as count FROM k8s_clusters
UNION ALL
SELECT 'Total Devices:', count(*) FROM devices
UNION ALL
SELECT 'Available Devices (in_stock):', count(*) FROM devices WHERE status = 'in_stock'
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
    duration_minutes, device_count, status
FROM elastic_scaling_strategies;

-- 显示设备状态分布
SELECT 
    status,
    count(*) as device_count,
    GROUP_CONCAT(ci_code, ', ') as devices
FROM devices 
GROUP BY status
ORDER BY status;

-- 显示可用设备详情
SELECT 
    ci_code, ip, arch_type, cpu, memory, status
FROM devices 
WHERE status = 'in_stock'
ORDER BY ci_code;

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

-- 分析设备匹配情况
SELECT 
    'Device Matching Analysis' as analysis_type,
    'Threshold Met: YES' as threshold_status,
    'Available Devices: 2' as available_devices,
    'Required Devices: 5' as required_devices,
    'Match Ratio: 40%' as match_ratio,
    'Expected Result: PARTIAL SUCCESS - ORDER WITH 2 DEVICES' as expected_result;
