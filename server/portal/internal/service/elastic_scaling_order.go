package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"navy-ng/models/portal"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateOrder 创建弹性伸缩订单
func (s *ElasticScalingService) CreateOrder(dto OrderDTO) (int64, error) {
	// 使用事务确保数据一致性
	var orderID int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 创建基础订单
		order := &portal.Order{
			OrderNumber: s.generateOrderNumber(),
			Name:        dto.Name,
			Description: dto.Description,
			Type:        portal.OrderTypeElasticScaling,
			Status:      portal.OrderStatusPending,
			CreatedBy:   dto.CreatedBy,
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}
		orderID = order.ID

		// 创建弹性伸缩订单详情
		detail := &portal.ElasticScalingOrderDetail{
			OrderID:                order.ID,
			ClusterID:              dto.ClusterID,
			StrategyID:             dto.StrategyID,
			ActionType:             dto.ActionType,
			DeviceCount:            dto.DeviceCount,
			StrategyTriggeredValue: dto.StrategyTriggeredValue,
			StrategyThresholdValue: dto.StrategyThresholdValue,
			ExternalTicketID:       dto.ExternalTicketID,
		}

		// 设置维护相关字段（如果是维护订单）
		if dto.ActionType == "maintenance_request" || dto.ActionType == "maintenance_uncordon" {
			if dto.MaintenanceStartTime != nil {
				navyStartTime := portal.NavyTime(*dto.MaintenanceStartTime)
				detail.MaintenanceStartTime = &navyStartTime
			}
			if dto.MaintenanceEndTime != nil {
				navyEndTime := portal.NavyTime(*dto.MaintenanceEndTime)
				detail.MaintenanceEndTime = &navyEndTime
			}
		}

		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		// 如果提供了设备列表，创建关联
		if len(dto.Devices) > 0 {
			for _, deviceID := range dto.Devices {
				orderDevice := portal.OrderDevice{
					OrderID:  order.ID,
					DeviceID: deviceID,
					Status:   "pending",
				}
				if err := tx.Create(&orderDevice).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return orderID, nil
}

// GetOrder 获取订单详情
func (s *ElasticScalingService) GetOrder(id int64) (*OrderDetailDTO, error) {
	// 获取基础订单信息
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("订单不存在: %d", id)
		}
		return nil, err
	}

	// 检查是否为弹性伸缩订单
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return nil, fmt.Errorf("订单类型不匹配或缺少详情信息: %d", id)
	}

	detail := order.ElasticScalingDetail

	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := "未知集群"
	if err := s.db.Select("clustername").First(&cluster, detail.ClusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	// 获取策略名称（如果有关联策略）
	strategyName := ""
	if detail.StrategyID != nil {
		var strategy portal.ElasticScalingStrategy
		if err := s.db.Select("name").First(&strategy, *detail.StrategyID).Error; err == nil {
			strategyName = strategy.Name
		}
	}

	// 获取关联设备
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", id).Find(&orderDevices).Error; err != nil {
		return nil, err
	}

	// 准备设备ID列表
	deviceIDs := make([]int64, len(orderDevices))
	for i, od := range orderDevices {
		deviceIDs[i] = od.DeviceID
	}

	// 获取设备详情
	var devices []portal.Device
	if len(deviceIDs) > 0 {
		if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
			return nil, err
		}
	}

	// 获取特定设备详情（如果是维护订单）
	var deviceInfo *DeviceDTO
	// 对于维护订单，我们从OrderDevice关联表中获取第一个设备作为主要设备
	if detail.ActionType == "maintenance_request" || detail.ActionType == "maintenance_uncordon" {
		if len(orderDevices) > 0 && len(devices) > 0 {
			// 使用第一个关联设备
			for _, device := range devices {
				deviceInfo = &DeviceDTO{
					ID:           device.ID,
					CICode:       device.CICode,
					IP:           device.IP,
					ArchType:     device.ArchType,
					CPU:          device.CPU,
					Memory:       device.Memory,
					Status:       device.Status,
					Role:         device.Role,
					Cluster:      device.Cluster,
					ClusterID:    device.ClusterID,
					IsSpecial:    device.IsSpecial,
					FeatureCount: device.FeatureCount,
				}
				break // 只取第一个设备
			}
		}
	}

	// 构建DTO
	dto := &OrderDetailDTO{
		OrderDTO: OrderDTO{
			ID:           order.ID,
			OrderNumber:  order.OrderNumber,
			Name:         order.Name,        // 订单名称
			Description:  order.Description, // 订单描述
			ClusterID:    detail.ClusterID,
			ClusterName:  clusterName,
			StrategyID:   detail.StrategyID,
			StrategyName: strategyName,
			ActionType:   detail.ActionType,
			Status:       string(order.Status),
			DeviceCount:  detail.DeviceCount,
			// DeviceID字段已移除，通过OrderDevice关联表获取设备信息
			DeviceInfo:           deviceInfo,
			Executor:             order.Executor,
			CreatedBy:            order.CreatedBy,
			CreatedAt:            time.Time(order.CreatedAt),
			FailureReason:        order.FailureReason,
			MaintenanceStartTime: nil,
			MaintenanceEndTime:   nil,
			ExternalTicketID:     detail.ExternalTicketID,
		},
		Devices: make([]DeviceDTO, len(devices)),
	}

	// Proper handling of maintenance time fields
	if detail.MaintenanceStartTime != nil {
		startTime := time.Time(*detail.MaintenanceStartTime)
		dto.MaintenanceStartTime = &startTime
	}

	if detail.MaintenanceEndTime != nil {
		endTime := time.Time(*detail.MaintenanceEndTime)
		dto.MaintenanceEndTime = &endTime
	}

	// Fix execution and completion time in GetOrder
	if order.ExecutionTime != nil {
		execTime := time.Time(*order.ExecutionTime)
		dto.ExecutionTime = &execTime
	}

	if order.CompletionTime != nil {
		complTime := time.Time(*order.CompletionTime)
		dto.CompletionTime = &complTime
	}

	// 转换设备列表
	deviceStatusMap := make(map[int64]string)
	for _, od := range orderDevices {
		deviceStatusMap[od.DeviceID] = od.Status
	}

	for i, device := range devices {
		deviceDTO := DeviceDTO{
			ID:           device.ID,
			CICode:       device.CICode,
			IP:           device.IP,
			ArchType:     device.ArchType,
			CPU:          device.CPU,
			Memory:       device.Memory,
			Status:       device.Status,
			Role:         device.Role,
			Cluster:      device.Cluster,
			ClusterID:    device.ClusterID,
			IsSpecial:    device.IsSpecial,
			FeatureCount: device.FeatureCount,
		}

		// 添加设备在订单中的状态
		if status, ok := deviceStatusMap[device.ID]; ok {
			deviceDTO.OrderStatus = status
		}

		dto.Devices[i] = deviceDTO
	}

	return dto, nil
}

// ListOrders 获取订单列表
func (s *ElasticScalingService) ListOrders(clusterID int64, strategyID int64, actionType string, status string, page, pageSize int) ([]OrderListItemDTO, int64, error) {
	var total int64

	// 构建查询，联合查询基础订单表和详情表
	query := s.db.Table("orders o").
		Joins("JOIN elastic_scaling_order_details d ON o.id = d.order_id").
		Where("o.type = ?", portal.OrderTypeElasticScaling)

	// 应用过滤条件
	if clusterID > 0 {
		query = query.Where("d.cluster_id = ?", clusterID)
	}
	if strategyID > 0 {
		query = query.Where("d.strategy_id = ?", strategyID)
	}
	if actionType != "" {
		query = query.Where("d.action_type = ?", actionType)
	}
	if status != "" {
		query = query.Where("o.status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，获取完整的订单信息
	var orders []portal.Order
	if err := s.db.Preload("ElasticScalingDetail").
		Where("type = ?", portal.OrderTypeElasticScaling).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	// 准备结果
	result := make([]OrderListItemDTO, 0, len(orders))
	for _, order := range orders {
		// 检查是否有详情信息
		if order.ElasticScalingDetail == nil {
			continue
		}

		detail := order.ElasticScalingDetail

		// 应用过滤条件（因为预加载可能包含不符合条件的记录）
		if clusterID > 0 && detail.ClusterID != clusterID {
			continue
		}
		if strategyID > 0 && (detail.StrategyID == nil || *detail.StrategyID != strategyID) {
			continue
		}
		if actionType != "" && detail.ActionType != actionType {
			continue
		}
		if status != "" && string(order.Status) != status {
			continue
		}

		// 获取集群名称
		var cluster portal.K8sCluster
		clusterName := "未知集群"
		if err := s.db.Select("clustername").First(&cluster, detail.ClusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		// 获取策略名称（如果有关联策略）
		var strategyName string
		if detail.StrategyID != nil {
			var strategy portal.ElasticScalingStrategy
			if err := s.db.Select("name").First(&strategy, *detail.StrategyID).Error; err == nil {
				strategyName = strategy.Name
			}
		}

		result = append(result, OrderListItemDTO{
			ID:           order.ID,
			OrderNumber:  order.OrderNumber,
			Name:         order.Name,        // 订单名称
			Description:  order.Description, // 订单描述
			ClusterID:    detail.ClusterID,
			ClusterName:  clusterName,
			StrategyID:   detail.StrategyID,
			StrategyName: strategyName,
			ActionType:   detail.ActionType,
			Status:       string(order.Status),
			DeviceCount:  detail.DeviceCount,
			CreatedBy:    order.CreatedBy,
			CreatedAt:    time.Time(order.CreatedAt),
		})
	}

	return result, int64(len(result)), nil
}

// UpdateOrderStatus 更新订单状态
func (s *ElasticScalingService) UpdateOrderStatus(id int64, status string, executor string, reason string) error {
	// 使用通用订单服务更新状态
	ctx := context.Background()
	orderStatus := portal.OrderStatus(status)

	// 验证状态
	validStatuses := map[portal.OrderStatus]bool{
		portal.OrderStatusPending:    true,
		portal.OrderStatusIgnored:    true,
		portal.OrderStatusProcessing: true,
		portal.OrderStatusCompleted:  true,
		portal.OrderStatusFailed:     true,
		portal.OrderStatusCancelled:  true,
	}

	if !validStatuses[orderStatus] {
		return fmt.Errorf("无效的订单状态: %s", status)
	}

	// 获取订单信息（包含详情）
	var order portal.Order
	if err := s.db.Preload("ElasticScalingDetail").First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("订单不存在: %d", id)
		}
		return err
	}

	// 检查是否为弹性伸缩订单
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return fmt.Errorf("订单类型不匹配: %d", id)
	}

	// 使用通用订单服务更新状态
	err := s.orderService.UpdateOrderStatus(ctx, id, orderStatus, executor, reason)
	if err != nil {
		return err
	}

	// 如果订单状态更新为 processing 或 completed，并且是由策略生成的，则记录策略执行历史
	detail := order.ElasticScalingDetail
	if (status == "processing" || status == "completed") && detail.StrategyID != nil {
		s.logger.Info("Order status updated, recording strategy execution history",
			zap.Int64("orderID", order.ID),
			zap.String("newStatus", status),
			zap.Int64p("strategyID", detail.StrategyID))

		var historyResult string
		var executionTimeForHistory portal.NavyTime

		if status == "processing" && order.ExecutionTime != nil {
			historyResult = StrategyExecutionResultOrderProcessingStarted
			executionTimeForHistory = *order.ExecutionTime
		} else if status == "completed" && order.CompletionTime != nil {
			historyResult = StrategyExecutionResultOrderCompleted
			executionTimeForHistory = *order.CompletionTime
		} else {
			// 如果时间戳缺失，则使用当前时间，但这不理想
			s.logger.Warn("Execution/Completion time is nil for order, using current time for history",
				zap.Int64("orderID", order.ID),
				zap.String("status", status))
			executionTimeForHistory = portal.NavyTime(time.Now())
			if status == "processing" {
				historyResult = StrategyExecutionResultOrderProcStartedNoExecTime
			} else {
				historyResult = StrategyExecutionResultOrderComplNoComplTime
			}
		}

		// 从订单详情中获取保存的触发值和阈值
		reasonForHistory := fmt.Sprintf("Order %s by strategy %d.", status, *detail.StrategyID)
		if order.FailureReason != "" && status == "failed" { // 虽然这里是 processing/completed, 但以防万一
			reasonForHistory = order.FailureReason
		}

		// 调用 recordStrategyExecution
		// 注意：recordStrategyExecution 内部的 ExecutionTime 将被我们这里提供的 executionTimeForHistory 覆盖
		// triggeredValue 和 thresholdValue 将从 detail 对象中获取
		errRecord := s.recordStrategyExecution(
			*detail.StrategyID,
			historyResult,
			&order.ID,
			reasonForHistory,
			detail.StrategyTriggeredValue, // 新增参数
			detail.StrategyThresholdValue, // 新增参数
			&executionTimeForHistory,      // 新增参数，传递实际的执行或完成时间
		)
		if errRecord != nil {
			s.logger.Error("Failed to record strategy execution history after order update",
				zap.Int64("orderID", order.ID),
				zap.Int64p("strategyID", detail.StrategyID),
				zap.Error(errRecord))
			// 不返回错误，因为主操作（更新订单状态）已成功
		}
	}

	return nil
}

// GetOrderDevices 获取订单关联的设备
func (s *ElasticScalingService) GetOrderDevices(orderID int64) ([]DeviceDTO, error) {
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", orderID).Find(&orderDevices).Error; err != nil {
		return nil, err
	}

	if len(orderDevices) == 0 {
		return []DeviceDTO{}, nil
	}

	// 提取设备ID
	deviceIDs := make([]int64, len(orderDevices))
	deviceStatusMap := make(map[int64]string)
	for i, od := range orderDevices {
		deviceIDs[i] = od.DeviceID
		deviceStatusMap[od.DeviceID] = od.Status
	}

	// 获取设备详情
	var devices []portal.Device
	if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		return nil, err
	}

	// 构建结果
	result := make([]DeviceDTO, len(devices))
	for i, device := range devices {
		result[i] = DeviceDTO{
			ID:           device.ID,
			CICode:       device.CICode,
			IP:           device.IP,
			ArchType:     device.ArchType,
			CPU:          device.CPU,
			Memory:       device.Memory,
			Status:       device.Status,
			Role:         device.Role,
			Cluster:      device.Cluster,
			ClusterID:    device.ClusterID,
			IsSpecial:    device.IsSpecial,
			FeatureCount: device.FeatureCount,
			OrderStatus:  deviceStatusMap[device.ID],
		}
	}

	return result, nil
}

// UpdateOrderDeviceStatus 更新订单中设备的状态
func (s *ElasticScalingService) UpdateOrderDeviceStatus(orderID int64, deviceID int64, status string) error {
	// 验证状态
	validStatuses := map[string]bool{
		"pending":    true,
		"processing": true,
		"completed":  true,
		"failed":     true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("无效的设备状态: %s", status)
	}

	var orderDevice portal.OrderDevice
	result := s.db.Where("order_id = ? AND device_id = ?", orderID, deviceID).First(&orderDevice)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("订单中不存在该设备")
		}
		return result.Error
	}

	orderDevice.Status = status
	return s.db.Save(&orderDevice).Error
}

// generateOrderNumber 生成唯一订单号
func (s *ElasticScalingService) generateOrderNumber() string {
	// 生成格式为 "ESO" + 年月日 + 6位随机数的订单号
	now := time.Now()
	dateStr := now.Format("20060102")
	randomStr := fmt.Sprintf("%06d", rand.Intn(1000000))
	return "ESO" + dateStr + randomStr
}
