-- 场景7：退池只能匹配部分设备测试数据
-- 测试目标：验证当满足退池条件但只能匹配到部分设备时，系统的处理逻辑

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
(7, 'partial-pool-devices-cluster', datetime('now'), datetime('now'));

-- 创建设备（只有部分在池设备，数量少于策略要求）
INSERT INTO devices (id, ci_code, ip, arch_type, cpu, memory, status, role, cluster, cluster_id, is_special, feature_count, created_at, updated_at) VALUES 
-- 在池设备（只有2台，但策略需要4台）
(24, 'DEV024', '192.168.7.10', 'x86_64', 8.0, 16.0, 'in_pool', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now')),
(25, 'DEV025', '192.168.7.11', 'x86_64', 16.0, 32.0, 'in_pool', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now')),
-- 非在池设备
(26, 'DEV026', '192.168.7.12', 'arm64', 12.0, 24.0, 'in_stock', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now')),
(27, 'DEV027', '192.168.7.13', 'x86_64', 8.0, 16.0, 'maintenance', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now')),
(28, 'DEV028', '192.168.7.14', 'x86_64', 16.0, 32.0, 'offline', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now')),
(29, 'DEV029', '192.168.7.15', 'arm64', 12.0, 24.0, 'reserved', 'worker', 'partial-pool-devices-cluster', 7, 0, 0, datetime('now'), datetime('now'));

-- 创建查询模板（查找在池设备）
INSERT INTO query_templates (id, name, groups, created_at, updated_at) VALUES 
(7, 'Find Pool Devices - Partial Match', '[{"id":"1","blocks":[{"id":"2","type":"device","key":"status","conditionType":"equal","value":"in_pool"}],"operator":"AND"}]', datetime('now'), datetime('now'));

-- 创建弹性伸缩策略（退池策略，要求4台设备但只能匹配到2台）
INSERT INTO elastic_scaling_strategies (
    id, name, description, threshold_trigger_action, 
    cpu_threshold_value, cpu_threshold_type, cpu_target_value,
    memory_threshold_value, memory_threshold_type, memory_target_value,
    condition_logic, duration_minutes, cooldown_minutes, device_count,
    node_selector, resource_types, status, exit_query_template_id,
    created_by, created_at, updated_at
) VALUES (
    7, 'Memory Low Partial Pool Devices', 'Test pool exit with partial device match', 'pool_exit',
    0, '', 0,
    20.0, 'allocated', 30.0,
    'AND', 2, 60, 4,  -- 要求4台设备
    '', 'total', 'enabled', 7,
    'admin', datetime('now'), datetime('now')
);

-- 创建策略集群关联
INSERT INTO strategy_cluster_associations (strategy_id, cluster_id, created_at, updated_at) VALUES 
(7, 7, datetime('now'), datetime('now'));

-- 创建资源快照数据（连续2天内存分配率低于20%，满足触发条件）
INSERT INTO resource_snapshots (
    cluster_id, resource_type, resource_pool,
    max_cpu_usage_ratio, max_memory_usage_ratio,
    cpu_request, cpu_capacity, mem_request, memory_capacity,
    created_at, updated_at
) VALUES 
-- 3天前：内存分配率 25%（不满足）
(7, 'total', 'total', 45.0, 35.0, 450.0, 1000.0, 2500.0, 10000.0, datetime('now', '-3 days'), datetime('now', '-3 days')),
-- 2天前：内存分配率 15%（满足）
(7, 'total', 'total', 45.0, 35.0, 450.0, 1000.0, 1500.0, 10000.0, datetime('now', '-2 days'), datetime('now', '-2 days')),
-- 1天前：内存分配率 18%（满足）
(7, 'total', 'total', 40.0, 32.0, 400.0, 1000.0, 1800.0, 10000.0, datetime('now', '-1 days'), datetime('now', '-1 days'));

-- 验证数据插入
SELECT 'Clusters:' as table_name, count(*) as count FROM k8s_clusters
UNION ALL
SELECT 'Total Devices:', count(*) FROM devices
UNION ALL
SELECT 'Pool Devices (in_pool):', count(*) FROM devices WHERE status = 'in_pool'
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

-- 显示在池设备详情
SELECT 
    ci_code, ip, arch_type, cpu, memory, status
FROM devices 
WHERE status = 'in_pool'
ORDER BY ci_code;

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

-- 分析设备匹配情况
SELECT 
    'Device Matching Analysis' as analysis_type,
    'Threshold Met: YES' as threshold_status,
    'Pool Devices: 2' as pool_devices,
    'Required Devices: 4' as required_devices,
    'Match Ratio: 50%' as match_ratio,
    'Expected Result: PARTIAL SUCCESS - ORDER WITH 2 DEVICES' as expected_result;
