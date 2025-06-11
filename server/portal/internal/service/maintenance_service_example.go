package service

import (
	"context"
	"navy-ng/server/portal/internal/service/events"
	"time"

	"go.uber.org/zap"
)

// MaintenanceService 维护服务示例
type MaintenanceService struct {
	logger       *zap.Logger
	eventManager *events.EventManager
}

// NewMaintenanceService 创建维护服务
func NewMaintenanceService(logger *zap.Logger, eventManager *events.EventManager) *MaintenanceService {
	return &MaintenanceService{
		logger:       logger,
		eventManager: eventManager,
	}
}

// CordonDevice 封锁设备示例方法
func (s *MaintenanceService) CordonDevice(orderID, deviceID int) error {
	s.logger.Info("Starting device cordon",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	// 发布维护开始事件
	err := s.publishMaintenanceEvent(orderID, deviceID, "cordon", "started", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance started event", zap.Error(err))
	}

	// 执行实际的cordon操作
	// ... 这里是具体的业务逻辑 ...

	// 模拟操作完成
	time.Sleep(1 * time.Second)

	// 发布维护完成事件
	err = s.publishMaintenanceEvent(orderID, deviceID, "cordon", "completed", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance completed event", zap.Error(err))
		return err
	}

	s.logger.Info("Device cordon completed",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	return nil
}

// DrainDevice 排空设备示例方法
func (s *MaintenanceService) DrainDevice(orderID, deviceID int) error {
	s.logger.Info("Starting device drain",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	// 发布维护开始事件
	err := s.publishMaintenanceEvent(orderID, deviceID, "drain", "started", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance started event", zap.Error(err))
	}

	// 执行实际的drain操作
	// ... 这里是具体的业务逻辑 ...

	// 模拟操作完成
	time.Sleep(2 * time.Second)

	// 发布维护完成事件
	err = s.publishMaintenanceEvent(orderID, deviceID, "drain", "completed", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance completed event", zap.Error(err))
		return err
	}

	s.logger.Info("Device drain completed",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	return nil
}

// UncordonDevice 解除封锁设备示例方法
func (s *MaintenanceService) UncordonDevice(orderID, deviceID int) error {
	s.logger.Info("Starting device uncordon",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	// 发布维护开始事件
	err := s.publishMaintenanceEvent(orderID, deviceID, "uncordon", "started", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance started event", zap.Error(err))
	}

	// 执行实际的uncordon操作
	// ... 这里是具体的业务逻辑 ...

	// 模拟操作完成
	time.Sleep(1 * time.Second)

	// 发布维护完成事件
	err = s.publishMaintenanceEvent(orderID, deviceID, "uncordon", "completed", nil, "")
	if err != nil {
		s.logger.Error("Failed to publish maintenance completed event", zap.Error(err))
		return err
	}

	s.logger.Info("Device uncordon completed",
		zap.Int("orderID", orderID),
		zap.Int("deviceID", deviceID))

	return nil
}

// publishMaintenanceEvent 发布维护事件的辅助方法
func (s *MaintenanceService) publishMaintenanceEvent(
	orderID, deviceID int,
	maintenanceType, status string,
	endTime *time.Time,
	errorMsg string,
) error {
	if s.eventManager == nil {
		return nil // 如果没有事件管理器，忽略事件发布
	}

	var eventType string
	switch status {
	case "started":
		eventType = events.EventTypeMaintenanceStarted
	case "completed":
		eventType = events.EventTypeMaintenanceCompleted
	case "failed":
		eventType = events.EventTypeMaintenanceFailed
	default:
		eventType = events.EventTypeMaintenanceStarted
	}

	now := time.Now()
	event := events.NewMaintenanceEvent(events.MaintenanceRequest{
		EventType:        eventType,
		OrderID:          orderID,
		DeviceID:         deviceID,
		MaintenanceType:  maintenanceType,
		Status:           status,
		StartTime:        &now,
		EndTime:          endTime,
		ExternalTicketID: "",
		Result:           "",
		Error:            errorMsg,
		Context:          events.NewEventContext(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.eventManager.Publish(events.PublishRequest{Event: event, Ctx: ctx})
}
