package routers

import (
	"net/http"
	"strconv"

	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Constants moved to constants.go

// ElasticScalingOrderHandler 处理弹性伸缩订单的 Handler
type ElasticScalingOrderHandler struct {
	service *service.ElasticScalingService
}

// NewElasticScalingOrderHandler 创建弹性伸缩订单处理器实例
func NewElasticScalingOrderHandler(db *gorm.DB) *ElasticScalingOrderHandler {
	redisHandler := redis.NewRedisHandler(RedisDefault)
	logger, _ := zap.NewProduction()
	keyBuilder := redis.NewKeyBuilder(RedisNamespace, RedisVersion)
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	return &ElasticScalingOrderHandler{
		service: service.NewElasticScalingService(db, redisHandler, logger, deviceCache),
	}
}

// RegisterRoutes 注册弹性伸缩订单路由
func (h *ElasticScalingOrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/elastic-scaling/orders")
	{
		orderGroup.POST("", h.CreateOrder)
		orderGroup.GET("", h.ListOrders)
		orderGroup.GET("/:id", h.GetOrder)
		orderGroup.PUT("/:id/status", h.UpdateOrderStatus)
		orderGroup.GET("/:id/devices", h.GetOrderDevices)
		orderGroup.PUT("/:id/devices/:device_id/status", h.UpdateOrderDeviceStatus)
	}
}

// ElasticScalingOrderQuery 定义了弹性伸缩订单列表的查询参数
type ElasticScalingOrderQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	ClusterID  int64  `form:"clusterId"`
	StrategyID int64  `form:"strategyId"`
	ActionType string `form:"actionType"`
	Status     string `form:"status"`
	Name       string `form:"name"`
}

// ListOrders 获取弹性伸缩订单列表
// @Summary 获取弹性伸缩订单列表
// @Description 获取弹性伸缩订单列表，支持分页和过滤
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页大小，默认10"
// @Param clusterId query int false "集群ID"
// @Param strategyId query int false "策略ID"
// @Param actionType query string false "动作类型"
// @Param status query string false "订单状态"
// @Param name query string false "订单名称（支持模糊搜索）"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders [get]
func (h *ElasticScalingOrderHandler) ListOrders(c *gin.Context) {
	var query ElasticScalingOrderQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, "参数解析失败: "+err.Error())
		return
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	orders, total, err := h.service.ListOrders(query.ClusterID, query.StrategyID, query.ActionType, query.Status, query.Name, query.Page, query.PageSize)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, gin.H{"list": orders, "total": total})
}

// CreateOrder 创建弹性伸缩订单
// @Summary 创建弹性伸缩订单
// @Description 创建新的弹性伸缩订单
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param order body service.OrderDTO true "订单数据"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders [post]
func (h *ElasticScalingOrderHandler) CreateOrder(c *gin.Context) {
	var dto service.OrderDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, "参数解析失败: "+err.Error())
		return
	}
	orderID, err := h.service.CreateOrder(dto)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, gin.H{"id": orderID})
}

// GetOrder 获取弹性伸缩订单详情
// @Summary 获取弹性伸缩订单详情
// @Description 获取指定订单的详细信息
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders/{id} [get]
func (h *ElasticScalingOrderHandler) GetOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	order, err := h.service.GetOrder(id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, order)
}

// UpdateOrderStatus 更新订单状态
// @Summary 更新订单状态
// @Description 更新指定订单的状态
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param request body object{status=string,reason=string} true "状态更新请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders/{id}/status [put]
func (h *ElasticScalingOrderHandler) UpdateOrderStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var reqBody struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}
	executor := "admin"
	err := h.service.UpdateOrderStatus(id, reqBody.Status, executor, reqBody.Reason)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, nil)
}

// GetOrderDevices 获取订单关联的设备
// @Summary 获取订单关联的设备
// @Description 获取指定订单关联的设备列表
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders/{id}/devices [get]
func (h *ElasticScalingOrderHandler) GetOrderDevices(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	devices, err := h.service.GetOrderDevices(id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, devices)
}

// UpdateOrderDeviceStatus 更新订单关联设备的状态
// @Summary 更新订单关联设备的状态
// @Description 更新指定订单中指定设备的状态
// @Tags 弹性伸缩订单
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param device_id path int true "设备ID"
// @Param request body object{status=string} true "设备状态更新请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/elastic-scaling/orders/{id}/devices/{device_id}/status [put]
func (h *ElasticScalingOrderHandler) UpdateOrderDeviceStatus(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	deviceID, _ := strconv.ParseInt(c.Param("device_id"), 10, 64)
	var reqBody struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}
	err := h.service.UpdateOrderDeviceStatus(orderID, deviceID, reqBody.Status)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(c, nil)
}
