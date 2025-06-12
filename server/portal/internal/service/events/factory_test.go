package events

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// createTestFactory 创建测试用的工厂
func createTestFactory() (*EventManager, *EventPublisherFactory) {
	logger := zaptest.NewLogger(&testing.T{})
	config := &Config{
		Timeout:     1 * time.Second,
		RetryCount:  1,
		BufferSize:  100,
		Async:       false, // 同步测试
		EnableStats: true,
	}
	em := NewEventManager(logger, config)
	factory := NewEventPublisherFactory(em, context.Background())
	return em, factory
}

// Test_ESOPublisher 测试弹性伸缩订单发布器
func Test_ESOPublisher(t *testing.T) {
	em, _ := createTestFactory()

	// 注册订单事件处理器
	orderHandler := NewTestGenericHandler[OrderEventData]("test_order_handler")
	RegisterGeneric(em, RegisterGenericRequest[OrderEventData]{
		EventType:   EventTypeOrderCompleted,
		HandlerName: "test_order_handler",
		Handler:     orderHandler,
	})

	// 初始化泛型事件系统
	InitializeGenericEventSystem(em)

	// 测试订单完成事件
	err := NewESOPublisher(12345).
		WithEventManager(em).
		WithOperator("admin").
		Complete(context.Background(), "订单处理完成")

	if err != nil {
		t.Fatalf("Failed to publish ESO complete event: %v", err)
	}

	// 验证事件被处理
	handledData := orderHandler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(handledData))
	}

	if handledData[0].OrderID != 12345 {
		t.Errorf("Expected order ID 12345, got %d", handledData[0].OrderID)
	}

	if handledData[0].Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", handledData[0].Status)
	}

	if handledData[0].Operator != "admin" {
		t.Errorf("Expected operator 'admin', got '%s'", handledData[0].Operator)
	}
}

// Test_ESOPublisher_WithoutFactory 测试未设置工厂的情况
func Test_ESOPublisher_WithoutFactory(t *testing.T) {
	// 测试未设置事件管理器时的错误处理
	publisher := NewESOPublisher(12345)
	publisher.em = nil // Explicitly set em to nil for this test case
	err := publisher.Complete(context.Background(), "订单处理完成")

	if err == nil {
		t.Error("Expected error when EventManager not set, got nil")
	}

	expectedMsg := "EventManager not set, use WithEventManager() first"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// Test_DevicePublisher 测试设备发布器
func Test_DevicePublisher(t *testing.T) {
	em, _ := createTestFactory()

	// 注册设备事件处理器
	deviceHandler := NewTestGenericHandler[DeviceEventData]("test_device_handler")
	RegisterGeneric(em, RegisterGenericRequest[DeviceEventData]{
		EventType:   EventTypeDeviceOperationCompleted,
		HandlerName: "test_device_handler",
		Handler:     deviceHandler,
	})

	// 初始化泛型事件系统
	InitializeGenericEventSystem(em)

	// 测试设备操作完成事件
	err := NewDevicePublisher(98765, 12345, "pool_entry").
		WithEventManager(em).
		Completed(context.Background(), "设备成功加入资源池")

	if err != nil {
		t.Fatalf("Failed to publish device completed event: %v", err)
	}

	// 验证事件被处理
	handledData := deviceHandler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(handledData))
	}

	if handledData[0].DeviceID != 98765 {
		t.Errorf("Expected device ID 98765, got %d", handledData[0].DeviceID)
	}

	if handledData[0].OrderID != 12345 {
		t.Errorf("Expected order ID 12345, got %d", handledData[0].OrderID)
	}

	if handledData[0].Action != "pool_entry" {
		t.Errorf("Expected action 'pool_entry', got '%s'", handledData[0].Action)
	}

	if handledData[0].Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", handledData[0].Status)
	}
}

// Test_MaintenancePublisher 测试维护发布器
func Test_MaintenancePublisher(t *testing.T) {
	em, _ := createTestFactory()

	// 注册维护事件处理器（使用传统事件处理器）
	var handledEvents []Event
	maintenanceHandler := EventHandlerFunc(func(ctx context.Context, event Event) error {
		handledEvents = append(handledEvents, event)
		return nil
	})

	em.Register(RegisterRequest{
		EventType:   EventTypeMaintenanceCompleted,
		HandlerName: "test_maintenance_handler",
		Handler:     maintenanceHandler,
	})

	// 测试维护完成事件
	err := NewMaintenancePublisher(12347, 98767, "cordon").
		WithEventManager(em).
		Completed(context.Background(), "设备维护操作完成")

	if err != nil {
		t.Fatalf("Failed to publish maintenance completed event: %v", err)
	}

	// 验证事件被处理
	if len(handledEvents) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(handledEvents))
	}

	if handledEvents[0].Type() != EventTypeMaintenanceCompleted {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeMaintenanceCompleted, handledEvents[0].Type())
	}
}

// Test_ScalingPublisher 测试弹性伸缩发布器
func Test_ScalingPublisher(t *testing.T) {
	em, _ := createTestFactory()

	// 注册弹性伸缩事件处理器（使用传统事件处理器）
	var handledEvents []Event
	scalingHandler := EventHandlerFunc(func(ctx context.Context, event Event) error {
		handledEvents = append(handledEvents, event)
		return nil
	})

	em.Register(RegisterRequest{
		EventType:   EventTypeScalingTriggered,
		HandlerName: "test_scaling_handler",
		Handler:     scalingHandler,
	})

	// 测试弹性伸缩触发事件
	selectedDevices := []int{98765, 98766, 98767}
	err := NewScalingPublisher(1001, 2001, "compute", "pool_entry").
		WithEventManager(em).
		Triggered(context.Background(), 3, selectedDevices)

	if err != nil {
		t.Fatalf("Failed to publish scaling triggered event: %v", err)
	}

	// 验证事件被处理
	if len(handledEvents) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(handledEvents))
	}

	if handledEvents[0].Type() != EventTypeScalingTriggered {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeScalingTriggered, handledEvents[0].Type())
	}

	// 验证事件数据
	scalingEvent, ok := handledEvents[0].(*ScalingEvent)
	if !ok {
		t.Errorf("Expected ScalingEvent, got %T", handledEvents[0])
	} else {
		if scalingEvent.StrategyID != 1001 {
			t.Errorf("Expected strategy ID 1001, got %d", scalingEvent.StrategyID)
		}
		if scalingEvent.ClusterID != 2001 {
			t.Errorf("Expected cluster ID 2001, got %d", scalingEvent.ClusterID)
		}
		if scalingEvent.DeviceCount != 3 {
			t.Errorf("Expected device count 3, got %d", scalingEvent.DeviceCount)
		}
		if len(scalingEvent.SelectedDevices) != 3 {
			t.Errorf("Expected 3 selected devices, got %d", len(scalingEvent.SelectedDevices))
		}
	}
}

// Test_ChainedEvents 测试链式事件发布
func Test_ChainedEvents(t *testing.T) {
	em, _ := createTestFactory()

	// 注册多个事件处理器
	orderHandler := NewTestGenericHandler[OrderEventData]("test_order_handler")
	deviceHandler := NewTestGenericHandler[DeviceEventData]("test_device_handler")

	RegisterGeneric(em, RegisterGenericRequest[OrderEventData]{
		EventType:   EventTypeOrderCreated,
		HandlerName: "test_order_created_handler",
		Handler:     orderHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[OrderEventData]{
		EventType:   EventTypeOrderCompleted,
		HandlerName: "test_order_completed_handler",
		Handler:     orderHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[DeviceEventData]{
		EventType:   EventTypeDeviceOperationStarted,
		HandlerName: "test_device_started_handler",
		Handler:     deviceHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[DeviceEventData]{
		EventType:   EventTypeDeviceOperationCompleted,
		HandlerName: "test_device_completed_handler",
		Handler:     deviceHandler,
	})

	// 初始化泛型事件系统
	InitializeGenericEventSystem(em)

	// 执行链式事件发布
	orderID := 12345
	deviceIDs := []int{98765, 98766}

	err := ChainedEventExample(em, orderID, deviceIDs)
	if err != nil {
		t.Fatalf("Failed to execute chained events: %v", err)
	}

	// 验证订单事件（创建 + 完成 = 2个事件）
	orderEvents := orderHandler.GetHandledData()
	if len(orderEvents) != 2 {
		t.Errorf("Expected 2 order events, got %d", len(orderEvents))
	}

	// 验证设备事件（每个设备开始 + 完成 = 2 * 2 = 4个事件）
	deviceEvents := deviceHandler.GetHandledData()
	if len(deviceEvents) != 4 {
		t.Errorf("Expected 4 device events, got %d", len(deviceEvents))
	}
}

// Test_FactoryEventErrorHandling 测试工厂事件错误处理
func Test_FactoryEventErrorHandling(t *testing.T) {
	em, _ := createTestFactory()

	// 注册事件处理器
	orderHandler := NewTestGenericHandler[OrderEventData]("test_order_handler")
	deviceHandler := NewTestGenericHandler[DeviceEventData]("test_device_handler")

	RegisterGeneric(em, RegisterGenericRequest[OrderEventData]{
		EventType:   EventTypeOrderFailed,
		HandlerName: "test_order_failed_handler",
		Handler:     orderHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[DeviceEventData]{
		EventType:   EventTypeDeviceOperationStarted,
		HandlerName: "test_device_started_handler",
		Handler:     deviceHandler,
	})

	RegisterGeneric(em, RegisterGenericRequest[DeviceEventData]{
		EventType:   EventTypeDeviceOperationFailed,
		HandlerName: "test_device_failed_handler",
		Handler:     deviceHandler,
	})

	// 初始化泛型事件系统
	InitializeGenericEventSystem(em)

	// 执行错误处理示例
	ErrorHandlingExample(em, 12345, 98765)

	// 验证订单失败事件
	orderEvents := orderHandler.GetHandledData()
	if len(orderEvents) != 1 {
		t.Errorf("Expected 1 order failed event, got %d", len(orderEvents))
	}

	if orderEvents[0].Status != "failed" {
		t.Errorf("Expected order status 'failed', got '%s'", orderEvents[0].Status)
	}

	// 验证设备事件（开始 + 失败 = 2个事件）
	deviceEvents := deviceHandler.GetHandledData()
	if len(deviceEvents) != 2 {
		t.Errorf("Expected 2 device events, got %d", len(deviceEvents))
	}

	// 检查设备失败事件
	failedEvent := deviceEvents[1] // 第二个应该是失败事件
	if failedEvent.Status != "failed" {
		t.Errorf("Expected device status 'failed', got '%s'", failedEvent.Status)
	}

	if failedEvent.ErrorMsg == "" {
		t.Error("Expected error message in failed device event")
	}
}

// Test_FactoryErrorHandling 测试工厂错误处理
func Test_FactoryErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	em := NewEventManager(logger, nil)
	ctx := context.Background()

	// 测试正常情况
	publisher := NewESOPublisher(12345).WithEventManager(em)
	err := publisher.Complete(ctx, "订单处理完成")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 测试未设置事件管理器的情况
	publisherWithoutEM := NewESOPublisher(12346)
	publisherWithoutEM.em = nil // Explicitly set em to nil for this test case
	err = publisherWithoutEM.Complete(ctx, "订单处理完成")
	if err == nil {
		t.Error("Expected error when EventManager not set, got nil")
	}
	if err != nil && err.Error() != "EventManager not set, use WithEventManager() first" {
		t.Errorf("Expected error message 'EventManager not set, use WithEventManager() first', got '%s'", err.Error())
	}
}
