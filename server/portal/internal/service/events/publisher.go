package events

import (
	"context"
	"reflect"

	"go.uber.org/zap"
)

// Publish 发布事件
func (em *EventManager) Publish(req PublishRequest) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	event := req.Event
	eventType := event.Type()

	// 首先尝试泛型处理器
	if err := em.publishGenericEvent(ctx, event); err != nil {
		em.logger.Error("Failed to publish generic event", zap.Error(err))
	}

	// 兼容旧的处理器
	em.mutex.RLock()
	handlers := make([]EventHandler, len(em.handlers[eventType]))
	copy(handlers, em.handlers[eventType])
	em.mutex.RUnlock()

	if len(handlers) == 0 {
		em.logger.Debug("No handlers found for event type",
			zap.String("eventType", eventType))
		return nil
	}

	em.logger.Debug("Publishing event",
		zap.String("eventType", eventType),
		zap.Int("handlerCount", len(handlers)))

	if em.config.Async {
		em.handleEventAsync(ctx, event, handlers)
		return nil
	} else {
		return em.handleEventSync(ctx, event, handlers)
	}
}

// publishGenericEvent 发布泛型事件
func (em *EventManager) publishGenericEvent(ctx context.Context, event Event) error {
	eventType := event.Type()

	em.mutex.RLock()
	defer em.mutex.RUnlock()

	typeMap, exists := em.genericHandlers[eventType]
	if !exists {
		em.logger.Debug("No generic handlers found for event type",
			zap.String("eventType", eventType))
		return nil
	}

	// 获取事件数据类型
	dataType := reflect.TypeOf(event.Data())
	handlers, exists := typeMap[dataType]
	if !exists {
		em.logger.Debug("No handlers found for data type",
			zap.String("eventType", eventType),
			zap.String("dataType", dataType.String()))
		return nil
	}

	em.logger.Debug("Publishing generic event",
		zap.String("eventType", eventType),
		zap.String("dataType", dataType.String()),
		zap.Int("handlerCount", len(handlers)))

	if em.config.Async {
		go em.handleGenericEventAsync(ctx, event, handlers)
		return nil
	} else {
		return em.handleGenericEventSync(ctx, event, handlers)
	}
}
