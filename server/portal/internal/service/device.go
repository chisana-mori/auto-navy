package service

import (
	"bytes"
	"context"
	"fmt"
	"navy-ng/models/portal"
	"strings"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm"
)

// Constants for DeviceService
const (
	ErrDeviceNotFoundMsg = "device with ID %d not found"
)

// DeviceService 设备服务
type DeviceService struct {
	db *gorm.DB
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
}

// buildDeviceBaseQuery 构建设备查询的基础查询，包含与 k8s_node 等表的关联
func (s *DeviceService) buildDeviceBaseQuery(ctx context.Context) *gorm.DB {
	// 获取今天的开始和结束时间
	currentTime := time.Now()
	todayStart := now.New(currentTime).BeginningOfDay()
	todayEnd := now.New(currentTime).EndOfDay()

	// 构建查询，添加 isSpecial 字段用于标识特殊设备
	query := s.db.WithContext(ctx).Model(&portal.Device{}).
		Select("device.*, " +
			// 添加 isSpecial 字段，当满足以下条件之一时为 true：
			// 1. 机器用途不为空
			// 2. 可以关联到 label_feature
			// 3. 可以关联到 taint_feature
			// 4. 当 device.group 和 cluster 为空，但可以关联到 device_app 时
			"CASE WHEN device.`group` != '' OR lf.id IS NOT NULL OR tf.id IS NOT NULL OR " +
			"     ((device.`group` = '' OR device.`group` IS NULL) AND (device.cluster = '' OR device.cluster IS NULL) AND da.name IS NOT NULL AND da.name != '') " +
			"THEN TRUE ELSE FALSE END AS is_special, " +
			// 添加特性计数字段，用于前端显示
			"(CASE WHEN device.`group` != '' THEN 1 ELSE 0 END + " +
			" CASE WHEN lf.id IS NOT NULL THEN 1 ELSE 0 END + " +
			" CASE WHEN tf.id IS NOT NULL THEN 1 ELSE 0 END + " +
			" CASE WHEN (device.`group` = '' OR device.`group` IS NULL) AND (device.cluster = '' OR device.cluster IS NULL) AND da.name IS NOT NULL AND da.name != '' THEN 1 ELSE 0 END" +
			") AS feature_count, " +
			// 当设备的 cluster 为空时，将 device_app.name 添加到 group 字段中
			"CASE WHEN device.cluster = '' OR device.cluster IS NULL THEN " +
			"  CASE WHEN device.`group` = '' OR device.`group` IS NULL THEN da.name " +
			"  ELSE CONCAT(device.`group`, ';', da.name) END " +
			"ELSE device.`group` END AS `group`")

	// 默认关联 k8s_node 表，并添加 status != Offline 筛选条件
	query = query.Joins("LEFT JOIN k8s_node ON LOWER(device.ci_code) = LOWER(k8s_node.nodename) AND (k8s_node.status != 'Offline' OR k8s_node.status IS NULL)")

	// 关联 k8s_node_label 表和 label_feature 表，用于判断是否为特殊设备
	query = query.Joins("LEFT JOIN k8s_node_label ON k8s_node.id = k8s_node_label.node_id AND k8s_node_label.created_at BETWEEN ? AND ?", todayStart, todayEnd)
	query = query.Joins("LEFT JOIN label_feature lf ON k8s_node_label.key = lf.key")

	// 关联 k8s_node_taint 表和 taint_feature 表，用于判断是否为特殊设备
	query = query.Joins("LEFT JOIN k8s_node_taint ON k8s_node.id = k8s_node_taint.node_id AND k8s_node_taint.created_at BETWEEN ? AND ?", todayStart, todayEnd)
	query = query.Joins("LEFT JOIN taint_feature tf ON k8s_node_taint.key = tf.key")

	// 关联 device_app 表，用于获取设备的来源信息
	query = query.Joins("LEFT JOIN device_app da ON device.appid = da.app_id")

	// 使用 GROUP BY 确保每个设备只返回一行
	query = query.Group("device.id")

	return query
}

// processMultilineKeyword 处理多行查询关键字
func processMultilineKeyword(keyword string) []string {
	// 检查是否包含常见的分隔符
	hasNewline := strings.Contains(keyword, "\n")
	hasComma := strings.Contains(keyword, ",")
	hasOr := strings.Contains(strings.ToLower(keyword), " or ")

	// 如果包含分隔符，则分割关键字
	if hasNewline || hasComma || hasOr {
		var lines []string

		// 首先按换行符分割
		if hasNewline {
			lines = strings.Split(keyword, "\n")
		} else if hasComma {
			// 如果没有换行符但有逗号，则按逗号分割
			lines = strings.Split(keyword, ",")
		} else if hasOr {
			// 如果没有换行符和逗号，但有 OR，则按 OR 分割
			lines = strings.Split(strings.ToLower(keyword), " or ")
		}

		// 过滤空行并去除空格
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}

		if len(result) > 0 {
			return result
		}
	}

	// 如果没有有效的多行关键字，返回原始关键字作为单个元素的切片
	return []string{keyword}
}

// ListDevices 获取设备列表

func (s *DeviceService) ListDevices(ctx context.Context, query *DeviceQuery) (*DeviceListResponse, error) {
	var models []portal.Device
	var total int64

	// 使用基础查询，包含与 k8s_node 等表的关联
	db := s.buildDeviceBaseQuery(ctx)

	// 应用关键字搜索
	if query.Keyword != emptyString {
		// 处理多行查询
		keywords := processMultilineKeyword(query.Keyword)

		// 检测是否是IP地址列表查询
		isIPList := true
		for _, keyword := range keywords {
			// 简单检查是否是IP地址格式（可以根据需要使用更严格的正则表达式）
			if !strings.Contains(keyword, ".") || strings.Contains(keyword, " ") {
				isIPList = false
				break
			}
		}

		// 如果是IP地址列表查询，则只在IP字段中查询
		if isIPList && len(keywords) > 1 {
			// 构建IP查询条件
			for i, ip := range keywords {
				ip = "%" + ip + "%"

				// 第一个IP使用 Where，后续IP使用 Or
				if i == 0 {
					db = db.Where("device.ip LIKE ?", ip)
				} else {
					db = db.Or("device.ip LIKE ?", ip)
				}
			}
		} else {
			// 常规关键字查询
			for i, keyword := range keywords {
				keyword = "%" + keyword + "%"

				// 第一个关键字使用 Where，后续关键字使用 Or
				if i == 0 {
					db = db.Where(
						"device.ci_code LIKE ? OR device.ip LIKE ? OR device.arch_type LIKE ? OR device.cluster LIKE ? OR "+
							"device.role LIKE ? OR device.idc LIKE ? OR device.room LIKE ? OR device.cabinet LIKE ? OR "+
							"device.cabinet_no LIKE ? OR device.infra_type LIKE ? OR device.net_zone LIKE ? OR device.appid LIKE ? OR device.`group` LIKE ?",
						keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
					)
				} else {
					db = db.Or(
						"device.ci_code LIKE ? OR device.ip LIKE ? OR device.arch_type LIKE ? OR device.cluster LIKE ? OR "+
							"device.role LIKE ? OR device.idc LIKE ? OR device.room LIKE ? OR device.cabinet LIKE ? OR "+
							"device.cabinet_no LIKE ? OR device.infra_type LIKE ? OR device.net_zone LIKE ? OR device.appid LIKE ? OR device.`group` LIKE ?",
						keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
					)
				}
			}
		}
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count devices: %w", err)
	}

	// 分页
	page := query.Page
	if page <= 0 {
		page = 1
	}

	size := query.Size
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	// 转换为响应格式
	responses := make([]DeviceResponse, len(models))
	for i, model := range models {
		responses[i] = DeviceResponse{
			ID:             model.ID,
			CICode:         model.CICode,
			IP:             model.IP,
			ArchType:       model.ArchType,
			IDC:            model.IDC,
			Room:           model.Room,
			Cabinet:        model.Cabinet,
			CabinetNO:      model.CabinetNO,
			InfraType:      model.InfraType,
			IsLocalization: model.IsLocalization,
			NetZone:        model.NetZone,
			Group:          model.Group,
			AppID:          model.AppID,
			OsCreateTime:   model.OsCreateTime,
			CPU:            model.CPU,
			Memory:         model.Memory,
			Model:          model.Model,
			KvmIP:          model.KvmIP,
			OS:             model.OS,
			Company:        model.Company,
			OSName:         model.OSName,
			OSIssue:        model.OSIssue,
			OSKernel:       model.OSKernel,
			Status:         model.Status,
			Role:           model.Role,
			Cluster:        model.Cluster,
			ClusterID:      model.ClusterID,
			AcceptanceTime: model.AcceptanceTime,
			DiskCount:      model.DiskCount,
			DiskDetail:     model.DiskDetail,
			NetworkSpeed:   model.NetworkSpeed,
			IsSpecial:      model.IsSpecial,
			FeatureCount:   model.FeatureCount,
			CreatedAt:      time.Time(model.CreatedAt),
			UpdatedAt:      time.Time(model.UpdatedAt),
		}
	}

	return &DeviceListResponse{
		List:  responses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetDevice 获取设备详情
// 简化实现，不再使用多表连接，只获取设备基本信息
func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*DeviceResponse, error) {
	var model portal.Device
	// 直接查询设备表，不使用多表连接
	db := s.db.WithContext(ctx).Model(&portal.Device{}).Where("id = ?", id)
	err := db.First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// 构建响应
	return &DeviceResponse{
		ID:             model.ID,
		CICode:         model.CICode,
		IP:             model.IP,
		ArchType:       model.ArchType,
		IDC:            model.IDC,
		Room:           model.Room,
		Cabinet:        model.Cabinet,
		CabinetNO:      model.CabinetNO,
		InfraType:      model.InfraType,
		IsLocalization: model.IsLocalization,
		NetZone:        model.NetZone,
		Group:          model.Group,
		AppID:          model.AppID,
		OsCreateTime:   model.OsCreateTime,
		CPU:            model.CPU,
		Memory:         model.Memory,
		Model:          model.Model,
		KvmIP:          model.KvmIP,
		OS:             model.OS,
		Company:        model.Company,
		OSName:         model.OSName,
		OSIssue:        model.OSIssue,
		OSKernel:       model.OSKernel,
		Status:         model.Status,
		Role:           model.Role,
		Cluster:        model.Cluster,
		ClusterID:      model.ClusterID,
		AcceptanceTime: model.AcceptanceTime,
		DiskCount:      model.DiskCount,
		DiskDetail:     model.DiskDetail,
		NetworkSpeed:   model.NetworkSpeed,
		// 设备详情页不需要特殊标记，设置为默认值
		IsSpecial:    false,
		FeatureCount: 0,
		CreatedAt:    time.Time(model.CreatedAt),
		UpdatedAt:    time.Time(model.UpdatedAt),
	}, nil
}

// UpdateDeviceRole 更新设备角色

func (s *DeviceService) UpdateDeviceRole(ctx context.Context, id int64, request *DeviceRoleUpdateRequest) error {
	// 查找设备
	var device portal.Device
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return fmt.Errorf("failed to find device: %w", result.Error)
	}

	// 更新角色字段
	result = s.db.WithContext(ctx).Model(&device).Update("role", request.Role)
	if result.Error != nil {
		return fmt.Errorf("failed to update device role: %w", result.Error)
	}

	return nil
}

// UpdateDeviceGroup 更新设备用途

func (s *DeviceService) UpdateDeviceGroup(ctx context.Context, id int64, request *DeviceGroupUpdateRequest) error {
	// 查找设备
	var device portal.Device
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return fmt.Errorf("failed to find device: %w", result.Error)
	}

	// 更新用途字段
	result = s.db.WithContext(ctx).Model(&device).Update("`group`", request.Group)
	if result.Error != nil {
		return fmt.Errorf("failed to update device group: %w", result.Error)
	}

	return nil
}

// ExportDevices 导出所有设备信息为Excel

func (s *DeviceService) ExportDevices(ctx context.Context) ([]byte, error) {
	var devices []portal.Device

	// 使用基础查询，包含与 k8s_node 等表的关联
	db := s.buildDeviceBaseQuery(ctx)

	// 获取所有设备信息
	if err := db.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to get devices for export: %w", err)
	}

	// 创建CSV格式的数据
	buffer := &bytes.Buffer{}

	// 写入表头
	headers := []string{
		"设备编码", "IP地址", "CPU架构", "IDC", "机房",
		"机柜", "机柜编号", "网络类型", "是否国产化", "网络区域",
		"机器类别", "APPID", "操作系统创建时间", "CPU", "内存",
		"型号", "KVM IP", "操作系统", "厂商", "操作系统名称",
		"操作系统版本", "操作系统内核", "状态", "角色", "集群",
		"集群ID", "创建时间", "更新时间",
	}

	// 写入CSV表头
	for i, header := range headers {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(header)
	}
	buffer.WriteString("\n")

	// 写入数据行
	for _, device := range devices {
		// 设备编码
		buffer.WriteString(device.CICode)
		buffer.WriteString(",")

		// IP地址
		buffer.WriteString(device.IP)
		buffer.WriteString(",")

		// CPU架构
		buffer.WriteString(device.ArchType)
		buffer.WriteString(",")

		// IDC
		buffer.WriteString(device.IDC)
		buffer.WriteString(",")

		// 机房
		buffer.WriteString(device.Room)
		buffer.WriteString(",")

		// 机柜
		buffer.WriteString(device.Cabinet)
		buffer.WriteString(",")

		// 机柜编号
		buffer.WriteString(device.CabinetNO)
		buffer.WriteString(",")

		// 网络类型
		buffer.WriteString(device.InfraType)
		buffer.WriteString(",")

		// 是否国产化
		if device.IsLocalization {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
		buffer.WriteString(",")

		// 网络区域
		buffer.WriteString(device.NetZone)
		buffer.WriteString(",")

		// 机器类别
		buffer.WriteString(device.Group)
		buffer.WriteString(",")

		// APPID
		buffer.WriteString(device.AppID)
		buffer.WriteString(",")

		// 操作系统创建时间
		buffer.WriteString(device.OsCreateTime)
		buffer.WriteString(",")

		// CPU
		buffer.WriteString(fmt.Sprintf("%f", device.CPU))
		buffer.WriteString(",")

		// 内存
		buffer.WriteString(fmt.Sprintf("%f", device.Memory))
		buffer.WriteString(",")

		// 型号
		buffer.WriteString(device.Model)
		buffer.WriteString(",")

		// KVM IP
		buffer.WriteString(device.KvmIP)
		buffer.WriteString(",")

		// 操作系统
		buffer.WriteString(device.OS)
		buffer.WriteString(",")

		// 厂商
		buffer.WriteString(device.Company)
		buffer.WriteString(",")

		// 操作系统名称
		buffer.WriteString(device.OSName)
		buffer.WriteString(",")

		// 操作系统版本
		buffer.WriteString(device.OSIssue)
		buffer.WriteString(",")

		// 操作系统内核
		buffer.WriteString(device.OSKernel)
		buffer.WriteString(",")

		// 状态
		buffer.WriteString(device.Status)
		buffer.WriteString(",")

		// 角色
		buffer.WriteString(device.Role)
		buffer.WriteString(",")

		// 集群
		buffer.WriteString(device.Cluster)
		buffer.WriteString(",")

		// 集群ID
		buffer.WriteString(fmt.Sprintf("%d", device.ClusterID))
		buffer.WriteString(",")

		// 创建时间
		buffer.WriteString(time.Time(device.CreatedAt).Format("2006-01-02"))
		buffer.WriteString(",")

		// 更新时间
		buffer.WriteString(time.Time(device.UpdatedAt).Format("2006-01-02"))

		buffer.WriteString("\n")
	}

	return buffer.Bytes(), nil
}
