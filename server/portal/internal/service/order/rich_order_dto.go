package order

import (
	"time"

	"navy-ng/models/portal"

	"gorm.io/gorm"
)

// DeviceDTO 设备DTO
type DeviceDTO struct {
	ID           int     `json:"id"`
	CICode       string  `json:"ciCode"`
	IP           string  `json:"ip"`
	ArchType     string  `json:"archType"`
	CPU          float64 `json:"cpu"`
	Memory       float64 `json:"memory"`
	Status       string  `json:"status"`
	Role         string  `json:"role"`
	Cluster      string  `json:"cluster"`
	ClusterID    int     `json:"clusterId"`
	IsSpecial    bool    `json:"isSpecial"`
	FeatureCount int     `json:"featureCount"`
	OrderStatus  string  `json:"orderStatus,omitempty"` // 在订单中的状态
}

// OrderDTO 弹性伸缩订单DTO
type OrderDTO struct {
	ID               int    `json:"id,omitempty"`
	OrderNumber      string `json:"orderNumber"`
	Name             string `json:"name"`        // 订单名称
	Description      string `json:"description"` // 订单描述
	ClusterID        int    `json:"clusterId"`
	ClusterName      string `json:"clusterName,omitempty"`
	StrategyID       *int   `json:"strategyId"`
	StrategyName     string `json:"strategyName,omitempty"`
	ActionType       string `json:"actionType"`       // pool_entry, pool_exit, maintenance_request, maintenance_uncordon
	ResourcePoolType string `json:"resourcePoolType"` // 资源池类型
	Status           string `json:"status"`
	DeviceCount      int    `json:"deviceCount"`
	// DeviceID字段已移除，使用Devices列表和OrderDevice关联表
	DeviceInfo             *DeviceDTO             `json:"deviceInfo,omitempty"`
	Executor               string                 `json:"executor"`
	ExecutionTime          *time.Time             `json:"executionTime"`
	CreatedBy              string                 `json:"createdBy"`
	CreatedAt              time.Time              `json:"createdAt"`
	CompletionTime         *time.Time             `json:"completionTime"`
	FailureReason          string                 `json:"failureReason"`
	MaintenanceStartTime   *time.Time             `json:"maintenanceStartTime,omitempty"`
	MaintenanceEndTime     *time.Time             `json:"maintenanceEndTime,omitempty"`
	ExternalTicketID       string                 `json:"externalTicketId,omitempty"`
	Devices                []int                  `json:"devices,omitempty"`   // 设备ID列表
	ExtraInfo              map[string]interface{} `json:"extraInfo,omitempty"` // 额外信息，用于存储维护原因等
	StrategyTriggeredValue string                 `json:"strategyTriggeredValue,omitempty"`
	StrategyThresholdValue string                 `json:"strategyThresholdValue,omitempty"`
}

// OrderListItemDTO 订单列表项
type OrderListItemDTO struct {
	ID               int       `json:"id"`
	OrderNumber      string    `json:"orderNumber"`
	Name             string    `json:"name"`        // 订单名称
	Description      string    `json:"description"` // 订单描述
	ClusterID        int       `json:"clusterId"`
	ClusterName      string    `json:"clusterName"`
	StrategyID       *int      `json:"strategyId"`
	StrategyName     string    `json:"strategyName"`
	ActionType       string    `json:"actionType"`
	ResourcePoolType string    `json:"resourcePoolType"` // 资源池类型
	Status           string    `json:"status"`
	DeviceCount      int       `json:"deviceCount"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
}

// OrderDetailDTO 订单详情
type OrderDetailDTO struct {
	OrderDTO
	Devices []DeviceDTO `json:"devices"` // 涉及设备的详细信息
}

// RichOrderDTO 丰富的订单DTO，包含完整的关联信息
type RichOrderDTO struct {
	// 基础订单信息
	ID             int                `json:"id"`
	OrderNumber    string             `json:"orderNumber"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Type           portal.OrderType   `json:"type"`
	Status         portal.OrderStatus `json:"status"`
	Executor       string             `json:"executor"`
	ExecutionTime  *time.Time         `json:"executionTime"`
	CreatedBy      string             `json:"createdBy"`
	CompletionTime *time.Time         `json:"completionTime"`
	FailureReason  string             `json:"failureReason"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`

	// 弹性伸缩详情（如果是弹性伸缩订单）
	ElasticScalingDetail *ElasticScalingDetailDTO `json:"elasticScalingDetail,omitempty"`

	// 关联的设备列表
	Devices []RichDeviceDTO `json:"devices,omitempty"`

	// 统计信息
	DeviceCount int `json:"deviceCount"` // 设备总数
}

// ElasticScalingDetailDTO 弹性伸缩详情DTO
type ElasticScalingDetailDTO struct {
	ID                     int        `json:"id"`
	OrderID                int        `json:"orderId"`
	ClusterID              int        `json:"clusterId"`
	ClusterName            string     `json:"clusterName,omitempty"` // 集群名称
	StrategyID             *int       `json:"strategyId"`
	StrategyName           string     `json:"strategyName,omitempty"` // 策略名称
	ActionType             string     `json:"actionType"`
	DeviceCount            int        `json:"deviceCount"`
	MaintenanceStartTime   *time.Time `json:"maintenanceStartTime"`
	MaintenanceEndTime     *time.Time `json:"maintenanceEndTime"`
	ExternalTicketID       string     `json:"externalTicketId"`
	StrategyTriggeredValue string     `json:"strategyTriggeredValue"`
	StrategyThresholdValue string     `json:"strategyThresholdValue"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

// RichDeviceDTO 丰富的设备DTO
type RichDeviceDTO struct {
	// 基础设备信息
	ID             int       `json:"id"`
	CICode         string    `json:"ciCode"`
	IP             string    `json:"ip"`
	ArchType       string    `json:"archType"`
	IDC            string    `json:"idc"`
	Room           string    `json:"room"`
	Cabinet        string    `json:"cabinet"`
	CabinetNO      string    `json:"cabinetNo"`
	InfraType      string    `json:"infraType"`
	IsLocalization bool      `json:"isLocalization"`
	NetZone        string    `json:"netZone"`
	Group          string    `json:"group"`
	AppID          string    `json:"appId"`
	AppName        string    `json:"appName"`
	OsCreateTime   string    `json:"osCreateTime"`
	CPU            float64   `json:"cpu"`
	Memory         float64   `json:"memory"`
	Model          string    `json:"model"`
	KvmIP          string    `json:"kvmIp"`
	OS             string    `json:"os"`
	Company        string    `json:"company"`
	OSName         string    `json:"osName"`
	OSIssue        string    `json:"osIssue"`
	OSKernel       string    `json:"osKernel"`
	Status         string    `json:"status"`
	Role           string    `json:"role"`
	Cluster        string    `json:"cluster"`
	ClusterID      int       `json:"clusterId"`
	AcceptanceTime string    `json:"acceptanceTime"`
	DiskCount      int       `json:"diskCount"`
	DiskDetail     string    `json:"diskDetail"`
	NetworkSpeed   string    `json:"networkSpeed"`
	IsSpecial      bool      `json:"isSpecial"`
	FeatureCount   int       `json:"featureCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// 订单中的状态
	OrderStatus string `json:"orderStatus,omitempty"` // 设备在订单中的处理状态
}

// ToRichOrderDTO 将数据库模型转换为丰富的DTO
func ToRichOrderDTO(order *portal.Order) *RichOrderDTO {
	if order == nil {
		return nil
	}

	dto := &RichOrderDTO{
		ID:            order.ID,
		OrderNumber:   order.OrderNumber,
		Name:          order.Name,
		Description:   order.Description,
		Type:          order.Type,
		Status:        order.Status,
		Executor:      order.Executor,
		CreatedBy:     order.CreatedBy,
		FailureReason: order.FailureReason,
		CreatedAt:     time.Time(order.CreatedAt),
		UpdatedAt:     time.Time(order.UpdatedAt),
	}

	// 转换时间字段
	if order.ExecutionTime != nil {
		execTime := time.Time(*order.ExecutionTime)
		dto.ExecutionTime = &execTime
	}

	if order.CompletionTime != nil {
		complTime := time.Time(*order.CompletionTime)
		dto.CompletionTime = &complTime
	}

	// 转换弹性伸缩详情
	if order.ElasticScalingDetail != nil {
		dto.ElasticScalingDetail = toElasticScalingDetailDTO(order.ElasticScalingDetail)
		dto.DeviceCount = order.ElasticScalingDetail.DeviceCount

		// Device conversion is removed from here as it requires a DB query.
		// This DTO is now primarily for non-device-specific details.
		// The full device list should be fetched by a dedicated service method if needed.
	}

	return dto
}

// toElasticScalingDetailDTO 转换弹性伸缩详情
func toElasticScalingDetailDTO(detail *portal.ElasticScalingOrderDetail) *ElasticScalingDetailDTO {
	if detail == nil {
		return nil
	}

	dto := &ElasticScalingDetailDTO{
		ID:                     detail.ID,
		OrderID:                detail.OrderID,
		ClusterID:              detail.ClusterID,
		StrategyID:             detail.StrategyID,
		ActionType:             detail.ActionType,
		DeviceCount:            detail.DeviceCount,
		ExternalTicketID:       "", // 维护相关字段已移至MaintenanceOrderDetail
		StrategyTriggeredValue: detail.StrategyTriggeredValue,
		StrategyThresholdValue: detail.StrategyThresholdValue,
		CreatedAt:              time.Time(detail.CreatedAt),
		UpdatedAt:              time.Time(detail.UpdatedAt),
	}

	// 维护时间字段现在由MaintenanceOrderDetail处理
	// MaintenanceStartTime和MaintenanceEndTime已移至MaintenanceOrderDetail

	return dto
}

// toRichDeviceDTO 转换设备信息
func toRichDeviceDTO(device *portal.Device) RichDeviceDTO {
	return RichDeviceDTO{
		ID:             device.ID,
		CICode:         device.CICode,
		IP:             device.IP,
		ArchType:       device.ArchType,
		IDC:            device.IDC,
		Room:           device.Room,
		Cabinet:        device.Cabinet,
		CabinetNO:      device.CabinetNO,
		InfraType:      device.InfraType,
		IsLocalization: device.IsLocalization,
		NetZone:        device.NetZone,
		Group:          device.Group,
		AppID:          device.AppID,
		AppName:        device.AppName,
		OsCreateTime:   device.OsCreateTime,
		CPU:            device.CPU,
		Memory:         device.Memory,
		Model:          device.Model,
		KvmIP:          device.KvmIP,
		OS:             device.OS,
		Company:        device.Company,
		OSName:         device.OSName,
		OSIssue:        device.OSIssue,
		OSKernel:       device.OSKernel,
		Status:         device.Status,
		Role:           device.Role,
		Cluster:        device.Cluster,
		ClusterID:      device.ClusterID,
		AcceptanceTime: device.AcceptanceTime,
		DiskCount:      device.DiskCount,
		DiskDetail:     device.DiskDetail,
		NetworkSpeed:   device.NetworkSpeed,
		IsSpecial:      device.IsSpecial,
		FeatureCount:   device.FeatureCount,
		CreatedAt:      time.Time(device.CreatedAt),
		UpdatedAt:      time.Time(device.UpdatedAt),
	}
}

// ToRichOrderDTOList 批量转换订单列表
func ToRichOrderDTOList(orders []*portal.Order) []RichOrderDTO {
	if len(orders) == 0 {
		return []RichOrderDTO{}
	}

	result := make([]RichOrderDTO, len(orders))
	for i, order := range orders {
		if dto := ToRichOrderDTO(order); dto != nil {
			result[i] = *dto
		}
	}

	return result
}

// EnrichOrderDTOWithNames 使用数据库连接丰富DTO中的名称信息
func EnrichOrderDTOWithNames(dto *RichOrderDTO, db *gorm.DB) {
	if dto == nil || dto.ElasticScalingDetail == nil {
		return
	}

	// 获取集群名称
	if dto.ElasticScalingDetail.ClusterID > 0 {
		var cluster portal.K8sCluster
		if err := db.Select("clustername").First(&cluster, dto.ElasticScalingDetail.ClusterID).Error; err == nil {
			dto.ElasticScalingDetail.ClusterName = cluster.ClusterName
		}
	}

	// 获取策略名称
	if dto.ElasticScalingDetail.StrategyID != nil && *dto.ElasticScalingDetail.StrategyID > 0 {
		var strategy portal.ElasticScalingStrategy
		if err := db.Select("name").First(&strategy, *dto.ElasticScalingDetail.StrategyID).Error; err == nil {
			dto.ElasticScalingDetail.StrategyName = strategy.Name
		}
	}
}

// EnrichOrderDTOListWithNames 批量丰富DTO列表中的名称信息
func EnrichOrderDTOListWithNames(dtos []RichOrderDTO, db *gorm.DB) {
	for i := range dtos {
		EnrichOrderDTOWithNames(&dtos[i], db)
	}
}
