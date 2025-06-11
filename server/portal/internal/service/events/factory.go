package events

import (
	"context"
	"fmt"
)

// EventPublisherFactory 事件发布器工厂
type EventPublisherFactory struct {
	em *EventManager
	ctx context.Context
}

// NewEventPublisherFactory 创建事件发布器工厂
func NewEventPublisherFactory(em *EventManager, ctx context.Context) *EventPublisherFactory {
	return &EventPublisherFactory{
		em:  em,
		ctx: ctx,
	}
}

// ESOPublisher 弹性伸缩订单发布器 (Elastic Scaling Order Publisher)
type ESOPublisher struct {
	em        *EventManager
	orderID   int
	orderType string
	operator  string
}

// NewESOPublisher 创建弹性伸缩订单发布器
func NewESOPublisher(orderID int, orderType string) *ESOPublisher {
	return &ESOPublisher{
		orderID:   orderID,
		orderType: orderType,
		operator:  "system", // 默认操作者
	}
}

// WithEventManager 设置事件管理器
func (p *ESOPublisher) WithEventManager(em *EventManager) *ESOPublisher {
	p.em = em
	return p
}

// WithOperator 设置操作者
func (p *ESOPublisher) WithOperator(operator string) *ESOPublisher {
	p.operator = operator
	return p
}

// Created 发布订单创建事件
func (p *ESOPublisher) Created(ctx context.Context, description string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishOrderEventDirect(p.em, OrderEventRequest{
		EventType:   EventTypeOrderCreated,
		OrderID:     p.orderID,
		OrderType:   p.orderType,
		Status:      "created",
		Operator:    p.operator,
		Description: description,
		Context:     ctx,
	})
}

// Complete 发布订单完成事件
func (p *ESOPublisher) Complete(ctx context.Context, description string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishOrderEventDirect(p.em, OrderEventRequest{
		EventType:   EventTypeOrderCompleted,
		OrderID:     p.orderID,
		OrderType:   p.orderType,
		Status:      "completed",
		Operator:    p.operator,
		Description: description,
		Context:     ctx,
	})
}

// Failed 发布订单失败事件
func (p *ESOPublisher) Failed(ctx context.Context, description string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishOrderEventDirect(p.em, OrderEventRequest{
		EventType:   EventTypeOrderFailed,
		OrderID:     p.orderID,
		OrderType:   p.orderType,
		Status:      "failed",
		Operator:    p.operator,
		Description: description,
		Context:     ctx,
	})
}

// Cancelled 发布订单取消事件
func (p *ESOPublisher) Cancelled(ctx context.Context, description string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishOrderEventDirect(p.em, OrderEventRequest{
		EventType:   EventTypeOrderCancelled,
		OrderID:     p.orderID,
		OrderType:   p.orderType,
		Status:      "cancelled",
		Operator:    p.operator,
		Description: description,
		Context:     ctx,
	})
}

// Returning 发布订单回收事件
func (p *ESOPublisher) Returning(ctx context.Context, description string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishOrderEventDirect(p.em, OrderEventRequest{
		EventType:   EventTypeOrderReturning,
		OrderID:     p.orderID,
		OrderType:   p.orderType,
		Status:      "returning",
		Operator:    p.operator,
		Description: description,
		Context:     ctx,
	})
}

// DevicePublisher 设备操作发布器
type DevicePublisher struct {
	em       *EventManager
	deviceID  int
	orderID   int
	action    string
}

// NewDevicePublisher 创建设备操作发布器
func NewDevicePublisher(deviceID, orderID int, action string) *DevicePublisher {
	return &DevicePublisher{
		deviceID: deviceID,
		orderID:  orderID,
		action:   action,
	}
}

// WithEventManager 设置事件管理器
func (p *DevicePublisher) WithEventManager(em *EventManager) *DevicePublisher {
	p.em = em
	return p
}

// Started 发布设备操作开始事件
func (p *DevicePublisher) Started(ctx context.Context, result string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishDeviceEventDirect(p.em, DeviceEventRequest{
		EventType: EventTypeDeviceOperationStarted,
		DeviceID:  p.deviceID,
		OrderID:   p.orderID,
		Action:    p.action,
		Status:    "started",
		Result:    result,
		Context:   ctx,
	})
}

// Completed 发布设备操作完成事件
func (p *DevicePublisher) Completed(ctx context.Context, result string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishDeviceEventDirect(p.em, DeviceEventRequest{
		EventType: EventTypeDeviceOperationCompleted,
		DeviceID:  p.deviceID,
		OrderID:   p.orderID,
		Action:    p.action,
		Status:    "completed",
		Result:    result,
		Context:   ctx,
	})
}

// Failed 发布设备操作失败事件
func (p *DevicePublisher) Failed(ctx context.Context, errorMsg string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	return PublishDeviceEventDirect(p.em, DeviceEventRequest{
		EventType: EventTypeDeviceOperationFailed,
		DeviceID:  p.deviceID,
		OrderID:   p.orderID,
		Action:    p.action,
		Status:    "failed",
		ErrorMsg:  errorMsg,
		Context:   ctx,
	})
}

// MaintenancePublisher 维护操作发布器
type MaintenancePublisher struct {
	em              *EventManager
	orderID         int
	deviceID        int
	maintenanceType string
}

// NewMaintenancePublisher 创建维护操作发布器
func NewMaintenancePublisher(orderID, deviceID int, maintenanceType string) *MaintenancePublisher {
	return &MaintenancePublisher{
		orderID:         orderID,
		deviceID:        deviceID,
		maintenanceType: maintenanceType,
	}
}

// WithEventManager 设置事件管理器
func (p *MaintenancePublisher) WithEventManager(em *EventManager) *MaintenancePublisher {
	p.em = em
	return p
}

// Started 发布维护开始事件
func (p *MaintenancePublisher) Started(ctx context.Context, result string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	// 使用现有的维护事件结构
	maintenanceEvent := NewMaintenanceEvent(MaintenanceRequest{
		EventType:       EventTypeMaintenanceStarted,
		OrderID:         p.orderID,
		DeviceID:        p.deviceID,
		MaintenanceType: p.maintenanceType,
		Status:          "started",
		Result:          result,
	})

	return p.em.Publish(PublishRequest{
		Event: maintenanceEvent,
		Ctx:   ctx,
	})
}

// Completed 发布维护完成事件
func (p *MaintenancePublisher) Completed(ctx context.Context, result string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	maintenanceEvent := NewMaintenanceEvent(MaintenanceRequest{
		EventType:       EventTypeMaintenanceCompleted,
		OrderID:         p.orderID,
		DeviceID:        p.deviceID,
		MaintenanceType: p.maintenanceType,
		Status:          "completed",
		Result:          result,
	})

	return p.em.Publish(PublishRequest{
		Event: maintenanceEvent,
		Ctx:   ctx,
	})
}

// Failed 发布维护失败事件
func (p *MaintenancePublisher) Failed(ctx context.Context, errorMsg string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	maintenanceEvent := NewMaintenanceEvent(MaintenanceRequest{
		EventType:       EventTypeMaintenanceFailed,
		OrderID:         p.orderID,
		DeviceID:        p.deviceID,
		MaintenanceType: p.maintenanceType,
		Status:          "failed",
		Error:           errorMsg,
	})

	return p.em.Publish(PublishRequest{
		Event: maintenanceEvent,
		Ctx:   ctx,
	})
}

// ScalingPublisher 弹性伸缩发布器
type ScalingPublisher struct {
	em           *EventManager
	strategyID   int
	clusterID    int
	resourceType string
	actionType   string
}

// NewScalingPublisher 创建弹性伸缩发布器
func NewScalingPublisher(strategyID, clusterID int, resourceType, actionType string) *ScalingPublisher {
	return &ScalingPublisher{
		strategyID:   strategyID,
		clusterID:    clusterID,
		resourceType: resourceType,
		actionType:   actionType,
	}
}

// WithEventManager 设置事件管理器
func (p *ScalingPublisher) WithEventManager(em *EventManager) *ScalingPublisher {
	p.em = em
	return p
}

// Triggered 发布弹性伸缩触发事件
func (p *ScalingPublisher) Triggered(ctx context.Context, deviceCount int, selectedDevices []int) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	scalingEvent := NewScalingEvent(ScalingRequest{
		EventType:       EventTypeScalingTriggered,
		StrategyID:      p.strategyID,
		ClusterID:       p.clusterID,
		ResourceType:    p.resourceType,
		ActionType:      p.actionType,
		DeviceCount:     deviceCount,
		SelectedDevices: selectedDevices,
		Status:          "triggered",
	})

	return p.em.Publish(PublishRequest{
		Event: scalingEvent,
		Ctx:   ctx,
	})
}

// Completed 发布弹性伸缩完成事件
func (p *ScalingPublisher) Completed(ctx context.Context, deviceCount int, selectedDevices []int, result string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	eventContext := NewEventContext()
	if result != "" {
		eventContext["result"] = result
	}

	scalingEvent := NewScalingEvent(ScalingRequest{
		EventType:       EventTypeScalingCompleted,
		StrategyID:      p.strategyID,
		ClusterID:       p.clusterID,
		ResourceType:    p.resourceType,
		ActionType:      p.actionType,
		DeviceCount:     deviceCount,
		SelectedDevices: selectedDevices,
		Status:          "completed",
		Context:         eventContext,
	})

	return p.em.Publish(PublishRequest{
		Event: scalingEvent,
		Ctx:   ctx,
	})
}

// Cancelled 发布弹性伸缩取消事件
func (p *ScalingPublisher) Cancelled(ctx context.Context, reason string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	eventContext := NewEventContext()
	if reason != "" {
		eventContext["reason"] = reason
	}

	scalingEvent := NewScalingEvent(ScalingRequest{
		EventType:    EventTypeScalingCancelled,
		StrategyID:   p.strategyID,
		ClusterID:    p.clusterID,
		ResourceType: p.resourceType,
		ActionType:   p.actionType,
		Status:       "cancelled",
		Context:      eventContext,
	})

	return p.em.Publish(PublishRequest{
		Event: scalingEvent,
		Ctx:   ctx,
	})
}

// Returning 发布弹性伸缩回收事件
func (p *ScalingPublisher) Returning(ctx context.Context, deviceCount int, selectedDevices []int, reason string) error {
	if p.em == nil {
		return fmt.Errorf("EventManager not set, use WithEventManager() first")
	}

	eventContext := NewEventContext()
	if reason != "" {
		eventContext["reason"] = reason
	}

	scalingEvent := NewScalingEvent(ScalingRequest{
		EventType:       EventTypeScalingReturning,
		StrategyID:      p.strategyID,
		ClusterID:       p.clusterID,
		ResourceType:    p.resourceType,
		ActionType:      p.actionType,
		DeviceCount:     deviceCount,
		SelectedDevices: selectedDevices,
		Status:          "returning",
		Context:         eventContext,
	})

	return p.em.Publish(PublishRequest{
		Event: scalingEvent,
		Ctx:   ctx,
	})
}