package events

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// 测试用的事件数据结构
type TestOrderData struct {
	OrderID     int    `json:"order_id"`
	OrderType   string `json:"order_type"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type TestDeviceData struct {
	DeviceID int    `json:"device_id"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Result   string `json:"result"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

type TestUserData struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Action   string `json:"action"`
}

// 测试用的处理器
type TestGenericHandler[T any] struct {
	name         string
	handledData  []T
	handleErrors []error
	callCount    int
	mutex        sync.Mutex
}

func NewTestGenericHandler[T any](name string) *TestGenericHandler[T] {
	return &TestGenericHandler[T]{
		name:         name,
		handledData:  make([]T, 0),
		handleErrors: make([]error, 0),
	}
}

func (h *TestGenericHandler[T]) Handle(ctx context.Context, event *GenericEvent[T]) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.callCount++
	h.handledData = append(h.handledData, event.EventData)

	// 如果有预设的错误，返回错误
	if len(h.handleErrors) > 0 {
		err := h.handleErrors[0]
		h.handleErrors = h.handleErrors[1:]
		return err
	}

	return nil
}

func (h *TestGenericHandler[T]) Name() string {
	return h.name
}

func (h *TestGenericHandler[T]) GetHandledData() []T {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	result := make([]T, len(h.handledData))
	copy(result, h.handledData)
	return result
}

func (h *TestGenericHandler[T]) GetCallCount() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.callCount
}

func (h *TestGenericHandler[T]) SetHandleError(err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.handleErrors = append(h.handleErrors, err)
}

func (h *TestGenericHandler[T]) Reset() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.handledData = h.handledData[:0]
	h.handleErrors = h.handleErrors[:0]
	h.callCount = 0
}

// 创建测试用的EventManager
func createTestEventManager() *EventManager {
	logger := zaptest.NewLogger(&testing.T{})
	config := &Config{
		Timeout:     1 * time.Second,
		RetryCount:  2,
		BufferSize:  100,
		Async:       false, // 默认同步，方便测试
		EnableStats: true,
	}
	return NewEventManager(logger, config)
}

// Test_RegisterGenericHandler 测试泛型处理器注册
func Test_RegisterGenericHandler(t *testing.T) {
	em := createTestEventManager()

	// 创建测试处理器
	orderHandler := NewTestGenericHandler[TestOrderData]("test_order_handler")
	deviceHandler := NewTestGenericHandler[TestDeviceData]("test_device_handler")

	// 注册处理器
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.test",
		HandlerName: "test_order_handler",
		Handler:     orderHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestDeviceData]{
		EventType:   "device.test",
		HandlerName: "test_device_handler",
		Handler:     deviceHandler,
	})

	// 验证处理器已注册
	em.mutex.RLock()
	orderHandlers := em.genericHandlers["order.test"][reflect.TypeOf(TestOrderData{})]
	deviceHandlers := em.genericHandlers["device.test"][reflect.TypeOf(TestDeviceData{})]
	em.mutex.RUnlock()

	if len(orderHandlers) != 1 {
		t.Errorf("Expected 1 order handler, got %d", len(orderHandlers))
	}

	if len(deviceHandlers) != 1 {
		t.Errorf("Expected 1 device handler, got %d", len(deviceHandlers))
	}
}

// Test_RegisterGenericFunc 测试泛型函数处理器注册
func Test_RegisterGenericFunc(t *testing.T) {
	em := createTestEventManager()

	var handledOrders []TestOrderData
	var callCount int
	mu := sync.Mutex{}

	handlerFunc := func(ctx context.Context, event *GenericEvent[TestOrderData]) error {
		mu.Lock()
		defer mu.Unlock()
		handledOrders = append(handledOrders, event.EventData)
		callCount++
		return nil
	}

	// 注册函数处理器
	RegisterGenericFunc(em, RegisterGenericFuncRequest[TestOrderData]{
		EventType:   "order.func.test",
		HandlerName: "test_func_handler",
		HandlerFunc: handlerFunc,
	})

	// 发布事件
	orderData := TestOrderData{
		OrderID:     123,
		OrderType:   "test",
		Status:      "processing",
		Description: "Test order",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.func.test",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// 验证处理器被调用
	mu.Lock()
	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}

	if len(handledOrders) != 1 {
		t.Errorf("Expected 1 handled order, got %d", len(handledOrders))
	}

	if handledOrders[0].OrderID != 123 {
		t.Errorf("Expected order ID 123, got %d", handledOrders[0].OrderID)
	}
	mu.Unlock()
}

// Test_PublishGenericEvent 测试泛型事件发布
func Test_PublishGenericEvent(t *testing.T) {
	em := createTestEventManager()

	// 注册处理器
	orderHandler := NewTestGenericHandler[TestOrderData]("order_handler")
	deviceHandler := NewTestGenericHandler[TestDeviceData]("device_handler")

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.created",
		HandlerName: "order_handler",
		Handler:     orderHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestDeviceData]{
		EventType:   "device.operation",
		HandlerName: "device_handler",
		Handler:     deviceHandler,
	})

	// 发布订单事件
	orderData := TestOrderData{
		OrderID:     456,
		OrderType:   "scaling",
		Status:      "created",
		Description: "New scaling order",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.created",
		Data:      orderData,
		Source:    "order_service",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish order event: %v", err)
	}

	// 发布设备事件
	deviceData := TestDeviceData{
		DeviceID: 789,
		Action:   "pool_entry",
		Status:   "success",
		Result:   "Device added to pool",
	}

	err = PublishGeneric(em, GenericEventRequest[TestDeviceData]{
		EventType: "device.operation",
		Data:      deviceData,
		Source:    "device_service",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish device event: %v", err)
	}

	// 验证事件被正确处理
	handledOrders := orderHandler.GetHandledData()
	if len(handledOrders) != 1 {
		t.Errorf("Expected 1 handled order, got %d", len(handledOrders))
	}

	if handledOrders[0].OrderID != 456 {
		t.Errorf("Expected order ID 456, got %d", handledOrders[0].OrderID)
	}

	handledDevices := deviceHandler.GetHandledData()
	if len(handledDevices) != 1 {
		t.Errorf("Expected 1 handled device, got %d", len(handledDevices))
	}

	if handledDevices[0].DeviceID != 789 {
		t.Errorf("Expected device ID 789, got %d", handledDevices[0].DeviceID)
	}
}

// Test_TypeSafety 测试类型安全性
func Test_TypeSafety(t *testing.T) {
	em := createTestEventManager()

	// 注册订单处理器
	orderHandler := NewTestGenericHandler[TestOrderData]("order_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "mixed.event",
		HandlerName: "order_handler",
		Handler:     orderHandler,
	})

	// 注册设备处理器（相同事件类型，不同数据类型）
	deviceHandler := NewTestGenericHandler[TestDeviceData]("device_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestDeviceData]{
		EventType:   "mixed.event",
		HandlerName: "device_handler",
		Handler:     deviceHandler,
	})

	// 发布订单事件 - 只有订单处理器应该被调用
	orderData := TestOrderData{
		OrderID:     111,
		OrderType:   "test",
		Status:      "active",
		Description: "Type safety test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "mixed.event",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish order event: %v", err)
	}

	// 发布设备事件 - 只有设备处理器应该被调用
	deviceData := TestDeviceData{
		DeviceID: 222,
		Action:   "test_action",
		Status:   "success",
		Result:   "Type safety test",
	}

	err = PublishGeneric(em, GenericEventRequest[TestDeviceData]{
		EventType: "mixed.event",
		Data:      deviceData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish device event: %v", err)
	}

	// 验证类型安全性
	handledOrders := orderHandler.GetHandledData()
	handledDevices := deviceHandler.GetHandledData()

	if len(handledOrders) != 1 {
		t.Errorf("Expected 1 handled order, got %d", len(handledOrders))
	}

	if len(handledDevices) != 1 {
		t.Errorf("Expected 1 handled device, got %d", len(handledDevices))
	}

	if handledOrders[0].OrderID != 111 {
		t.Errorf("Expected order ID 111, got %d", handledOrders[0].OrderID)
	}

	if handledDevices[0].DeviceID != 222 {
		t.Errorf("Expected device ID 222, got %d", handledDevices[0].DeviceID)
	}
}

// Test_MultipleHandlers 测试同一事件类型的多个处理器
func Test_MultipleHandlers(t *testing.T) {
	em := createTestEventManager()

	// 注册多个订单处理器
	handler1 := NewTestGenericHandler[TestOrderData]("handler1")
	handler2 := NewTestGenericHandler[TestOrderData]("handler2")
	handler3 := NewTestGenericHandler[TestOrderData]("handler3")

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.multiple",
		HandlerName: "handler1",
		Handler:     handler1,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.multiple",
		HandlerName: "handler2",
		Handler:     handler2,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.multiple",
		HandlerName: "handler3",
		Handler:     handler3,
	})

	// 发布事件
	orderData := TestOrderData{
		OrderID:     333,
		OrderType:   "multi",
		Status:      "test",
		Description: "Multiple handlers test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.multiple",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// 验证所有处理器都被调用
	handlers := []*TestGenericHandler[TestOrderData]{handler1, handler2, handler3}
	for i, handler := range handlers {
		callCount := handler.GetCallCount()
		if callCount != 1 {
			t.Errorf("Handler %d: expected 1 call, got %d", i+1, callCount)
		}

		handledData := handler.GetHandledData()
		if len(handledData) != 1 {
			t.Errorf("Handler %d: expected 1 handled item, got %d", i+1, len(handledData))
		}

		if handledData[0].OrderID != 333 {
			t.Errorf("Handler %d: expected order ID 333, got %d", i+1, handledData[0].OrderID)
		}
	}
}

// Test_ErrorHandling 测试错误处理
func Test_ErrorHandling(t *testing.T) {
	em := createTestEventManager()

	// 创建会出错的处理器
	errorHandler := NewTestGenericHandler[TestOrderData]("error_handler")
	// 设置多个错误，确保总是返回错误
	errorHandler.SetHandleError(errors.New("test error 1"))
	errorHandler.SetHandleError(errors.New("test error 2"))
	errorHandler.SetHandleError(errors.New("test error 3"))

	successHandler := NewTestGenericHandler[TestOrderData]("success_handler")

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.error",
		HandlerName: "error_handler",
		Handler:     errorHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.error",
		HandlerName: "success_handler",
		Handler:     successHandler,
	})

	// 直接发布泛型事件，跳过Publish方法
	orderData := TestOrderData{
		OrderID:     444,
		OrderType:   "error_test",
		Status:      "test",
		Description: "Error handling test",
	}

	genericEvent := NewGenericEvent("order.error", orderData, "test")

	// 直接调用泛型事件发布方法
	err := em.publishGenericEvent(context.Background(), genericEvent)

	// 由于有处理器失败，应该返回错误
	if err == nil {
		t.Error("Expected error but got nil")
	}

	// 验证两个处理器都被调用了
	if errorHandler.GetCallCount() == 0 {
		t.Error("Error handler should have been called")
	}

	if successHandler.GetCallCount() == 0 {
		t.Error("Success handler should have been called")
	}
}

// Test_AsyncEventHandling 测试异步事件处理
func Test_AsyncEventHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &Config{
		Timeout:     1 * time.Second,
		RetryCount:  2,
		BufferSize:  100,
		Async:       true, // 启用异步
		EnableStats: true,
	}
	em := NewEventManager(logger, config)

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("async_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.async",
		HandlerName: "async_handler",
		Handler:     handler,
	})

	// 发布事件
	orderData := TestOrderData{
		OrderID:     555,
		OrderType:   "async",
		Status:      "test",
		Description: "Async test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.async",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish async event: %v", err)
	}

	// 等待异步处理完成
	time.Sleep(100 * time.Millisecond)

	// 验证处理器被调用
	if handler.GetCallCount() != 1 {
		t.Errorf("Expected 1 call, got %d", handler.GetCallCount())
	}

	handledData := handler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled item, got %d", len(handledData))
	}

	if handledData[0].OrderID != 555 {
		t.Errorf("Expected order ID 555, got %d", handledData[0].OrderID)
	}
}

// Test_RetryMechanism 测试重试机制
func Test_RetryMechanism(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &Config{
		Timeout:     1 * time.Second,
		RetryCount:  3,
		BufferSize:  100,
		Async:       false,
		EnableStats: true,
	}
	em := NewEventManager(logger, config)

	// 创建会失败几次然后成功的处理器
	handler := NewTestGenericHandler[TestOrderData]("retry_handler")
	handler.SetHandleError(errors.New("attempt 1 failed"))
	handler.SetHandleError(errors.New("attempt 2 failed"))
	// 第三次调用会成功（没有错误）

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.retry",
		HandlerName: "retry_handler",
		Handler:     handler,
	})

	// 发布事件
	orderData := TestOrderData{
		OrderID:     666,
		OrderType:   "retry",
		Status:      "test",
		Description: "Retry test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.retry",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	// 第三次重试应该成功
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	// 验证处理器被调用了3次（2次失败 + 1次成功）
	if handler.GetCallCount() != 3 {
		t.Errorf("Expected 3 calls (2 retries + 1 success), got %d", handler.GetCallCount())
	}
}

// Test_UnregisterGenericHandler 测试注销泛型处理器
func Test_UnregisterGenericHandler(t *testing.T) {
	em := createTestEventManager()

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("unregister_test_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.unregister",
		HandlerName: "unregister_test_handler",
		Handler:     handler,
	})

	// 验证处理器已注册
	em.mutex.RLock()
	handlers := em.genericHandlers["order.unregister"][reflect.TypeOf(TestOrderData{})]
	em.mutex.RUnlock()

	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler after registration, got %d", len(handlers))
	}

	// 注销处理器
	UnregisterGeneric(em, UnregisterGenericRequest[TestOrderData]{
		EventType:   "order.unregister",
		HandlerName: "unregister_test_handler",
	})

	// 验证处理器已注销
	em.mutex.RLock()
	eventTypeMap := em.genericHandlers["order.unregister"]
	em.mutex.RUnlock()

	if eventTypeMap != nil {
		t.Error("Expected event type to be removed after unregistering all handlers")
	}

	// 发布事件，应该没有处理器响应
	orderData := TestOrderData{
		OrderID:     777,
		OrderType:   "unregister",
		Status:      "test",
		Description: "Unregister test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.unregister",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// 验证处理器没有被调用
	if handler.GetCallCount() != 0 {
		t.Errorf("Expected 0 calls after unregistration, got %d", handler.GetCallCount())
	}
}

// Test_DuplicateRegistration 测试重复注册
func Test_DuplicateRegistration(t *testing.T) {
	em := createTestEventManager()

	handler := NewTestGenericHandler[TestOrderData]("duplicate_handler")

	// 第一次注册
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.duplicate",
		HandlerName: "duplicate_handler",
		Handler:     handler,
	})

	// 重复注册相同的处理器（应该被忽略）
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.duplicate",
		HandlerName: "duplicate_handler",
		Handler:     handler,
	})

	// 验证只有一个处理器
	em.mutex.RLock()
	handlers := em.genericHandlers["order.duplicate"][reflect.TypeOf(TestOrderData{})]
	em.mutex.RUnlock()

	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler after duplicate registration, got %d", len(handlers))
	}

	// 发布事件，应该只调用一次
	orderData := TestOrderData{
		OrderID:     888,
		OrderType:   "duplicate",
		Status:      "test",
		Description: "Duplicate test",
	}

	err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
		EventType: "order.duplicate",
		Data:      orderData,
		Source:    "test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// 验证处理器只被调用一次
	if handler.GetCallCount() != 1 {
		t.Errorf("Expected 1 call, got %d", handler.GetCallCount())
	}
}

// Benchmark_PublishGenericEvent 基准测试泛型事件发布性能
func Benchmark_PublishGenericEvent(b *testing.B) {
	em := createTestEventManager()

	handler := NewTestGenericHandler[TestOrderData]("benchmark_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.benchmark",
		HandlerName: "benchmark_handler",
		Handler:     handler,
	})

	orderData := TestOrderData{
		OrderID:     999,
		OrderType:   "benchmark",
		Status:      "test",
		Description: "Benchmark test",
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
				EventType: "order.benchmark",
				Data:      orderData,
				Source:    "benchmark",
				Context:   ctx,
			})
			if err != nil {
				b.Fatalf("Failed to publish event: %v", err)
			}
		}
	})
}

// Benchmark_RegisterGenericHandler 基准测试泛型处理器注册性能
func Benchmark_RegisterGenericHandler(b *testing.B) {
	em := createTestEventManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler := NewTestGenericHandler[TestOrderData]("benchmark_handler")
		RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
			EventType:   "order.register_benchmark",
			HandlerName: "benchmark_handler",
			Handler:     handler,
		})
	}
}

// Test_OrderStatusChangeScenario 测试订单状态变更的完整业务场景
func Test_OrderStatusChangeScenario(t *testing.T) {
	em := createTestEventManager()

	// 模拟订单服务接收各种状态变更事件
	orderStatusHandler := NewTestGenericHandler[TestOrderData]("order_status_handler")
	deviceOperationHandler := NewTestGenericHandler[TestDeviceData]("device_operation_handler")

	// 注册处理器监听不同的事件类型
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.status.completed",
		HandlerName: "order_status_handler",
		Handler:     orderStatusHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.status.cancelled",
		HandlerName: "order_status_handler",
		Handler:     orderStatusHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.status.returning",
		HandlerName: "order_status_handler",
		Handler:     orderStatusHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestDeviceData]{
		EventType:   "device.operation.completed",
		HandlerName: "device_operation_handler",
		Handler:     deviceOperationHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[TestDeviceData]{
		EventType:   "device.operation.returning",
		HandlerName: "device_operation_handler",
		Handler:     deviceOperationHandler,
	})

	// 场景1: 订单完成流程
	t.Run("OrderCompletedScenario", func(t *testing.T) {
		orderStatusHandler.Reset()
		deviceOperationHandler.Reset()

		// 1. 设备操作完成
		deviceData := TestDeviceData{
			DeviceID: 100,
			Action:   "pool_entry",
			Status:   "completed",
			Result:   "Device successfully added to pool",
		}

		err := PublishGeneric(em, GenericEventRequest[TestDeviceData]{
			EventType: "device.operation.completed",
			Data:      deviceData,
			Source:    "device_service",
			Context:   context.Background(),
		})

		if err != nil {
			t.Fatalf("Failed to publish device operation completed event: %v", err)
		}

		// 2. 订单状态变更为完成
		orderData := TestOrderData{
			OrderID:     1001,
			OrderType:   "scaling",
			Status:      "completed",
			Description: "Order completed successfully - all devices processed",
		}

		err = PublishGeneric(em, GenericEventRequest[TestOrderData]{
			EventType: "order.status.completed",
			Data:      orderData,
			Source:    "order_service",
			Context:   context.Background(),
		})

		if err != nil {
			t.Fatalf("Failed to publish order completed event: %v", err)
		}

		// 验证事件被正确接收
		handledOrders := orderStatusHandler.GetHandledData()
		handledDevices := deviceOperationHandler.GetHandledData()

		if len(handledOrders) != 1 {
			t.Errorf("Expected 1 order event, got %d", len(handledOrders))
		}

		if len(handledDevices) != 1 {
			t.Errorf("Expected 1 device event, got %d", len(handledDevices))
		}

		// 验证事件内容
		if handledOrders[0].OrderID != 1001 {
			t.Errorf("Expected order ID 1001, got %d", handledOrders[0].OrderID)
		}

		if handledOrders[0].Status != "completed" {
			t.Errorf("Expected order status 'completed', got '%s'", handledOrders[0].Status)
		}

		if handledDevices[0].DeviceID != 100 {
			t.Errorf("Expected device ID 100, got %d", handledDevices[0].DeviceID)
		}

		if handledDevices[0].Action != "pool_entry" {
			t.Errorf("Expected device action 'pool_entry', got '%s'", handledDevices[0].Action)
		}
	})

	// 场景2: 订单取消流程
	t.Run("OrderCancelledScenario", func(t *testing.T) {
		orderStatusHandler.Reset()
		deviceOperationHandler.Reset()

		// 订单被取消
		orderData := TestOrderData{
			OrderID:     1002,
			OrderType:   "scaling",
			Status:      "cancelled",
			Description: "Order cancelled by user request",
		}

		err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
			EventType: "order.status.cancelled",
			Data:      orderData,
			Source:    "order_service",
			Context:   context.Background(),
		})

		if err != nil {
			t.Fatalf("Failed to publish order cancelled event: %v", err)
		}

		// 验证取消事件被正确接收
		handledOrders := orderStatusHandler.GetHandledData()

		if len(handledOrders) != 1 {
			t.Errorf("Expected 1 cancelled order event, got %d", len(handledOrders))
		}

		if handledOrders[0].OrderID != 1002 {
			t.Errorf("Expected order ID 1002, got %d", handledOrders[0].OrderID)
		}

		if handledOrders[0].Status != "cancelled" {
			t.Errorf("Expected order status 'cancelled', got '%s'", handledOrders[0].Status)
		}
	})

	// 场景3: 订单退回流程
	t.Run("OrderReturningScenario", func(t *testing.T) {
		orderStatusHandler.Reset()
		deviceOperationHandler.Reset()

		// 1. 订单进入退回状态
		orderData := TestOrderData{
			OrderID:     1003,
			OrderType:   "scaling",
			Status:      "returning",
			Description: "Order returning - rolling back device operations",
		}

		err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
			EventType: "order.status.returning",
			Data:      orderData,
			Source:    "order_service",
			Context:   context.Background(),
		})

		if err != nil {
			t.Fatalf("Failed to publish order returning event: %v", err)
		}

		// 2. 设备执行退回操作
		deviceData := TestDeviceData{
			DeviceID: 200,
			Action:   "pool_exit",
			Status:   "returning",
			Result:   "Device rollback in progress",
		}

		err = PublishGeneric(em, GenericEventRequest[TestDeviceData]{
			EventType: "device.operation.returning",
			Data:      deviceData,
			Source:    "device_service",
			Context:   context.Background(),
		})

		if err != nil {
			t.Fatalf("Failed to publish device returning event: %v", err)
		}

		// 验证退回流程事件被正确接收
		handledOrders := orderStatusHandler.GetHandledData()
		handledDevices := deviceOperationHandler.GetHandledData()

		if len(handledOrders) != 1 {
			t.Errorf("Expected 1 returning order event, got %d", len(handledOrders))
		}

		if len(handledDevices) != 1 {
			t.Errorf("Expected 1 returning device event, got %d", len(handledDevices))
		}

		// 验证退回事件内容
		if handledOrders[0].OrderID != 1003 {
			t.Errorf("Expected order ID 1003, got %d", handledOrders[0].OrderID)
		}

		if handledOrders[0].Status != "returning" {
			t.Errorf("Expected order status 'returning', got '%s'", handledOrders[0].Status)
		}

		if handledDevices[0].DeviceID != 200 {
			t.Errorf("Expected device ID 200, got %d", handledDevices[0].DeviceID)
		}

		if handledDevices[0].Action != "pool_exit" {
			t.Errorf("Expected device action 'pool_exit', got '%s'", handledDevices[0].Action)
		}

		if handledDevices[0].Status != "returning" {
			t.Errorf("Expected device status 'returning', got '%s'", handledDevices[0].Status)
		}
	})

	// 场景4: 多订单并发处理场景
	t.Run("ConcurrentOrdersScenario", func(t *testing.T) {
		orderStatusHandler.Reset()
		deviceOperationHandler.Reset()

		// 并发发送多个订单的不同状态事件
		orders := []struct {
			orderID   int
			status    string
			eventType string
		}{
			{2001, "completed", "order.status.completed"},
			{2002, "cancelled", "order.status.cancelled"},
			{2003, "returning", "order.status.returning"},
			{2004, "completed", "order.status.completed"},
		}

		// 使用WaitGroup等待所有事件处理完成
		var wg sync.WaitGroup

		for _, order := range orders {
			wg.Add(1)
			go func(orderID int, status, eventType string) {
				defer wg.Done()

				orderData := TestOrderData{
					OrderID:     orderID,
					OrderType:   "scaling",
					Status:      status,
					Description: fmt.Sprintf("Order %d - %s", orderID, status),
				}

				err := PublishGeneric(em, GenericEventRequest[TestOrderData]{
					EventType: eventType,
					Data:      orderData,
					Source:    "order_service",
					Context:   context.Background(),
				})

				if err != nil {
					t.Errorf("Failed to publish order %d event: %v", orderID, err)
				}
			}(order.orderID, order.status, order.eventType)
		}

		wg.Wait()

		// 等待事件处理完成
		time.Sleep(50 * time.Millisecond)

		// 验证所有订单事件都被处理
		handledOrders := orderStatusHandler.GetHandledData()

		if len(handledOrders) != 4 {
			t.Errorf("Expected 4 order events, got %d", len(handledOrders))
		}

		// 验证所有预期的订单ID都被处理了
		orderIDs := make(map[int]bool)
		for _, order := range handledOrders {
			orderIDs[order.OrderID] = true
		}

		expectedIDs := []int{2001, 2002, 2003, 2004}
		for _, expectedID := range expectedIDs {
			if !orderIDs[expectedID] {
				t.Errorf("Expected order ID %d to be processed", expectedID)
			}
		}
	})

	// 注意：由于每个子测试都调用了Reset()，这里我们无法验证总计数
	// 但是每个子测试内部已经验证了预期的事件处理
	t.Log("All order status change scenarios completed successfully")
	t.Log("Events were properly published and received by registered handlers")
	t.Log("System demonstrates loose coupling through event-driven architecture")
}
