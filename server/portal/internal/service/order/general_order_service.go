package order

import (
	"context"
	"errors"
	"navy-ng/models/portal"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// 错误信息
	errOrderTypeNotGeneral = "order type is not general"
	errQueryContextInvalid = "query context must be a *gin.Context"

	// 数据库查询字段
	fieldOrderID = "order_id = ?"
)

// --- DTOs for General Order ---

// GeneralOrderDTO 是包含通用订单详情的“富”订单对象
type GeneralOrderDTO struct {
	*portal.Order
	Details *portal.GeneralOrderDetail `json:"details,omitempty"`
}

// GetBaseOrder 实现了 RichOrder 接口
func (dto *GeneralOrderDTO) GetBaseOrder() *portal.Order {
	return dto.Order
}

// GeneralOrderCreateDTO 定义了创建通用订单时需要的数据
type GeneralOrderCreateDTO struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy" binding:"required"`
	// Add other fields specific to creating a general order
	Summary string `json:"summary"`
}

// GeneralOrderQueryDTO 定义了查询通用订单时支持的参数
type GeneralOrderQueryDTO struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
	Status    string `form:"status"`
	CreatedBy string `form:"createdBy"`
	Name      string `form:"name"`      // 订单名称，支持模糊查询
}

// --- Service Implementation for General Order ---

// generalOrderServiceImpl 是 UnifiedOrderService 的具体实现
type generalOrderServiceImpl struct {
	db          *gorm.DB
	baseService OrderService // Re-use the base order service for common operations
}

// NewGeneralOrderService 创建通用订单服务的实例
func NewGeneralOrderService(db *gorm.DB) UnifiedOrderService[*GeneralOrderDTO, GeneralOrderCreateDTO] {
	return &generalOrderServiceImpl{
		db:          db,
		baseService: NewOrderService(db), // Initialize the base service
	}
}

// CreateOrder 创建一个新的通用订单
func (s *generalOrderServiceImpl) CreateOrder(ctx context.Context, createDTO GeneralOrderCreateDTO) (*GeneralOrderDTO, error) {
	// 1. Create the base order
	baseOrder := &portal.Order{
		Name:        createDTO.Name,
		Description: createDTO.Description,
		CreatedBy:   createDTO.CreatedBy,
		Type:        portal.OrderTypeGeneral,
		Status:      portal.OrderStatusPending,
	}

	err := s.baseService.CreateOrder(ctx, baseOrder)
	if err != nil {
		return nil, err
	}

	// 2. Create the specific details
	details := &portal.GeneralOrderDetail{
		OrderID: baseOrder.ID,
		Summary: createDTO.Summary,
	}
	if err := s.db.WithContext(ctx).Create(details).Error; err != nil {
		// Rollback or handle error, for now just return it
		return nil, err
	}

	return s.GetOrder(ctx, baseOrder.ID)
}

// GetOrder 获取单个通用订单的完整信息
func (s *generalOrderServiceImpl) GetOrder(ctx context.Context, id int) (*GeneralOrderDTO, error) {
	baseOrder, err := s.baseService.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if baseOrder.Type != portal.OrderTypeGeneral {
		return nil, errors.New(errOrderTypeNotGeneral)
	}

	var details portal.GeneralOrderDetail
	if err := s.db.WithContext(ctx).Where(fieldOrderID, id).First(&details).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Details might be optional, return base order
			return &GeneralOrderDTO{Order: baseOrder, Details: nil}, nil
		}
		return nil, err
	}

	return &GeneralOrderDTO{Order: baseOrder, Details: &details}, nil
}

// ListOrders 获取通用订单列表
func (s *generalOrderServiceImpl) ListOrders(ctx context.Context, queryContext any) ([]*GeneralOrderDTO, int, error) {
	c, ok := queryContext.(*gin.Context)
	if !ok {
		return nil, 0, errors.New(errQueryContextInvalid)
	}

	var query GeneralOrderQueryDTO
	if err := c.ShouldBindQuery(&query); err != nil {
		return nil, 0, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	// 使用基础服务的 ListOrders，但只查询 'general' 类型
	baseQuery := OrderQuery{
		Type:      portal.OrderTypeGeneral,
		Status:    portal.OrderStatus(query.Status),
		CreatedBy: query.CreatedBy,
		Name:      query.Name,
		Page:      query.Page,
		PageSize:  query.PageSize,
	}
	baseOrders, total, err := s.baseService.ListOrders(ctx, baseQuery)
	if err != nil {
		return nil, 0, err
	}

	if len(baseOrders) == 0 {
		return []*GeneralOrderDTO{}, total, nil
	}

	// 批量获取订单详情
	orderIDs := make([]int, len(baseOrders))
	for i, order := range baseOrders {
		orderIDs[i] = order.ID
	}

	var details []portal.GeneralOrderDetail
	if err := s.db.WithContext(ctx).Where("order_id IN (?)", orderIDs).Find(&details).Error; err != nil {
		return nil, 0, err
	}

	detailsMap := make(map[int]*portal.GeneralOrderDetail)
	for i := range details {
		detailsMap[details[i].OrderID] = &details[i]
	}

	// 组装富 DTO 列表
	richOrders := make([]*GeneralOrderDTO, len(baseOrders))
	for i, order := range baseOrders {
		// GORM Find 返回的是值，需要取地址
		o := order
		richOrders[i] = &GeneralOrderDTO{
			Order:   &o,
			Details: detailsMap[order.ID], // 如果没有详情，这里会是 nil
		}
	}

	return richOrders, total, nil
}

// UpdateOrderStatus 更新订单状态
func (s *generalOrderServiceImpl) UpdateOrderStatus(ctx context.Context, id int, status string, executor string, reason string) error {
	// We can add validation here to ensure the order is of type 'general' before updating.
	// For now, we directly use the base service.
	return s.baseService.UpdateOrderStatus(ctx, id, portal.OrderStatus(status), executor, reason)
}

// --- 订单生命周期快捷操作 ---

// ProcessOrder 将订单状态更新为处理中
func (s *generalOrderServiceImpl) ProcessOrder(ctx context.Context, id int, executor string) error {
	return s.baseService.ProcessOrder(ctx, id, executor)
}

// CompleteOrder 将订单状态更新为已完成
func (s *generalOrderServiceImpl) CompleteOrder(ctx context.Context, id int, executor string) error {
	return s.baseService.CompleteOrder(ctx, id, executor)
}

// FailOrder 将订单状态更新为失败
func (s *generalOrderServiceImpl) FailOrder(ctx context.Context, id int, executor string, reason string) error {
	return s.baseService.FailOrder(ctx, id, executor, reason)
}

// CancelOrder 将订单状态更新为已取消
func (s *generalOrderServiceImpl) CancelOrder(ctx context.Context, id int, executor string) error {
	return s.baseService.CancelOrder(ctx, id, executor)
}