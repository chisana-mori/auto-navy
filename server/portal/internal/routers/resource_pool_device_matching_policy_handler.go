package routers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/service"
)

// ResourcePoolDeviceMatchingPolicyHandler 资源池设备匹配策略处理器
type ResourcePoolDeviceMatchingPolicyHandler struct {
	policyService *service.ResourcePoolDeviceMatchingPolicyService
}

// NewResourcePoolDeviceMatchingPolicyHandler 创建资源池设备匹配策略处理器
func NewResourcePoolDeviceMatchingPolicyHandler(db *gorm.DB) *ResourcePoolDeviceMatchingPolicyHandler {
	// 创建 Redis 客户端和键构建器
	redisHandler := redis.NewRedisHandler(RedisDefault)
	keyBuilder := redis.NewKeyBuilder(RedisNamespace, service.CacheVersion)

	// 创建设备缓存服务
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	// 创建资源池设备匹配策略服务
	policyService := service.NewResourcePoolDeviceMatchingPolicyService(db, deviceCache)

	return &ResourcePoolDeviceMatchingPolicyHandler{
		policyService: policyService,
	}
}

// RegisterRoutes 注册路由
func (h *ResourcePoolDeviceMatchingPolicyHandler) RegisterRoutes(router *gin.RouterGroup) {
	resourcePoolGroup := router.Group(RouteGroupResourcePool)
	{
		// 匹配策略相关接口
		matchingPoliciesGroup := resourcePoolGroup.Group(SubRouteMatchingPolicies)
		{
			matchingPoliciesGroup.GET("", h.GetResourcePoolDeviceMatchingPolicies)
			matchingPoliciesGroup.POST("", h.CreateResourcePoolDeviceMatchingPolicy)
			matchingPoliciesGroup.GET(RouteParamID, h.GetResourcePoolDeviceMatchingPolicy)
			matchingPoliciesGroup.PUT(RouteParamID, h.UpdateResourcePoolDeviceMatchingPolicy)
			matchingPoliciesGroup.DELETE(RouteParamID, h.DeleteResourcePoolDeviceMatchingPolicy)
			matchingPoliciesGroup.PUT(RouteParamIDStatus, h.UpdateResourcePoolDeviceMatchingPolicyStatus)
			matchingPoliciesGroup.GET(SubRouteByType, h.GetResourcePoolDeviceMatchingPoliciesByType)
		}
	}
}

// GetResourcePoolDeviceMatchingPolicies 获取资源池设备匹配策略列表
// @Summary 获取资源池设备匹配策略列表
// @Description 获取资源池设备匹配策略列表
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页数量，默认10"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies [get]
func (h *ResourcePoolDeviceMatchingPolicyHandler) GetResourcePoolDeviceMatchingPolicies(c *gin.Context) {
	// 获取分页参数
	page, err := strconv.Atoi(c.DefaultQuery(ParamPage, DefaultPageValue))
	if err != nil || page <= 0 {
		page = DefaultPageInt
	}

	size, err := strconv.Atoi(c.DefaultQuery(ParamSize, DefaultSizeValue))
	if err != nil || size <= 0 || size > MaxSizeInt {
		size = DefaultSizeInt
	}

	// 创建一个带有超时的上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// 调用服务获取策略列表
	response, err := h.policyService.GetResourcePoolDeviceMatchingPolicies(ctx, page, size)
	if err != nil {
		// 记录详细错误信息
		fmt.Printf("Error getting matching policies: %v\n", err)

		// 检查是否是超时错误
		if ctx.Err() == context.DeadlineExceeded {
			render.InternalServerError(c, MsgRequestTimeout)
			return
		}

		render.InternalServerError(c, MsgFailedToGetPolicies+err.Error())
		return
	}

	render.Success(c, response)
}

// GetResourcePoolDeviceMatchingPolicy 获取资源池设备匹配策略详情
// @Summary 获取资源池设备匹配策略详情
// @Description 获取资源池设备匹配策略详情
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 404 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies/{id} [get]
func (h *ResourcePoolDeviceMatchingPolicyHandler) GetResourcePoolDeviceMatchingPolicy(c *gin.Context) {
	// 获取策略ID
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidPolicyID)
		return
	}

	// 调用服务获取策略详情
	policy, err := h.policyService.GetResourcePoolDeviceMatchingPolicy(c.Request.Context(), id)
	if err != nil {
		render.NotFound(c, err.Error())
		return
	}

	render.Success(c, policy)
}

// CreateResourcePoolDeviceMatchingPolicy 创建资源池设备匹配策略
// @Summary 创建资源池设备匹配策略
// @Description 创建资源池设备匹配策略
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param request body service.CreateResourcePoolDeviceMatchingPolicyRequest true "创建策略请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies [post]
func (h *ResourcePoolDeviceMatchingPolicyHandler) CreateResourcePoolDeviceMatchingPolicy(c *gin.Context) {
	// 解析请求
	var req service.CreateResourcePoolDeviceMatchingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(c)

	// 转换为服务层模型
	policy := req.ToServiceModel(username)

	// 调用服务创建策略
	if err := h.policyService.CreateResourcePoolDeviceMatchingPolicy(c.Request.Context(), policy); err != nil {
		render.InternalServerError(c, err.Error())
		return
	}

	render.Success(c, policy)
}

// UpdateResourcePoolDeviceMatchingPolicy 更新资源池设备匹配策略
// @Summary 更新资源池设备匹配策略
// @Description 更新资源池设备匹配策略
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param request body service.UpdateResourcePoolDeviceMatchingPolicyRequest true "更新策略请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 404 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies/{id} [put]
func (h *ResourcePoolDeviceMatchingPolicyHandler) UpdateResourcePoolDeviceMatchingPolicy(c *gin.Context) {
	// 获取策略ID
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidPolicyID)
		return
	}

	// 解析请求
	var req service.UpdateResourcePoolDeviceMatchingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(c)

	// 转换为服务层模型
	policy := req.ToServiceModel(id, username)

	// 调用服务更新策略
	if err := h.policyService.UpdateResourcePoolDeviceMatchingPolicy(c.Request.Context(), policy); err != nil {
		render.InternalServerError(c, err.Error())
		return
	}

	render.SuccessWithMessage(c, MsgPolicyUpdatedSuccess, nil)
}

// DeleteResourcePoolDeviceMatchingPolicy 删除资源池设备匹配策略
// @Summary 删除资源池设备匹配策略
// @Description 删除资源池设备匹配策略
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 404 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies/{id} [delete]
func (h *ResourcePoolDeviceMatchingPolicyHandler) DeleteResourcePoolDeviceMatchingPolicy(c *gin.Context) {
	// 获取策略ID
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidPolicyID)
		return
	}

	// 调用服务删除策略
	if err := h.policyService.DeleteResourcePoolDeviceMatchingPolicy(c.Request.Context(), id); err != nil {
		render.InternalServerError(c, err.Error())
		return
	}

	render.SuccessWithMessage(c, MsgPolicyDeletedSuccess, nil)
}

// UpdateResourcePoolDeviceMatchingPolicyStatus 更新资源池设备匹配策略状态
// @Summary 更新资源池设备匹配策略状态
// @Description 更新资源池设备匹配策略状态
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param id path int true "策略ID"
// @Param request body service.UpdateResourcePoolDeviceMatchingPolicyStatusRequest true "更新状态请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 404 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies/{id}/status [put]
func (h *ResourcePoolDeviceMatchingPolicyHandler) UpdateResourcePoolDeviceMatchingPolicyStatus(c *gin.Context) {
	// 获取策略ID
	id, err := strconv.ParseInt(c.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidPolicyID)
		return
	}

	// 解析请求
	var req service.UpdateResourcePoolDeviceMatchingPolicyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		render.BadRequest(c, err.Error())
		return
	}

	// 调用服务更新策略状态
	if err := h.policyService.UpdateResourcePoolDeviceMatchingPolicyStatus(c.Request.Context(), id, req.Status); err != nil {
		render.InternalServerError(c, err.Error())
		return
	}

	render.SuccessWithMessage(c, MsgPolicyStatusUpdatedSuccess, nil)
}

// GetResourcePoolDeviceMatchingPoliciesByType 根据资源池类型和动作类型获取匹配策略
// @Summary 根据资源池类型和动作类型获取匹配策略
// @Description 根据资源池类型和动作类型获取匹配策略
// @Tags 资源池设备匹配策略
// @Accept json
// @Produce json
// @Param resourcePoolType query string true "资源池类型"
// @Param actionType query string true "动作类型：pool_entry 或 pool_exit"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /resource-pool/matching-policies/by-type [get]
type PolicyTypeQuery struct {
	ResourcePoolType string `form:"resourcePoolType" binding:"required"`
	ActionType       string `form:"actionType" binding:"required"`
}

func (h *ResourcePoolDeviceMatchingPolicyHandler) GetResourcePoolDeviceMatchingPoliciesByType(c *gin.Context) {
	var query PolicyTypeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	// 调用服务获取策略
	policies, err := h.policyService.GetResourcePoolDeviceMatchingPoliciesByType(c.Request.Context(), query.ResourcePoolType, query.ActionType)
	if err != nil {
		render.InternalServerError(c, err.Error())
		return
	}

	render.Success(c, policies)
}
