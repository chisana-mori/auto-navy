package service

import (
	"context"
	"fmt"
	"navy-ng/models/portal"
	"time"
)

// UnifiedOrderService 统一的订单服务接口
type UnifiedOrderService interface {
	// 基础订单操作
	CreateOrder(ctx context.Context, dto interface{}) (int64, error)
	GetOrder(ctx context.Context, id int64) (interface{}, error)
	ListOrders(ctx context.Context, query UnifiedOrderQuery) (interface{}, int64, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string, executor string, reason string) error
	DeleteOrder(ctx context.Context, id int64) error

	// 扩展操作（可选实现）
	GetOrderDevices(ctx context.Context, id int64) (interface{}, error)
	UpdateOrderDeviceStatus(ctx context.Context, orderID int64, deviceID int64, status string) error
}

// UnifiedOrderQuery 统一的订单查询参数
type UnifiedOrderQuery struct {
	Type      string     `json:"type"`      // 订单类型
	Status    string     `json:"status"`    // 订单状态
	CreatedBy string     `json:"createdBy"` // 创建者
	Page      int        `json:"page"`      // 页码
	PageSize  int        `json:"pageSize"`  // 每页大小
	StartTime *time.Time `json:"startTime"` // 开始时间
	EndTime   *time.Time `json:"endTime"`   // 结束时间

	// 弹性伸缩特有参数
	ClusterID  int64  `json:"clusterId"`  // 集群ID
	StrategyID int64  `json:"strategyId"` // 策略ID
	ActionType string `json:"actionType"` // 动作类型
}

// OrderServiceRegistry 订单服务注册器
type OrderServiceRegistry struct {
	services map[string]UnifiedOrderService
}

// NewOrderServiceRegistry 创建订单服务注册器
func NewOrderServiceRegistry() *OrderServiceRegistry {
	return &OrderServiceRegistry{
		services: make(map[string]UnifiedOrderService),
	}
}

// RegisterService 注册订单服务
func (r *OrderServiceRegistry) RegisterService(orderType string, service UnifiedOrderService) {
	r.services[orderType] = service
}

// GetService 获取订单服务
func (r *OrderServiceRegistry) GetService(orderType string) (UnifiedOrderService, bool) {
	service, exists := r.services[orderType]
	return service, exists
}

// GetSupportedTypes 获取支持的订单类型
func (r *OrderServiceRegistry) GetSupportedTypes() []string {
	types := make([]string, 0, len(r.services))
	for orderType := range r.services {
		types = append(types, orderType)
	}
	return types
}

// ElasticScalingOrderAdapter 弹性伸缩订单适配器
type ElasticScalingOrderAdapter struct {
	elasticScalingService *ElasticScalingService
}

// NewElasticScalingOrderAdapter 创建弹性伸缩订单适配器
func NewElasticScalingOrderAdapter(elasticScalingService *ElasticScalingService) *ElasticScalingOrderAdapter {
	return &ElasticScalingOrderAdapter{
		elasticScalingService: elasticScalingService,
	}
}

// CreateOrder 创建订单
func (a *ElasticScalingOrderAdapter) CreateOrder(ctx context.Context, dto interface{}) (int64, error) {
	orderDTO, ok := dto.(OrderDTO)
	if !ok {
		return 0, ErrInvalidOrderType
	}
	return a.elasticScalingService.CreateOrder(orderDTO)
}

// GetOrder 获取订单详情
func (a *ElasticScalingOrderAdapter) GetOrder(ctx context.Context, id int64) (interface{}, error) {
	return a.elasticScalingService.GetOrder(id)
}

// ListOrders 获取订单列表
func (a *ElasticScalingOrderAdapter) ListOrders(ctx context.Context, query UnifiedOrderQuery) (interface{}, int64, error) {
	return a.elasticScalingService.ListOrders(query.ClusterID, query.StrategyID, query.ActionType, query.Status, query.Page, query.PageSize)
}

// UpdateOrderStatus 更新订单状态
func (a *ElasticScalingOrderAdapter) UpdateOrderStatus(ctx context.Context, id int64, status string, executor string, reason string) error {
	return a.elasticScalingService.UpdateOrderStatus(id, status, executor, reason)
}

// DeleteOrder 删除订单
func (a *ElasticScalingOrderAdapter) DeleteOrder(ctx context.Context, id int64) error {
	// 弹性伸缩订单通常不支持删除，返回错误
	return ErrOperationNotSupported
}

// GetOrderDevices 获取订单设备
func (a *ElasticScalingOrderAdapter) GetOrderDevices(ctx context.Context, id int64) (interface{}, error) {
	return a.elasticScalingService.GetOrderDevices(id)
}

// UpdateOrderDeviceStatus 更新订单设备状态
func (a *ElasticScalingOrderAdapter) UpdateOrderDeviceStatus(ctx context.Context, orderID int64, deviceID int64, status string) error {
	return a.elasticScalingService.UpdateOrderDeviceStatus(orderID, deviceID, status)
}

// GeneralOrderAdapter 通用订单适配器
type GeneralOrderAdapter struct {
	orderService OrderService
}

// NewGeneralOrderAdapter 创建通用订单适配器
func NewGeneralOrderAdapter(orderService OrderService) *GeneralOrderAdapter {
	return &GeneralOrderAdapter{
		orderService: orderService,
	}
}

// CreateOrder 创建订单
func (a *GeneralOrderAdapter) CreateOrder(ctx context.Context, dto interface{}) (int64, error) {
	order, ok := dto.(*portal.Order)
	if !ok {
		return 0, ErrInvalidOrderType
	}
	err := a.orderService.CreateOrder(ctx, order)
	return order.ID, err
}

// GetOrder 获取订单详情
func (a *GeneralOrderAdapter) GetOrder(ctx context.Context, id int64) (interface{}, error) {
	return a.orderService.GetOrderByID(ctx, id)
}

// ListOrders 获取订单列表
func (a *GeneralOrderAdapter) ListOrders(ctx context.Context, query UnifiedOrderQuery) (interface{}, int64, error) {
	orderQuery := OrderQuery{
		Type:      portal.OrderType(query.Type),
		Status:    portal.OrderStatus(query.Status),
		CreatedBy: query.CreatedBy,
		Page:      query.Page,
		PageSize:  query.PageSize,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
	}
	return a.orderService.ListOrders(ctx, orderQuery)
}

// UpdateOrderStatus 更新订单状态
func (a *GeneralOrderAdapter) UpdateOrderStatus(ctx context.Context, id int64, status string, executor string, reason string) error {
	return a.orderService.UpdateOrderStatus(ctx, id, portal.OrderStatus(status), executor, reason)
}

// DeleteOrder 删除订单
func (a *GeneralOrderAdapter) DeleteOrder(ctx context.Context, id int64) error {
	return a.orderService.DeleteOrder(ctx, id)
}

// GetOrderDevices 获取订单设备（通用订单不支持）
func (a *GeneralOrderAdapter) GetOrderDevices(ctx context.Context, id int64) (interface{}, error) {
	return nil, ErrOperationNotSupported
}

// UpdateOrderDeviceStatus 更新订单设备状态（通用订单不支持）
func (a *GeneralOrderAdapter) UpdateOrderDeviceStatus(ctx context.Context, orderID int64, deviceID int64, status string) error {
	return ErrOperationNotSupported
}

// 错误定义
var (
	ErrInvalidOrderType      = fmt.Errorf("invalid order type")
	ErrOperationNotSupported = fmt.Errorf("operation not supported")
	ErrOrderServiceNotFound  = fmt.Errorf("order service not found")
)
