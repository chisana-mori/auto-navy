package service

import (
	"context"
	"navy-ng/models/portal"
)

// RichOrder 是一个所有订单详情DTO都必须实现的接口。
// 它提供了一个多态的方式来访问基础订单信息。
type RichOrder interface {
	GetBaseOrder() *portal.Order
}

// UnifiedOrderService 定义了统一订单管理服务的契约。
// 它使用泛型来处理不同订单类型的创建（C）和返回（T）数据结构。
type UnifiedOrderService[T RichOrder, C any] interface {
	CreateOrder(ctx context.Context, createDTO C) (T, error)
	GetOrder(ctx context.Context, id int64) (T, error)
	ListOrders(ctx context.Context, query any) ([]T, int64, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string, executor string, reason string) error

	// 订单生命周期快捷操作
	ProcessOrder(ctx context.Context, id int64, executor string) error
	CompleteOrder(ctx context.Context, id int64, executor string) error
	FailOrder(ctx context.Context, id int64, executor string, reason string) error
	CancelOrder(ctx context.Context, id int64, executor string) error
}

// serviceRegistry 用于存储不同订单类型对应的服务实例。
var serviceRegistry = make(map[string]any)

// RegisterOrderService 注册一个订单服务。
func RegisterOrderService(orderType string, service any) {
	serviceRegistry[orderType] = service
}

// GetOrderService 从注册表中获取一个订单服务。
func GetOrderService(orderType string) (any, bool) {
	service, found := serviceRegistry[orderType]
	return service, found
}