package order

import (
	"context"
	"fmt"
	"navy-ng/models/portal"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MaintenanceRequestDTO 维护请求DTO
type MaintenanceRequestDTO struct {
	DeviceID             int64     `json:"deviceId"`
	CICode               string    `json:"ciCode"`
	MaintenanceStartTime time.Time `json:"maintenanceStartTime"`
	MaintenanceEndTime   time.Time `json:"maintenanceEndTime"`
	ExternalTicketID     string    `json:"externalTicketId"`
	Priority             string    `json:"priority"`
	Reason               string    `json:"reason"`
	Comments             string    `json:"comments"`
}

// MaintenanceResponseDTO 维护响应DTO
type MaintenanceResponseDTO struct {
	Success         bool   `json:"success"`
	OrderID         int64  `json:"orderId"`
	OrderNumber     string `json:"orderNumber"`
	ScheduledTime   string `json:"scheduledTime"`
	Status          string `json:"status"`
	Message         string `json:"message"`
	UncordonOrderID *int64 `json:"uncordonOrderId,omitempty"`
}

// MaintenanceCallbackDTO 维护回调DTO
type MaintenanceCallbackDTO struct {
	ExternalTicketID string `json:"externalTicketId"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	CompletedAt      string `json:"completedAt"`
}

// MaintenanceOrderDTO 维护订单DTO
type MaintenanceOrderDTO struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	ClusterID            int64                  `json:"clusterId"`
	Devices              []int64                `json:"devices"`
	MaintenanceStartTime *time.Time             `json:"maintenanceStartTime"`
	MaintenanceEndTime   *time.Time             `json:"maintenanceEndTime"`
	ExternalTicketID     string                 `json:"externalTicketId"`
	MaintenanceType      portal.MaintenanceType `json:"maintenanceType"`
	Priority             string                 `json:"priority"`
	Reason               string                 `json:"reason"`
	Comments             string                 `json:"comments"`
	CreatedBy            string                 `json:"createdBy"`
}

// MaintenanceOrderDetailDTO 维护订单详情DTO
type MaintenanceOrderDetailDTO struct {
	Order             *portal.Order                  `json:"order"`
	MaintenanceDetail *portal.MaintenanceOrderDetail `json:"maintenanceDetail"`
	Devices           []portal.Device                `json:"devices"`
}

// MaintenanceOrderService 维护订单服务接口
type MaintenanceOrderService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMaintenanceOrderService 创建维护订单服务
func NewMaintenanceOrderService(db *gorm.DB, logger *zap.Logger) *MaintenanceOrderService {
	return &MaintenanceOrderService{
		db:     db,
		logger: logger,
	}
}

// CreateOrder 创建维护订单
func (s *MaintenanceOrderService) CreateOrder(ctx context.Context, dto MaintenanceOrderDTO) (*MaintenanceOrderDetailDTO, error) {
	// 创建基础订单
	order := &portal.Order{
		OrderNumber: fmt.Sprintf("MAINT-%d", time.Now().Unix()),
		Name:        dto.Name,
		Description: dto.Description,
		Type:        portal.OrderTypeMaintenance,
		Status:      portal.OrderStatusPending,
		CreatedBy:   dto.CreatedBy,
	}

	if err := s.db.Create(order).Error; err != nil {
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	// 创建维护订单详情
	maintenanceDetail := &portal.MaintenanceOrderDetail{
		OrderID:              order.ID,
		ClusterID:            dto.ClusterID,
		MaintenanceStartTime: (*portal.NavyTime)(dto.MaintenanceStartTime),
		MaintenanceEndTime:   (*portal.NavyTime)(dto.MaintenanceEndTime),
		ExternalTicketID:     dto.ExternalTicketID,
		MaintenanceType:      string(dto.MaintenanceType),
		Priority:             dto.Priority,
		Reason:               dto.Reason,
		Comments:             dto.Comments,
	}

	if err := s.db.Create(maintenanceDetail).Error; err != nil {
		return nil, fmt.Errorf("创建维护详情失败: %w", err)
	}

	// 获取设备信息
	var devices []portal.Device
	if len(dto.Devices) > 0 {
		if err := s.db.Where("id IN ?", dto.Devices).Find(&devices).Error; err != nil {
			return nil, fmt.Errorf("查询设备失败: %w", err)
		}
	}

	return &MaintenanceOrderDetailDTO{
		Order:             order,
		MaintenanceDetail: maintenanceDetail,
		Devices:           devices,
	}, nil
}

// ListOrders 列出维护订单
func (s *MaintenanceOrderService) ListOrders(ctx context.Context, filter interface{}) ([]*MaintenanceOrderDetailDTO, int64, error) {
	var orders []portal.Order
	var total int64

	query := s.db.Where("type = ?", portal.OrderTypeMaintenance)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	var results []*MaintenanceOrderDetailDTO
	for _, order := range orders {
		var maintenanceDetail portal.MaintenanceOrderDetail
		if err := s.db.Where("order_id = ?", order.ID).First(&maintenanceDetail).Error; err != nil {
			continue
		}

		var devices []portal.Device
		// 这里需要根据实际的设备关联逻辑来查询设备

		results = append(results, &MaintenanceOrderDetailDTO{
			Order:             &order,
			MaintenanceDetail: &maintenanceDetail,
			Devices:           devices,
		})
	}

	return results, total, nil
}

// ConfirmMaintenance 确认维护
func (s *MaintenanceOrderService) ConfirmMaintenance(ctx context.Context, orderID int64, operatorID string) error {
	return s.db.Model(&portal.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status":   "confirmed",
		"executor": operatorID,
	}).Error
}

// StartMaintenance 开始维护
func (s *MaintenanceOrderService) StartMaintenance(ctx context.Context, orderID int64, operatorID string) error {
	return s.db.Model(&portal.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status":   "in_progress",
		"executor": operatorID,
	}).Error
}

// ExecuteUncordon 执行Uncordon操作
func (s *MaintenanceOrderService) ExecuteUncordon(ctx context.Context, orderID int64, operatorID string) error {
	return s.db.Model(&portal.Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"status":   portal.OrderStatusCompleted,
		"executor": operatorID,
	}).Error
}

// CompleteMaintenance 完成维护
func (s *MaintenanceOrderService) CompleteMaintenance(ctx context.Context, externalTicketID string, message string) (*MaintenanceOrderDetailDTO, error) {
	// 查找对应的维护订单
	var maintenanceDetail portal.MaintenanceOrderDetail
	if err := s.db.Where("external_ticket_id = ?", externalTicketID).First(&maintenanceDetail).Error; err != nil {
		return nil, fmt.Errorf("未找到对应的维护订单: %w", err)
	}

	// 更新订单状态为完成
	if err := s.db.Model(&portal.Order{}).Where("id = ?", maintenanceDetail.OrderID).Update("status", portal.OrderStatusCompleted).Error; err != nil {
		return nil, fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 获取完整的订单信息
	var order portal.Order
	if err := s.db.First(&order, maintenanceDetail.OrderID).Error; err != nil {
		return nil, err
	}

	return &MaintenanceOrderDetailDTO{
		Order:             &order,
		MaintenanceDetail: &maintenanceDetail,
		Devices:           []portal.Device{},
	}, nil
}

// MaintenanceServiceV2 设备维护服务V2版本
// 保持原有接口不变，但内部使用新的维护订单服务架构
type MaintenanceServiceV2 struct {
	db                      *gorm.DB
	maintenanceOrderService *MaintenanceOrderService
	logger                  *zap.Logger
}

// convertDevicesToDTO 转换Device到DeviceDTO
func convertDevicesToDTO(devices []portal.Device) []DeviceDTO {
	result := make([]DeviceDTO, len(devices))
	for i, device := range devices {
		result[i] = DeviceDTO{
			ID:       device.ID,
			CICode:   device.CICode,
			IP:       device.IP,
			ArchType: device.ArchType,
			CPU:      device.CPU,
			Memory:   device.Memory,
			Status:   device.Status,
			Role:     device.Role,
			Cluster:  device.Cluster,
		}
	}
	return result
}

// 使用原有的DTO定义，避免重复声明

// NewMaintenanceServiceV2 创建设备维护服务V2
func NewMaintenanceServiceV2(db *gorm.DB, logger *zap.Logger) *MaintenanceServiceV2 {
	return &MaintenanceServiceV2{
		db:                      db,
		maintenanceOrderService: NewMaintenanceOrderService(db, logger),
		logger:                  logger,
	}
}

// RequestMaintenance 请求设备维护
func (s *MaintenanceServiceV2) RequestMaintenance(request *MaintenanceRequestDTO) (*MaintenanceResponseDTO, error) {
	ctx := context.Background()

	// 获取设备信息
	var device portal.Device
	if err := s.db.First(&device, request.DeviceID).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	// 转换为新的维护订单DTO
	maintenanceOrderDTO := MaintenanceOrderDTO{
		Name:                 fmt.Sprintf("设备维护 - %s", device.IP),
		Description:          fmt.Sprintf("设备 %s 的维护请求，工单号: %s", device.IP, request.ExternalTicketID),
		ClusterID:            int64(device.ClusterID),
		Devices:              []int64{request.DeviceID},
		MaintenanceStartTime: &request.MaintenanceStartTime,
		MaintenanceEndTime:   &request.MaintenanceEndTime,
		ExternalTicketID:     request.ExternalTicketID,
		MaintenanceType:      portal.MaintenanceTypeCordon, // 默认为cordon类型
		Priority:             request.Priority,
		Reason:               request.Reason,
		Comments:             request.Comments,
		CreatedBy:            "external_system",
	}

	// 创建维护订单
	orderDetail, err := s.maintenanceOrderService.CreateOrder(ctx, maintenanceOrderDTO)
	if err != nil {
		return nil, fmt.Errorf("创建维护订单失败: %w", err)
	}

	// 构建响应
	response := &MaintenanceResponseDTO{
		Success:       true,
		OrderID:       orderDetail.Order.ID,
		OrderNumber:   orderDetail.Order.OrderNumber,
		ScheduledTime: request.MaintenanceStartTime.Format(time.RFC3339),
		Status:        string(orderDetail.Order.Status),
		Message:       "维护请求已接收，等待确认",
	}

	return response, nil
}

// ConfirmMaintenance 确认维护请求
func (s *MaintenanceServiceV2) ConfirmMaintenance(orderID int64, operatorID string) error {
	ctx := context.Background()
	return s.maintenanceOrderService.ConfirmMaintenance(ctx, orderID, operatorID)
}

// StartMaintenance 开始设备维护，执行Cordon操作
func (s *MaintenanceServiceV2) StartMaintenance(orderID int64, operatorID string) error {
	ctx := context.Background()
	return s.maintenanceOrderService.StartMaintenance(ctx, orderID, operatorID)
}

// CompleteMaintenance 完成设备维护，创建Uncordon订单
func (s *MaintenanceServiceV2) CompleteMaintenance(externalTicketID string, message string) (*MaintenanceResponseDTO, error) {
	ctx := context.Background()

	// 完成维护并可能创建uncordon订单
	orderDetail, err := s.maintenanceOrderService.CompleteMaintenance(ctx, externalTicketID, message)
	if err != nil {
		return nil, err
	}

	// 查找是否有自动创建的uncordon订单
	var uncordonOrder *MaintenanceOrderDetailDTO
	orders, _, err := s.maintenanceOrderService.ListOrders(ctx, nil)
	if err == nil {
		for _, order := range orders {
			if order.MaintenanceDetail.ExternalTicketID == externalTicketID &&
				order.MaintenanceDetail.MaintenanceType == string(portal.MaintenanceTypeUncordon) &&
				order.Order.Status == portal.OrderStatusPending {
				uncordonOrder = order
				break
			}
		}
	}

	// 构建响应
	response := &MaintenanceResponseDTO{
		Success:     true,
		OrderID:     orderDetail.Order.ID,
		OrderNumber: orderDetail.Order.OrderNumber,
		Status:      string(orderDetail.Order.Status),
		Message:     "维护完成",
	}

	if uncordonOrder != nil {
		response.OrderID = uncordonOrder.Order.ID
		response.OrderNumber = uncordonOrder.Order.OrderNumber
		response.Status = string(uncordonOrder.Order.Status)
		response.Message = "维护完成，已创建节点恢复订单"
	}

	return response, nil
}

// ExecuteUncordon 执行Uncordon操作
func (s *MaintenanceServiceV2) ExecuteUncordon(orderID int64, operatorID string) error {
	ctx := context.Background()
	return s.maintenanceOrderService.ExecuteUncordon(ctx, orderID, operatorID)
}

// GetPendingMaintenanceRequests 获取所有待处理的维护请求
func (s *MaintenanceServiceV2) GetPendingMaintenanceRequests() ([]OrderDetailDTO, error) {
	ctx := context.Background()

	// 获取所有维护订单
	orders, _, err := s.maintenanceOrderService.ListOrders(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("查询待处理维护请求失败: %w", err)
	}

	var results []OrderDetailDTO
	for _, order := range orders {
		// 只返回待确认和已安排维护的订单
		if order.Order.Status == portal.OrderStatusPending ||
			order.Order.Status == "scheduled_for_maintenance" {

			// 转换为原有的OrderDetailDTO格式
			orderDetailDTO := OrderDetailDTO{
				OrderDTO: OrderDTO{
					ID:          order.Order.ID,
					OrderNumber: order.Order.OrderNumber,
					Name:        order.Order.Name,
					Description: order.Order.Description,
					Status:      string(order.Order.Status),
					Executor:    order.Order.Executor,
					CreatedBy:   order.Order.CreatedBy,
					CreatedAt:   time.Time(order.Order.CreatedAt),
				},
				Devices: convertDevicesToDTO(order.Devices),
			}

			if order.Order.ExecutionTime != nil {
				execTime := time.Time(*order.Order.ExecutionTime)
				orderDetailDTO.ExecutionTime = &execTime
			}
			if order.Order.CompletionTime != nil {
				compTime := time.Time(*order.Order.CompletionTime)
				orderDetailDTO.CompletionTime = &compTime
			}

			// 设置维护相关字段
			if order.MaintenanceDetail != nil {
				orderDetailDTO.ClusterID = order.MaintenanceDetail.ClusterID
				orderDetailDTO.ExternalTicketID = order.MaintenanceDetail.ExternalTicketID
				if order.MaintenanceDetail.MaintenanceStartTime != nil {
					startTime := time.Time(*order.MaintenanceDetail.MaintenanceStartTime)
					orderDetailDTO.MaintenanceStartTime = &startTime
				}
				if order.MaintenanceDetail.MaintenanceEndTime != nil {
					endTime := time.Time(*order.MaintenanceDetail.MaintenanceEndTime)
					orderDetailDTO.MaintenanceEndTime = &endTime
				}
			}

			results = append(results, orderDetailDTO)
		}
	}

	return results, nil
}

// GetPendingUncordonRequests 获取所有待处理的Uncordon请求
func (s *MaintenanceServiceV2) GetPendingUncordonRequests() ([]OrderDetailDTO, error) {
	ctx := context.Background()

	// 获取所有维护订单
	orders, _, err := s.maintenanceOrderService.ListOrders(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("查询待处理Uncordon请求失败: %w", err)
	}

	var results []OrderDetailDTO
	for _, order := range orders {
		// 只返回uncordon类型的待处理订单
		if order.MaintenanceDetail != nil &&
			order.MaintenanceDetail.MaintenanceType == string(portal.MaintenanceTypeUncordon) &&
			(order.Order.Status == portal.OrderStatusPending || order.Order.Status == portal.OrderStatusProcessing) {

			// 转换为原有的OrderDetailDTO格式
			orderDetailDTO := OrderDetailDTO{
				OrderDTO: OrderDTO{
					ID:          order.Order.ID,
					OrderNumber: order.Order.OrderNumber,
					Name:        order.Order.Name,
					Description: order.Order.Description,
					Status:      string(order.Order.Status),
					Executor:    order.Order.Executor,
					CreatedBy:   order.Order.CreatedBy,
					CreatedAt:   time.Time(order.Order.CreatedAt),
				},
				Devices: convertDevicesToDTO(order.Devices),
			}

			if order.Order.ExecutionTime != nil {
				execTime := time.Time(*order.Order.ExecutionTime)
				orderDetailDTO.ExecutionTime = &execTime
			}
			if order.Order.CompletionTime != nil {
				compTime := time.Time(*order.Order.CompletionTime)
				orderDetailDTO.CompletionTime = &compTime
			}

			// 设置维护相关字段
			orderDetailDTO.ClusterID = order.MaintenanceDetail.ClusterID
			orderDetailDTO.ExternalTicketID = order.MaintenanceDetail.ExternalTicketID
			if order.MaintenanceDetail.MaintenanceStartTime != nil {
				startTime := time.Time(*order.MaintenanceDetail.MaintenanceStartTime)
				orderDetailDTO.MaintenanceStartTime = &startTime
			}
			if order.MaintenanceDetail.MaintenanceEndTime != nil {
				endTime := time.Time(*order.MaintenanceDetail.MaintenanceEndTime)
				orderDetailDTO.MaintenanceEndTime = &endTime
			}

			results = append(results, orderDetailDTO)
		}
	}

	return results, nil
}
