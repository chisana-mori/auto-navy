-- 添加额外动态条件字段到资源池设备匹配策略表
ALTER TABLE resource_pool_device_matching_policy ADD COLUMN addition_conds TEXT DEFAULT NULL COMMENT '额外动态条件，JSON格式存储';
