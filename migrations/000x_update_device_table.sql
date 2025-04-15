-- 更新设备表结构

-- 重命名旧表作为备份
ALTER TABLE devices RENAME TO devices_backup;

-- 创建新表
CREATE TABLE devices (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    ci_code VARCHAR(255) NULL COMMENT '设备编码',
    ip VARCHAR(50) NULL COMMENT 'IP地址',
    arch_type VARCHAR(50) NULL COMMENT 'CPU架构',
    idc VARCHAR(100) NULL COMMENT 'IDC',
    room VARCHAR(100) NULL COMMENT '机房',
    cabinet VARCHAR(100) NULL COMMENT '所属机柜',
    cabinet_no VARCHAR(100) NULL COMMENT '机柜编号',
    infra_type VARCHAR(100) NULL COMMENT '网络类型',
    is_localization BOOLEAN DEFAULT FALSE COMMENT '是否国产化',
    net_zone VARCHAR(100) NULL COMMENT '网络区域',
    `group` VARCHAR(100) NULL COMMENT '机器类别',
    appid VARCHAR(100) NULL COMMENT 'APPID',
    os_create_time VARCHAR(100) NULL COMMENT '操作系统创建时间',
    cpu FLOAT NULL COMMENT 'CPU数量',
    memory FLOAT NULL COMMENT '内存大小',
    model VARCHAR(100) NULL COMMENT '型号',
    kvm_ip VARCHAR(50) NULL COMMENT 'KVM IP',
    os VARCHAR(100) NULL COMMENT '操作系统',
    company VARCHAR(100) NULL COMMENT '厂商',
    os_name VARCHAR(100) NULL COMMENT '操作系统名称',
    os_issue VARCHAR(100) NULL COMMENT '操作系统版本',
    os_kernel VARCHAR(100) NULL COMMENT '操作系统内核',
    status VARCHAR(50) NULL COMMENT '状态',
    role VARCHAR(100) NULL COMMENT '角色',
    cluster VARCHAR(255) NULL COMMENT '所属集群',
    cluster_id INT NULL COMMENT '集群ID',
    INDEX idx_ci_code (ci_code),
    INDEX idx_ip (ip)
);

-- 迁移数据（根据字段映射关系）
INSERT INTO devices (
    id, created_at, updated_at, deleted_at,
    ci_code, ip, arch_type, idc, room, cabinet,
    net_zone, role, cluster
)
SELECT
    id, created_at, updated_at, deleted_at,
    device_id, ip, arch, idc, room, cabinet,
    network, role, cluster
FROM devices_backup;

-- 更新其他字段的默认值
UPDATE devices SET
    cabinet_no = '',
    infra_type = '',
    is_localization = FALSE,
    `group` = '',
    appid = IFNULL(app_id, ''),
    os_create_time = '',
    cpu = 0,
    memory = 0,
    model = '',
    kvm_ip = '',
    os = '',
    company = '',
    os_name = '',
    os_issue = '',
    os_kernel = '',
    status = '',
    cluster_id = 0
WHERE id > 0;

-- 如果一切正常，可以删除备份表
-- DROP TABLE devices_backup;
