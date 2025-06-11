package events

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
)

// DefaultTypeConverter 默认类型转换器
type DefaultTypeConverter[T any] struct{}

func (c *DefaultTypeConverter[T]) Convert(data interface{}) (T, error) {
	var zero T
	if converted, ok := data.(T); ok {
		return converted, nil
	}
	return zero, fmt.Errorf("cannot convert %T to %T", data, zero)
}

func (c *DefaultTypeConverter[T]) CanConvert(data interface{}) bool {
	var zero T
	_, ok := data.(T)
	return ok || reflect.TypeOf(data).AssignableTo(reflect.TypeOf(zero))
}

// RegisterConverter 注册类型转换器
func RegisterConverter[T any](em *EventManager, converter TypeConverter[T]) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	var zero T
	targetType := reflect.TypeOf(zero)
	em.converters[targetType] = converter

	em.logger.Info("Type converter registered",
		zap.String("targetType", targetType.String()))
}

// GetConverter 获取类型转换器
func GetConverter[T any](em *EventManager) (TypeConverter[T], bool) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	var zero T
	targetType := reflect.TypeOf(zero)

	if converter, exists := em.converters[targetType]; exists {
		if typedConverter, ok := converter.(TypeConverter[T]); ok {
			return typedConverter, true
		}
	}

	// 返回默认转换器
	return &DefaultTypeConverter[T]{}, false
}
