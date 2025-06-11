package events

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
)

// GenericEvent 泛型事件结构
type GenericEvent[T any] struct {
	EventType string
	EventData T
	EventTime time.Time
	Source    string
}

func (e *GenericEvent[T]) Type() string {
	return e.EventType
}

func (e *GenericEvent[T]) Data() interface{} {
	return e.EventData
}

func (e *GenericEvent[T]) Timestamp() time.Time {
	return e.EventTime
}

// GenericEventRequest 泛型事件发布请求
type GenericEventRequest[T any] struct {
	EventType string
	Data      T
	Source    string
	Context   context.Context
}

// GenericConvertRequest 带转换器的泛型发布请求
type GenericConvertRequest[T any] struct {
	EventType string
	RawData   interface{}
	Source    string
	Context   context.Context
}

// RegisterGenericRequest 泛型注册请求
type RegisterGenericRequest[T any] struct {
	EventType   string
	HandlerName string
	Handler     GenericEventHandler[T]
}

// RegisterGenericFuncRequest 泛型函数注册请求
type RegisterGenericFuncRequest[T any] struct {
	EventType   string
	HandlerName string
	HandlerFunc GenericEventHandlerFunc[T]
}

// UnregisterGenericRequest 泛型注销请求
type UnregisterGenericRequest[T any] struct {
	EventType   string
	HandlerName string
}

// NewGenericEvent 创建泛型事件
func NewGenericEvent[T any](eventType string, data T, source string) *GenericEvent[T] {
	return &GenericEvent[T]{
		EventType: eventType,
		EventData: data,
		EventTime: time.Now(),
		Source:    source,
	}
}

// PublishGeneric 泛型发布事件方法（优化版）
func PublishGeneric[T any](em *EventManager, req GenericEventRequest[T]) error {
	// 创建泛型事件
	event := NewGenericEvent(req.EventType, req.Data, req.Source)

	// 使用现有的Publish方法
	return em.Publish(PublishRequest{
		Event: event,
		Ctx:   req.Context,
	})
}

// PublishGenericWithConverter 使用转换器的泛型发布方法（优化版）
func PublishGenericWithConverter[T any](em *EventManager, req GenericConvertRequest[T]) error {
	// 获取转换器
	converter, hasCustomConverter := GetConverter[T](em)

	// 转换数据
	convertedData, err := converter.Convert(req.RawData)
	if err != nil {
		em.logger.Error("Failed to convert data for generic event",
			zap.String("eventType", req.EventType),
			zap.String("sourceType", reflect.TypeOf(req.RawData).String()),
			zap.String("targetType", reflect.TypeOf(convertedData).String()),
			zap.Bool("hasCustomConverter", hasCustomConverter),
			zap.Error(err))
		return fmt.Errorf("data conversion failed: %w", err)
	}

	em.logger.Debug("Data converted for generic event",
		zap.String("eventType", req.EventType),
		zap.String("sourceType", reflect.TypeOf(req.RawData).String()),
		zap.String("targetType", reflect.TypeOf(convertedData).String()),
		zap.Bool("hasCustomConverter", hasCustomConverter))

	// 发布转换后的事件
	return PublishGeneric(em, GenericEventRequest[T]{
		EventType: req.EventType,
		Data:      convertedData,
		Source:    req.Source,
		Context:   req.Context,
	})
}
