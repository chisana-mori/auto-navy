package events

import (
	"context"
	"time"
)

// GenericEventHandler 泛型事件处理器接口
type GenericEventHandler[T any] interface {
	Handle(ctx context.Context, event *GenericEvent[T]) error
	Name() string
}

// GenericEventHandlerFunc 泛型函数类型的事件处理器
type GenericEventHandlerFunc[T any] func(ctx context.Context, event *GenericEvent[T]) error

func (f GenericEventHandlerFunc[T]) Handle(ctx context.Context, event *GenericEvent[T]) error {
	return f(ctx, event)
}

func (f GenericEventHandlerFunc[T]) Name() string {
	return "anonymous_generic_handler"
}

// Event 事件接口
type Event interface {
	Type() string
	Data() interface{}
	Timestamp() time.Time
}

// EventHandler 事件处理器接口 (保留用于兼容)
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	Name() string
}

// EventHandlerFunc 函数类型的事件处理器 (保留用于兼容)
type EventHandlerFunc func(ctx context.Context, event Event) error

func (f EventHandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

func (f EventHandlerFunc) Name() string {
	return "anonymous_handler"
}

// TypeConverter 类型转换器接口
type TypeConverter[T any] interface {
	Convert(data interface{}) (T, error)
	CanConvert(data interface{}) bool
}
