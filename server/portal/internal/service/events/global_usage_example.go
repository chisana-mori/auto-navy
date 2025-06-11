package events

import (
	"context"
	"fmt"
)

// GlobalEventManagerUsageExample 展示如何在其他服务中使用全局 EventManager
func GlobalEventManagerUsageExample() {
	// 在任何地方都可以获取全局 EventManager 实例
	em := GetGlobalEventManager()

	// 使用工厂模式创建 publisher
	esoPublisher := NewESOPublisher(12345).WithEventManager(em)

	// 发布事件
	if err := esoPublisher.Created(context.Background(), "弹性伸缩订单已创建"); err != nil {
		fmt.Printf("发布事件失败: %v\n", err)
	}
}

// ServiceLayerExample 展示在服务层中如何使用全局 EventManager
type OrderService struct {
	// 不需要在结构体中存储 EventManager
}

func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) CreateOrder(orderID int, orderType string) error {
	// 业务逻辑...

	// 直接获取全局 EventManager 发布事件
	em := GetGlobalEventManager()
	esoPublisher := NewESOPublisher(orderID).WithEventManager(em)

	return esoPublisher.Created(context.Background(), "订单创建成功")
}

func (s *OrderService) CompleteOrder(orderID int) error {
	// 业务逻辑...

	// 使用全局 EventManager
	em := GetGlobalEventManager()
	esoPublisher := NewESOPublisher(orderID).WithEventManager(em)

	return esoPublisher.Complete(context.Background(), "订单完成")
}

// HandlerRegistrationExample 展示如何注册事件处理器
func HandlerRegistrationExample() {
	// 获取全局 EventManager
	em := GetGlobalEventManager()

	// 注册泛型事件处理器
	RegisterGenericFunc(em, RegisterGenericFuncRequest[map[string]interface{}]{
		EventType:   "order.created",
		HandlerName: "order_notification_handler",
		HandlerFunc: func(ctx context.Context, event *GenericEvent[map[string]interface{}]) error {
			fmt.Printf("处理订单创建事件: %+v\n", event.EventData)
			return nil
		},
	})
}

// MiddlewareExample 展示在中间件中如何使用全局 EventManager
func MiddlewareExample() {
	// 在 HTTP 中间件中记录请求事件
	em := GetGlobalEventManager()

	// 发布请求事件
	err := PublishGeneric(em, GenericEventRequest[map[string]interface{}]{
		EventType: "http.request",
		Data: map[string]interface{}{
			"method": "POST",
			"path":   "/api/orders",
			"user_id": 123,
		},
		Source: "http_middleware",
	})

	if err != nil {
		fmt.Printf("发布 HTTP 请求事件失败: %v\n", err)
	}
}

// BackgroundJobExample 展示在后台任务中如何使用全局 EventManager
func BackgroundJobExample() {
	// 在后台任务中使用全局 EventManager
	em := GetGlobalEventManager()

	// 发布任务完成事件
	err := PublishGeneric(em, GenericEventRequest[map[string]interface{}]{
		EventType: "job.completed",
		Data: map[string]interface{}{
			"job_id":   "cleanup-001",
			"duration": "5m30s",
			"status":   "success",
		},
		Source: "background_job",
	})

	if err != nil {
		fmt.Printf("发布任务完成事件失败: %v\n", err)
	}
}