package order

import (
	"fmt"
	"net/http"

	"navy-ng/models/portal"
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/routers"
	. "navy-ng/server/portal/internal/routers"
	"navy-ng/server/portal/internal/service/order"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Constants moved to constants.go

// UnifiedOrderHandler 处理所有新类型订单的通用 Handler
type UnifiedOrderHandler struct {
	db *gorm.DB // db instance, may be needed for some operations
}

// NewUnifiedOrderHandler 创建通用订单处理器实例
func NewUnifiedOrderHandler(db *gorm.DB) *UnifiedOrderHandler {
	return &UnifiedOrderHandler{
		db: db,
	}
}

// InitServices 初始化并注册所有订单服务
func (h *UnifiedOrderHandler) InitServices(logger *zap.Logger) {
	// 注册 GeneralOrderService
	generalOrderService := order.NewGeneralOrderService(h.db)
	order.RegisterOrderService("general", generalOrderService)

	// 注册维护订单服务
	maintenanceService := order.NewMaintenanceOrderService(h.db, logger)
	order.RegisterOrderService(string(portal.OrderTypeMaintenance), maintenanceService)

	// 未来新的订单服务可以在这里继续注册
	// e.g. elasticScalingService := service.NewElasticScalingService(db, redisHandler, logger, deviceCache)
	//      service.RegisterOrderService(string(portal.OrderTypeElasticScaling), elasticScalingService)
}

// RegisterRoutes 注册通用订单路由
// 路由设计为 /orders/{orderType}/... 的格式
func (h *UnifiedOrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/orders/:" + routers.ParamOrderType)
	{
		orderGroup.POST("", h.CreateOrder)
		orderGroup.GET("", h.ListOrders)
		orderGroup.GET("/:"+ParamID, h.GetOrder)
		orderGroup.PUT("/:"+ParamID+"/status", h.UpdateOrderStatus)
		// 快捷操作路由
		orderGroup.POST("/:"+ParamID+"/process", h.ProcessOrder)
		orderGroup.POST("/:"+ParamID+"/complete", h.CompleteOrder)
		orderGroup.POST("/:"+ParamID+"/fail", h.FailOrder)
		orderGroup.POST("/:"+ParamID+"/cancel", h.CancelOrder)
	}
}

// GetOrderRequest 定义了获取订单时从 URI 绑定的参数
type GetOrderRequest struct {
	OrderType string `uri:"orderType" binding:"required"`
	ID        int64  `uri:"id" binding:"required"`
}

// GetOrder 获取指定类型的订单详情
// @Summary 获取特定类型的订单详情
// @Description 根据订单类型和ID获取订单的详细信息
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型 (e.g., general)"
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response{data=order.GeneralOrderDTO} "成功时返回通用订单详情"
// @Failure 400 {object} render.ErrorResponse "请求参数错误"
// @Failure 404 {object} render.ErrorResponse "订单不存在"
// @Failure 500 {object} render.ErrorResponse "服务器内部错误"
// @Router /fe-v1/orders/{orderType}/{id} [get]
func (h *UnifiedOrderHandler) GetOrder(c *gin.Context) {
	var req GetOrderRequest
	if err := c.ShouldBindUri(&req); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgParamBindFailed, err.Error()))
		return
	}

	serviceInstance, found := order.GetOrderService(req.OrderType)
	if !found {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidOrderType, req.OrderType))
		return
	}

	switch req.OrderType {
	case OrderTypeGeneral:
		handleGetOrder[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO](c, serviceInstance, req.ID)
	default:
		render.BadRequest(c, fmt.Sprintf(MsgUnsupportedOrderType, req.OrderType))
	}
}

// handleGetOrder 是一个泛型辅助函数，用于处理获取订单的通用逻辑
func handleGetOrder[T order.RichOrder, C any](c *gin.Context, serviceInstance any, id int64) {
	// 类型断言，确保服务实例实现了正确的泛型接口
	s, ok := serviceInstance.(order.UnifiedOrderService[T, C])
	if !ok {
		render.Fail(c, http.StatusInternalServerError, MsgServiceTypeMismatch)
		return
	}

	// 调用具体的服务方法
	order, err := s.GetOrder(c.Request.Context(), int(id))
	if err != nil {
		// 可以在这里处理特定的错误类型，例如 gorm.ErrRecordNotFound
		render.Fail(c, http.StatusInternalServerError, fmt.Sprintf(MsgGetOrderFailed, err.Error()))
		return
	}

	render.Success(c, order)
}

// --- CreateOrder ---

// CreateOrderRequest 定义了创建订单时从 URI 绑定的参数
type CreateOrderRequest struct {
	OrderType string `uri:"orderType" binding:"required"`
}

// CreateOrder 创建一个新订单
// @Summary 创建一个新订单
// @Description 根据指定的订单类型创建一个新的订单
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型 (e.g., general)"
// @Param order body order.GeneralOrderCreateDTO true "创建通用订单所需的数据"
// @Success 201 {object} render.Response{data=order.GeneralOrderDTO} "成功时返回创建的订单详情"
// @Failure 400 {object} render.ErrorResponse "请求参数错误"
// @Failure 500 {object} render.ErrorResponse "服务器内部错误"
// @Router /fe-v1/orders/{orderType} [post]
func (h *UnifiedOrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindUri(&req); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgParamBindFailed, err.Error()))
		return
	}

	serviceInstance, found := order.GetOrderService(req.OrderType)
	if !found {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidOrderType, req.OrderType))
		return
	}

	switch req.OrderType {
	case OrderTypeGeneral:
		handleCreateOrder[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO](c, serviceInstance)
	default:
		render.BadRequest(c, fmt.Sprintf(MsgUnsupportedOrderType, req.OrderType))
	}
}

// handleCreateOrder 是一个泛型辅助函数，用于处理创建订单的通用逻辑
func handleCreateOrder[T order.RichOrder, C any](c *gin.Context, serviceInstance any) {
	s, ok := serviceInstance.(order.UnifiedOrderService[T, C])
	if !ok {
		render.Fail(c, http.StatusInternalServerError, MsgServiceTypeMismatch)
		return
	}

	var createDTO C
	if err := c.ShouldBindJSON(&createDTO); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgParamBindFailed, err.Error()))
		return
	}

	order, err := s.CreateOrder(c.Request.Context(), createDTO)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, fmt.Sprintf(MsgCreateOrderFailed, err.Error()))
		return
	}

	render.Success(c, order)
}

// --- ListOrders ---

// ListOrdersRequest 定义了列出订单时从 URI 绑定的参数
type ListOrdersRequest struct {
	OrderType string `uri:"orderType" binding:"required"`
}

// ListOrders 获取订单列表
// @Summary 获取指定类型的订单列表
// @Description 根据订单类型和查询参数获取订单列表
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型 (e.g., general)"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param status query string false "订单状态"
// @Param createdBy query string false "创建者"
// @Param name query string false "订单名称，支持模糊查询"
// @Success 200 {object} render.Response{data=order.GeneralOrderQueryDTO} "成功时返回订单列表和总数"
// @Failure 400 {object} render.ErrorResponse "请求参数错误"
// @Failure 500 {object} render.ErrorResponse "服务器内部错误"
// @Router /fe-v1/orders/{orderType} [get]
func (h *UnifiedOrderHandler) ListOrders(c *gin.Context) {
	var req ListOrdersRequest
	if err := c.ShouldBindUri(&req); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgParamBindFailed, err.Error()))
		return
	}

	serviceInstance, found := order.GetOrderService(req.OrderType)
	if !found {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidOrderType, req.OrderType))
		return
	}

	switch req.OrderType {
	case OrderTypeGeneral:
		handleListOrders[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO](c, serviceInstance)
	default:
		render.BadRequest(c, fmt.Sprintf(MsgUnsupportedOrderType, req.OrderType))
	}
}

// handleListOrders 是一个泛型辅助函数，用于处理获取订单列表的通用逻辑
func handleListOrders[T order.RichOrder, C any](c *gin.Context, serviceInstance any) {
	s, ok := serviceInstance.(order.UnifiedOrderService[T, C])
	if !ok {
		render.Fail(c, http.StatusInternalServerError, MsgServiceTypeMismatch)
		return
	}

	// 服务层将从 'c' 中解析出自己需要的查询参数
	orders, total, err := s.ListOrders(c.Request.Context(), c)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, fmt.Sprintf(MsgListOrdersFailed, err.Error()))
		return
	}

	render.Success(c, gin.H{"list": orders, "total": total})
}

// --- UpdateOrderStatus ---

// UpdateOrderStatusRequest 定义了更新订单状态的请求体
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason"`
}

// UpdateOrderStatusURIRequest 定义了更新状态时从 URI 绑定的参数
type UpdateOrderStatusURIRequest struct {
	OrderType string `uri:"orderType" binding:"required"`
	ID        int64  `uri:"id" binding:"required"`
}

// UpdateOrderStatus 更新订单状态
// @Summary 更新订单状态
// @Description 更新指定订单的状态
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型 (e.g., general)"
// @Param id path int true "订单ID"
// @Param statusUpdate body UpdateOrderStatusRequest true "状态更新请求"
// @Success 200 {object} render.Response "成功"
// @Failure 400 {object} render.ErrorResponse "请求参数错误"
// @Failure 500 {object} render.ErrorResponse "服务器内部错误"
// @Router /fe-v1/orders/{orderType}/{id}/status [put]
func (h *UnifiedOrderHandler) UpdateOrderStatus(c *gin.Context) {
	var uriReq UpdateOrderStatusURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgURIBindFailed, err.Error()))
		return
	}

	var bodyReq UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&bodyReq); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgBodyBindFailed, err.Error()))
		return
	}

	serviceInstance, found := order.GetOrderService(uriReq.OrderType)
	if !found {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidOrderType, uriReq.OrderType))
		return
	}

	switch uriReq.OrderType {
	case OrderTypeGeneral:
		handleUpdateOrderStatus[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO](c, serviceInstance, uriReq.ID, bodyReq.Status, bodyReq.Reason)
	default:
		render.BadRequest(c, fmt.Sprintf(MsgUnsupportedOrderType, uriReq.OrderType))
	}
}

// handleUpdateOrderStatus 是一个泛型辅助函数，用于处理更新订单状态的通用逻辑
func handleUpdateOrderStatus[T order.RichOrder, C any](c *gin.Context, serviceInstance any, id int64, status, reason string) {
	s, ok := serviceInstance.(order.UnifiedOrderService[T, C])
	if !ok {
		render.Fail(c, http.StatusInternalServerError, MsgServiceTypeMismatch)
		return
	}
	// To-Do: Get executor from request context (e.g., JWT middleware)
	executor := DefaultUsername
	err := s.UpdateOrderStatus(c.Request.Context(), int(id), status, executor, reason)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, fmt.Sprintf(MsgUpdateStatusFailed, err.Error()))
		return
	}

	render.Success(c, nil)
}

// --- 快捷操作 Handlers ---

// ExecutorRequest 定义了需要执行者的请求体
type ExecutorRequest struct {
	Executor string `json:"executor" binding:"required"`
}

// FailRequest 定义了需要失败原因的请求体
type FailRequest struct {
	Executor string `json:"executor" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// ProcessOrder 处理订单
// @Summary 处理订单
// @Description 将订单状态设置为“处理中”
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型"
// @Param id path int true "订单ID"
// @Param executor body ExecutorRequest true "执行者信息"
// @Success 200 {object} render.Response "成功"
// @Router /fe-v1/orders/{orderType}/{id}/process [post]
func (h *UnifiedOrderHandler) ProcessOrder(c *gin.Context) {
	h.handleSimpleLifecycleAction(c, func(srv order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO], id int64, executor string) error {
		return srv.ProcessOrder(c.Request.Context(), int(id), executor)
	})
}

// CompleteOrder 完成订单
// @Summary 完成订单
// @Description 将订单状态设置为“已完成”
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型"
// @Param id path int true "订单ID"
// @Param executor body ExecutorRequest true "执行者信息"
// @Success 200 {object} render.Response "成功"
// @Router /fe-v1/orders/{orderType}/{id}/complete [post]
func (h *UnifiedOrderHandler) CompleteOrder(c *gin.Context) {
	h.handleSimpleLifecycleAction(c, func(srv order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO], id int64, executor string) error {
		return srv.CompleteOrder(c.Request.Context(), int(id), executor)
	})
}

// FailOrder 失败订单
// @Summary 失败订单
// @Description 将订单状态设置为“失败”
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型"
// @Param id path int true "订单ID"
// @Param failInfo body FailRequest true "失败信息"
// @Success 200 {object} render.Response "成功"
// @Router /fe-v1/orders/{orderType}/{id}/fail [post]
func (h *UnifiedOrderHandler) FailOrder(c *gin.Context) {
	var uriReq UpdateOrderStatusURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		render.BadRequest(c, err.Error())
		return
	}
	var bodyReq FailRequest
	if err := c.ShouldBindJSON(&bodyReq); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	serviceInstance, found := order.GetOrderService(uriReq.OrderType)
	if !found {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	switch uriReq.OrderType {
	case "general":
		s, _ := serviceInstance.(order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO])
		err := s.FailOrder(c.Request.Context(), int(uriReq.ID), bodyReq.Executor, bodyReq.Reason)
		if err != nil {
			render.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		render.Success(c, nil)
	default:
		render.BadRequest(c, "不支持的订单类型")
	}
}

// CancelOrder 取消订单
// @Summary 取消订单
// @Description 将订单状态设置为“已取消”
// @Tags 统一订单
// @Accept json
// @Produce json
// @Param orderType path string true "订单类型"
// @Param id path int true "订单ID"
// @Param executor body ExecutorRequest true "执行者信息"
// @Success 200 {object} render.Response "成功"
// @Router /fe-v1/orders/{orderType}/{id}/cancel [post]
func (h *UnifiedOrderHandler) CancelOrder(c *gin.Context) {
	h.handleSimpleLifecycleAction(c, func(srv order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO], id int64, executor string) error {
		return srv.CancelOrder(c.Request.Context(), int(id), executor)
	})
}

// handleSimpleLifecycleAction 是一个高阶函数，用于处理只需要 executor 的简单生命周期操作
func (h *UnifiedOrderHandler) handleSimpleLifecycleAction(c *gin.Context, actionFunc func(srv order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO], id int64, executor string) error) {
	var uriReq UpdateOrderStatusURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		render.BadRequest(c, err.Error())
		return
	}
	var bodyReq ExecutorRequest
	if err := c.ShouldBindJSON(&bodyReq); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	serviceInstance, found := order.GetOrderService(uriReq.OrderType)
	if !found {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	switch uriReq.OrderType {
	case "general":
		srv, ok := serviceInstance.(order.UnifiedOrderService[*order.GeneralOrderDTO, order.GeneralOrderCreateDTO])
		if !ok {
			render.Fail(c, http.StatusInternalServerError, "服务接口类型不匹配")
			return
		}
		err := actionFunc(srv, uriReq.ID, bodyReq.Executor)
		if err != nil {
			render.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		render.Success(c, nil)
	default:
		render.BadRequest(c, "不支持的订单类型")
	}
}
