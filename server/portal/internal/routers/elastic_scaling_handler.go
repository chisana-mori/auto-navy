package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis" // Import redis package
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // Add zap import
	"gorm.io/gorm"
)

// Constants moved to constants.go

// ElasticScalingHandler 弹性伸缩处理器
type ElasticScalingHandler struct {
	service *service.ElasticScalingService
}

// NewElasticScalingHandler 创建弹性伸缩处理器
// NewElasticScalingHandler 创建弹性伸缩处理器
// 接受数据库连接和 Redis 处理器作为参数
func NewElasticScalingHandler(db *gorm.DB) *ElasticScalingHandler {
	redisHandler := redis.NewRedisHandler(RedisDefault)
	// Create a logger for the service
	logger, _ := zap.NewProduction()
	// 创建设备缓存
	deviceCache := service.NewDeviceCache(redisHandler, redis.NewKeyBuilder(RedisNamespace, RedisVersion))

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

	// 统计接口
	statsGroup := elasticGroup.Group("/stats")
	{
		statsGroup.GET("/dashboard", h.GetDashboardStats)
		statsGroup.GET("/resource-trend", h.GetResourceAllocationTrend)
		statsGroup.GET("/orders", h.GetOrderStats)
		statsGroup.GET("/resource-pool-types", h.GetResourcePoolTypes) // 新增获取资源池类型接口
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
type StrategyListQuery struct {
	Name     string `form:"name"`
	Status   string `form:"status"`
	Action   string `form:"action"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

func (h *ElasticScalingHandler) ListStrategies(c *gin.Context) {
	var query StrategyListQuery
	// 设置默认值
	query.Page = 1
	query.PageSize = 10

	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidParams, err.Error()))
		return
	}

	// 验证分页参数
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	strategies, total, err := h.service.ListStrategies(query.Name, query.Status, query.Action, query.Page, query.PageSize)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, gin.H{
		"list":  strategies,
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
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
	dto.CreatedBy = DefaultExecutor // 实际环境中应该从认证信息获取

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
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidStrategyID)
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
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidStrategyID)
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
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidStrategyID)
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
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidStrategyID)
		return
	}

	var reqBody struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	if reqBody.Status != StatusEnabled && reqBody.Status != StatusDisabled {
		render.BadRequest(c, MsgInvalidStatus)
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
// @Description 获取指定策略的执行历史记录，支持分页和集群名字模糊查询
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param clusterName query string false "集群名字（模糊查询）"
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/strategies/{id}/execution-history [get]
func (h *ElasticScalingHandler) GetStrategyExecutionHistory(c *gin.Context) {
	// 解析策略ID
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidStrategyID)
		return
	}

	// 解析分页参数
	var pagination service.PaginationRequest
	if err = c.ShouldBindQuery(&pagination); err != nil {
		render.BadRequest(c, "分页参数错误")
		return
	}
	pagination.AdjustPagination()

	// 获取集群名字查询参数
	clusterName := c.Query("clusterName")

	// 获取策略执行历史
	histories, total, err := h.service.GetStrategyExecutionHistoryWithPagination(id, &pagination, clusterName)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, service.ToPaginationResponseWithData(&pagination, total, histories))
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
type ResourceTrendQuery struct {
	ClusterID     int64  `form:"clusterId" binding:"required,min=1"`
	TimeRange     string `form:"timeRange"`
	ResourceTypes string `form:"resourceTypes"`
}

func (h *ElasticScalingHandler) GetResourceAllocationTrend(c *gin.Context) {
	var query ResourceTrendQuery
	// 设置默认值
	query.TimeRange = "24h"
	query.ResourceTypes = "total"

	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidParams, err.Error()))
		return
	}

	trend, err := h.service.GetResourceAllocationTrend(query.ClusterID, query.TimeRange, query.ResourceTypes)
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
	timeRange := c.DefaultQuery(ParamTimeRange, "30d")

	stats, err := h.service.GetOrderStats(timeRange)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, stats)
}

// GetResourcePoolTypes 获取资源池类型列表
// @Summary 获取资源池类型列表
// @Description 获取当天所有资源池类型
// @Tags 弹性伸缩
// @Accept json
// @Produce json
// @Success 200 {object} render.Response
// @Router /fe-v1/elastic-scaling/stats/resource-pool-types [get]
func (h *ElasticScalingHandler) GetResourcePoolTypes(c *gin.Context) {
	types, err := h.service.GetResourcePoolTypes()
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(c, types)
}
