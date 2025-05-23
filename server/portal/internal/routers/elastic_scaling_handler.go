package routers

import (
	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis" // Import redis package
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // Add zap import
	"gorm.io/gorm"
)

// ElasticScalingHandler 弹性伸缩处理器
type ElasticScalingHandler struct {
	service *service.ElasticScalingService
}

// NewElasticScalingHandler 创建弹性伸缩处理器
// NewElasticScalingHandler 创建弹性伸缩处理器
// 接受数据库连接和 Redis 处理器作为参数
func NewElasticScalingHandler(db *gorm.DB) *ElasticScalingHandler {
	redisHandler := redis.NewRedisHandler("default")
	// Create a logger for the service
	logger, _ := zap.NewProduction()
	// 创建设备缓存
	deviceCache := service.NewDeviceCache(redisHandler, redis.NewKeyBuilder("navy", "v1"))

	return &ElasticScalingHandler{
		service: service.NewElasticScalingService(db, redisHandler, logger, deviceCache), // Pass redisHandler, logger and cache
	}
}

// RegisterRoutes 注册路由
func (h *ElasticScalingHandler) RegisterRoutes(api *gin.RouterGroup) {
	elasticGroup := api.Group("/elastic-scaling")

	// 策略相关接口
	strategyGroup := elasticGroup.Group("/strategies")
	{
		strategyGroup.GET("", h.ListStrategies)
		strategyGroup.POST("", h.CreateStrategy)
		strategyGroup.GET("/:id", h.GetStrategy)
		strategyGroup.PUT("/:id", h.UpdateStrategy)
		strategyGroup.DELETE("/:id", h.DeleteStrategy)
		strategyGroup.PUT("/:id/status", h.UpdateStrategyStatus)
		strategyGroup.GET("/:id/execution-history", h.GetStrategyExecutionHistory)
	}

	// 订单相关接口
	orderGroup := elasticGroup.Group("/orders")
	{
		orderGroup.GET("", h.ListOrders)
		orderGroup.POST("", h.CreateOrder)
		orderGroup.GET("/:id", h.GetOrder)
		orderGroup.PUT("/:id/status", h.UpdateOrderStatus)
		orderGroup.GET("/:id/devices", h.GetOrderDevices)
		orderGroup.PUT("/:id/devices/:deviceId/status", h.UpdateOrderDeviceStatus)
	}

	// 统计接口
	statsGroup := elasticGroup.Group("/stats")
	{
		statsGroup.GET("/dashboard", h.GetDashboardStats)
		statsGroup.GET("/resource-trend", h.GetResourceAllocationTrend)
		statsGroup.GET("/orders", h.GetOrderStats)
	}
}

// ListStrategies 获取策略列表
// @Summary 获取策略列表
// @Description 获取弹性伸缩策略列表，支持分页和过滤
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param name query string false "策略名称（模糊搜索）"
// @Param status query string false "策略状态（enabled/disabled）"
// @Param action query string false "触发动作（pool_entry/pool_exit）"
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页大小，默认10"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies [get]
func (h *ElasticScalingHandler) ListStrategies(c *gin.Context) {
	name := c.Query("name")
	status := c.Query("status")
	action := c.Query("action")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	strategies, total, err := h.service.ListStrategies(name, status, action, page, pageSize)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{
		"list":  strategies,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// CreateStrategy 创建策略
// @Summary 创建策略
// @Description 创建新的弹性伸缩策略
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param strategy body service.StrategyDTO true "策略数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies [post]
func (h *ElasticScalingHandler) CreateStrategy(c *gin.Context) {
	var dto service.StrategyDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 设置创建者
	dto.CreatedBy = "admin" // 实际环境中应该从认证信息获取

	id, err := h.service.CreateStrategy(dto)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{"id": id})
}

// GetStrategy 获取策略详情
// @Summary 获取策略详情
// @Description 获取指定策略的详细信息
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id} [get]
func (h *ElasticScalingHandler) GetStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的策略ID")
		return
	}

	strategy, err := h.service.GetStrategy(id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, strategy)
}

// UpdateStrategy 更新策略
// @Summary 更新策略
// @Description 更新指定策略的信息
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param strategy body service.StrategyDTO true "策略数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id} [put]
func (h *ElasticScalingHandler) UpdateStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的策略ID")
		return
	}

	var dto service.StrategyDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateStrategy(id, dto); err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// DeleteStrategy 删除策略
// @Summary 删除策略
// @Description 删除指定的策略
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id} [delete]
func (h *ElasticScalingHandler) DeleteStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的策略ID")
		return
	}

	if err := h.service.DeleteStrategy(id); err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// UpdateStrategyStatus 更新策略状态
// @Summary 更新策略状态
// @Description 更新指定策略的启用/禁用状态
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param status body map[string]string true "状态数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id}/status [put]
func (h *ElasticScalingHandler) UpdateStrategyStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的策略ID")
		return
	}

	var reqBody struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	if reqBody.Status != "enabled" && reqBody.Status != "disabled" {
		render.BadRequest(c, "状态必须为 enabled 或 disabled")
		return
	}

	if err := h.service.UpdateStrategyStatus(id, reqBody.Status); err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// GetStrategyExecutionHistory 获取策略执行历史
// @Summary 获取策略执行历史
// @Description 获取指定策略的执行历史记录
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id}/execution-history [get]
func (h *ElasticScalingHandler) GetStrategyExecutionHistory(c *gin.Context) {
	// 简化实现：直接从策略详情中获取执行历史
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的策略ID")
		return
	}

	strategy, err := h.service.GetStrategy(id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, strategy.ExecutionHistory)
}

// ListOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取弹性伸缩订单列表，支持分页和过滤
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param clusterId query int false "集群ID"
// @Param strategyId query int false "策略ID"
// @Param actionType query string false "订单类型（pool_entry/pool_exit等）"
// @Param status query string false "订单状态"
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页大小，默认10"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders [get]
func (h *ElasticScalingHandler) ListOrders(c *gin.Context) {
	clusterID, _ := strconv.ParseInt(c.Query("clusterId"), 10, 64)
	strategyID, _ := strconv.ParseInt(c.Query("strategyId"), 10, 64)
	actionType := c.Query("actionType")
	status := c.Query("status")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	orders, total, err := h.service.ListOrders(clusterID, strategyID, actionType, status, page, pageSize)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{
		"list":  orders,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// CreateOrder 创建订单
// @Summary 创建订单
// @Description 创建新的弹性伸缩订单
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param order body service.OrderDTO true "订单数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders [post]
func (h *ElasticScalingHandler) CreateOrder(c *gin.Context) {
	var dto service.OrderDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 设置创建者
	dto.CreatedBy = "admin" // 实际环境中应该从认证信息获取

	id, err := h.service.CreateOrder(dto)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{"id": id})
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 获取指定订单的详细信息
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders/{id} [get]
func (h *ElasticScalingHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

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
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param status body map[string]string true "状态数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders/{id}/status [put]
func (h *ElasticScalingHandler) UpdateOrderStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
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

	// 设置执行者为当前用户
	executor := "admin" // 实际环境中应该从认证信息获取

	if err := h.service.UpdateOrderStatus(id, reqBody.Status, executor, reqBody.Reason); err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// GetOrderDevices 获取订单关联的设备
// @Summary 获取订单关联的设备
// @Description 获取指定订单关联的设备列表
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders/{id}/devices [get]
func (h *ElasticScalingHandler) GetOrderDevices(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	devices, err := h.service.GetOrderDevices(id)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, devices)
}

// UpdateOrderDeviceStatus 更新订单中设备的状态
// @Summary 更新订单中设备的状态
// @Description 更新指定订单中特定设备的状态
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Param deviceId path int true "设备ID"
// @Param status body map[string]string true "状态数据"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/orders/{id}/devices/{deviceId}/status [put]
func (h *ElasticScalingHandler) UpdateOrderDeviceStatus(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	deviceID, err := strconv.ParseInt(c.Param("deviceId"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的设备ID")
		return
	}

	var reqBody struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateOrderDeviceStatus(orderID, deviceID, reqBody.Status); err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, nil)
}

// GetDashboardStats 获取工作台统计数据
// @Summary 获取工作台统计数据
// @Description 获取工作台概览统计数据
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/stats/dashboard [get]
func (h *ElasticScalingHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.service.GetDashboardStats()
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, stats)
}

// GetResourceAllocationTrend 获取资源分配趋势
// @Summary 获取资源分配趋势
// @Description 获取指定集群的资源分配趋势数据
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param timeRange query string false "时间范围（24h/7d/30d）"
// @Param resourceTypes query string false "资源类型，多个类型用逗号分隔（如：total,compute,memory）"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/stats/resource-trend [get]
func (h *ElasticScalingHandler) GetResourceAllocationTrend(c *gin.Context) {
	clusterID, err := strconv.ParseInt(c.Query("clusterId"), 10, 64)
	if err != nil || clusterID <= 0 {
		render.BadRequest(c, "无效的集群ID")
		return
	}

	timeRange := c.DefaultQuery("timeRange", "24h")
	resourceTypes := c.DefaultQuery("resourceTypes", "total")

	trend, err := h.service.GetResourceAllocationTrend(clusterID, timeRange, resourceTypes)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, trend)
}

// GetOrderStats 获取订单统计
// @Summary 获取订单统计
// @Description 获取不同时间范围的订单统计数据
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param timeRange query string false "时间范围（7d/30d/90d）"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/stats/orders [get]
func (h *ElasticScalingHandler) GetOrderStats(c *gin.Context) {
	timeRange := c.DefaultQuery("timeRange", "30d")

	stats, err := h.service.GetOrderStats(timeRange)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, stats)
}
