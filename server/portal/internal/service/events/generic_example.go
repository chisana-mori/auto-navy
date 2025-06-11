package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"go.uber.org/zap"
)

// OrderEventData 订单事件数据结构
type OrderEventData struct {
	OrderID     int    `json:"order_id"`
	OrderType   string `json:"order_type"`
	Status      string `json:"status"`
	Operator    string `json:"operator"`
	Description string `json:"description"`
}

// DeviceEventData 设备事件数据结构
type DeviceEventData struct {
	DeviceID int    `json:"device_id"`
	OrderID  int    `json:"order_id"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Result   string `json:"result"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// OrderEventConverter 订单事件转换器
type OrderEventConverter struct{}

func (c *OrderEventConverter) Convert(data interface{}) (OrderEventData, error) {
	switch v := data.(type) {
	case OrderEventData:
		return v, nil
	case map[string]interface{}:
		return c.convertFromMap(v)
	case string:
		return c.convertFromJSON(v)
	case *OrderStatusChangedEvent:
		return OrderEventData{
			OrderID:     v.OrderID,
			OrderType:   v.OrderType,
			Status:      v.NewStatus,
			Operator:    v.Executor,
			Description: v.Reason,
		}, nil
	default:
		return OrderEventData{}, fmt.Errorf("unsupported data type: %T", data)
	}
}

func (c *OrderEventConverter) CanConvert(data interface{}) bool {
	switch data.(type) {
	case OrderEventData, map[string]interface{}, string, *OrderStatusChangedEvent:
		return true
	default:
		return false
	}
}

func (c *OrderEventConverter) convertFromMap(m map[string]interface{}) (OrderEventData, error) {
	result := OrderEventData{}

	if id, ok := m["order_id"]; ok {
		if idInt, err := parseIntValue(id); err == nil {
			result.OrderID = idInt
		}
	}

	if orderType, ok := m["order_type"].(string); ok {
		result.OrderType = orderType
	}

	if status, ok := m["status"].(string); ok {
		result.Status = status
	}

	if operator, ok := m["operator"].(string); ok {
		result.Operator = operator
	}

	if desc, ok := m["description"].(string); ok {
		result.Description = desc
	}

	return result, nil
}

func (c *OrderEventConverter) convertFromJSON(jsonStr string) (OrderEventData, error) {
	var result OrderEventData
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return OrderEventData{}, err
	}
	return result, nil
}

// DeviceEventConverter 设备事件转换器
type DeviceEventConverter struct{}

func (c *DeviceEventConverter) Convert(data interface{}) (DeviceEventData, error) {
	switch v := data.(type) {
	case DeviceEventData:
		return v, nil
	case map[string]interface{}:
		return c.convertFromMap(v)
	case string:
		return c.convertFromJSON(v)
	case *DeviceOperationEvent:
		return DeviceEventData{
			DeviceID: v.DeviceID,
			OrderID:  v.OrderID,
			Action:   v.OperationType,
			Status:   v.Status,
			Result:   v.Result,
			ErrorMsg: v.Error,
		}, nil
	default:
		return DeviceEventData{}, fmt.Errorf("unsupported data type: %T", data)
	}
}

func (c *DeviceEventConverter) CanConvert(data interface{}) bool {
	switch data.(type) {
	case DeviceEventData, map[string]interface{}, string, *DeviceOperationEvent:
		return true
	default:
		return false
	}
}

func (c *DeviceEventConverter) convertFromMap(m map[string]interface{}) (DeviceEventData, error) {
	result := DeviceEventData{}

	if id, ok := m["device_id"]; ok {
		if idInt, err := parseIntValue(id); err == nil {
			result.DeviceID = idInt
		}
	}

	if orderID, ok := m["order_id"]; ok {
		if orderIDInt, err := parseIntValue(orderID); err == nil {
			result.OrderID = orderIDInt
		}
	}

	if action, ok := m["action"].(string); ok {
		result.Action = action
	}

	if status, ok := m["status"].(string); ok {
		result.Status = status
	}

	if resultValue, ok := m["result"].(string); ok {
		result.Result = resultValue
	}

	if errorMsg, ok := m["error_msg"].(string); ok {
		result.ErrorMsg = errorMsg
	}

	return result, nil
}

func (c *DeviceEventConverter) convertFromJSON(jsonStr string) (DeviceEventData, error) {
	var result DeviceEventData
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return DeviceEventData{}, err
	}
	return result, nil
}

// parseIntValue 解析整数值的辅助函数
func parseIntValue(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot parse %T to int", value)
	}
}

// InitializeGenericEventSystem 初始化泛型事件系统
func InitializeGenericEventSystem(em *EventManager) {
	// 注册转换器
	RegisterConverter(em, &OrderEventConverter{})
	RegisterConverter(em, &DeviceEventConverter{})

	em.logger.Info("Generic event system initialized with converters")
}

// PublishOrderEvent 发布订单事件的便利方法（优化版）
func PublishOrderEvent(em *EventManager, ctx context.Context, eventType string, orderData interface{}) error {
	return PublishGenericWithConverter[OrderEventData](em, GenericConvertRequest[OrderEventData]{
		EventType: eventType,
		RawData:   orderData,
		Source:    "order_service",
		Context:   ctx,
	})
}

// PublishDeviceEvent 发布设备事件的便利方法（优化版）
func PublishDeviceEvent(em *EventManager, ctx context.Context, eventType string, deviceData interface{}) error {
	return PublishGenericWithConverter[DeviceEventData](em, GenericConvertRequest[DeviceEventData]{
		EventType: eventType,
		RawData:   deviceData,
		Source:    "device_service",
		Context:   ctx,
	})
}

// OrderEventRequest 订单事件发布请求
type OrderEventRequest struct {
	EventType   string
	OrderID     int
	OrderType   string
	Status      string
	Operator    string
	Description string
	Context     context.Context
}

// PublishOrderEventDirect 直接发布订单事件的便利方法
func PublishOrderEventDirect(em *EventManager, req OrderEventRequest) error {
	orderData := OrderEventData{
		OrderID:     req.OrderID,
		OrderType:   req.OrderType,
		Status:      req.Status,
		Operator:    req.Operator,
		Description: req.Description,
	}

	return PublishGeneric(em, GenericEventRequest[OrderEventData]{
		EventType: req.EventType,
		Data:      orderData,
		Source:    "order_service",
		Context:   req.Context,
	})
}

// DeviceEventRequest 设备事件发布请求
type DeviceEventRequest struct {
	EventType string
	DeviceID  int
	OrderID   int
	Action    string
	Status    string
	Result    string
	ErrorMsg  string
	Context   context.Context
}

// PublishDeviceEventDirect 直接发布设备事件的便利方法
func PublishDeviceEventDirect(em *EventManager, req DeviceEventRequest) error {
	deviceData := DeviceEventData{
		DeviceID: req.DeviceID,
		OrderID:  req.OrderID,
		Action:   req.Action,
		Status:   req.Status,
		Result:   req.Result,
		ErrorMsg: req.ErrorMsg,
	}

	return PublishGeneric(em, GenericEventRequest[DeviceEventData]{
		EventType: req.EventType,
		Data:      deviceData,
		Source:    "device_service",
		Context:   req.Context,
	})
}

// ExampleUsage 使用示例（优化版）
func ExampleUsage(em *EventManager) {
	ctx := context.Background()

	// 初始化泛型事件系统
	InitializeGenericEventSystem(em)

	// 示例1：使用结构体直接发布（优化版）
	orderData := OrderEventData{
		OrderID:     12345,
		OrderType:   "elastic_scaling",
		Status:      "completed",
		Operator:    "system",
		Description: "订单处理完成",
	}

	err := PublishGeneric(em, GenericEventRequest[OrderEventData]{
		EventType: "order.completed",
		Data:      orderData,
		Source:    "order_service",
		Context:   ctx,
	})
	if err != nil {
		em.logger.Error("Failed to publish order event", zap.Error(err))
	}

	// 示例2：使用便利方法发布订单事件
	err = PublishOrderEventDirect(em, OrderEventRequest{
		EventType:   "order.processing",
		OrderID:     12346,
		OrderType:   "maintenance",
		Status:      "processing",
		Operator:    "admin",
		Description: "维护订单处理中",
		Context:     ctx,
	})
	if err != nil {
		em.logger.Error("Failed to publish order event", zap.Error(err))
	}

	// 示例3：使用便利方法发布设备事件
	err = PublishDeviceEventDirect(em, DeviceEventRequest{
		EventType: "device.operation.completed",
		DeviceID:  98765,
		OrderID:   12345,
		Action:    "pool_entry",
		Status:    "success",
		Result:    "设备成功加入资源池",
		Context:   ctx,
	})
	if err != nil {
		em.logger.Error("Failed to publish device event", zap.Error(err))
	}

	// 示例4：使用map数据发布（会自动转换）
	mapData := map[string]interface{}{
		"device_id": 98766,
		"order_id":  12345,
		"action":    "pool_exit",
		"status":    "success",
		"result":    "设备成功退出资源池",
	}

	err = PublishGenericWithConverter[DeviceEventData](em, GenericConvertRequest[DeviceEventData]{
		EventType: "device.operation.completed",
		RawData:   mapData,
		Source:    "device_service",
		Context:   ctx,
	})
	if err != nil {
		em.logger.Error("Failed to publish device event from map", zap.Error(err))
	}

	// 示例5：使用JSON字符串发布（会自动转换）
	jsonData := `{
		"order_id": 12347,
		"order_type": "elastic_scaling",
		"status": "failed",
		"operator": "system",
		"description": "订单处理失败"
	}`

	err = PublishGenericWithConverter[OrderEventData](em, GenericConvertRequest[OrderEventData]{
		EventType: "order.failed",
		RawData:   jsonData,
		Source:    "order_service",
		Context:   ctx,
	})
	if err != nil {
		em.logger.Error("Failed to publish order event from JSON", zap.Error(err))
	}
}

// RegisterGenericHandler 注册泛型事件处理器的便利方法
func RegisterGenericHandler[T any](em *EventManager, eventType, handlerName string, handler func(ctx context.Context, data T) error) {
	handlerFunc := func(ctx context.Context, event *GenericEvent[T]) error {
		return handler(ctx, event.EventData)
	}

	RegisterGenericFunc(em, RegisterGenericFuncRequest[T]{
		EventType:   eventType,
		HandlerName: handlerName,
		HandlerFunc: handlerFunc,
	})
}

// ExampleHandlerRegistration 处理器注册示例
func ExampleHandlerRegistration(em *EventManager) {
	// 注册订单事件处理器
	RegisterGenericHandler(em, "order.completed", "order_completion_handler",
		func(ctx context.Context, data OrderEventData) error {
			em.logger.Info("Handling order completion",
				zap.Int("orderID", data.OrderID),
				zap.String("orderType", data.OrderType),
				zap.String("operator", data.Operator))

			// 处理订单完成逻辑
			// ... 业务逻辑代码 ...

			return nil
		})

	// 注册设备事件处理器
	RegisterGenericHandler(em, "device.operation.completed", "device_operation_handler",
		func(ctx context.Context, data DeviceEventData) error {
			em.logger.Info("Handling device operation",
				zap.Int("deviceID", data.DeviceID),
				zap.Int("orderID", data.OrderID),
				zap.String("action", data.Action),
				zap.String("status", data.Status))

			// 处理设备操作逻辑
			// ... 业务逻辑代码 ...

			return nil
		})
}
