-- 创建设备维护订单详情表
CREATE TABLE IF NOT EXISTS maintenance_order_details (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL UNIQUE COMMENT '关联订单ID（外键）',
    cluster_id BIGINT NOT NULL COMMENT '关联集群ID',
    maintenance_start_time DATETIME NULL COMMENT '维护开始时间',
    maintenance_end_time DATETIME NULL COMMENT '维护结束时间',
    external_ticket_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '外部工单号',
    maintenance_type VARCHAR(50) NOT NULL DEFAULT 'general' COMMENT '维护类型（cordon/uncordon/general）',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium' COMMENT '优先级（high/medium/low）',
    reason TEXT NOT NULL DEFAULT '' COMMENT '维护原因',
    comments TEXT NOT NULL DEFAULT '' COMMENT '附加说明',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    deleted_at TIMESTAMP NULL COMMENT '删除时间',
    
    -- 外键约束
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    
    -- 索引
    INDEX idx_maintenance_order_details_order_id (order_id),
    INDEX idx_maintenance_order_details_cluster_id (cluster_id),
    INDEX idx_maintenance_order_details_external_ticket_id (external_ticket_id),
    INDEX idx_maintenance_order_details_maintenance_type (maintenance_type),
    INDEX idx_maintenance_order_details_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='设备维护订单详情表';

-- 从弹性伸缩订单详情表中移除维护相关字段（如果存在）
-- 注意：在生产环境中执行前，请先备份数据
-- ALTER TABLE elastic_scaling_order_details DROP COLUMN IF EXISTS maintenance_start_time;
-- ALTER TABLE elastic_scaling_order_details DROP COLUMN IF EXISTS maintenance_end_time;
-- ALTER TABLE elastic_scaling_order_details DROP COLUMN IF EXISTS external_ticket_id;

-- 修改order_device表结构，将order_detail_id改为order_id（如果需要）
-- ALTER TABLE order_device CHANGE COLUMN order_detail_id order_id BIGINT;
-- ALTER TABLE order_device ADD INDEX idx_order_device_order_id (order_id);