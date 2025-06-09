package service

import (
	"context"
	"fmt"
	"time"

	"navy-ng/models/portal"

	"gorm.io/gorm"
)

const (
	// 预加载字段
	preloadElasticScalingDetail = "ElasticScalingDetail"
	preloadDevices              = "Devices"

	// 数据库字段
	fieldOrderNumber    = "order_number = ?"
	fieldID             = "id = ?"
	fieldType           = "type = ?"
	fieldStatus         = "status"
	fieldStatusEq       = "status = ?"
	fieldCreatedBy      = "created_by = ?"
	fieldNameLike       = "name LIKE ?"
	fieldCreatedAtGTE   = "created_at >= ?"
	fieldCreatedAtLTE   = "created_at <= ?"
	fieldUpdatedAt      = "updated_at"
	fieldExecutor       = "executor"
	fieldExecutionTime  = "execution_time"
	fieldCompletionTime = "completion_time"
	fieldFailureReason  = "failure_reason"

	// 默认消息
	msgOrderCancelled = "订单已取消"
	msgOrderIgnored   = "订单已忽略"

	// 订单号前缀
	orderNumberPrefix = "ORD%d"
)

// OrderQuery 订单查询参数
type OrderQuery struct {
	Type      portal.OrderType   `json:"type"`
	Status    portal.OrderStatus `json:"status"`
	CreatedBy string             `json:"createdBy"`
	Name      string             `json:"name"` // 订单名称，支持模糊查询
	Page      int                `json:"page"`
	PageSize  int                `json:"pageSize"`
	StartTime *time.Time         `json:"startTime"`
	EndTime   *time.Time         `json:"endTime"`
}

// BaseOrderDTO 基础订单DTO
type BaseOrderDTO struct {
	ID             int64              `json:"id"`
	OrderNumber    string             `json:"orderNumber"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Type           portal.OrderType   `json:"type"`
	Status         portal.OrderStatus `json:"status"`
	Executor       string             `json:"executor"`
	ExecutionTime  *time.Time         `json:"executionTime"`
	CreatedBy      string             `json:"createdBy"`
	CompletionTime *time.Time         `json:"completionTime"`
	FailureReason  string             `json:"failureReason"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

// OrderService 通用订单服务接口
type OrderService interface {
	// 通用订单操作
	CreateOrder(ctx context.Context, order *portal.Order) error
	GetOrderByID(ctx context.Context, id int64) (*portal.Order, error)
	GetOrderByNumber(ctx context.Context, orderNumber string) (*portal.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status portal.OrderStatus, executor string, reason string) error
	ListOrders(ctx context.Context, query OrderQuery) ([]portal.Order, int64, error)
	DeleteOrder(ctx context.Context, id int64) error

	// 订单处理流程
	ProcessOrder(ctx context.Context, id int64, executor string) error
	CompleteOrder(ctx context.Context, id int64, executor string) error
	FailOrder(ctx context.Context, id int64, executor string, reason string) error
	CancelOrder(ctx context.Context, id int64, executor string) error
	IgnoreOrder(ctx context.Context, id int64, executor string) error
}

// orderServiceImpl 通用订单服务实现
type orderServiceImpl struct {
	db *gorm.DB
}

// NewOrderService 创建订单服务实例
func NewOrderService(db *gorm.DB) OrderService {
	return &orderServiceImpl{
		db: db,
	}
}

// CreateOrder 创建订单
func (s *orderServiceImpl) CreateOrder(ctx context.Context, order *portal.Order) error {
	if order.OrderNumber == "" {
		order.OrderNumber = s.generateOrderNumber()
	}

	if order.Status == "" {
		order.Status = portal.OrderStatusPending
	}

	return s.db.WithContext(ctx).Create(order).Error
}

// GetOrderByID 根据ID获取订单
func (s *orderServiceImpl) GetOrderByID(ctx context.Context, id int64) (*portal.Order, error) {
	var order portal.Order
	err := s.db.WithContext(ctx).
		Preload(preloadElasticScalingDetail).
		Preload(preloadDevices).
		First(&order, id).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetOrderByNumber 根据订单号获取订单
func (s *orderServiceImpl) GetOrderByNumber(ctx context.Context, orderNumber string) (*portal.Order, error) {
	var order portal.Order
	err := s.db.WithContext(ctx).
		Preload(preloadElasticScalingDetail).
		Preload(preloadDevices).
		Where(fieldOrderNumber, orderNumber).First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// UpdateOrderStatus 更新订单状态
func (s *orderServiceImpl) UpdateOrderStatus(ctx context.Context, id int64, status portal.OrderStatus, executor string, reason string) error {
	updates := map[string]interface{}{
		fieldStatus:    status,
		fieldExecutor:  executor,
		fieldUpdatedAt: time.Now(),
	}

	// 根据状态设置相应的时间字段
	switch status {
	case portal.OrderStatusProcessing:
		updates[fieldExecutionTime] = time.Now()
	case portal.OrderStatusCompleted:
		updates[fieldCompletionTime] = time.Now()
	case portal.OrderStatusReturning:
		// 归还中状态，设置执行时间（如果还没有设置的话）
		if updates[fieldExecutionTime] == nil {
			updates[fieldExecutionTime] = time.Now()
		}
	case portal.OrderStatusReturnCompleted:
		updates[fieldCompletionTime] = time.Now()
	case portal.OrderStatusNoReturn:
		updates[fieldCompletionTime] = time.Now()
	case portal.OrderStatusFailed:
		updates[fieldCompletionTime] = time.Now()
		if reason != "" {
			updates[fieldFailureReason] = reason
		}
	case portal.OrderStatusCancelled:
		updates[fieldCompletionTime] = time.Now()
		if reason != "" {
			updates[fieldFailureReason] = reason
		}
	case portal.OrderStatusIgnored:
		updates[fieldCompletionTime] = time.Now()
		if reason != "" {
			updates[fieldFailureReason] = reason
		}
		// portal.OrderStatusPending 状态不需要特殊处理，只更新基本字段
	}

	return s.db.WithContext(ctx).Model(&portal.Order{}).Where(fieldID, id).Updates(updates).Error
}

// ListOrders 获取订单列表
func (s *orderServiceImpl) ListOrders(ctx context.Context, query OrderQuery) ([]portal.Order, int64, error) {
	var orders []portal.Order
	var total int64

	db := s.db.WithContext(ctx).Model(&portal.Order{})

	// 应用过滤条件
	if query.Type != "" {
		db = db.Where(fieldType, query.Type)
	}
	if query.Status != "" {
		db = db.Where(fieldStatusEq, query.Status)
	}
	if query.CreatedBy != "" {
		db = db.Where(fieldCreatedBy, query.CreatedBy)
	}
	if query.Name != "" {
		db = db.Where(fieldNameLike, "%"+query.Name+"%")
	}
	if query.StartTime != nil {
		db = db.Where(fieldCreatedAtGTE, query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where(fieldCreatedAtLTE, query.EndTime)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 预加载关联数据
	err := db.Preload(preloadElasticScalingDetail).
		Preload(preloadDevices).
		Order(OrderByCreatedAtDesc).
		Find(&orders).Error

	return orders, total, err
}

// DeleteOrder 删除订单（软删除）
func (s *orderServiceImpl) DeleteOrder(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&portal.Order{}, id).Error
}

// ProcessOrder 处理订单
func (s *orderServiceImpl) ProcessOrder(ctx context.Context, id int64, executor string) error {
	return s.UpdateOrderStatus(ctx, id, portal.OrderStatusProcessing, executor, "")
}

// CompleteOrder 完成订单
func (s *orderServiceImpl) CompleteOrder(ctx context.Context, id int64, executor string) error {
	return s.UpdateOrderStatus(ctx, id, portal.OrderStatusCompleted, executor, "")
}

// FailOrder 订单失败
func (s *orderServiceImpl) FailOrder(ctx context.Context, id int64, executor string, reason string) error {
	return s.UpdateOrderStatus(ctx, id, portal.OrderStatusFailed, executor, reason)
}

// CancelOrder 取消订单
func (s *orderServiceImpl) CancelOrder(ctx context.Context, id int64, executor string) error {
	return s.UpdateOrderStatus(ctx, id, portal.OrderStatusCancelled, executor, msgOrderCancelled)
}

// IgnoreOrder 忽略订单
func (s *orderServiceImpl) IgnoreOrder(ctx context.Context, id int64, executor string) error {
	return s.UpdateOrderStatus(ctx, id, portal.OrderStatusIgnored, executor, msgOrderIgnored)
}

// generateOrderNumber 生成唯一订单号
func (s *orderServiceImpl) generateOrderNumber() string {
	return fmt.Sprintf(orderNumberPrefix, time.Now().UnixNano()/1000000)
}
