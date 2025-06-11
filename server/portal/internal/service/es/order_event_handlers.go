package es

import (
	"context"
	"fmt"
	"navy-ng/models/portal"
	"navy-ng/server/portal/internal/service/events"
	"time"

	"go.uber.org/zap"
)

// OrderEventHandler 订单事件处理器
type OrderEventHandler struct {
	orderService *ElasticScalingService
	logger       *zap.Logger
}

// NewOrderEventHandler 创建订单事件处理器
func NewOrderEventHandler(orderService *ElasticScalingService, logger *zap.Logger) *OrderEventHandler {
	return &OrderEventHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// RegisterEventHandlers 注册所有订单相关的事件处理器（使用泛型方式）
func (s *ElasticScalingService) RegisterEventHandlers(eventManager *events.EventManager) {
	handler := NewOrderEventHandler(s, s.logger)

	// 注册设备操作完成事件处理器
	events.RegisterGenericFunc(eventManager, events.RegisterGenericFuncRequest[events.DeviceEventData]{
		EventType:   events.EventTypeDeviceOperationCompleted,
		HandlerName: "order_device_operation_completed_handler",
		HandlerFunc: handler.HandleDeviceOperationCompletedGeneric,
	})

	// 注册设备操作失败事件处理器
	events.RegisterGenericFunc(eventManager, events.RegisterGenericFuncRequest[events.DeviceEventData]{
		EventType:   events.EventTypeDeviceOperationFailed,
		HandlerName: "order_device_operation_failed_handler",
		HandlerFunc: handler.HandleDeviceOperationFailedGeneric,
	})

	// 注册订单取消事件处理器
	events.RegisterGenericFunc(eventManager, events.RegisterGenericFuncRequest[events.OrderEventData]{
		EventType:   events.EventTypeOrderCancelled,
		HandlerName: "order_cancelled_handler",
		HandlerFunc: handler.HandleOrderCancelledGeneric,
	})

	// 注册设备归还事件处理器
	events.RegisterGenericFunc(eventManager, events.RegisterGenericFuncRequest[events.DeviceEventData]{
		EventType:   "device.returning",
		HandlerName: "device_returning_handler",
		HandlerFunc: handler.HandleDeviceReturningGeneric,
	})

	// 注册订单处理完成事件处理器
	events.RegisterGenericFunc(eventManager, events.RegisterGenericFuncRequest[events.OrderEventData]{
		EventType:   "order.processing.completed",
		HandlerName: "order_processing_completed_handler",
		HandlerFunc: handler.HandleOrderProcessingCompletedGeneric,
	})

	s.logger.Info("Order event handlers registered successfully using generic registration")
}

// 泛型事件处理器方法

// HandleDeviceOperationCompletedGeneric 处理设备操作完成事件（泛型版本）
func (h *OrderEventHandler) HandleDeviceOperationCompletedGeneric(ctx context.Context, event *events.GenericEvent[events.DeviceEventData]) error {
	deviceData := event.EventData

	h.logger.Info("Handling device operation completed event (generic)",
		zap.Int("orderID", deviceData.OrderID),
		zap.Int("deviceID", deviceData.DeviceID),
		zap.String("action", deviceData.Action),
		zap.String("result", deviceData.Result))

	// 更新订单中设备的状态
	err := h.orderService.UpdateOrderDeviceStatus(deviceData.OrderID, deviceData.DeviceID, StatusSuccess)
	if err != nil {
		h.logger.Error("Failed to update order device status",
			zap.Int("orderID", deviceData.OrderID),
			zap.Int("deviceID", deviceData.DeviceID),
			zap.Error(err))
		return err
	}

	// 检查订单中所有设备是否都已完成
	allCompleted, err := h.checkAllDevicesCompleted(deviceData.OrderID)
	if err != nil {
		h.logger.Error("Failed to check if all devices completed",
			zap.Int("orderID", deviceData.OrderID),
			zap.Error(err))
		return err
	}

	// 如果所有设备都已完成，更新订单状态为已完成
	if allCompleted {
		reason := fmt.Sprintf("所有设备操作已完成 - %s", deviceData.Action)
		err = h.orderService.UpdateOrderStatus(
			deviceData.OrderID,
			string(portal.OrderStatusCompleted),
			"system",
			reason,
		)
		if err != nil {
			h.logger.Error("Failed to update order status to completed",
				zap.Int("orderID", deviceData.OrderID),
				zap.Error(err))
			return err
		}

		h.logger.Info("Order completed - all devices finished",
			zap.Int("orderID", deviceData.OrderID))
	}

	return nil
}

// HandleDeviceOperationFailedGeneric 处理设备操作失败事件（泛型版本）
func (h *OrderEventHandler) HandleDeviceOperationFailedGeneric(ctx context.Context, event *events.GenericEvent[events.DeviceEventData]) error {
	deviceData := event.EventData

	h.logger.Warn("Handling device operation failed event (generic)",
		zap.Int("orderID", deviceData.OrderID),
		zap.Int("deviceID", deviceData.DeviceID),
		zap.String("action", deviceData.Action),
		zap.String("error", deviceData.ErrorMsg))

	// 更新订单中设备的状态为失败
	err := h.orderService.UpdateOrderDeviceStatus(deviceData.OrderID, deviceData.DeviceID, "failed")
	if err != nil {
		h.logger.Error("Failed to update order device status to failed",
			zap.Int("orderID", deviceData.OrderID),
			zap.Int("deviceID", deviceData.DeviceID),
			zap.Error(err))
		return err
	}

	// 检查是否需要标记整个订单为失败
	shouldFailOrder, err := h.shouldFailOrder(deviceData.OrderID)
	if err != nil {
		h.logger.Error("Failed to determine if order should fail",
			zap.Int("orderID", deviceData.OrderID),
			zap.Error(err))
		return err
	}

	if shouldFailOrder {
		reason := fmt.Sprintf("设备操作失败 - 设备ID: %d, 错误: %s", deviceData.DeviceID, deviceData.ErrorMsg)
		err = h.orderService.UpdateOrderStatus(
			deviceData.OrderID,
			string(portal.OrderStatusFailed),
			"system",
			reason,
		)
		if err != nil {
			h.logger.Error("Failed to update order status to failed",
				zap.Int("orderID", deviceData.OrderID),
				zap.Error(err))
			return err
		}

		h.logger.Error("Order failed due to device operation failure",
			zap.Int("orderID", deviceData.OrderID),
			zap.Int("deviceID", deviceData.DeviceID))
	}

	return nil
}

// HandleOrderCancelledGeneric 处理订单取消事件（泛型版本）
func (h *OrderEventHandler) HandleOrderCancelledGeneric(ctx context.Context, event *events.GenericEvent[events.OrderEventData]) error {
	orderData := event.EventData

	h.logger.Info("Handling order cancelled event (generic)",
		zap.Int("orderID", orderData.OrderID),
		zap.String("description", orderData.Description))

	// 取消所有正在进行的设备操作
	err := h.cancelAllDeviceOperations(orderData.OrderID, orderData.Description)
	if err != nil {
		h.logger.Error("Failed to cancel device operations",
			zap.Int("orderID", orderData.OrderID),
			zap.Error(err))
		return err
	}

	// 发布弹性伸缩取消事件
	// TODO: 可以使用泛型事件发布
	h.logger.Info("Order cancellation handled successfully",
		zap.Int("orderID", orderData.OrderID))

	return nil
}

// HandleDeviceReturningGeneric 处理设备归还事件（泛型版本）
func (h *OrderEventHandler) HandleDeviceReturningGeneric(ctx context.Context, event *events.GenericEvent[events.DeviceEventData]) error {
	deviceData := event.EventData

	h.logger.Info("Handling device returning event (generic)",
		zap.Int("orderID", deviceData.OrderID),
		zap.Int("deviceID", deviceData.DeviceID))

	// 更新设备状态为归还中
	err := h.orderService.UpdateOrderDeviceStatus(deviceData.OrderID, deviceData.DeviceID, "returning")
	if err != nil {
		h.logger.Error("Failed to update device status to returning",
			zap.Int("orderID", deviceData.OrderID),
			zap.Int("deviceID", deviceData.DeviceID),
			zap.Error(err))
		return err
	}

	// 检查该订单的所有设备是否都在归还中
	allReturning, err := h.checkAllDevicesReturning(deviceData.OrderID)
	if err != nil {
		h.logger.Error("Failed to check if all devices returning",
			zap.Int("orderID", deviceData.OrderID),
			zap.Error(err))
		return err
	}

	if allReturning {
		// 更新订单状态为归还中
		err = h.orderService.UpdateOrderStatus(
			deviceData.OrderID,
			"returning",
			"system",
			"所有设备开始归还流程",
		)
		if err != nil {
			h.logger.Error("Failed to update order status to returning",
				zap.Int("orderID", deviceData.OrderID),
				zap.Error(err))
			return err
		}

		// 开始设备回收流程
		err = h.checkAndStartDeviceReclaim(deviceData.OrderID)
		if err != nil {
			h.logger.Error("Failed to start device reclaim",
				zap.Int("orderID", deviceData.OrderID),
				zap.Error(err))
			return err
		}

		h.logger.Info("All devices returning, order status updated",
			zap.Int("orderID", deviceData.OrderID))
	}

	return nil
}

// HandleOrderProcessingCompletedGeneric 处理订单处理完成事件（泛型版本）
func (h *OrderEventHandler) HandleOrderProcessingCompletedGeneric(ctx context.Context, event *events.GenericEvent[events.OrderEventData]) error {
	orderData := event.EventData

	h.logger.Info("Handling order processing completed event (generic)",
		zap.Int("orderID", orderData.OrderID))

	// 检查订单类型，执行相应的后处理逻辑
	if orderData.OrderType == "pool_exit" {
		// 对于pool_exit订单，可能需要启动设备回收流程
		err := h.checkAndStartDeviceReclaim(orderData.OrderID)
		if err != nil {
			h.logger.Error("Failed to start device reclaim for pool_exit order",
				zap.Int("orderID", orderData.OrderID),
				zap.Error(err))
			return err
		}
	}

	h.logger.Info("Order processing completed handling finished",
		zap.Int("orderID", orderData.OrderID))

	return nil
}

// HandleMaintenanceCompleted 处理维护完成事件
func (h *OrderEventHandler) HandleMaintenanceCompleted(ctx context.Context, event events.Event) error {
	maintenanceEvent, ok := event.Data().(*events.MaintenanceEvent)
	if !ok {
		return fmt.Errorf("invalid event data type for maintenance completed event")
	}

	h.logger.Info("Handling maintenance completed event",
		zap.Int("orderID", maintenanceEvent.OrderID),
		zap.Int("deviceID", maintenanceEvent.DeviceID),
		zap.String("maintenanceType", maintenanceEvent.MaintenanceType))

	// 根据维护类型决定订单状态更新逻辑
	switch maintenanceEvent.MaintenanceType {
	case "cordon":
		// Cordon完成，更新订单状态为处理中
		err := h.orderService.UpdateOrderStatus(
			maintenanceEvent.OrderID,
			string(portal.OrderStatusProcessing),
			"system",
			"设备已成功cordon，开始执行后续操作",
		)
		if err != nil {
			return err
		}

	case "drain":
		// Drain完成，继续等待uncordon或其他操作
		h.logger.Info("Device drain completed, waiting for next step",
			zap.Int("orderID", maintenanceEvent.OrderID),
			zap.Int("deviceID", maintenanceEvent.DeviceID))

	case "uncordon":
		// Uncordon完成，标记维护订单完成
		err := h.orderService.UpdateOrderStatus(
			maintenanceEvent.OrderID,
			string(portal.OrderStatusCompleted),
			"system",
			"设备维护完成，已成功uncordon",
		)
		if err != nil {
			return err
		}

		h.logger.Info("Maintenance order completed",
			zap.Int("orderID", maintenanceEvent.OrderID),
			zap.Int("deviceID", maintenanceEvent.DeviceID))
	}

	return nil
}

// HandleMaintenanceFailed 处理维护失败事件
func (h *OrderEventHandler) HandleMaintenanceFailed(ctx context.Context, event events.Event) error {
	maintenanceEvent, ok := event.Data().(*events.MaintenanceEvent)
	if !ok {
		return fmt.Errorf("invalid event data type for maintenance failed event")
	}

	h.logger.Error("Handling maintenance failed event",
		zap.Int("orderID", maintenanceEvent.OrderID),
		zap.Int("deviceID", maintenanceEvent.DeviceID),
		zap.String("maintenanceType", maintenanceEvent.MaintenanceType),
		zap.String("error", maintenanceEvent.Error))

	// 维护失败，标记订单失败
	reason := fmt.Sprintf("维护操作失败 - 类型: %s, 设备ID: %d, 错误: %s",
		maintenanceEvent.MaintenanceType, maintenanceEvent.DeviceID, maintenanceEvent.Error)

	err := h.orderService.UpdateOrderStatus(
		maintenanceEvent.OrderID,
		string(portal.OrderStatusFailed),
		"system",
		reason,
	)
	if err != nil {
		h.logger.Error("Failed to update order status to failed after maintenance failure",
			zap.Int("orderID", maintenanceEvent.OrderID),
			zap.Error(err))
		return err
	}

	return nil
}

// cancelAllDeviceOperations 取消订单中所有设备的操作
func (h *OrderEventHandler) cancelAllDeviceOperations(orderID int, reason string) error {
	devices, err := h.orderService.GetOrderDevices(orderID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		// 如果设备操作还在进行中，则取消
		if device.OrderStatus == "executing" || device.OrderStatus == "pending" {
			err = h.orderService.UpdateOrderDeviceStatus(orderID, device.ID, "cancelled")
			if err != nil {
				h.logger.Error("Failed to cancel device operation",
					zap.Int("orderID", orderID),
					zap.Int("deviceID", device.ID),
					zap.Error(err))
				continue // 继续取消其他设备
			}

			h.logger.Info("Device operation cancelled",
				zap.Int("orderID", orderID),
				zap.Int("deviceID", device.ID),
				zap.String("reason", reason))
		}
	}

	return nil
}

// startRollbackProcess 启动回滚流程
func (h *OrderEventHandler) startRollbackProcess(orderID int, rollbackActionType, reason string) error {
	// 更新订单状态为回滚中
	err := h.orderService.UpdateOrderStatus(
		orderID,
		string(portal.OrderStatusReturning),
		"system",
		fmt.Sprintf("开始回滚操作: %s, 原因: %s", rollbackActionType, reason),
	)
	if err != nil {
		return err
	}

	// 获取需要回滚的设备列表
	devices, err := h.orderService.GetOrderDevices(orderID)
	if err != nil {
		return err
	}

	// 为每个设备启动回滚操作
	for _, device := range devices {
		// 只对已成功的设备进行回滚
		if device.OrderStatus == "success" {
			// 发布设备回滚事件，让设备服务处理具体的回滚逻辑
			err = h.publishDeviceRollbackEvent(orderID, device.ID, rollbackActionType)
			if err != nil {
				h.logger.Error("Failed to publish device rollback event",
					zap.Int("orderID", orderID),
					zap.Int("deviceID", device.ID),
					zap.Error(err))
				continue
			}

			// 更新设备状态为回滚中
			err = h.orderService.UpdateOrderDeviceStatus(orderID, device.ID, "rollback_executing")
			if err != nil {
				h.logger.Error("Failed to update device status to rollback_executing",
					zap.Int("orderID", orderID),
					zap.Int("deviceID", device.ID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// publishDeviceRollbackEvent 发布设备回滚事件
func (h *OrderEventHandler) publishDeviceRollbackEvent(orderID, deviceID int, rollbackActionType string) error {
	// 发布设备回滚事件，让设备服务处理具体的回滚逻辑
	h.logger.Info("Device rollback event published",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID),
		zap.String("rollbackActionType", rollbackActionType))

	// 这里可以使用全局事件管理器发布事件
	// 具体的实现取决于如何获取全局事件管理器实例
	return nil
}

// checkAllDevicesCompleted 检查订单中所有设备是否都已完成
func (h *OrderEventHandler) checkAllDevicesCompleted(orderID int) (bool, error) {
	devices, err := h.orderService.GetOrderDevices(orderID)
	if err != nil {
		return false, err
	}

	for _, device := range devices {
		// 如果有设备不是成功状态，则订单未完成
		if device.OrderStatus != "success" {
			return false, nil
		}
	}

	return len(devices) > 0, nil // 至少有一个设备且都成功
}

// shouldFailOrder 判断是否应该标记订单失败
func (h *OrderEventHandler) shouldFailOrder(orderID int) (bool, error) {
	devices, err := h.orderService.GetOrderDevices(orderID)
	if err != nil {
		return false, err
	}

	failedCount := 0
	for _, device := range devices {
		if device.OrderStatus == "failed" {
			failedCount++
		}
	}

	// 业务规则：如果超过50%的设备失败，或者只有一个设备且失败了，则标记订单失败
	if len(devices) == 1 && failedCount == 1 {
		return true, nil
	}

	if len(devices) > 1 && float64(failedCount)/float64(len(devices)) > 0.5 {
		return true, nil
	}

	return false, nil
}

// checkAllDevicesReturning 检查订单中所有设备是否都在归还中
func (h *OrderEventHandler) checkAllDevicesReturning(orderID int) (bool, error) {
	devices, err := h.orderService.GetOrderDevices(orderID)
	if err != nil {
		return false, err
	}

	for _, device := range devices {
		if device.OrderStatus != "returning" && device.OrderStatus != "returned" {
			return false, nil
		}
	}

	return len(devices) > 0, nil
}

// checkAndStartDeviceReclaim 检查并启动设备回收流程
func (h *OrderEventHandler) checkAndStartDeviceReclaim(orderID int) error {
	// 获取订单详情
	orderDetail, err := h.orderService.GetOrder(orderID)
	if err != nil {
		return err
	}

	// 如果是pool_exit操作，需要启动设备回收
	if orderDetail.ActionType == "pool_exit" {
		h.logger.Info("Starting device reclaim process for pool_exit order",
			zap.Int("orderID", orderID))

		// 这里可以发布设备回收事件或直接调用回收服务
		// 具体实现取决于业务需求
	}

	return nil
}

// PublishOrderStatusChangedEvent 发布订单状态变更事件
func (s *ElasticScalingService) PublishOrderStatusChangedEvent(
	eventManager *events.EventManager,
	req events.OrderStatusChangeRequest,
) error {
	if eventManager == nil {
		return nil // 如果没有事件管理器，忽略事件发布
	}

	event := events.NewOrderStatusChangedEvent(req)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return eventManager.Publish(events.PublishRequest{Event: event, Ctx: ctx})
}

// PublishScalingEvent 发布弹性伸缩事件
func (s *ElasticScalingService) PublishScalingEvent(
	eventManager *events.EventManager,
	req events.ScalingRequest,
) error {
	if eventManager == nil {
		return nil
	}

	event := events.NewScalingEvent(req)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return eventManager.Publish(events.PublishRequest{Event: event, Ctx: ctx})
}
