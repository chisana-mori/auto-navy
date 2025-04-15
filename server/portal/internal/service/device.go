package service

import (
	"bytes"
	"context"
	"fmt"
	"navy-ng/models/portal"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Constants for DeviceService
const (
	ErrDeviceNotFoundMsg = "device with ID %d not found"
)

// DeviceQuery 设备查询参数
type DeviceQuery struct {
	Page    int    `form:"page" json:"page"`       // 页码
	Size    int    `form:"size" json:"size"`       // 每页数量
	Keyword string `form:"keyword" json:"keyword"` // 搜索关键字
}

// DeviceResponse 设备响应
type DeviceResponse struct {
	ID             int64     `json:"id"`             // ID
	CICode         string    `json:"ciCode"`         // 设备编码
	IP             string    `json:"ip"`             // IP地址
	ArchType       string    `json:"archType"`       // CPU架构
	IDC            string    `json:"idc"`            // IDC
	Room           string    `json:"room"`           // 机房
	Cabinet        string    `json:"cabinet"`        // 所属机柜
	CabinetNO      string    `json:"cabinetNo"`      // 机柜编号
	InfraType      string    `json:"infraType"`      // 网络类型
	IsLocalization bool      `json:"isLocalization"` // 是否国产化
	NetZone        string    `json:"netZone"`        // 网络区域
	Group          string    `json:"group"`          // 机器类别
	AppID          string    `json:"appId"`          // APPID
	OsCreateTime   string    `json:"osCreateTime"`   // 操作系统创建时间
	CPU            float64   `json:"cpu"`            // CPU数量
	Memory         float64   `json:"memory"`         // 内存大小
	Model          string    `json:"model"`          // 型号
	KvmIP          string    `json:"kvmIp"`          // KVM IP
	OS             string    `json:"os"`             // 操作系统
	Company        string    `json:"company"`        // 厂商
	OSName         string    `json:"osName"`         // 操作系统名称
	OSIssue        string    `json:"osIssue"`        // 操作系统版本
	OSKernel       string    `json:"osKernel"`       // 操作系统内核
	Status         string    `json:"status"`         // 状态
	Role           string    `json:"role"`           // 角色
	Cluster        string    `json:"cluster"`        // 所属集群
	ClusterID      int       `json:"clusterId"`      // 集群ID
	CreatedAt      time.Time `json:"createdAt"`      // 创建时间
	UpdatedAt      time.Time `json:"updatedAt"`      // 更新时间
}

// DeviceRoleUpdateRequest 设备角色更新请求
type DeviceRoleUpdateRequest struct {
	Role string `json:"role" binding:"required"` // 新的角色值
}

// DeviceGroupUpdateRequest 设备用途更新请求
type DeviceGroupUpdateRequest struct {
	Group string `json:"group" binding:"required"` // 新的用途值
}

// DeviceService 设备服务
type DeviceService struct {
	db *gorm.DB
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
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

	db := s.db.WithContext(ctx).Model(&portal.Device{})

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
					db = db.Where("ip LIKE ?", ip)
				} else {
					db = db.Or("ip LIKE ?", ip)
				}
			}
		} else {
			// 常规关键字查询
			for i, keyword := range keywords {
				keyword = "%" + keyword + "%"

				// 第一个关键字使用 Where，后续关键字使用 Or
				if i == 0 {
					db = db.Where(
						"ci_code LIKE ? OR ip LIKE ? OR arch_type LIKE ? OR cluster LIKE ? OR "+
							"role LIKE ? OR idc LIKE ? OR room LIKE ? OR cabinet LIKE ? OR "+
							"cabinet_no LIKE ? OR infra_type LIKE ? OR net_zone LIKE ? OR appid LIKE ? OR `group` LIKE ?",
						keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
					)
				} else {
					db = db.Or(
						"ci_code LIKE ? OR ip LIKE ? OR arch_type LIKE ? OR cluster LIKE ? OR "+
							"role LIKE ? OR idc LIKE ? OR room LIKE ? OR cabinet LIKE ? OR "+
							"cabinet_no LIKE ? OR infra_type LIKE ? OR net_zone LIKE ? OR appid LIKE ? OR `group` LIKE ?",
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

func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*DeviceResponse, error) {
	var model portal.Device
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

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
		CreatedAt:      time.Time(model.CreatedAt),
		UpdatedAt:      time.Time(model.UpdatedAt),
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

	// 获取所有设备信息
	if err := s.db.WithContext(ctx).Find(&devices).Error; err != nil {
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
