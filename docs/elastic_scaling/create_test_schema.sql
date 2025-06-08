-- 弹性伸缩测试数据库表结构
-- 用于创建测试所需的基础表结构

-- 创建集群表
CREATE TABLE IF NOT EXISTS k8s_clusters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建设备表
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ci_code TEXT NOT NULL,
    ip TEXT NOT NULL,
    arch_type TEXT NOT NULL,
    cpu REAL NOT NULL,
    memory REAL NOT NULL,
    status TEXT NOT NULL,
    role TEXT NOT NULL,
    cluster TEXT NOT NULL,
    cluster_id INTEGER NOT NULL,
    is_special INTEGER DEFAULT 0,
    feature_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES k8s_clusters(id)
);

-- 创建查询模板表
CREATE TABLE IF NOT EXISTS query_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    groups TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建弹性伸缩策略表
CREATE TABLE IF NOT EXISTS elastic_scaling_strategies (
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
    device_count INTEGER DEFAULT 0,
    node_selector TEXT DEFAULT '',
    resource_types TEXT DEFAULT '',
    status TEXT DEFAULT 'enabled',
    entry_query_template_id INTEGER,
    exit_query_template_id INTEGER,
    created_by TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (entry_query_template_id) REFERENCES query_templates(id),
    FOREIGN KEY (exit_query_template_id) REFERENCES query_templates(id)
);

-- 创建策略集群关联表
CREATE TABLE IF NOT EXISTS strategy_cluster_associations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (strategy_id) REFERENCES elastic_scaling_strategies(id),
    FOREIGN KEY (cluster_id) REFERENCES k8s_clusters(id)
);

-- 创建资源快照表
CREATE TABLE IF NOT EXISTS resource_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id INTEGER NOT NULL,
    resource_type TEXT NOT NULL,
    resource_pool TEXT NOT NULL,
    max_cpu_usage_ratio REAL DEFAULT 0,
    max_memory_usage_ratio REAL DEFAULT 0,
    cpu_request REAL DEFAULT 0,
    cpu_capacity REAL DEFAULT 0,
    mem_request REAL DEFAULT 0,
    memory_capacity REAL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_id) REFERENCES k8s_clusters(id)
);

-- 创建订单表
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending',
    created_by TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建弹性伸缩订单详情表
CREATE TABLE IF NOT EXISTS elastic_scaling_order_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    strategy_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    device_count INTEGER DEFAULT 0,
    resource_pool_type TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (strategy_id) REFERENCES elastic_scaling_strategies(id),
    FOREIGN KEY (cluster_id) REFERENCES k8s_clusters(id)
);

-- 创建订单设备关联表
CREATE TABLE IF NOT EXISTS order_device (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    device_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (device_id) REFERENCES devices(id)
);

-- 创建策略执行历史表
CREATE TABLE IF NOT EXISTS strategy_execution_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    strategy_id INTEGER NOT NULL,
    execution_result TEXT NOT NULL,
    execution_details TEXT,
    order_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (strategy_id) REFERENCES elastic_scaling_strategies(id),
    FOREIGN KEY (order_id) REFERENCES orders(id)
);

-- 创建序列表（用于自增ID管理）
CREATE TABLE IF NOT EXISTS sqlite_sequence (
    name TEXT PRIMARY KEY,
    seq INTEGER
);
