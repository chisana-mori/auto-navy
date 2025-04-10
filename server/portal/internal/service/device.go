package service

import (
	"bytes"
	"context"
	"fmt"
	"navy-ng/models/portal"
	"time"

	"gorm.io/gorm"
)

// Constants for DeviceService
const (
	ErrDeviceNotFoundMsg = "device with ID %d not found"
)

// DeviceQuery 设备查询参数
type DeviceQuery struct {
	Page    int    `form:"page" json:"page"`         // 页码
	Size    int    `form:"size" json:"size"`         // 每页数量
	Keyword string `form:"keyword" json:"keyword"`   // 搜索关键字
}

// DeviceListResponse 设备列表响应
type DeviceListResponse struct {
	List  []DeviceResponse `json:"list"`  // 设备列表
	Total int64            `json:"total"` // 总数
	Page  int              `json:"page"`  // 当前页码
	Size  int              `json:"size"`  // 每页数量
}

// DeviceResponse 设备响应
type DeviceResponse struct {
	ID           int64     `json:"id"`            // ID
	DeviceID     string    `json:"deviceId"`     // 设备ID
	IP           string    `json:"ip"`            // IP地址
	MachineType  string    `json:"machineType"`  // 机器类型
	Cluster      string    `json:"cluster"`       // 所属集群
	Role         string    `json:"role"`          // 集群角色
	Arch         string    `json:"arch"`          // 架构
	IDC          string    `json:"idc"`           // IDC
	Room         string    `json:"room"`          // Room
	Datacenter   string    `json:"datacenter"`    // 机房
	Cabinet      string    `json:"cabinet"`       // 机柜号
	Network      string    `json:"network"`       // 网络区域
	AppID        string    `json:"appId"`        // APPID
	ResourcePool string    `json:"resourcePool"` // 资源池/产品
	CreatedAt    time.Time `json:"createdAt"`    // 创建时间
	UpdatedAt    time.Time `json:"updatedAt"`    // 更新时间
}

// DeviceRoleUpdateRequest 设备角色更新请求
type DeviceRoleUpdateRequest struct {
	Role string `json:"role" binding:"required"` // 新的角色值
}

// DeviceService 设备服务
type DeviceService struct {
	db *gorm.DB
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
}

// ListDevices 获取设备列表
// @Summary 获取设备列表
// @Description 获取设备列表，支持分页和关键字搜索
// @Tags Device
// @Accept json
// @Produce json
// @Param query query DeviceQuery true "查询参数"
// @Success 200 {object} DeviceListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /device [get]
func (s *DeviceService) ListDevices(ctx context.Context, query *DeviceQuery) (*DeviceListResponse, error) {
	var models []portal.Device
	var total int64

	db := s.db.WithContext(ctx).Model(&portal.Device{}).Where("deleted = ?", emptyString)

	// 应用关键字搜索
	if query.Keyword != emptyString {
		keyword := "%" + query.Keyword + "%"
		db = db.Where(
			"device_id LIKE ? OR ip LIKE ? OR machine_type LIKE ? OR cluster LIKE ? OR "+
				"role LIKE ? OR arch LIKE ? OR idc LIKE ? OR room LIKE ? OR "+
				"datacenter LIKE ? OR cabinet LIKE ? OR network LIKE ? OR app_id LIKE ? OR resource_pool LIKE ?",
			keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword,
		)
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
			ID:           model.ID,
			DeviceID:     model.DeviceID,
			IP:           model.IP,
			MachineType:  model.MachineType,
			Cluster:      model.Cluster,
			Role:         model.Role,
			Arch:         model.Arch,
			IDC:          model.IDC,
			Room:         model.Room,
			Datacenter:   model.Datacenter,
			Cabinet:      model.Cabinet,
			Network:      model.Network,
			AppID:        model.AppID,
			ResourcePool: model.ResourcePool,
			CreatedAt:    time.Time(model.CreatedAt),
			UpdatedAt:    time.Time(model.UpdatedAt),
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
// @Summary 获取设备详情
// @Description 根据ID获取设备详情
// @Tags Device
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} DeviceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /device/{id} [get]
func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*DeviceResponse, error) {
	var model portal.Device
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = ?", id, emptyString).First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return &DeviceResponse{
		ID:           model.ID,
		DeviceID:     model.DeviceID,
		IP:           model.IP,
		MachineType:  model.MachineType,
		Cluster:      model.Cluster,
		Role:         model.Role,
		Arch:         model.Arch,
		IDC:          model.IDC,
		Room:         model.Room,
		Datacenter:   model.Datacenter,
		Cabinet:      model.Cabinet,
		Network:      model.Network,
		AppID:        model.AppID,
		ResourcePool: model.ResourcePool,
		CreatedAt:    time.Time(model.CreatedAt),
		UpdatedAt:    time.Time(model.UpdatedAt),
	}, nil
}

// UpdateDeviceRole 更新设备角色
// @Summary 更新设备角色
// @Description 只更新设备的角色字段
// @Tags Device
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body DeviceRoleUpdateRequest true "角色更新请求"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /device/{id}/role [patch]
func (s *DeviceService) UpdateDeviceRole(ctx context.Context, id int64, request *DeviceRoleUpdateRequest) error {
	// 查找设备
	var device portal.Device
	result := s.db.WithContext(ctx).Where("id = ? AND deleted = ?", id, emptyString).First(&device)
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

// ExportDevices 导出所有设备信息为Excel
// @Summary 导出设备信息
// @Description 导出所有设备信息为Excel文件
// @Tags Device
// @Accept json
// @Produce application/octet-stream
// @Success 200 {file} file "设备信息.xlsx"
// @Failure 500 {object} ErrorResponse
// @Router /device/export [get]
func (s *DeviceService) ExportDevices(ctx context.Context) ([]byte, error) {
	var devices []portal.Device

	// 获取所有设备信息
	if err := s.db.WithContext(ctx).Where("deleted = ?", emptyString).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to get devices for export: %w", err)
	}

	// 创建CSV格式的数据
	buffer := &bytes.Buffer{}

	// 写入表头
	headers := []string{
		"设备ID", "IP地址", "机器类型", "所属集群", "集群角色",
		"架构", "IDC", "Room", "机房", "机柜号",
		"网络区域", "APPID", "资源池/产品", "创建时间", "更新时间",
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
		// 设备ID
		buffer.WriteString(device.DeviceID)
		buffer.WriteString(",")

		// IP地址
		buffer.WriteString(device.IP)
		buffer.WriteString(",")

		// 机器类型
		buffer.WriteString(device.MachineType)
		buffer.WriteString(",")

		// 所属集群
		buffer.WriteString(device.Cluster)
		buffer.WriteString(",")

		// 集群角色
		buffer.WriteString(device.Role)
		buffer.WriteString(",")

		// 架构
		buffer.WriteString(device.Arch)
		buffer.WriteString(",")

		// IDC
		buffer.WriteString(device.IDC)
		buffer.WriteString(",")

		// Room
		buffer.WriteString(device.Room)
		buffer.WriteString(",")

		// 机房
		buffer.WriteString(device.Datacenter)
		buffer.WriteString(",")

		// 机柜号
		buffer.WriteString(device.Cabinet)
		buffer.WriteString(",")

		// 网络区域
		buffer.WriteString(device.Network)
		buffer.WriteString(",")

		// APPID
		buffer.WriteString(device.AppID)
		buffer.WriteString(",")

		// 资源池/产品
		buffer.WriteString(device.ResourcePool)
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
