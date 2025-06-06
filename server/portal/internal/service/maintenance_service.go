package service

import (
	"fmt"
	// "log" // This might become unused - Removed
	"navy-ng/models/portal"
	"navy-ng/pkg/redis"
	"time"

	"go.uber.org/zap" // Added zap import
	"gorm.io/gorm"
)

// MaintenanceService 设备维护服务
type MaintenanceService struct {
	db             *gorm.DB
	scalingService *ElasticScalingService
	logger         *zap.Logger // Added logger
}

// MaintenanceRequestDTO 维护请求数据传输对象
type MaintenanceRequestDTO struct {
	DeviceID             int64     `json:"deviceId"`             // 设备ID
	CICode               string    `json:"ciCode"`               // 设备CI编码（可选）
	MaintenanceStartTime time.Time `json:"maintenanceStartTime"` // 维护开始时间
	MaintenanceEndTime   time.Time `json:"maintenanceEndTime"`   // 维护结束时间
	ExternalTicketID     string    `json:"externalTicketID"`     // 外部工单号
	Reason               string    `json:"reason"`               // 维护原因
	Priority             string    `json:"priority,omitempty"`   // 优先级：high, medium, low
	Comments             string    `json:"comments,omitempty"`   // 附加说明
}

// MaintenanceCallbackDTO 维护回调数据传输对象
type MaintenanceCallbackDTO struct {
	ExternalTicketID string `json:"externalTicketID"` // 外部工单号
	OrderID          int64  `json:"orderId"`          // 内部订单号
	Status           string `json:"status"`           // 维护状态：completed, cancelled, delayed
	Message          string `json:"message"`          // 状态描述
}

// MaintenanceResponseDTO 维护请求响应数据传输对象
type MaintenanceResponseDTO struct {
	Success       bool   `json:"success"`       // 是否成功
	OrderID       int64  `json:"orderId"`       // 内部订单号
	OrderNumber   string `json:"orderNumber"`   // 订单号
	ScheduledTime string `json:"scheduledTime"` // 确认的维护时间
	Status        string `json:"status"`        // 订单状态
	Message       string `json:"message"`       // 响应消息
}

// NewMaintenanceService 创建设备维护服务
func NewMaintenanceService(db *gorm.DB, logger *zap.Logger) *MaintenanceService {
	// 创建Redis处理器
	redisHandler := redis.NewRedisHandler("default")

	// 创建设备缓存
	deviceCache := NewDeviceCache(redisHandler, redis.NewKeyBuilder("navy", "v1"))

	return &MaintenanceService{
		db:             db,
		scalingService: NewElasticScalingService(db, redisHandler, logger, deviceCache),
		logger:         logger,
	}
}

// RequestMaintenance 请求设备维护
func (s *MaintenanceService) RequestMaintenance(request *MaintenanceRequestDTO) (*MaintenanceResponseDTO, error) {
	// 使用Redis锁确保同一设备不会重复提交维护请求
	lockKey := fmt.Sprintf("maintenance:device:%d:lock", request.DeviceID)
	redisHandler := redis.NewRedisHandler("default")
	redisHandler.Expire(30 * time.Second)

	lockSuccess, err := redisHandler.AcquireLock(lockKey, fmt.Sprintf("maint_req:%d", time.Now().UnixNano()), 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("获取维护锁失败: %w", err)
	}

	if !lockSuccess {
		return nil, fmt.Errorf("该设备已有维护请求正在处理中，请稍后再试")
	}

	defer redisHandler.Delete(lockKey)

	// 检查设备是否存在
	var device portal.Device
	if err := s.db.First(&device, request.DeviceID).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	// 检查是否已存在该设备的待处理维护订单（使用新的订单表结构）
	var existingOrderCount int64
	err = s.db.Table("orders o").
		Joins("JOIN elastic_scaling_order_details d ON o.id = d.order_id").
		Joins("JOIN order_device od ON o.id = od.order_id").
		Where("od.device_id = ? AND d.action_type = ? AND o.type = ? AND o.status IN (?)",
			request.DeviceID, "maintenance_request", portal.OrderTypeElasticScaling,
			[]string{"pending_confirmation", "scheduled_for_maintenance", "maintenance_in_progress"}).
		Count(&existingOrderCount).Error
	if err != nil {
		return nil, fmt.Errorf("查询现有维护订单失败: %w", err)
	}

	if existingOrderCount > 0 {
		return nil, fmt.Errorf("该设备已有待处理的维护订单")
	}

	// 创建维护请求订单
	orderDTO := OrderDTO{
		ClusterID:            int64(device.ClusterID), // 将int转换为int64
		ActionType:           "maintenance_request",
		DeviceCount:          1,                         // 维护订单只针对单台设备
		Devices:              []int64{request.DeviceID}, // 使用设备列表而不是单个DeviceID
		MaintenanceStartTime: &request.MaintenanceStartTime,
		MaintenanceEndTime:   &request.MaintenanceEndTime,
		ExternalTicketID:     request.ExternalTicketID,
		CreatedBy:            "external_system", // 由外部系统创建
		ExtraInfo: map[string]interface{}{
			"reason":   request.Reason,
			"priority": request.Priority,
			"comments": request.Comments,
		},
	}

	orderID, err := s.scalingService.CreateOrder(orderDTO)
	if err != nil {
		return nil, fmt.Errorf("创建维护订单失败: %w", err)
	}

	// 获取创建的订单信息
	var order portal.Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("获取创建的订单失败: %w", err)
	}

	// 构建响应
	response := &MaintenanceResponseDTO{
		Success:       true,
		OrderID:       orderID,
		OrderNumber:   order.OrderNumber,
		ScheduledTime: request.MaintenanceStartTime.Format(time.RFC3339),
		Status:        "pending_confirmation",
		Message:       "维护请求已接收，等待确认",
	}

	return response, nil
}

// ConfirmMaintenance 确认维护请求
func (s *MaintenanceService) ConfirmMaintenance(orderID int64, operatorID string) error {
	// 获取订单信息
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").First(&order, orderID).Error; err != nil {
		return fmt.Errorf("订单不存在: %w", err)
	}

	// 验证订单类型和状态
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return fmt.Errorf("不是弹性伸缩订单")
	}

	if order.ElasticScalingDetail.ActionType != "maintenance_request" {
		return fmt.Errorf("不是维护订单")
	}

	if string(order.Status) != "pending_confirmation" {
		return fmt.Errorf("订单状态不是待确认")
	}

	// 更新订单状态为已确认，等待维护
	err := s.db.Model(&order).Updates(map[string]interface{}{
		"status":     "scheduled_for_maintenance",
		"executor":   operatorID,
		"updated_at": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("确认维护请求失败: %w", err)
	}

	return nil
}

// StartMaintenance 开始设备维护，执行Cordon操作
func (s *MaintenanceService) StartMaintenance(orderID int64, operatorID string) error {
	// 获取订单信息
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").First(&order, orderID).Error; err != nil {
		return fmt.Errorf("订单不存在: %w", err)
	}

	// 验证订单类型和状态
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return fmt.Errorf("不是弹性伸缩订单")
	}

	if order.ElasticScalingDetail.ActionType != "maintenance_request" {
		return fmt.Errorf("不是维护订单")
	}

	if string(order.Status) != "scheduled_for_maintenance" {
		return fmt.Errorf("订单状态不正确，当前状态: %s, 预期状态: scheduled_for_maintenance", order.Status)
	}

	// 获取设备信息（通过OrderDevice关联表）
	var orderDevice portal.OrderDevice
	if err := s.db.Where("order_id = ?", orderID).First(&orderDevice).Error; err != nil {
		return fmt.Errorf("订单没有关联设备: %w", err)
	}

	var device portal.Device
	if err := s.db.First(&device, orderDevice.DeviceID).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// TODO: 调用Kubernetes客户端执行Cordon操作
	// 此处只是示例，实际应集成K8s客户端
	s.logger.Info("Executing node Cordon operation",
		zap.String("deviceIP", device.IP),
		zap.Int("clusterID", device.ClusterID), // Changed to zap.Int
		zap.Int64("orderID", orderID))

	// 更新订单状态为维护中
	err := s.db.Model(&order).Updates(map[string]interface{}{
		"status":         "maintenance_in_progress",
		"executor":       operatorID,
		"execution_time": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	return nil
}

// CompleteMaintenance 完成设备维护，创建Uncordon订单
func (s *MaintenanceService) CompleteMaintenance(externalTicketID string, message string) (*MaintenanceResponseDTO, error) {
	// 根据外部工单号查找维护订单（使用新的订单表结构）
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").
		Joins("JOIN elastic_scaling_order_details d ON orders.id = d.order_id").
		Where("d.external_ticket_id = ? AND d.action_type = ? AND orders.type = ?",
			externalTicketID, "maintenance_request", portal.OrderTypeElasticScaling).
		First(&order).Error; err != nil {
		return nil, fmt.Errorf("找不到对应的维护订单: %w", err)
	}

	// 验证订单状态
	if string(order.Status) != "maintenance_in_progress" {
		return nil, fmt.Errorf("订单状态不是维护中，当前状态: %s", order.Status)
	}

	// 更新维护请求订单状态为已完成
	err := s.db.Model(&order).Updates(map[string]interface{}{
		"status":          "completed",
		"completion_time": time.Now(),
	}).Error

	if err != nil {
		return nil, fmt.Errorf("更新维护订单状态失败: %w", err)
	}
	// 获取原订单的设备信息
	var orderDevice portal.OrderDevice
	if err := s.db.Where("order_id = ?", order.ID).First(&orderDevice).Error; err != nil {
		return nil, fmt.Errorf("获取订单关联设备失败: %w", err)
	}

	uncordonOrderDTO := OrderDTO{
		ClusterID:        order.ElasticScalingDetail.ClusterID,
		ActionType:       "maintenance_uncordon",
		DeviceCount:      1,
		Devices:          []int64{orderDevice.DeviceID}, // 使用相同的设备
		ExternalTicketID: externalTicketID,
		CreatedBy:        "system",
	}

	uncordonOrderID, err := s.scalingService.CreateOrder(uncordonOrderDTO)
	if err != nil {
		return nil, fmt.Errorf("创建Uncordon订单失败: %w", err)
	}

	// 获取创建的Uncordon订单
	var uncordonOrder portal.Order
	if err := s.db.First(&uncordonOrder, uncordonOrderID).Error; err != nil {
		return nil, fmt.Errorf("获取创建的Uncordon订单失败: %w", err)
	}

	// 构建响应
	response := &MaintenanceResponseDTO{
		Success:     true,
		OrderID:     uncordonOrderID,
		OrderNumber: uncordonOrder.OrderNumber,
		Status:      "pending",
		Message:     "维护完成，已创建节点恢复订单",
	}

	return response, nil
}

// ExecuteUncordon 执行Uncordon操作
func (s *MaintenanceService) ExecuteUncordon(orderID int64, operatorID string) error {
	// 获取订单信息
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").First(&order, orderID).Error; err != nil {
		return fmt.Errorf("订单不存在: %w", err)
	}

	// 验证订单类型和状态
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return fmt.Errorf("不是弹性伸缩订单")
	}

	if order.ElasticScalingDetail.ActionType != "maintenance_uncordon" {
		return fmt.Errorf("不是Uncordon订单")
	}

	if string(order.Status) != "pending" && string(order.Status) != "processing" {
		return fmt.Errorf("订单状态不正确，当前状态: %s", order.Status)
	}

	// 获取设备信息（通过OrderDevice关联表）
	var orderDevice portal.OrderDevice
	if err := s.db.Where("order_id = ?", orderID).First(&orderDevice).Error; err != nil {
		return fmt.Errorf("订单没有关联设备: %w", err)
	}

	var device portal.Device
	if err := s.db.First(&device, orderDevice.DeviceID).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 更新订单状态为处理中
	if order.Status == portal.OrderStatusPending {
		err := s.db.Model(&order).Updates(map[string]interface{}{
			"status":         "processing",
			"executor":       operatorID,
			"execution_time": time.Now(),
		}).Error

		if err != nil {
			return fmt.Errorf("更新订单状态失败: %w", err)
		}
	}

	// TODO: 调用Kubernetes客户端执行Uncordon操作
	// 此处只是示例，实际应集成K8s客户端
	s.logger.Info("Executing node Uncordon operation",
		zap.String("deviceIP", device.IP),
		zap.Int("clusterID", device.ClusterID), // Changed to zap.Int
		zap.Int64("orderID", orderID))

	// 更新订单状态为已完成
	err := s.db.Model(&order).Updates(map[string]interface{}{
		"status":          "completed",
		"completion_time": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	return nil
}

// GetPendingMaintenanceRequests 获取所有待处理的维护请求
func (s *MaintenanceService) GetPendingMaintenanceRequests() ([]OrderDetailDTO, error) {
	// 查询所有待确认的维护请求（使用新的订单表结构）
	var orderIDs []int64
	if err := s.db.Table("orders o").
		Joins("JOIN elastic_scaling_order_details d ON o.id = d.order_id").
		Where("o.type = ? AND d.action_type = ? AND o.status IN (?)",
			portal.OrderTypeElasticScaling, "maintenance_request",
			[]string{"pending_confirmation", "scheduled_for_maintenance"}).
		Pluck("o.id", &orderIDs).Error; err != nil {
		return nil, fmt.Errorf("查询待处理维护请求失败: %w", err)
	}

	var results []OrderDetailDTO
	// 使用现有服务获取订单详情
	for _, orderID := range orderIDs {
		orderDetail, err := s.scalingService.GetOrder(orderID)
		if err != nil {
			s.logger.Error("Failed to get order details for pending maintenance request",
				zap.Int64("orderID", orderID),
				zap.Error(err))
			continue
		}

		results = append(results, *orderDetail)
	}

	return results, nil
}

// GetPendingUncordonRequests 获取所有待处理的Uncordon请求
func (s *MaintenanceService) GetPendingUncordonRequests() ([]OrderDetailDTO, error) {
	// 查询所有待处理的Uncordon请求（使用新的订单表结构）
	var orderIDs []int64
	if err := s.db.Table("orders o").
		Joins("JOIN elastic_scaling_order_details d ON o.id = d.order_id").
		Where("o.type = ? AND d.action_type = ? AND o.status IN (?)",
			portal.OrderTypeElasticScaling, "maintenance_uncordon",
			[]string{"pending", "processing"}).
		Pluck("o.id", &orderIDs).Error; err != nil {
		return nil, fmt.Errorf("查询待处理Uncordon请求失败: %w", err)
	}

	var results []OrderDetailDTO
	// 使用现有服务获取订单详情
	for _, orderID := range orderIDs {
		orderDetail, err := s.scalingService.GetOrder(orderID)
		if err != nil {
			s.logger.Error("Failed to get order details for pending uncordon request",
				zap.Int64("orderID", orderID),
				zap.Error(err))
			continue
		}

		results = append(results, *orderDetail)
	}

	return results, nil
}
