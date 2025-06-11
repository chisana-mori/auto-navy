-- 弹性伸缩策略表
CREATE TABLE `ng_elastic_scaling_strategy` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `name` varchar(128) NOT NULL COMMENT '策略名称',
    `description` varchar(500) DEFAULT NULL COMMENT '策略描述',
    `threshold_trigger_action` varchar(20) NOT NULL COMMENT '阈值触发动作(pool_entry/pool_exit)',
    `cpu_threshold_value` double DEFAULT 0 COMMENT 'CPU阈值',
    `cpu_threshold_type` varchar(20) DEFAULT NULL COMMENT 'CPU阈值类型(usage/allocated)',
    `cpu_target_value` double DEFAULT 0 COMMENT 'CPU目标值',
    `memory_threshold_value` double DEFAULT 0 COMMENT '内存阈值',
    `memory_threshold_type` varchar(20) DEFAULT NULL COMMENT '内存阈值类型(usage/allocated)',
    `memory_target_value` double DEFAULT 0 COMMENT '内存目标值',
    `condition_logic` varchar(10) DEFAULT 'OR' COMMENT '条件逻辑(AND/OR)',
    `duration_minutes` int(11) NOT NULL COMMENT '持续时间(分钟)',
    `cooldown_minutes` int(11) NOT NULL COMMENT '冷却时间(分钟)',
    `resource_types` varchar(255) DEFAULT NULL COMMENT '资源类型列表(逗号分隔)',
    `status` varchar(20) NOT NULL COMMENT '状态(enabled/disabled)',
    `created_by` varchar(50) NOT NULL COMMENT '创建人',
    PRIMARY KEY (`id`),
    KEY `idx_name` (`name`),
    KEY `idx_status` (`status`),
    KEY `idx_created_by` (`created_by`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='弹性伸缩策略表';

-- 策略集群关联表
CREATE TABLE `ng_strategy_cluster_association` (
    `strategy_id` int(11) NOT NULL COMMENT '策略ID',
    `cluster_id` int(11) NOT NULL COMMENT '集群ID',
    PRIMARY KEY (`strategy_id`, `cluster_id`),
    KEY `idx_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='策略集群关联表';

-- 基础订单表
CREATE TABLE `ng_orders` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_number` varchar(50) NOT NULL UNIQUE COMMENT '唯一订单号',
    `name` varchar(255) DEFAULT NULL COMMENT '订单名称',
    `description` text COMMENT '订单描述',
    `type` varchar(50) DEFAULT NULL COMMENT '订单类型',
    `status` varchar(50) DEFAULT NULL COMMENT '订单状态',
    `executor` varchar(100) DEFAULT NULL COMMENT '执行人',
    `execution_time` datetime DEFAULT NULL COMMENT '执行时间',
    `created_by` varchar(100) DEFAULT NULL COMMENT '创建人',
    `completion_time` datetime DEFAULT NULL COMMENT '完成时间',
    `failure_reason` text COMMENT '失败原因',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_number` (`order_number`),
    KEY `idx_type` (`type`),
    KEY `idx_status` (`status`),
    KEY `idx_created_by` (`created_by`),
    KEY `idx_executor` (`executor`),
    KEY `idx_execution_time` (`execution_time`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='基础订单表';

-- 弹性伸缩订单详情表
CREATE TABLE `ng_elastic_scaling_order_details` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_id` bigint(20) NOT NULL UNIQUE COMMENT '关联订单ID',
    `cluster_id` bigint(20) DEFAULT NULL COMMENT '关联集群ID',
    `strategy_id` bigint(20) DEFAULT NULL COMMENT '关联策略ID',
    `action_type` varchar(50) DEFAULT NULL COMMENT '订单操作类型(入池/退池)',
    `resource_pool_type` varchar(50) DEFAULT NULL COMMENT '资源池类型',
    `device_count` int(11) DEFAULT NULL COMMENT '请求的设备数量',
    `strategy_triggered_value` varchar(255) DEFAULT NULL COMMENT '策略触发时的具体指标值',
    `strategy_threshold_value` varchar(255) DEFAULT NULL COMMENT '策略触发时的阈值设定',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_id` (`order_id`),
    KEY `idx_cluster_id` (`cluster_id`),
    KEY `idx_strategy_id` (`strategy_id`),
    KEY `idx_action_type` (`action_type`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_elastic_scaling_order_id` FOREIGN KEY (`order_id`) REFERENCES `ng_orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='弹性伸缩订单详情表';

-- 通用订单详情表
CREATE TABLE `general_order_details` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_id` bigint(20) NOT NULL UNIQUE COMMENT '关联订单ID',
    `summary` varchar(255) DEFAULT NULL COMMENT '订单摘要',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_id` (`order_id`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_general_order_id` FOREIGN KEY (`order_id`) REFERENCES `ng_orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通用订单详情表';

-- 设备维护订单详情表
CREATE TABLE `maintenance_order_details` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_id` bigint(20) NOT NULL UNIQUE COMMENT '关联订单ID',
    `cluster_id` bigint(20) DEFAULT NULL COMMENT '关联集群ID',
    `maintenance_start_time` datetime DEFAULT NULL COMMENT '维护开始时间',
    `maintenance_end_time` datetime DEFAULT NULL COMMENT '维护结束时间',
    `external_ticket_id` varchar(100) DEFAULT NULL COMMENT '外部工单号',
    `maintenance_type` varchar(50) DEFAULT NULL COMMENT '维护类型(cordon/uncordon/general)',
    `priority` varchar(20) DEFAULT NULL COMMENT '优先级(high/medium/low)',
    `reason` text COMMENT '维护原因',
    `comments` text COMMENT '附加说明',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_id` (`order_id`),
    KEY `idx_cluster_id` (`cluster_id`),
    KEY `idx_maintenance_type` (`maintenance_type`),
    KEY `idx_priority` (`priority`),
    KEY `idx_external_ticket_id` (`external_ticket_id`),
    KEY `idx_maintenance_start_time` (`maintenance_start_time`),
    KEY `idx_deleted_at` (`deleted_at`),
    CONSTRAINT `fk_maintenance_order_id` FOREIGN KEY (`order_id`) REFERENCES `ng_orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='设备维护订单详情表';

-- 资源池设备匹配策略表
CREATE TABLE `ng_resource_pool_device_matching_policy` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `name` varchar(255) NOT NULL COMMENT '策略名称',
    `description` text COMMENT '策略描述',
    `resource_pool_type` varchar(255) NOT NULL COMMENT '资源池类型',
    `action_type` varchar(50) NOT NULL COMMENT '动作类型(pool_entry/pool_exit)',
    `query_template_id` int(10) unsigned NOT NULL COMMENT '关联的查询模板ID',
    `status` varchar(50) NOT NULL DEFAULT 'enabled' COMMENT '状态(enabled/disabled)',
    `addition_conds` text COMMENT '额外动态条件(JSON格式)',
    `created_by` varchar(255) DEFAULT NULL COMMENT '创建者',
    `updated_by` varchar(255) DEFAULT NULL COMMENT '更新者',
    PRIMARY KEY (`id`),
    KEY `idx_name` (`name`),
    KEY `idx_resource_pool_type` (`resource_pool_type`),
    KEY `idx_action_type` (`action_type`),
    KEY `idx_status` (`status`),
    KEY `idx_query_template_id` (`query_template_id`),
    KEY `idx_created_by` (`created_by`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源池设备匹配策略表';

-- 订单设备关联表
CREATE TABLE `ng_order_device` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_id` int(11) NOT NULL COMMENT '订单ID',
    `device_id` int(11) NOT NULL COMMENT '设备ID',
    `status` varchar(50) DEFAULT 'pending' COMMENT '状态',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_order_device` (`order_id`, `device_id`),
    KEY `idx_device_id` (`device_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单设备关联表';

-- 策略执行历史表
CREATE TABLE `ng_strategy_execution_history` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `strategy_id` bigint(20) DEFAULT NULL COMMENT '策略ID',
    `cluster_id` bigint(20) DEFAULT NULL COMMENT '集群ID',
    `resource_type` varchar(100) DEFAULT NULL COMMENT '资源池名称',
    `execution_time` datetime NOT NULL COMMENT '执行时间',
    `triggered_value` varchar(255) DEFAULT NULL COMMENT '触发策略时的具体指标值',
    `threshold_value` varchar(255) DEFAULT NULL COMMENT '触发策略时的阈值设定',
    `result` varchar(50) DEFAULT NULL COMMENT '执行结果',
    `order_id` bigint(20) DEFAULT NULL COMMENT '关联订单ID',
    `reason` text COMMENT '执行结果的原因',
    PRIMARY KEY (`id`),
    KEY `idx_strategy_id` (`strategy_id`),
    KEY `idx_cluster_id` (`cluster_id`),
    KEY `idx_execution_time` (`execution_time`),
    KEY `idx_result` (`result`),
    KEY `idx_order_id` (`order_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='策略执行历史表';

-- 通知日志表
CREATE TABLE `notification_log` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `order_id` bigint(20) DEFAULT NULL COMMENT '关联订单ID',
    `strategy_id` bigint(20) DEFAULT NULL COMMENT '关联策略ID',
    `notification_type` varchar(50) DEFAULT NULL COMMENT '通知类型',
    `recipient` varchar(255) DEFAULT NULL COMMENT '接收人信息',
    `content` text COMMENT '通知内容',
    `status` varchar(50) DEFAULT NULL COMMENT '发送状态',
    `send_time` datetime NOT NULL COMMENT '发送时间',
    `error_message` text COMMENT '错误信息',
    PRIMARY KEY (`id`),
    KEY `idx_order_id` (`order_id`),
    KEY `idx_strategy_id` (`strategy_id`),
    KEY `idx_notification_type` (`notification_type`),
    KEY `idx_status` (`status`),
    KEY `idx_send_time` (`send_time`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知日志表';

-- 值班表
CREATE TABLE `duty_roster` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    `user_id` varchar(100) DEFAULT NULL COMMENT '用户ID/名称',
    `duty_date` date DEFAULT NULL COMMENT '值班日期',
    `start_time` time DEFAULT NULL COMMENT '开始时间',
    `end_time` time DEFAULT NULL COMMENT '结束时间',
    `contact_info` varchar(255) DEFAULT NULL COMMENT '联系方式',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_duty_date` (`duty_date`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='值班表';