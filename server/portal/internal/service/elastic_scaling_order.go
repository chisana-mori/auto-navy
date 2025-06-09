package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"navy-ng/models/portal"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// 动作类型
	actionTypeMaintenanceRequest  = "maintenance_request"
	actionTypeMaintenanceUncordon = "maintenance_uncordon"

	// 错误信息
	errOrderNotFound           = "订单不存在: %d"
	errOrderTypeMismatch       = "订单类型不匹配或缺少详情信息: %d"
	errInvalidOrderStatus      = "无效的订单状态: %s"
	errDeviceNotInOrder        = "订单中不存在该设备"
	errFailedToFindOrderDetail = "failed to find order detail for order %d: %w"
	errInvalidDeviceStatus     = "无效的设备状态: %s"

	// 日志消息
	logOrderStatusUpdated    = "Order status updated, recording strategy execution history"
	logTimeIsNil             = "Execution/Completion time is nil for order, using current time for history"
	logFailedToRecordHistory = "Failed to record strategy execution history after order update"

	// Zap 日志字段键
	zapKeyOrderID   = "orderID"
	zapKeyNewStatus = "newStatus"
	zapKeyStatus    = "status"

	// GORM 查询字段
	fieldClusterName = "clustername"
	fieldName        = "name"
	queryClusterID   = "d.cluster_id = ?"
	queryStrategyID  = "d.strategy_id = ?"
	queryActionType  = "d.action_type = ?"
	queryOrderStatus = "o.status = ?"
	queryOrderName   = "o.name LIKE ?"

	// 默认值
	unknownCluster = "未知集群"

	// 格式化字符串
	reasonOrderUpdatedByStrategy = "Order %s by strategy %d."

	// 邮件相关常量
	emailSubjectTemplate = "【弹性伸缩】%s变更通知 - 订单号：%s"
	actionTypePoolEntry  = "pool_entry"
	actionTypePoolExit   = "pool_exit"
	actionNamePoolEntry  = "入池"
	actionNamePoolExit   = "退池"
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
			Status:      portal.OrderStatus(StatusPending),
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
			ResourcePoolType:       dto.ResourcePoolType,
			DeviceCount:            dto.DeviceCount,
			StrategyTriggeredValue: dto.StrategyTriggeredValue,
			StrategyThresholdValue: dto.StrategyThresholdValue,
		}

		// 维护相关字段现在由MaintenanceOrderDetail处理
		// ExternalTicketID, MaintenanceStartTime, MaintenanceEndTime已移至MaintenanceOrderDetail

		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		// 如果提供了设备列表，创建关联
		if len(dto.Devices) > 0 {
			for _, deviceID := range dto.Devices {
				orderDevice := portal.OrderDevice{
					OrderID:  order.ID,
					DeviceID: deviceID,
					Status:   StatusPending,
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

	// 生成邮件正文通知值班人员
	emailContent := s.generateOrderNotificationEmail(orderID, dto)
	s.logger.Info("Generated order notification email",
		zap.Int64("orderID", orderID),
		zap.String("emailContent", emailContent))

	// TODO: 实现邮件发送功能
	// 这里需要用户自定义实现邮件发送逻辑
	// 可以集成企业邮件系统、钉钉、企业微信等通知渠道
	// 示例：
	// err = s.sendEmail(emailContent, getOnDutyPersons())
	// if err != nil {
	//     s.logger.Error("Failed to send notification email", zap.Error(err))
	// }

	return orderID, nil
}

// GetOrder 获取订单详情
func (s *ElasticScalingService) GetOrder(id int64) (*OrderDetailDTO, error) {
	// 获取基础订单信息
	var order portal.Order
	if err := s.db.Preload(preloadElasticScalingDetail).First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf(errOrderNotFound, id)
		}
		return nil, err
	}

	// 检查是否为弹性伸缩订单
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return nil, fmt.Errorf(errOrderTypeMismatch, id)
	}

	detail := order.ElasticScalingDetail

	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := unknownCluster
	if err := s.db.Select(fieldClusterName).First(&cluster, detail.ClusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	// 获取策略名称（如果有关联策略）
	strategyName := ""
	if detail.StrategyID != nil {
		var strategy portal.ElasticScalingStrategy
		if err := s.db.Select(fieldName).First(&strategy, *detail.StrategyID).Error; err == nil {
			strategyName = strategy.Name
		}
	}

	// 获取关联设备
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", order.ID).Find(&orderDevices).Error; err != nil {
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
	if detail.ActionType == actionTypeMaintenanceRequest || detail.ActionType == actionTypeMaintenanceUncordon {
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
			ID:               order.ID,
			OrderNumber:      order.OrderNumber,
			Name:             order.Name,        // 订单名称
			Description:      order.Description, // 订单描述
			ClusterID:        detail.ClusterID,
			ClusterName:      clusterName,
			StrategyID:       detail.StrategyID,
			StrategyName:     strategyName,
			ActionType:       detail.ActionType,
			ResourcePoolType: detail.ResourcePoolType,
			Status:           string(order.Status),
			DeviceCount:      detail.DeviceCount,
			// DeviceID字段已移除，通过OrderDevice关联表获取设备信息
			DeviceInfo:           deviceInfo,
			Executor:             order.Executor,
			CreatedBy:            order.CreatedBy,
			CreatedAt:            time.Time(order.CreatedAt),
			FailureReason:        order.FailureReason,
			MaintenanceStartTime: nil,
			MaintenanceEndTime:   nil,
			ExternalTicketID:     "",
		},
		Devices: make([]DeviceDTO, len(devices)),
	}

	// Maintenance time fields are now handled by MaintenanceOrderDetail
	// These fields are no longer part of ElasticScalingOrderDetail

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
func (s *ElasticScalingService) ListOrders(clusterID int64, strategyID int64, actionType string, status string, name string, page, pageSize int) ([]OrderListItemDTO, int64, error) {
	var total int64

	// 构建查询，联合查询基础订单表和详情表
	query := s.db.Table("orders o").
		Joins("JOIN ng_elastic_scaling_order_details d ON o.id = d.order_id").
		Where("o.type = ?", portal.OrderTypeElasticScaling)

	// 应用过滤条件
	if clusterID > 0 {
		query = query.Where(queryClusterID, clusterID)
	}
	if strategyID > 0 {
		query = query.Where(queryStrategyID, strategyID)
	}
	if actionType != "" {
		query = query.Where(queryActionType, actionType)
	}
	if status != "" {
		query = query.Where(queryOrderStatus, status)
	}
	if name != "" {
		query = query.Where(queryOrderName, "%"+name+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，获取完整的订单信息
	var orders []portal.Order
	orderQuery := s.db.Preload(preloadElasticScalingDetail).
		Where("type = ?", portal.OrderTypeElasticScaling)

	// 添加订单名称过滤条件
	if name != "" {
		orderQuery = orderQuery.Where("name LIKE ?", "%"+name+"%")
	}

	if err := orderQuery.Order(OrderByCreatedAtDesc).
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
		if name != "" && !strings.Contains(strings.ToLower(order.Name), strings.ToLower(name)) {
			continue
		}

		// 获取集群名称
		var cluster portal.K8sCluster
		clusterName := unknownCluster
		if err := s.db.Select(fieldClusterName).First(&cluster, detail.ClusterID).Error; err == nil {
			clusterName = cluster.ClusterName
		}

		// 获取策略名称（如果有关联策略）
		var strategyName string
		if detail.StrategyID != nil {
			var strategy portal.ElasticScalingStrategy
			if err := s.db.Select(fieldName).First(&strategy, *detail.StrategyID).Error; err == nil {
				strategyName = strategy.Name
			}
		}

		result = append(result, OrderListItemDTO{
			ID:               order.ID,
			OrderNumber:      order.OrderNumber,
			Name:             order.Name,        // 订单名称
			Description:      order.Description, // 订单描述
			ClusterID:        detail.ClusterID,
			ClusterName:      clusterName,
			StrategyID:       detail.StrategyID,
			StrategyName:     strategyName,
			ActionType:       detail.ActionType,
			ResourcePoolType: detail.ResourcePoolType,
			Status:           string(order.Status),
			DeviceCount:      detail.DeviceCount,
			CreatedBy:        order.CreatedBy,
			CreatedAt:        time.Time(order.CreatedAt),
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
		portal.OrderStatusPending:         true,
		portal.OrderStatusIgnored:         true,
		portal.OrderStatusProcessing:      true,
		portal.OrderStatusReturning:       true,
		portal.OrderStatusReturnCompleted: true,
		portal.OrderStatusNoReturn:        true,
		portal.OrderStatusCompleted:       true,
		portal.OrderStatusFailed:          true,
		portal.OrderStatusCancelled:       true,
	}

	if !validStatuses[orderStatus] {
		return fmt.Errorf(errInvalidOrderStatus, status)
	}

	// 获取订单信息（包含详情）
	var order portal.Order
	if err := s.db.Preload(preloadElasticScalingDetail).First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf(errOrderNotFound, id)
		}
		return err
	}

	// 检查是否为弹性伸缩订单
	if order.Type != portal.OrderTypeElasticScaling || order.ElasticScalingDetail == nil {
		return fmt.Errorf(errOrderTypeMismatch, id)
	}

	// 使用通用订单服务更新状态
	err := s.orderService.UpdateOrderStatus(ctx, id, orderStatus, executor, reason)
	if err != nil {
		return err
	}

	// 如果订单状态更新为 processing 或 completed，并且是由策略生成的，则记录策略执行历史
	detail := order.ElasticScalingDetail
	if (status == string(portal.OrderStatusProcessing) || status == string(portal.OrderStatusCompleted)) && detail.StrategyID != nil {
		s.logger.Info(logOrderStatusUpdated,
			zap.Int64(zapKeyOrderID, order.ID),
			zap.String(zapKeyNewStatus, status),
			zap.Int64p(zapKeyStrategyID, detail.StrategyID))

		var historyResult string
		var executionTimeForHistory portal.NavyTime

		if status == string(portal.OrderStatusProcessing) && order.ExecutionTime != nil {
			historyResult = StrategyExecutionResultOrderProcessingStarted
			executionTimeForHistory = *order.ExecutionTime
		} else if status == string(portal.OrderStatusCompleted) && order.CompletionTime != nil {
			historyResult = StrategyExecutionResultOrderCompleted
			executionTimeForHistory = *order.CompletionTime
		} else {
			// 如果时间戳缺失，则使用当前时间，但这不理想
			s.logger.Warn(logTimeIsNil,
				zap.Int64(zapKeyOrderID, order.ID),
				zap.String(zapKeyStatus, status))
			executionTimeForHistory = portal.NavyTime(time.Now())
			if status == string(portal.OrderStatusProcessing) {
				historyResult = StrategyExecutionResultOrderProcStartedNoExecTime
			} else {
				historyResult = StrategyExecutionResultOrderComplNoComplTime
			}
		}

		// 从订单详情中获取保存的触发值和阈值
		reasonForHistory := fmt.Sprintf(reasonOrderUpdatedByStrategy, status, *detail.StrategyID)
		if order.FailureReason != "" && status == string(portal.OrderStatusFailed) { // 虽然这里是 processing/completed, 但以防万一
			reasonForHistory = order.FailureReason
		}

		// 调用 recordStrategyExecution
		// 注意：recordStrategyExecution 内部的 ExecutionTime 将被我们这里提供的 executionTimeForHistory 覆盖
		// triggeredValue 和 thresholdValue 将从 detail 对象中获取
		errRecord := s.recordStrategyExecution(
			*detail.StrategyID,
			detail.ClusterID,        // clusterID 参数
			detail.ResourcePoolType, // resourceType 参数
			historyResult,
			&order.ID,
			reasonForHistory,
			detail.StrategyTriggeredValue, // 新增参数
			detail.StrategyThresholdValue, // 新增参数
			&executionTimeForHistory,      // 新增参数，传递实际的执行或完成时间
		)
		if errRecord != nil {
			s.logger.Error(logFailedToRecordHistory,
				zap.Int64(zapKeyOrderID, order.ID),
				zap.Error(errRecord))
		}
	}

	return nil
}

// UpdateOrderDeviceStatus 更新订单中单个设备的状态
func (s *ElasticScalingService) UpdateOrderDeviceStatus(orderID int64, deviceID int64, status string) error {
	// 验证状态
	validStatuses := map[string]bool{
		StatusPending:   true,
		StatusSuccess:   true,
		StatusFailed:    true,
		StatusExecuting: true,
	}
	if !validStatuses[status] {
		return fmt.Errorf(errInvalidDeviceStatus, status)
	}

	// 查找订单设备关联记录
	var orderDevice portal.OrderDevice
	err := s.db.Where("order_id = ? AND device_id = ?", orderID, deviceID).First(&orderDevice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf(errDeviceNotInOrder)
		}
		return err
	}

	// 更新状态
	orderDevice.Status = status
	return s.db.Save(&orderDevice).Error
}

// GetOrderDevices 获取订单中的所有设备
func (s *ElasticScalingService) GetOrderDevices(orderID int64) ([]DeviceDTO, error) {
	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_id = ?", orderID).Find(&orderDevices).Error; err != nil {
		return nil, err
	}

	if len(orderDevices) == 0 {
		return []DeviceDTO{}, nil
	}

	deviceIDs := make([]int64, len(orderDevices))
	for i, od := range orderDevices {
		deviceIDs[i] = od.DeviceID
	}

	var devices []portal.Device
	if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		return nil, err
	}

	// 转换设备列表
	deviceStatusMap := make(map[int64]string)
	for _, od := range orderDevices {
		deviceStatusMap[od.DeviceID] = od.Status
	}

	deviceDTOs := make([]DeviceDTO, len(devices))
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
		if status, ok := deviceStatusMap[device.ID]; ok {
			deviceDTO.OrderStatus = status
		}
		deviceDTOs[i] = deviceDTO
	}

	return deviceDTOs, nil
}

// generateOrderNumber 生成唯一的订单号
func (s *ElasticScalingService) generateOrderNumber() string {
	// 格式: ES-YYYYMMDD-HHMMSS-random
	return fmt.Sprintf("ES-%s-%d", time.Now().Format("20060102-150405"), rand.Intn(1000))
}

// generateOrderNotificationEmail 生成订单创建的邮件通知内容
func (s *ElasticScalingService) generateOrderNotificationEmail(orderID int64, dto OrderDTO) string {
	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := "未知集群"
	if err := s.db.Select("clustername").First(&cluster, dto.ClusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	// 获取设备信息
	devices := s.getDeviceInfoForEmail(dto.Devices)

	// 获取动作名称
	actionName := s.getActionName(dto.ActionType)

	// 生成邮件主题
	subject := fmt.Sprintf(emailSubjectTemplate, actionName, dto.Name)

	// 构建邮件正文
	return s.buildEmailHTML(subject, actionName, clusterName, dto, devices)
}

// getActionName 将动作类型转换为可读的中文名称
func (s *ElasticScalingService) getActionName(actionType string) string {
	switch actionType {
	case actionTypePoolEntry:
		return actionNamePoolEntry
	case actionTypePoolExit:
		return actionNamePoolExit
	case "maintenance_request":
		return "维护申请"
	case "maintenance_uncordon":
		return "维护解除"
	default:
		if actionType == "unknown" || actionType == "" {
			return "未知操作"
		}
		return actionType
	}
}

// getDeviceInfoForEmail 获取用于邮件通知的设备信息
func (s *ElasticScalingService) getDeviceInfoForEmail(deviceIDs []int64) []DeviceDTO {
	if len(deviceIDs) == 0 {
		return nil
	}
	var devices []portal.Device
	if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		s.logger.Error("Failed to get device info for email", zap.Error(err))
		return nil
	}

	deviceDTOs := make([]DeviceDTO, len(devices))
	for i, d := range devices {
		deviceDTOs[i] = DeviceDTO{
			ID:      d.ID,
			CICode:  d.CICode,
			IP:      d.IP,
			CPU:     d.CPU,
			Memory:  d.Memory,
			Status:  d.Status,
			Cluster: d.Cluster,
		}
	}
	return deviceDTOs
}

// buildEmailHTML 构建邮件正文的HTML结构
func (s *ElasticScalingService) buildEmailHTML(subject, actionName, clusterName string, dto OrderDTO, devices []DeviceDTO) string {
	var builder strings.Builder

	// 确定主题色彩
	var headerColor, actionColor, actionIcon string
	switch actionName {
	case actionNamePoolEntry:
		headerColor = "linear-gradient(135deg, #52c41a 0%, #389e0d 100%)"
		actionColor = "#52c41a"
		actionIcon = "🚀"
	case actionNamePoolExit:
		headerColor = "linear-gradient(135deg, #ff7a45 0%, #d4380d 100%)"
		actionColor = "#ff7a45"
		actionIcon = "📉"
	default:
		headerColor = "linear-gradient(135deg, #1890ff 0%, #0050b3 100%)"
		actionColor = "#1890ff"
		actionIcon = "⚡"
	}

	// 如果无设备，使用警告色
	if len(devices) == 0 {
		headerColor = "linear-gradient(135deg, #ff7a45 0%, #d4380d 100%)"
		actionIcon = "⚠️"
	}

	// HTML头部和样式
	builder.WriteString(`<!DOCTYPE html>
		<html>
		<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + subject + `</title>
		</head>
<body style="margin: 0; padding: 20px; font-family: 'Microsoft YaHei', Arial, sans-serif; background-color: #f5f7fa;">
    <div style="max-width: 800px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">`)

	// 邮件头部
	builder.WriteString(fmt.Sprintf(`
        <!-- 邮件头部 -->
        <div style="background: %s; color: white; padding: 24px; border-radius: 8px 8px 0 0;">
            <h1 style="margin: 0; font-size: 24px; font-weight: 600;">
                %s %s变更通知`, headerColor, actionIcon, actionName))

	if len(devices) == 0 {
		builder.WriteString("（设备不足）")
	}

	builder.WriteString(fmt.Sprintf(`
            </h1>
            <p style="margin: 8px 0 0 0; font-size: 14px; opacity: 0.9;">
                订单号：%s | 创建时间：%s
            </p>
        </div>`, dto.Name, time.Now().Format("2006-01-02 15:04:05")))

	// 邮件正文
	builder.WriteString(`
        <!-- 邮件正文 -->
        <div style="padding: 32px;">`)

	// 问候语
	var greetingText string
	if len(devices) == 0 {
		greetingText = fmt.Sprintf("系统检测到集群资源需要进行<strong style=\"color: %s;\">%s</strong>变更操作，但<strong style=\"color: #ff4d4f;\">无法找到可用设备</strong>，请协调处理相关工作。", actionColor, actionName)
	} else {
		greetingText = fmt.Sprintf("系统检测到集群资源需要进行<strong style=\"color: %s;\">%s</strong>变更操作，已匹配到 %d 台设备，请及时处理。", actionColor, actionName, len(devices))
	}

	builder.WriteString(fmt.Sprintf(`
            <!-- 问候语 -->
            <div style="margin-bottom: 24px;">
                <h2 style="color: #333; font-size: 18px; margin: 0 0 12px 0;">👋 值班同事，您好！</h2>
                <p style="color: #666; font-size: 14px; line-height: 1.6; margin: 0;">
                    %s
                </p>
            </div>`, greetingText))

	// 订单详情
	builder.WriteString(`
            <!-- 订单详情 -->
            <div style="margin-bottom: 24px;">
                <h3 style="color: #262626; margin: 0 0 20px 0; font-size: 18px; font-weight: 600; display: flex; align-items: center;">
                    <span style="margin-right: 8px;">📋</span>
                    <span>订单详情</span>
                </h3>
                <div style="background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%); border-radius: 8px; padding: 24px; border: 1px solid #e9ecef; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
                    <table style="width: 100%; border-collapse: separate; border-spacing: 0;">`)

	builder.WriteString(fmt.Sprintf(`
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; width: 140px; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #1890ff; margin-right: 8px; border-radius: 2px;"></span>
                                订单号
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 600; color: #262626; font-family: 'Courier New', monospace;">%s</td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: %s; margin-right: 8px; border-radius: 2px;"></span>
                                操作类型
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 700;">
                                <span style="color: %s; background-color: %s20; padding: 4px 12px; border-radius: 16px; font-size: 13px;">%s</span>
                            </td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #52c41a; margin-right: 8px; border-radius: 2px;"></span>
                                目标集群
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 600; color: #262626;">%s</td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #722ed1; margin-right: 8px; border-radius: 2px;"></span>
                                资源池类型
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 600; color: #262626;">
                                <span style="background-color: #f6f6f6; color: #595959; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-family: monospace;">%s</span>
                            </td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #fa8c16; margin-right: 8px; border-radius: 2px;"></span>
                                需要设备数量
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 16px; font-weight: 700; color: #262626;">
                                <span style="color: #fa8c16;">%d</span> <span style="font-size: 12px; color: #8c8c8c; font-weight: 400;">台</span>
                            </td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: middle;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #9254de; margin-right: 8px; border-radius: 2px;"></span>
                                创建人
                            </td>
                            <td style="padding: 12px 16px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 600; color: #262626;">
                                <span style="background-color: #f0f5ff; color: #1890ff; padding: 2px 8px; border-radius: 4px; font-size: 12px;">%s</span>
                            </td>
                        </tr>
                        <tr>
                            <td style="padding: 12px 16px; color: #8c8c8c; font-size: 13px; font-weight: 500; vertical-align: top; padding-top: 16px;">
                                <span style="display: inline-block; width: 4px; height: 16px; background-color: #13c2c2; margin-right: 8px; border-radius: 2px;"></span>
                                触发原因
                            </td>
                            <td style="padding: 12px 16px; font-size: 14px; line-height: 1.6; color: #595959; padding-top: 16px;">
                                <div style="background-color: #fafafa; padding: 12px; border-radius: 4px; border-left: 3px solid #13c2c2;">
                                    %s
                                </div>
                            </td>
                        </tr>`,
		dto.Name, actionColor, actionColor, actionColor, actionName, clusterName, dto.ResourcePoolType, dto.DeviceCount, dto.CreatedBy, dto.Description))

	builder.WriteString(`
                    </table>
                </div>
            </div>`)

	// 设备列表或设备不足提醒
	if len(devices) > 0 {
		builder.WriteString(`
            <!-- 涉及设备列表 -->
            <div style="margin-bottom: 24px;">
                <h3 style="color: #262626; margin: 0 0 16px 0; font-size: 18px; font-weight: 600; display: flex; align-items: center;">
                    <span style="margin-right: 8px;">🖥️</span>
                    <span>涉及设备列表</span>
                    <span style="background-color: #1890ff; color: white; padding: 2px 8px; border-radius: 12px; font-size: 12px; margin-left: 12px; font-weight: 500;">` + fmt.Sprintf("%d台", len(devices)) + `</span>
                </h3>
                <div style="background-color: #ffffff; border-radius: 8px; border: 1px solid #e8e8e8; overflow: hidden;">
                    <div style="overflow-x: auto;">
                        <table style="width: 100%; border-collapse: collapse;">
                            <thead>
                                <tr style="background-color: #fafafa;">
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">设备ID</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">CI编码</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">IP地址</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">CPU核心</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">内存(GB)</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">当前状态</th>
                                    <th style="padding: 12px 16px; text-align: left; font-weight: 600; color: #595959; font-size: 13px; border-bottom: 1px solid #e8e8e8;">所属集群</th>
                                </tr>
                            </thead>
                            <tbody>`)

		for i, d := range devices {
			// 交替背景色
			bgColor := "#ffffff"
			if i%2 == 1 {
				bgColor = "#fafafa"
			}

			// 设备状态样式 - 简化版本
			var statusBadge string
			switch d.Status {
			case "available":
				statusBadge = `<span style="background-color: #52c41a; color: white; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500;">可用</span>`
			case "in_use":
				statusBadge = `<span style="background-color: #ff7a45; color: white; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500;">使用中</span>`
			case "maintenance":
				statusBadge = `<span style="background-color: #722ed1; color: white; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500;">维护中</span>`
			default:
				statusBadge = `<span style="background-color: #8c8c8c; color: white; padding: 2px 8px; border-radius: 4px; font-size: 12px; font-weight: 500;">未知</span>`
			}

			builder.WriteString(fmt.Sprintf(`
                                <tr style="background-color: %s;">
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; font-weight: 600; color: #1890ff; font-family: monospace;">%d</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; font-family: monospace; font-weight: 500;">%s</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; font-family: monospace;">%s</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; font-weight: 600;">%.1f核</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; font-weight: 600;">%.1fGB</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8;">%s</td>
                                    <td style="padding: 12px 16px; border-bottom: 1px solid #e8e8e8; color: #595959;">%s</td>
                                </tr>`,
				bgColor,
				d.ID,
				d.CICode,
				d.IP,
				d.CPU,
				d.Memory/1024,
				statusBadge,
				d.Cluster))
		}

		builder.WriteString(`
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>`)

		// 操作指引
		var instructions string
		if actionName == actionNamePoolEntry {
			instructions = `
                    <li>登录设备管理系统，确认以上设备状态为可用</li>
                    <li>执行设备入池操作，将设备加入目标集群</li>
                    <li>验证设备在集群中的运行状态</li>
                    <li>更新订单状态为"已完成"</li>
                    <li>如遇问题，请及时联系技术支持团队</li>`
		} else {
			instructions = `
                    <li>确认集群负载低于阈值且适合执行退池操作</li>
                    <li>发维护邮件并提前对节点做cordon操作（16点前）</li>
                    <li>18点后执行设备的排空（drain）操作</li>
                    <li>将设备从集群中移除</li>
                    <li>视情况更新归还或无须归还设备状态</li>`
		}

		builder.WriteString(fmt.Sprintf(`
            <!-- 操作指引 -->
            <div style="background: linear-gradient(135deg, #1890ff 0%%, #0050b3 100%%); border-radius: 6px; padding: 20px; margin-bottom: 24px; color: white;">
                <h3 style="margin: 0 0 12px 0; font-size: 16px;">⚡ 处理指引</h3>
                <ul style="margin: 0; padding-left: 20px; line-height: 1.8;">%s
                </ul>
            </div>`, instructions))

		// 时间要求或安全提醒
		if actionName == actionNamePoolEntry {
			builder.WriteString(`
            <!-- 时间要求 -->
            <div style="border-left: 4px solid #52c41a; background-color: #f6ffed; padding: 16px; margin-bottom: 24px;">
                <h4 style="color: #389e0d; margin: 0 0 8px 0; font-size: 14px;">⏰ 时间要求</h4>
                <p style="color: #52c41a; font-size: 13px; line-height: 1.6; margin: 0;">
                    请在<strong>2小时内</strong>完成设备入池操作，以确保集群资源能够及时缓解负载压力。
                </p>
            </div>
            <!-- 重要提醒 -->
            <div style="border-left: 4px solid #ff4d4f; background-color: #fff2f0; padding: 16px; margin-bottom: 24px;">
                <h4 style="color: #cf1322; margin: 0 0 8px 0; font-size: 14px;">⚠️ 重要提醒</h4>
                <p style="color: #a8071a; font-size: 13px; line-height: 1.6; margin: 0;">
                    集群资源分配率已超过阈值，建议<strong>尽快完成入池操作</strong>以确保集群稳定运行。
                    请按照处理指引的步骤进行操作，如遇问题及时联系技术支持团队。
                </p>
            </div>`)
		} else {
			builder.WriteString(`
            <!-- 安全提醒 -->
            <div style="border-left: 4px solid #ff7a45; background-color: #fff2e8; padding: 16px; margin-bottom: 24px;">
                <h4 style="color: #d4380d; margin: 0 0 8px 0; font-size: 14px;">⚠️ 安全提醒</h4>
                <p style="color: #ff7a45; font-size: 13px; line-height: 1.6; margin: 0;">
                    请确保在执行退池操作前，<strong>充分验证工作负载已成功迁移</strong>，避免服务中断。建议在低峰期执行此操作。
                </p>
            </div>`)
		}
	} else {
		// 设备不足提醒
		builder.WriteString(`
            <!-- 设备不足提醒 -->
            <div style="border-left: 4px solid #ff4d4f; background-color: #fff2f0; padding: 20px; margin-bottom: 24px;">
                <h3 style="color: #cf1322; margin: 0 0 12px 0; font-size: 16px;">🚫 设备不足情况</h3>
                <div style="background-color: white; border-radius: 6px; padding: 16px; margin-bottom: 16px; text-align: center;">
                    <div style="font-size: 48px; color: #ff4d4f; margin-bottom: 16px;">⚠️</div>
                    <p style="font-size: 16px; color: #cf1322; font-weight: 600; margin: 0;">
                        未找到可用设备执行` + actionName + `操作
                    </p>
                </div>
                <p style="color: #a8071a; font-size: 13px; line-height: 1.6; margin: 0;">
                    <strong>注意：</strong> 本次操作未匹配到具体设备，请相关人员关注并手动处理。
                </p>
            </div>

            <!-- 处理指引 -->
            <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border-radius: 6px; padding: 20px; margin-bottom: 24px; color: white;">
                <h3 style="margin: 0 0 12px 0; font-size: 16px;">⚡ 处理指引</h3>
                <ul style="margin: 0; padding-left: 20px; line-height: 1.8;">
                    <li>检查集群中可用设备状态，确认是否有可调配的设备</li>
                    <li>联系设备管理团队，申请新的可用设备</li>
                    <li>评估是否可以从其他集群调配设备资源</li>
                    <li>考虑临时增加虚拟机节点作为过渡方案</li>
                    <li>如无法及时获得设备，评估是否可以忽略此次扩容需求</li>
                    <li>完成设备协调后，请手动创建入池订单或重新触发策略评估</li>
                </ul>
            </div>

            <!-- 重要提醒 -->
            <div style="border-left: 4px solid #ff4d4f; background-color: #fff2f0; padding: 16px; margin-bottom: 24px;">
                <h4 style="color: #cf1322; margin: 0 0 8px 0; font-size: 14px;">⚠️ 重要提醒</h4>
                <p style="color: #a8071a; font-size: 13px; line-height: 1.6; margin: 0;">
                    集群资源分配率已连续超过阈值，建议<strong>尽快协调设备资源</strong>以避免性能问题。
                    如短期内无法获得设备，请考虑其他优化措施或临时扩容方案。
                </p>
            </div>`)
	}

	// 联系信息
	builder.WriteString(`
            <!-- 联系信息 -->
            <div style="text-align: center; padding: 20px; background-color: #f8f9fa; border-radius: 6px;">
                <p style="color: #666; font-size: 13px; margin: 0 0 8px 0;">
                    如有疑问，请联系技术支持团队：support@company.com | 紧急热线：400-800-8888
                </p>
                <p style="color: #999; font-size: 12px; margin: 0;">
                    此邮件由弹性伸缩系统自动发送，请勿直接回复
                </p>
            </div>
        </div>`)

	// 邮件底部
	builder.WriteString(fmt.Sprintf(`
        <!-- 邮件底部 -->
        <div style="background-color: #f1f3f4; padding: 16px 32px; border-radius: 0 0 8px 8px; text-align: center;">
            <p style="color: #666; font-size: 12px; margin: 0;">
                © 2024 弹性伸缩管理系统 | 发送时间：%s
            </p>
				</div>
			</div>
		</body>
</html>`, time.Now().Format("2006-01-02 15:04:05")))

	return builder.String()
}

// generateOrderName generates a descriptive name for the order.
func (s *ElasticScalingService) generateOrderName(strategy *portal.ElasticScalingStrategy, deviceCount int) string {
	actionStr := "扩容"
	if strategy.ThresholdTriggerAction == TriggerActionPoolExit {
		actionStr = "缩容"
	}
	return fmt.Sprintf("弹性%s-%s-%s", actionStr, strategy.Name, time.Now().Format("20060102-1504"))
}

// generateOrderDescription generates a detailed description for the order.
func (s *ElasticScalingService) generateOrderDescription(
	strategy *portal.ElasticScalingStrategy,
	clusterID int64,
	resourceType string,
	selectedDeviceIDs []int64,
	latestSnapshot *portal.ResourceSnapshot,
) string {
	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := "未知集群"
	if err := s.db.Select("clustername").First(&cluster, clusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	actionName := s.getActionName(strategy.ThresholdTriggerAction)
	baseDescription := fmt.Sprintf("策略 [%s] 为集群 [%s]（%s类型）触发%s操作。",
		strategy.Name, clusterName, resourceType, actionName)

	if len(selectedDeviceIDs) == 0 {
		return baseDescription + "\n但未匹配到合适设备，请关注。"
	}

	// 如果没有快照信息，无法计算预测值，返回基础描述
	if latestSnapshot == nil {
		return fmt.Sprintf("%s\n匹配到 %d 台设备。", baseDescription, len(selectedDeviceIDs))
	}

	// 获取匹配到的设备的总资源
	var devices []portal.Device
	if err := s.db.Where("id IN ?", selectedDeviceIDs).Find(&devices).Error; err != nil {
		s.logger.Error("Failed to fetch selected devices for description", zap.Error(err))
		return fmt.Sprintf("%s\n匹配到 %d 台设备，但获取设备详情失败。", baseDescription, len(selectedDeviceIDs))
	}

	var totalCPU, totalMemory float64
	for _, d := range devices {
		totalCPU += d.CPU
		totalMemory += d.Memory
	}

	// 计算新的分配率
	currentCPUAllocation := safePercentage(latestSnapshot.CpuRequest, latestSnapshot.CpuCapacity)
	currentMemAllocation := safePercentage(latestSnapshot.MemRequest, latestSnapshot.MemoryCapacity)
	newCPUAllocationRate, newMemAllocationRate := s.calculateProjectedAllocation(latestSnapshot, totalCPU, totalMemory, strategy.ThresholdTriggerAction)

	var changeVerb, direction string
	if strategy.ThresholdTriggerAction == TriggerActionPoolEntry {
		changeVerb = "降低"
		direction = "至"
	} else {
		changeVerb = "提升"
		direction = "至"
	}

	projectionDescription := fmt.Sprintf("\n匹配到 %d 台设备（总CPU: %.1f, 总内存: %.1f GB）。\n预计操作后：\n- CPU分配率将由 %.2f%% %s %s %.2f%%\n- 内存分配率将由 %.2f%% %s %s %.2f%%",
		len(selectedDeviceIDs), totalCPU, totalMemory/1024,
		currentCPUAllocation, changeVerb, direction, newCPUAllocationRate,
		currentMemAllocation, changeVerb, direction, newMemAllocationRate)

	return baseDescription + projectionDescription
}

// calculateProjectedAllocation calculates the projected resource allocation rates after the scaling action.
func (s *ElasticScalingService) calculateProjectedAllocation(snapshot *portal.ResourceSnapshot, deviceTotalCPU, deviceTotalMemory float64, action string) (cpuRate float64, memRate float64) {
	currentCPURequest := snapshot.CpuRequest
	currentMemRequest := snapshot.MemRequest
	currentCPUCapacity := snapshot.CpuCapacity
	currentMemCapacity := snapshot.MemoryCapacity

	var newCPUCapacity, newMemCapacity float64

	if action == TriggerActionPoolEntry {
		newCPUCapacity = currentCPUCapacity + deviceTotalCPU
		newMemCapacity = currentMemCapacity + deviceTotalMemory
	} else { // TriggerActionPoolExit
		newCPUCapacity = currentCPUCapacity - deviceTotalCPU
		newMemCapacity = currentMemCapacity - deviceTotalMemory
	}

	// 使用 safePercentage 函数安全地计算百分比
	cpuRate = safePercentage(currentCPURequest, newCPUCapacity)
	memRate = safePercentage(currentMemRequest, newMemCapacity)

	return cpuRate, memRate
}
