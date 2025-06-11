package events

import (
	"time"
)

// 事件类型常量
const (
	// 订单状态相关事件
	EventTypeOrderStatusChanged = "order.status.changed"
	EventTypeOrderCreated       = "order.created"
	EventTypeOrderCompleted     = "order.completed"
	EventTypeOrderFailed        = "order.failed"
	EventTypeOrderCancelled     = "order.cancelled"
	EventTypeOrderReturning     = "order.returning"

	// 设备操作相关事件
	EventTypeDeviceOperationStarted   = "device.operation.started"
	EventTypeDeviceOperationCompleted = "device.operation.completed"
	EventTypeDeviceOperationFailed    = "device.operation.failed"

	// 维护相关事件
	EventTypeMaintenanceStarted   = "maintenance.started"
	EventTypeMaintenanceCompleted = "maintenance.completed"
	EventTypeMaintenanceFailed    = "maintenance.failed"

	// 弹性伸缩相关事件
	EventTypeScalingTriggered = "scaling.triggered"
	EventTypeScalingCompleted = "scaling.completed"
	EventTypeScalingCancelled = "scaling.cancelled"
	EventTypeScalingReturning = "scaling.returning"
)

// BaseEvent 基础事件结构
type BaseEvent struct {
	EventType string      `json:"event_type"`
	EventData interface{} `json:"event_data"`
	EventTime time.Time   `json:"event_time"`
	Source    string      `json:"source"`
	TraceID   string      `json:"trace_id,omitempty"`
}

func (e *BaseEvent) Type() string {
	return e.EventType
}

func (e *BaseEvent) Data() interface{} {
	return e.EventData
}

func (e *BaseEvent) Timestamp() time.Time {
	return e.EventTime
}

// OrderStatusChangedEvent 订单状态变更事件
type OrderStatusChangedEvent struct {
	BaseEvent
	OrderID   int                    `json:"order_id"`
	OrderType string                 `json:"order_type"`
	OldStatus string                 `json:"old_status"`
	NewStatus string                 `json:"new_status"`
	Executor  string                 `json:"executor"`
	Reason    string                 `json:"reason"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// OrderStatusChangeRequest 订单状态变更事件创建请求
type OrderStatusChangeRequest struct {
	OrderID   int
	OrderType string
	OldStatus string
	NewStatus string
	Executor  string
	Reason    string
	Context   map[string]interface{}
}

func NewOrderStatusChangedEvent(req OrderStatusChangeRequest) *OrderStatusChangedEvent {
	event := &OrderStatusChangedEvent{
		BaseEvent: BaseEvent{
			EventType: EventTypeOrderStatusChanged,
			EventTime: time.Now(),
			Source:    "order_service",
		},
		OrderID:   req.OrderID,
		OrderType: req.OrderType,
		OldStatus: req.OldStatus,
		NewStatus: req.NewStatus,
		Executor:  req.Executor,
		Reason:    req.Reason,
		Context:   req.Context,
	}
	event.BaseEvent.EventData = event
	return event
}

// DeviceOperationEvent 设备操作事件
type DeviceOperationEvent struct {
	BaseEvent
	OrderID       int                    `json:"order_id"`
	DeviceID      int                    `json:"device_id"`
	OperationType string                 `json:"operation_type"` // pool_entry, pool_exit, maintenance, etc.
	Status        string                 `json:"status"`         // started, completed, failed
	Result        string                 `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// DeviceOperationRequest 设备操作事件创建请求
type DeviceOperationRequest struct {
	EventType     string
	OrderID       int
	DeviceID      int
	OperationType string
	Status        string
	Result        string
	Error         string
	Context       map[string]interface{}
}

func NewDeviceOperationEvent(req DeviceOperationRequest) *DeviceOperationEvent {
	event := &DeviceOperationEvent{
		BaseEvent: BaseEvent{
			EventType: req.EventType,
			EventTime: time.Now(),
			Source:    "device_service",
		},
		OrderID:       req.OrderID,
		DeviceID:      req.DeviceID,
		OperationType: req.OperationType,
		Status:        req.Status,
		Result:        req.Result,
		Error:         req.Error,
		Context:       req.Context,
	}
	event.BaseEvent.EventData = event
	return event
}

// MaintenanceEvent 维护事件
type MaintenanceEvent struct {
	BaseEvent
	OrderID          int                    `json:"order_id"`
	DeviceID         int                    `json:"device_id"`
	MaintenanceType  string                 `json:"maintenance_type"` // cordon, drain, uncordon
	Status           string                 `json:"status"`           // started, completed, failed
	StartTime        *time.Time             `json:"start_time,omitempty"`
	EndTime          *time.Time             `json:"end_time,omitempty"`
	ExternalTicketID string                 `json:"external_ticket_id,omitempty"`
	Result           string                 `json:"result,omitempty"`
	Error            string                 `json:"error,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
}

// MaintenanceRequest 维护事件创建请求
type MaintenanceRequest struct {
	EventType        string
	OrderID          int
	DeviceID         int
	MaintenanceType  string
	Status           string
	StartTime        *time.Time
	EndTime          *time.Time
	ExternalTicketID string
	Result           string
	Error            string
	Context          map[string]interface{}
}

func NewMaintenanceEvent(req MaintenanceRequest) *MaintenanceEvent {
	event := &MaintenanceEvent{
		BaseEvent: BaseEvent{
			EventType: req.EventType,
			EventTime: time.Now(),
			Source:    "maintenance_service",
		},
		OrderID:          req.OrderID,
		DeviceID:         req.DeviceID,
		MaintenanceType:  req.MaintenanceType,
		Status:           req.Status,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		ExternalTicketID: req.ExternalTicketID,
		Result:           req.Result,
		Error:            req.Error,
		Context:          req.Context,
	}
	event.BaseEvent.EventData = event
	return event
}

// ScalingEvent 弹性伸缩事件
type ScalingEvent struct {
	BaseEvent
	StrategyID      int                    `json:"strategy_id"`
	ClusterID       int                    `json:"cluster_id"`
	ResourceType    string                 `json:"resource_type"`
	ActionType      string                 `json:"action_type"` // pool_entry, pool_exit
	DeviceCount     int                    `json:"device_count"`
	SelectedDevices []int                  `json:"selected_devices,omitempty"`
	OrderID         *int                   `json:"order_id,omitempty"`
	Status          string                 `json:"status"` // triggered, completed, failed
	Context         map[string]interface{} `json:"context,omitempty"`
}

// ScalingRequest 弹性伸缩事件创建请求
type ScalingRequest struct {
	EventType       string
	StrategyID      int
	ClusterID       int
	ResourceType    string
	ActionType      string
	DeviceCount     int
	SelectedDevices []int
	OrderID         *int
	Status          string
	Context         map[string]interface{}
}

func NewScalingEvent(req ScalingRequest) *ScalingEvent {
	event := &ScalingEvent{
		BaseEvent: BaseEvent{
			EventType: req.EventType,
			EventTime: time.Now(),
			Source:    "elastic_scaling_service",
		},
		StrategyID:      req.StrategyID,
		ClusterID:       req.ClusterID,
		ResourceType:    req.ResourceType,
		ActionType:      req.ActionType,
		DeviceCount:     req.DeviceCount,
		SelectedDevices: req.SelectedDevices,
		OrderID:         req.OrderID,
		Status:          req.Status,
		Context:         req.Context,
	}
	event.BaseEvent.EventData = event
	return event
}

// EventContext 事件上下文辅助函数
type EventContext map[string]interface{}

func (ctx EventContext) WithOrderNumber(orderNumber string) EventContext {
	ctx["order_number"] = orderNumber
	return ctx
}

func (ctx EventContext) WithClusterName(clusterName string) EventContext {
	ctx["cluster_name"] = clusterName
	return ctx
}

func (ctx EventContext) WithDeviceInfo(ciCode, ip string) EventContext {
	ctx["device_ci_code"] = ciCode
	ctx["device_ip"] = ip
	return ctx
}

func (ctx EventContext) WithStrategyInfo(strategyName string, thresholdValue, triggeredValue float64) EventContext {
	ctx["strategy_name"] = strategyName
	ctx["threshold_value"] = thresholdValue
	ctx["triggered_value"] = triggeredValue
	return ctx
}

func (ctx EventContext) WithError(err error) EventContext {
	if err != nil {
		ctx["error"] = err.Error()
	}
	return ctx
}

func NewEventContext() EventContext {
	return make(EventContext)
}
