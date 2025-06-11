package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
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

// TypeConverter 类型转换器接口
type TypeConverter[T any] interface {
	Convert(data interface{}) (T, error)
	CanConvert(data interface{}) bool
}

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

// EventManager 事件管理器
type EventManager struct {
	genericHandlers map[string]map[reflect.Type][]interface{} // 存储泛型处理器
	handlers        map[string][]EventHandler                 // 保留用于兼容
	converters      map[reflect.Type]interface{}              // 存储类型转换器
	mutex           sync.RWMutex
	logger          *zap.Logger
	config          *Config
}

// Config 事件管理器配置
type Config struct {
	Timeout     time.Duration // 事件处理超时时间
	RetryCount  int           // 重试次数
	BufferSize  int           // 事件队列缓冲大小
	Async       bool          // 是否异步处理
	EnableStats bool          // 是否启用统计
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:     30 * time.Second,
		RetryCount:  3,
		BufferSize:  1000,
		Async:       true,
		EnableStats: true,
	}
}

// NewEventManager 创建新的事件管理器
func NewEventManager(logger *zap.Logger, config *Config) *EventManager {
	if config == nil {
		config = DefaultConfig()
	}

	return &EventManager{
		genericHandlers: make(map[string]map[reflect.Type][]interface{}),
		handlers:        make(map[string][]EventHandler),
		converters:      make(map[reflect.Type]interface{}),
		logger:          logger,
		config:          config,
	}
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

// GenericConvertRequest 带转换器的泛型发布请求
type GenericConvertRequest[T any] struct {
	EventType string
	RawData   interface{}
	Source    string
	Context   context.Context
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

// UnregisterGenericRequest 泛型注销请求
type UnregisterGenericRequest[T any] struct {
	EventType   string
	HandlerName string
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

// PublishRequest 发布事件的请求结构体
type PublishRequest struct {
	Event Event
	Ctx   context.Context
}

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

// GetHandlersRequest 获取处理器列表的请求结构体
type GetHandlersRequest struct {
	EventType string
}

// GetHandlers 获取指定事件类型的处理器列表
func (em *EventManager) GetHandlers(req GetHandlersRequest) []string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	handlers := em.handlers[req.EventType]
	names := make([]string, len(handlers))
	for i, handler := range handlers {
		names[i] = handler.Name()
	}

	return names
}

// GetAllEventTypes 获取所有已注册的事件类型
func (em *EventManager) GetAllEventTypes() []string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	types := make([]string, 0, len(em.handlers))
	for eventType := range em.handlers {
		types = append(types, eventType)
	}

	return types
}

// ShutdownRequest 关闭的请求结构体
type ShutdownRequest struct {
	Ctx context.Context
}

// Shutdown 优雅关闭事件管理器
func (em *EventManager) Shutdown(req ShutdownRequest) error {
	em.logger.Info("Shutting down event manager")

	// 等待正在处理的事件完成
	// 这里可以添加更复杂的等待逻辑

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// 清空处理器
	em.handlers = make(map[string][]EventHandler)

	em.logger.Info("Event manager shutdown completed")
	return nil
}
