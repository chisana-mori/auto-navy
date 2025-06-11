package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// 测试用转换器
type TestOrderConverter struct{}

func (c *TestOrderConverter) Convert(data interface{}) (TestOrderData, error) {
	switch v := data.(type) {
	case TestOrderData:
		return v, nil
	case map[string]interface{}:
		return c.convertFromMap(v)
	case string:
		return c.convertFromJSON(v)
	case *OrderStatusChangedEvent:
		return TestOrderData{
			OrderID:     v.OrderID,
			OrderType:   v.OrderType,
			Status:      v.NewStatus,
			Description: v.Reason,
		}, nil
	default:
		var zero TestOrderData
		return zero, fmt.Errorf("unsupported data type: %T", data)
	}
}

func (c *TestOrderConverter) CanConvert(data interface{}) bool {
	switch data.(type) {
	case TestOrderData, map[string]interface{}, string, *OrderStatusChangedEvent:
		return true
	default:
		return false
	}
}

func (c *TestOrderConverter) convertFromMap(m map[string]interface{}) (TestOrderData, error) {
	result := TestOrderData{}

	if id, ok := m["order_id"]; ok {
		if idFloat, ok := id.(float64); ok {
			result.OrderID = int(idFloat)
		} else if idInt, ok := id.(int); ok {
			result.OrderID = idInt
		}
	}

	if orderType, ok := m["order_type"].(string); ok {
		result.OrderType = orderType
	}

	if status, ok := m["status"].(string); ok {
		result.Status = status
	}

	if description, ok := m["description"].(string); ok {
		result.Description = description
	}

	return result, nil
}

func (c *TestOrderConverter) convertFromJSON(jsonStr string) (TestOrderData, error) {
	var result TestOrderData
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return TestOrderData{}, err
	}
	return result, nil
}

// Test_RegisterConverter 测试转换器注册
func Test_RegisterConverter(t *testing.T) {
	em := createTestEventManager()

	converter := &TestOrderConverter{}
	RegisterConverter(em, converter)

	// 验证转换器已注册
	retrievedConverter, exists := GetConverter[TestOrderData](em)
	if !exists {
		t.Error("Expected converter to be registered")
	}

	if retrievedConverter == nil {
		t.Error("Retrieved converter should not be nil")
	}

	// 测试转换器功能
	testData := TestOrderData{
		OrderID:     123,
		OrderType:   "test",
		Status:      "active",
		Description: "Test order",
	}

	converted, err := retrievedConverter.Convert(testData)
	if err != nil {
		t.Fatalf("Failed to convert data: %v", err)
	}

	if converted.OrderID != 123 {
		t.Errorf("Expected order ID 123, got %d", converted.OrderID)
	}
}

// Test_PublishGenericWithConverter 测试使用转换器发布事件
func Test_PublishGenericWithConverter(t *testing.T) {
	em := createTestEventManager()

	// 注册转换器
	converter := &TestOrderConverter{}
	RegisterConverter(em, converter)

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("converter_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.converter.test",
		HandlerName: "converter_handler",
		Handler:     handler,
	})

	// 测试从map转换
	mapData := map[string]interface{}{
		"order_id":    float64(456), // JSON中数字被解析为float64
		"order_type":  "scaling",
		"status":      "processing",
		"description": "Converted from map",
	}

	err := PublishGenericWithConverter[TestOrderData](em, GenericConvertRequest[TestOrderData]{
		EventType: "order.converter.test",
		RawData:   mapData,
		Source:    "converter_test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event with converter: %v", err)
	}

	// 验证转换和处理
	handledData := handler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled item, got %d", len(handledData))
	}

	if handledData[0].OrderID != 456 {
		t.Errorf("Expected order ID 456, got %d", handledData[0].OrderID)
	}

	if handledData[0].OrderType != "scaling" {
		t.Errorf("Expected order type 'scaling', got '%s'", handledData[0].OrderType)
	}
}

// Test_ConvertFromJSON 测试从JSON字符串转换
func Test_ConvertFromJSON(t *testing.T) {
	em := createTestEventManager()

	// 注册转换器
	converter := &TestOrderConverter{}
	RegisterConverter(em, converter)

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("json_converter_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.json.test",
		HandlerName: "json_converter_handler",
		Handler:     handler,
	})

	// 测试从JSON字符串转换
	jsonData := `{
		"order_id": 789,
		"order_type": "maintenance",
		"status": "completed",
		"description": "Converted from JSON"
	}`

	err := PublishGenericWithConverter[TestOrderData](em, GenericConvertRequest[TestOrderData]{
		EventType: "order.json.test",
		RawData:   jsonData,
		Source:    "json_test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish event from JSON: %v", err)
	}

	// 验证转换和处理
	handledData := handler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled item, got %d", len(handledData))
	}

	if handledData[0].OrderID != 789 {
		t.Errorf("Expected order ID 789, got %d", handledData[0].OrderID)
	}

	if handledData[0].Description != "Converted from JSON" {
		t.Errorf("Expected description 'Converted from JSON', got '%s'", handledData[0].Description)
	}
}

// Test_ConvertFromLegacyEvent 测试从旧事件结构转换
func Test_ConvertFromLegacyEvent(t *testing.T) {
	em := createTestEventManager()

	// 注册转换器
	converter := &TestOrderConverter{}
	RegisterConverter(em, converter)

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("legacy_converter_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.legacy.test",
		HandlerName: "legacy_converter_handler",
		Handler:     handler,
	})

	// 创建旧的事件结构
	legacyEvent := &OrderStatusChangedEvent{
		BaseEvent: BaseEvent{
			EventType: "order.legacy.test",
			Source:    "legacy_system",
		},
		OrderID:   999,
		OrderType: "legacy_order",
		OldStatus: "pending",
		NewStatus: "active",
		Executor:  "system",
		Reason:    "Converted from legacy event",
	}

	err := PublishGenericWithConverter[TestOrderData](em, GenericConvertRequest[TestOrderData]{
		EventType: "order.legacy.test",
		RawData:   legacyEvent,
		Source:    "legacy_test",
		Context:   context.Background(),
	})

	if err != nil {
		t.Fatalf("Failed to publish legacy event: %v", err)
	}

	// 验证转换和处理
	handledData := handler.GetHandledData()
	if len(handledData) != 1 {
		t.Errorf("Expected 1 handled item, got %d", len(handledData))
	}

	if handledData[0].OrderID != 999 {
		t.Errorf("Expected order ID 999, got %d", handledData[0].OrderID)
	}

	if handledData[0].Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", handledData[0].Status)
	}

	if handledData[0].Description != "Converted from legacy event" {
		t.Errorf("Expected description 'Converted from legacy event', got '%s'", handledData[0].Description)
	}
}

// Test_DefaultConverter 测试默认转换器
func Test_DefaultConverter(t *testing.T) {
	em := createTestEventManager()

	// 不注册自定义转换器，使用默认转换器
	defaultConverter, exists := GetConverter[TestOrderData](em)
	if exists {
		t.Error("Should not have custom converter registered")
	}

	// 测试直接类型匹配
	testData := TestOrderData{
		OrderID:     100,
		OrderType:   "direct",
		Status:      "test",
		Description: "Direct conversion",
	}

	converted, err := defaultConverter.Convert(testData)
	if err != nil {
		t.Fatalf("Default converter should handle direct type match: %v", err)
	}

	if converted.OrderID != 100 {
		t.Errorf("Expected order ID 100, got %d", converted.OrderID)
	}

	// 测试不兼容类型转换
	invalidData := "invalid data"
	_, err = defaultConverter.Convert(invalidData)
	if err == nil {
		t.Error("Expected error for invalid data type conversion")
	}
}

// Test_ConverterCanConvert 测试转换器的CanConvert方法
func Test_ConverterCanConvert(t *testing.T) {
	converter := &TestOrderConverter{}

	// 测试支持的类型
	supportedTypes := []interface{}{
		TestOrderData{},
		map[string]interface{}{},
		"json string",
		&OrderStatusChangedEvent{},
	}

	for i, data := range supportedTypes {
		if !converter.CanConvert(data) {
			t.Errorf("Converter should support type %d: %T", i, data)
		}
	}

	// 测试不支持的类型
	unsupportedTypes := []interface{}{
		123,
		struct{}{},
		[]int{},
	}

	for i, data := range unsupportedTypes {
		if converter.CanConvert(data) {
			t.Errorf("Converter should not support type %d: %T", i, data)
		}
	}
}

// Test_ConversionError 测试转换错误处理
func Test_ConversionError(t *testing.T) {
	em := createTestEventManager()

	// 注册转换器
	converter := &TestOrderConverter{}
	RegisterConverter(em, converter)

	// 注册处理器
	handler := NewTestGenericHandler[TestOrderData]("error_converter_handler")
	RegisterGeneric(em, RegisterGenericRequest[TestOrderData]{
		EventType:   "order.conversion.error",
		HandlerName: "error_converter_handler",
		Handler:     handler,
	})

	// 尝试转换无效的JSON
	invalidJSON := `{"order_id": "invalid", "incomplete json`

	err := PublishGenericWithConverter[TestOrderData](em, GenericConvertRequest[TestOrderData]{
		EventType: "order.conversion.error",
		RawData:   invalidJSON,
		Source:    "error_test",
		Context:   context.Background(),
	})

	if err == nil {
		t.Error("Expected conversion error for invalid JSON")
	}

	// 验证处理器没有被调用
	if handler.GetCallCount() != 0 {
		t.Errorf("Handler should not be called when conversion fails, got %d calls", handler.GetCallCount())
	}
}

// Test_MultipleConverters 测试多个转换器
func Test_MultipleConverters(t *testing.T) {
	em := createTestEventManager()

	// 注册不同类型的转换器
	orderConverter := &TestOrderConverter{}
	RegisterConverter(em, orderConverter)

	// 自定义设备转换器
	deviceConverter := &DefaultTypeConverter[TestDeviceData]{}
	RegisterConverter(em, deviceConverter)

	// 验证不同类型的转换器都被正确注册
	retrievedOrderConverter, orderExists := GetConverter[TestOrderData](em)
	if !orderExists {
		t.Error("Order converter should be registered")
	}

	retrievedDeviceConverter, deviceExists := GetConverter[TestDeviceData](em)
	if !deviceExists {
		t.Error("Device converter should be registered")
	}

	// 验证转换器类型正确
	if reflect.TypeOf(retrievedOrderConverter) != reflect.TypeOf(orderConverter) {
		t.Error("Retrieved order converter type mismatch")
	}

	if reflect.TypeOf(retrievedDeviceConverter) != reflect.TypeOf(deviceConverter) {
		t.Error("Retrieved device converter type mismatch")
	}
}

// Benchmark_ConvertFromMap 基准测试从map转换性能
func Benchmark_ConvertFromMap(b *testing.B) {
	converter := &TestOrderConverter{}

	mapData := map[string]interface{}{
		"order_id":    float64(123),
		"order_type":  "benchmark",
		"status":      "active",
		"description": "Benchmark test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(mapData)
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}

// Benchmark_ConvertFromJSON 基准测试从JSON转换性能
func Benchmark_ConvertFromJSON(b *testing.B) {
	converter := &TestOrderConverter{}

	jsonData := `{
		"order_id": 123,
		"order_type": "benchmark",
		"status": "active",
		"description": "Benchmark test"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.Convert(jsonData)
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}
