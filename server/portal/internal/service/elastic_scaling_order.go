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
	if err := s.db.Where("order_detail_id = ?", detail.ID).Find(&orderDevices).Error; err != nil {
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
		Joins("JOIN elastic_scaling_order_details d ON o.id = d.order_id").
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
				zap.Int64p(zapKeyStrategyID, detail.StrategyID),
				zap.Error(errRecord))
			// 不返回错误，因为主操作（更新订单状态）已成功
		}
	}

	return nil
}

// GetOrderDevices 获取订单关联的设备
func (s *ElasticScalingService) GetOrderDevices(orderID int64) ([]DeviceDTO, error) {
	var detail portal.ElasticScalingOrderDetail
	if err := s.db.Where(fieldOrderID, orderID).First(&detail).Error; err != nil {
		return nil, fmt.Errorf(errFailedToFindOrderDetail, orderID, err)
	}

	var orderDevices []portal.OrderDevice
	if err := s.db.Where("order_detail_id = ?", detail.ID).Find(&orderDevices).Error; err != nil {
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
		StatusPending:                        true,
		string(portal.OrderStatusProcessing): true,
		StatusCompleted:                      true,
		StatusFailed:                         true,
	}

	if !validStatuses[status] {
		return fmt.Errorf(errInvalidDeviceStatus, status)
	}

	// First, find the order detail ID from the order ID
	var detail portal.ElasticScalingOrderDetail
	if err := s.db.Where(fieldOrderID, orderID).First(&detail).Error; err != nil {
		return fmt.Errorf(errFailedToFindOrderDetail, orderID, err)
	}

	var orderDevice portal.OrderDevice
	// Now use the order_detail_id
	result := s.db.Where("order_detail_id = ? AND device_id = ?", detail.ID, deviceID).First(&orderDevice)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New(errDeviceNotInOrder)
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

// generateOrderNotificationEmail 生成订单通知邮件正文
func (s *ElasticScalingService) generateOrderNotificationEmail(orderID int64, dto OrderDTO) string {
	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := unknownCluster
	if err := s.db.Select(fieldClusterName).First(&cluster, dto.ClusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	// 获取设备信息
	deviceInfo := s.getDeviceInfoForEmail(dto.Devices)

	// 确定变更工作类型
	actionName := s.getActionName(dto.ActionType)

	// 生成邮件主题
	subject := fmt.Sprintf(emailSubjectTemplate, actionName, fmt.Sprintf("ESO%d", orderID))

	// 生成HTML邮件正文
	emailContent := s.buildEmailHTML(subject, actionName, clusterName, dto, deviceInfo)

	return emailContent
}

// getActionName 根据动作类型获取中文名称
func (s *ElasticScalingService) getActionName(actionType string) string {
	switch actionType {
	case actionTypePoolEntry:
		return actionNamePoolEntry
	case actionTypePoolExit:
		return actionNamePoolExit
	case actionTypeMaintenanceRequest:
		return "维护申请"
	case actionTypeMaintenanceUncordon:
		return "维护解除"
	default:
		return "未知操作"
	}
}

// getDeviceInfoForEmail 获取设备信息用于邮件显示
func (s *ElasticScalingService) getDeviceInfoForEmail(deviceIDs []int64) []DeviceDTO {
	if len(deviceIDs) == 0 {
		return []DeviceDTO{}
	}

	var devices []portal.Device
	if err := s.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		s.logger.Error("Failed to get device info for email", zap.Error(err))
		return []DeviceDTO{}
	}

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
		}
	}

	return result
}

// buildEmailHTML 构建HTML格式的邮件正文
func (s *ElasticScalingService) buildEmailHTML(subject, actionName, clusterName string, dto OrderDTO, devices []DeviceDTO) string {
	// 获取当前时间
	now := time.Now()
	currentTime := now.Format("2006-01-02 15:04:05")

	// 检查是否为无设备情况
	isNoDevicesSituation := len(devices) == 0 && dto.DeviceCount == 0

	// 构建设备信息表格
	deviceTableRows := ""
	if len(devices) > 0 {
		for _, device := range devices {
			deviceTableRows += fmt.Sprintf(`
				<tr>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%s</td>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%s</td>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%s</td>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%.1f核</td>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%.1fGB</td>
					<td style="padding: 8px; border: 1px solid #e0e0e0; text-align: center;">%s</td>
				</tr>`,
				device.CICode, device.IP, device.ArchType, device.CPU, device.Memory, device.Status)
		}
	} else if isNoDevicesSituation {
		// 无设备情况的特殊提醒
		deviceTableRows = `
			<tr>
				<td colspan="6" style="padding: 16px; border: 1px solid #e0e0e0; text-align: center; color: #ff4d4f; font-weight: bold;">
					⚠️ 找不到要处理的设备，请自行协调处理
				</td>
			</tr>`
	} else {
		deviceTableRows = `
			<tr>
				<td colspan="6" style="padding: 16px; border: 1px solid #e0e0e0; text-align: center; color: #666;">
					设备数量：%d 台（具体设备信息请查看系统详情）
				</td>
			</tr>`
		deviceTableRows = fmt.Sprintf(deviceTableRows, dto.DeviceCount)
	}

	// 确定操作颜色和图标
	actionColor := "#1890ff"
	actionIcon := "🔄"
	if isNoDevicesSituation {
		// 无设备情况使用警告颜色
		actionColor = "#ff7a45"
		actionIcon = "⚠️"
	} else if dto.ActionType == actionTypePoolEntry {
		actionColor = "#52c41a"
		actionIcon = "⬆️"
	} else if dto.ActionType == actionTypePoolExit {
		actionColor = "#ff7a45"
		actionIcon = "⬇️"
	}

	// 构建完整的HTML邮件
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
</head>
<body style="margin: 0; padding: 20px; font-family: 'Microsoft YaHei', Arial, sans-serif; background-color: #f5f7fa;">
    <div style="max-width: 800px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
        <!-- 邮件头部 -->
        <div style="background: linear-gradient(135deg, %s 0%%, %s 100%%); color: white; padding: 24px; border-radius: 8px 8px 0 0;">
            <h1 style="margin: 0; font-size: 24px; font-weight: 600;">
                %s %s变更通知%s
            </h1>
            <p style="margin: 8px 0 0 0; font-size: 14px; opacity: 0.9;">
                订单号：%s | 创建时间：%s
            </p>
        </div>`,
		subject, actionColor, actionColor, actionIcon, actionName,
		func() string {
			if isNoDevicesSituation {
				return "（设备不足）"
			}
			return ""
		}(), dto.OrderNumber, currentTime)

	// 构建问候语内容
	greetingContent := ""
	if isNoDevicesSituation {
		greetingContent = fmt.Sprintf(`系统检测到集群资源需要进行<strong style="color: %s;">%s</strong>变更操作，但<strong style="color: #ff4d4f;">无法找到可用设备</strong>，请协调处理相关工作。`, actionColor, actionName)
	} else {
		greetingContent = fmt.Sprintf(`系统检测到集群资源需要进行<strong style="color: %s;">%s</strong>变更操作，请及时处理相关工作。`, actionColor, actionName)
	}

	// 继续构建邮件正文
	htmlContent += fmt.Sprintf(`
        <!-- 邮件正文 -->
        <div style="padding: 32px;">
            <!-- 问候语 -->
            <div style="margin-bottom: 24px;">
                <h2 style="color: #333; font-size: 18px; margin: 0 0 12px 0;">👋 值班同事，您好！</h2>
                <p style="color: #666; font-size: 14px; line-height: 1.6; margin: 0;">
                    %s
                </p>
            </div>

            <!-- 变更详情 -->
            <div style="background-color: #f8f9fa; border-radius: 6px; padding: 20px; margin-bottom: 24px;">
                <h3 style="color: #333; font-size: 16px; margin: 0 0 16px 0;">📋 变更详情</h3>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
                    <div>
                        <span style="color: #666; font-size: 13px;">目标集群：</span>
                        <strong style="color: #333; font-size: 14px;">%s</strong>
                    </div>
                    <div>
                        <span style="color: #666; font-size: 13px;">变更类型：</span>
                        <strong style="color: %s; font-size: 14px;">%s</strong>
                    </div>
                    <div>
                        <span style="color: #666; font-size: 13px;">%s：</span>
                        <strong style="color: %s; font-size: 14px;">%d 台</strong>
                    </div>
                    <div>
                        <span style="color: #666; font-size: 13px;">创建人：</span>
                        <strong style="color: #333; font-size: 14px;">%s</strong>
                    </div>
                </div>
            </div>`,
		greetingContent, clusterName, actionColor, actionName,
		func() string {
			if isNoDevicesSituation {
				return "需要设备"
			}
			return "设备数量"
		}(),
		func() string {
			if isNoDevicesSituation {
				return "#ff4d4f"
			}
			return "#333"
		}(),
		func() int {
			if isNoDevicesSituation && dto.ActionType == actionTypePoolEntry {
				// 对于入池操作，显示策略要求的设备数量
				return 2 // 这里应该从策略中获取，暂时硬编码
			}
			return dto.DeviceCount
		}(),
		dto.CreatedBy)

	// 添加无设备情况的特殊提醒
	if isNoDevicesSituation {
		htmlContent += `
            <!-- 设备不足提醒 -->
            <div style="border-left: 4px solid #ff4d4f; background-color: #fff2f0; padding: 20px; margin-bottom: 24px;">
                <h3 style="color: #cf1322; margin: 0 0 12px 0; font-size: 16px;">🚫 设备不足情况</h3>
                <p style="color: #a8071a; font-size: 13px; line-height: 1.6; margin: 0;">
                    <strong>找不到要处理的设备，请自行协调处理。</strong>建议联系设备管理团队申请新设备或调整现有设备状态。
                </p>
            </div>

            <!-- 处理指引 -->
            <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border-radius: 6px; padding: 20px; margin-bottom: 24px; color: white;">
                <h3 style="margin: 0 0 12px 0; font-size: 16px;">⚡ 处理指引</h3>
                <ul style="margin: 0; padding-left: 20px; line-height: 1.8;">
                    <li>联系设备管理团队申请新的可用设备</li>
                    <li>检查现有设备状态，评估是否可以调整为可用状态</li>
                    <li>考虑从其他集群调配设备资源</li>
                    <li>如无法及时获得设备，可选择忽略此次扩容需求</li>
                    <li>完成设备协调后，请手动创建订单或重新触发策略评估</li>
                </ul>
            </div>

            <!-- 重要提醒 -->
            <div style="border-left: 4px solid #ff4d4f; background-color: #fff2f0; padding: 16px; margin-bottom: 24px;">
                <h4 style="color: #cf1322; margin: 0 0 8px 0; font-size: 14px;">⚠️ 重要提醒</h4>
                <p style="color: #a8071a; font-size: 13px; line-height: 1.6; margin: 0;">
                    集群资源使用率已连续超过阈值，建议<strong>尽快协调设备资源</strong>以避免性能问题。
                    如短期内无法获得设备，请考虑其他优化措施或临时扩容方案。
                </p>
            </div>`
	} else {
		// 正常情况下的设备信息
		htmlContent += fmt.Sprintf(`
            <!-- 设备信息 -->
            <div style="margin-bottom: 24px;">
                <h3 style="color: #333; font-size: 16px; margin: 0 0 16px 0;">🖥️ 涉及设备</h3>
                <div style="overflow-x: auto;">
                    <table style="width: 100%%; border-collapse: collapse; background-color: white; border-radius: 6px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1);">
                        <thead>
                            <tr style="background-color: #f1f3f4;">
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">CI编码</th>
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">IP地址</th>
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">架构</th>
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">CPU</th>
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">内存</th>
                                <th style="padding: 12px 8px; border: 1px solid #e0e0e0; font-size: 13px; font-weight: 600; color: #333;">状态</th>
                            </tr>
                        </thead>
                        <tbody>
                            %s
                        </tbody>
                    </table>
                </div>
            </div>`, deviceTableRows)
	}

	// 添加邮件底部
	htmlContent += `
            <!-- 联系信息 -->
            <div style="text-align: center; padding: 20px; background-color: #f8f9fa; border-radius: 6px;">
                <p style="color: #666; font-size: 13px; margin: 0 0 8px 0;">
                    如有疑问，请联系设备管理团队或系统管理员
                </p>
                <p style="color: #999; font-size: 12px; margin: 0;">
                    此邮件由弹性伸缩系统自动发送，请勿直接回复
                </p>
            </div>
        </div>

        <!-- 邮件底部 -->
        <div style="background-color: #f1f3f4; padding: 16px 32px; border-radius: 0 0 8px 8px; text-align: center;">
            <p style="color: #666; font-size: 12px; margin: 0;">
                © 2024 弹性伸缩管理系统 | 发送时间：` + currentTime + `
            </p>
        </div>
    </div>
</body>
</html>`

	return htmlContent
}

// generateOrderName 生成订单名称
func (s *ElasticScalingService) generateOrderName(strategy *portal.ElasticScalingStrategy, deviceCount int) string {
	actionName := s.getActionName(strategy.ThresholdTriggerAction)

	if deviceCount == 0 {
		return fmt.Sprintf("%s变更提醒（设备不足）", actionName)
	}

	return fmt.Sprintf("%s变更订单", actionName)
}

// generateOrderDescription 生成订单描述
func (s *ElasticScalingService) generateOrderDescription(strategy *portal.ElasticScalingStrategy, clusterID int64, resourceType string, deviceCount int) string {
	// 获取集群名称
	var cluster portal.K8sCluster
	clusterName := unknownCluster
	if err := s.db.Select(fieldClusterName).First(&cluster, clusterID).Error; err == nil {
		clusterName = cluster.ClusterName
	}

	actionName := s.getActionName(strategy.ThresholdTriggerAction)

	if deviceCount == 0 {
		return fmt.Sprintf("策略 '%s' 触发%s操作，但无法找到可用设备。集群：%s，资源类型：%s。请协调处理设备资源。",
			strategy.Name, actionName, clusterName, resourceType)
	}

	return fmt.Sprintf("策略 '%s' 触发%s操作。集群：%s，资源类型：%s，涉及设备：%d台。",
		strategy.Name, actionName, clusterName, resourceType, deviceCount)
}
