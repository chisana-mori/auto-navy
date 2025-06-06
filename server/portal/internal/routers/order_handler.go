package routers

import (
	"net/http"
	"strconv"
	"time"

	"navy-ng/models/portal"
	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// OrderHandler 统一订单处理器
type OrderHandler struct {
	registry *service.OrderServiceRegistry
	db       *gorm.DB
}

// NewOrderHandler 创建订单处理器实例
func NewOrderHandler(db *gorm.DB) *OrderHandler {
	registry := service.NewOrderServiceRegistry()

	// 创建Redis处理器和logger
	redisHandler := redis.NewRedisHandler("default")
	logger, _ := zap.NewProduction()
	keyBuilder := redis.NewKeyBuilder("navy", "v1")
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	// 注册弹性伸缩订单服务
	elasticScalingService := service.NewElasticScalingService(
		db,
		redisHandler,
		logger,
		deviceCache,
	)
	elasticAdapter := service.NewElasticScalingOrderAdapter(elasticScalingService)
	registry.RegisterService(string(portal.OrderTypeElasticScaling), elasticAdapter)

	// 注册通用订单服务
	generalOrderService := service.NewOrderService(db)
	generalAdapter := service.NewGeneralOrderAdapter(generalOrderService)
	registry.RegisterService("general", generalAdapter)

	return &OrderHandler{
		registry: registry,
		db:       db,
	}
}

// RegisterOrderRoutes 注册通用订单路由
func RegisterOrderRoutes(router *gin.RouterGroup, handler *OrderHandler) {
	orderGroup := router.Group("/orders")
	{
		orderGroup.GET("", handler.ListOrders)
		orderGroup.POST("", handler.CreateOrder)
		orderGroup.GET("/:id", handler.GetOrder)
		orderGroup.PUT("/:id/status", handler.UpdateOrderStatus)
		orderGroup.DELETE("/:id", handler.DeleteOrder)
	}
}

// ListOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取通用订单列表，支持分页和过滤
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param type query string false "订单类型"
// @Param status query string false "订单状态"
// @Param createdBy query string false "创建人"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} render.Response
// @Router /fe-v1/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	// 解析查询参数
	query := service.UnifiedOrderQuery{
		Type:      string(c.Query("type")),
		Status:    string(c.Query("status")),
		CreatedBy: c.Query("createdBy"),
		Page:      1,
		PageSize:  10,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			query.Page = p
		}
	}

	if pageSize := c.Query("pageSize"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			query.PageSize = ps
		}
	}

	// 解析时间范围
	if startTime := c.Query("startTime"); startTime != "" {
		if t, err := time.Parse("2006-01-02", startTime); err == nil {
			query.StartTime = &t
		}
	}

	if endTime := c.Query("endTime"); endTime != "" {
		if t, err := time.Parse("2006-01-02", endTime); err == nil {
			query.EndTime = &t
		}
	}

	// 调用服务层
	orderService, exists := h.registry.GetService(string(query.Type))
	if !exists {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	ordersInterface, total, err := orderService.ListOrders(c.Request.Context(), query)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 类型断言转换为具体类型
	orders, ok := ordersInterface.([]*portal.Order)
	if !ok {
		render.Fail(c, http.StatusInternalServerError, "订单数据类型转换失败")
		return
	}

	// 转换为丰富的DTO
	result := service.ToRichOrderDTOList(orders)

	// 丰富DTO中的名称信息
	service.EnrichOrderDTOListWithNames(result, h.db)

	render.Success(c, gin.H{
		"list":  result,
		"total": total,
	})
}

// CreateOrder 创建订单
// @Summary 创建订单
// @Description 创建新的通用订单
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param order body portal.Order true "订单数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var order portal.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 设置创建者
	if order.CreatedBy == "" {
		order.CreatedBy = "admin" // 实际环境中应该从认证信息获取
	}

	// 获取对应的订单服务
	orderService, exists := h.registry.GetService(string(order.Type))
	if !exists {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	_, err := orderService.CreateOrder(c.Request.Context(), order)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{"id": order.ID})
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 根据ID获取订单详情
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	// 获取订单类型参数，如果没有提供则使用general
	orderType := c.Query("type")
	if orderType == "" {
		orderType = "general"
	}

	// 获取对应的订单服务
	orderService, exists := h.registry.GetService(orderType)
	if !exists {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	orderInterface, err := orderService.GetOrder(c.Request.Context(), id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 类型断言转换为具体类型
	order, ok := orderInterface.(*portal.Order)
	if !ok {
		render.Fail(c, http.StatusInternalServerError, "订单数据类型转换失败")
		return
	}

	// 转换为丰富的DTO
	dto := service.ToRichOrderDTO(order)
	if dto == nil {
		render.Fail(c, http.StatusInternalServerError, "订单数据转换失败")
		return
	}

	// 丰富DTO中的名称信息
	service.EnrichOrderDTOWithNames(dto, h.db)

	render.Success(c, *dto)
}

// UpdateOrderStatus 更新订单状态
// @Summary 更新订单状态
// @Description 更新订单状态
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param request body object true "状态更新请求"
// @Success 200 {object} render.Response
// @Router /fe-v1/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	var reqBody struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 设置执行者
	executor := "admin" // 实际环境中应该从认证信息获取

	// 获取订单类型参数，如果没有提供则使用general
	orderType := c.Query("type")
	if orderType == "" {
		orderType = "general"
	}

	// 获取对应的订单服务
	orderService, exists := h.registry.GetService(orderType)
	if !exists {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	err = orderService.UpdateOrderStatus(c.Request.Context(), id, reqBody.Status, executor, reqBody.Reason)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// DeleteOrder 删除订单
// @Summary 删除订单
// @Description 删除订单（软删除）
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/orders/{id} [delete]
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	// 获取订单类型参数，如果没有提供则使用general
	orderType := c.Query("type")
	if orderType == "" {
		orderType = "general"
	}

	// 获取对应的订单服务
	orderService, exists := h.registry.GetService(orderType)
	if !exists {
		render.BadRequest(c, "不支持的订单类型")
		return
	}

	err = orderService.DeleteOrder(c.Request.Context(), id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}
