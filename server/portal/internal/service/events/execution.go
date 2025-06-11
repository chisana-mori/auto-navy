package events

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
)

// handleGenericEventSync 同步处理泛型事件
func (em *EventManager) handleGenericEventSync(ctx context.Context, event Event, handlers []interface{}) error {
	var lastErr error
	for _, h := range handlers {
		if err := em.executeGenericHandlerWithRetry(ctx, h, event); err != nil {
			em.logger.Error("Generic handler execution failed",
				zap.String("eventType", event.Type()),
				zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}

// handleGenericEventAsync 异步处理泛型事件
func (em *EventManager) handleGenericEventAsync(ctx context.Context, event Event, handlers []interface{}) {
	for _, h := range handlers {
		go func(handler interface{}) {
			if err := em.executeGenericHandlerWithRetry(ctx, handler, event); err != nil {
				em.logger.Error("Async generic handler execution failed",
					zap.String("eventType", event.Type()),
					zap.Error(err))
			}
		}(h)
	}
}

// executeGenericHandlerWithRetry 使用重试机制执行泛型处理器
func (em *EventManager) executeGenericHandlerWithRetry(ctx context.Context, handler interface{}, event Event) error {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, em.config.Timeout)
	defer cancel()

	var lastErr error

	// 使用反射来调用泛型处理器
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// 检查是否有Handle方法
	handleMethod := handlerValue.MethodByName("Handle")
	if !handleMethod.IsValid() {
		return fmt.Errorf("handler does not have Handle method")
	}

	for attempt := 0; attempt <= em.config.RetryCount; attempt++ {
		if attempt > 0 {
			em.logger.Warn("Retrying generic handler execution",
				zap.String("eventType", event.Type()),
				zap.String("handlerType", handlerType.String()),
				zap.Int("attempt", attempt),
				zap.Error(lastErr))

			// 等待一小段时间再重试
			select {
			case <-timeoutCtx.Done():
				return fmt.Errorf("timeout during retry: %w", timeoutCtx.Err())
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
		}

		// 调用Handle方法
		// 泛型处理器期望的是实际的泛型事件对象，不是Event接口
		results := handleMethod.Call([]reflect.Value{
			reflect.ValueOf(timeoutCtx),
			reflect.ValueOf(event),
		})

		// 检查返回值是否有错误
		if len(results) > 0 && !results[0].IsNil() {
			if err, ok := results[0].Interface().(error); ok {
				lastErr = err
				continue
			}
		}

		// 成功执行
		return nil
	}

	return fmt.Errorf("generic handler failed after %d retries: %w", em.config.RetryCount, lastErr)
}

// handleEventSync 同步处理事件
func (em *EventManager) handleEventSync(ctx context.Context, event Event, handlers []EventHandler) error {
	var lastErr error

	for _, handler := range handlers {
		if err := em.executeHandlerWithRetry(ctx, handler, event); err != nil {
			em.logger.Error("Event handler failed",
				zap.String("eventType", event.Type()),
				zap.String("handlerName", handler.Name()),
				zap.Error(err))
			lastErr = err
		}
	}

	return lastErr
}

// handleEventAsync 异步处理事件
func (em *EventManager) handleEventAsync(ctx context.Context, event Event, handlers []EventHandler) {
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := em.executeHandlerWithRetry(ctx, h, event); err != nil {
				em.logger.Error("Async event handler failed",
					zap.String("eventType", event.Type()),
					zap.String("handlerName", h.Name()),
					zap.Error(err))
			}
		}(handler)
	}
}

// executeHandlerWithRetry 带重试的处理器执行
func (em *EventManager) executeHandlerWithRetry(ctx context.Context, handler EventHandler, event Event) error {
	var lastErr error

	for i := 0; i <= em.config.RetryCount; i++ {
		// 创建带超时的上下文
		timeoutCtx, cancel := context.WithTimeout(ctx, em.config.Timeout)

		err := handler.Handle(timeoutCtx, event)
		cancel()

		if err == nil {
			if i > 0 {
				em.logger.Info("Event handler succeeded after retry",
					zap.String("eventType", event.Type()),
					zap.String("handlerName", handler.Name()),
					zap.Int("retryCount", i))
			}
			return nil
		}

		lastErr = err

		if i < em.config.RetryCount {
			em.logger.Warn("Event handler failed, will retry",
				zap.String("eventType", event.Type()),
				zap.String("handlerName", handler.Name()),
				zap.Int("attempt", i+1),
				zap.Int("maxRetries", em.config.RetryCount),
				zap.Error(err))

			// 指数退避
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
	}

	return fmt.Errorf("event handler failed after %d retries: %w", em.config.RetryCount, lastErr)
}
