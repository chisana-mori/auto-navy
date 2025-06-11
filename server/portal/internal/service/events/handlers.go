package events

import (
	"context"
	"reflect"

	"go.uber.org/zap"
)

// NamedGenericEventHandler 带名称的泛型事件处理器
type NamedGenericEventHandler[T any] struct {
	HandlerName string
	HandlerFunc GenericEventHandlerFunc[T]
}

func (h *NamedGenericEventHandler[T]) Handle(ctx context.Context, event *GenericEvent[T]) error {
	return h.HandlerFunc(ctx, event)
}

func (h *NamedGenericEventHandler[T]) Name() string {
	return h.HandlerName
}

// NamedEventHandler 带名称的事件处理器 (保留用于兼容)
type NamedEventHandler struct {
	HandlerName string
	HandlerFunc EventHandlerFunc
}

func (h *NamedEventHandler) Handle(ctx context.Context, event Event) error {
	return h.HandlerFunc(ctx, event)
}

func (h *NamedEventHandler) Name() string {
	return h.HandlerName
}

// RegisterGeneric 注册泛型事件处理器
func RegisterGeneric[T any](em *EventManager, req RegisterGenericRequest[T]) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	eventType := req.EventType
	handlerName := req.HandlerName
	handler := req.Handler

	// 初始化事件类型的map
	if em.genericHandlers[eventType] == nil {
		em.genericHandlers[eventType] = make(map[reflect.Type][]interface{})
	}

	// 获取数据类型
	var zero T
	dataType := reflect.TypeOf(zero)

	// 检查处理器是否已注册
	if em.isGenericHandlerRegistered(eventType, dataType, handlerName) {
		em.logger.Warn("Generic handler already registered, skipping",
			zap.String("eventType", eventType),
			zap.String("dataType", dataType.String()),
			zap.String("handlerName", handlerName))
		return
	}

	// 添加处理器
	em.genericHandlers[eventType][dataType] = append(em.genericHandlers[eventType][dataType], handler)

	em.logger.Info("Generic handler registered",
		zap.String("eventType", eventType),
		zap.String("dataType", dataType.String()),
		zap.String("handlerName", handlerName))
}

// RegisterGenericFunc 注册泛型函数事件处理器
func RegisterGenericFunc[T any](em *EventManager, req RegisterGenericFuncRequest[T]) {
	namedHandler := &NamedGenericEventHandler[T]{
		HandlerName: req.HandlerName,
		HandlerFunc: req.HandlerFunc,
	}

	RegisterGeneric(em, RegisterGenericRequest[T]{
		EventType:   req.EventType,
		HandlerName: req.HandlerName,
		Handler:     namedHandler,
	})
}

// isGenericHandlerRegistered 检查泛型处理器是否已注册
func (em *EventManager) isGenericHandlerRegistered(eventType string, dataType reflect.Type, handlerName string) bool {
	if typeMap, exists := em.genericHandlers[eventType]; exists {
		if handlers, exists := typeMap[dataType]; exists {
			for _, h := range handlers {
				// 使用反射获取处理器的Name方法
				handlerValue := reflect.ValueOf(h)
				nameMethod := handlerValue.MethodByName("Name")
				if nameMethod.IsValid() {
					results := nameMethod.Call(nil)
					if len(results) > 0 {
						if name, ok := results[0].Interface().(string); ok {
							if name == handlerName {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// UnregisterGeneric 注销泛型事件处理器
func UnregisterGeneric[T any](em *EventManager, req UnregisterGenericRequest[T]) {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	eventType := req.EventType
	handlerName := req.HandlerName

	var zero T
	dataType := reflect.TypeOf(zero)

	if typeMap, exists := em.genericHandlers[eventType]; exists {
		if handlers, exists := typeMap[dataType]; exists {
			for i, h := range handlers {
				if handler, ok := h.(GenericEventHandler[T]); ok {
					if handler.Name() == handlerName {
						// 移除处理器
						em.genericHandlers[eventType][dataType] = append(handlers[:i], handlers[i+1:]...)

						// 如果该类型没有处理器了，删除类型映射
						if len(em.genericHandlers[eventType][dataType]) == 0 {
							delete(em.genericHandlers[eventType], dataType)
						}

						// 如果该事件类型没有任何处理器了，删除事件类型映射
						if len(em.genericHandlers[eventType]) == 0 {
							delete(em.genericHandlers, eventType)
						}

						em.logger.Info("Generic handler unregistered",
							zap.String("eventType", eventType),
							zap.String("dataType", dataType.String()),
							zap.String("handlerName", handlerName))
						return
					}
				}
			}
		}
	}

	em.logger.Warn("Generic handler not found for unregistration",
		zap.String("eventType", eventType),
		zap.String("dataType", dataType.String()),
		zap.String("handlerName", handlerName))
}
