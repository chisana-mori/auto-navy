-- 弹性伸缩测试数据库表结构
-- 用于创建测试所需的基础表结构


-- 创建弹性伸缩策略表
CREATE TABLE IF NOT EXISTS ng_elastic_scaling_strategy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    threshold_trigger_action TEXT NOT NULL,
    cpu_threshold_value REAL DEFAULT 0,
    cpu_threshold_type TEXT DEFAULT '',
    cpu_target_value REAL DEFAULT 0,
    memory_threshold_value REAL DEFAULT 0,
    memory_threshold_type TEXT DEFAULT '',
    memory_target_value REAL DEFAULT 0,
    condition_logic TEXT DEFAULT 'AND',
    duration_minutes INTEGER DEFAULT 0,
    cooldown_minutes INTEGER DEFAULT 0,
    resource_types TEXT DEFAULT '',
    status TEXT DEFAULT 'enabled',
    created_by TEXT DEFAULT '',
);

-- 创建策略集群关联表
CREATE TABLE IF NOT EXISTS ng_strategy_cluster_association (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);

-- 创建订单表
CREATE TABLE IF NOT EXISTS ng_orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending',
    created_by TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);

-- 创建弹性伸缩订单详情表
CREATE TABLE IF NOT EXISTS ng_elastic_scaling_order_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    strategy_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    device_count INTEGER DEFAULT 0,
    resource_pool_type TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);

-- 创建订单设备关联表
CREATE TABLE IF NOT EXISTS ng_order_device (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);

-- 创建策略执行历史表
CREATE TABLE IF NOT EXISTS ng_strategy_execution_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_id INTEGER NOT NULL,
    execution_result TEXT NOT NULL,
    execution_details TEXT,
    order_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
);

-- 创建资源池设备匹配策略表
CREATE TABLE IF NOT EXISTS ng_resource_pool_device_matching_policy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    resource_pool_type TEXT NOT NULL,
    action_type TEXT NOT NULL,
    query_template_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'enabled',
    addition_conds TEXT,
    created_by TEXT,
    updated_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
