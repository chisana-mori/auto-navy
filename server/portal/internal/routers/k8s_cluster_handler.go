package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
)

// K8sClusterHandler K8s集群处理器
type K8sClusterHandler struct {
	service *service.K8sClusterService
}

// NewK8sClusterHandler 创建K8s集群处理器
func NewK8sClusterHandler(db *gorm.DB) *K8sClusterHandler {
	// 创建服务
	k8sClusterService := service.NewK8sClusterService(db)

	return &K8sClusterHandler{
		service: k8sClusterService,
	}
}

// RegisterRoutes 注册路由
func (h *K8sClusterHandler) RegisterRoutes(router *gin.RouterGroup) {
	k8sClusterGroup := router.Group("/fe-v1/k8s-clusters")
	{
		k8sClusterGroup.GET("", h.GetK8sClusters)
		k8sClusterGroup.GET(":id", h.GetK8sClusterByID)
		k8sClusterGroup.POST("", h.CreateK8sCluster)
		k8sClusterGroup.PUT(":id", h.UpdateK8sCluster)
		k8sClusterGroup.DELETE(":id", h.DeleteK8sCluster)
		k8sClusterGroup.GET(":id/nodes", h.GetK8sClusterNodes)
	}
}

// GetK8sClusters 获取K8s集群列表
// @Summary 获取K8s集群列表
// @Description 获取K8s集群列表，支持分页和过滤
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param size query int true "每页数量"
// @Param cluster_name query string false "集群名称"
// @Param cluster_name_cn query string false "集群中文名称"
// @Param status query string false "状态"
// @Param cluster_type query string false "集群类型"
// @Param idc query string false "IDC"
// @Param zone query string false "可用区"
// @Success 200 {object} service.K8sClusterListResponse
// @Failure 400 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters [get]
func (h *K8sClusterHandler) GetK8sClusters(ctx *gin.Context) {
	var query service.K8sClusterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		render.BadRequest(ctx, "无效的查询参数: "+err.Error())
		return
	}

	// 设置默认值
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 {
		query.Size = 10
	}

	response, err := h.service.GetK8sClusters(ctx.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(ctx, "获取集群列表失败: "+err.Error())
		return
	}

	render.Success(ctx, response)
}

// GetK8sClusterByID 根据ID获取K8s集群
// @Summary 根据ID获取K8s集群
// @Description 根据ID获取K8s集群详细信息
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} service.K8sClusterResponse
// @Failure 400 {object} service.ErrorResponse
// @Failure 404 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters/{id} [get]
func (h *K8sClusterHandler) GetK8sClusterByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(ctx, "无效的ID: "+err.Error())
		return
	}

	response, err := h.service.GetK8sClusterByID(ctx.Request.Context(), id)
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, "获取集群失败: "+err.Error())
		return
	}

	render.Success(ctx, response)
}

// CreateK8sCluster 创建K8s集群
// @Summary 创建K8s集群
// @Description 创建新的K8s集群
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param cluster body service.CreateK8sClusterRequest true "集群信息"
// @Success 201 {object} service.K8sClusterResponse
// @Failure 400 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters [post]
func (h *K8sClusterHandler) CreateK8sCluster(ctx *gin.Context) {
	var req service.CreateK8sClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		render.BadRequest(ctx, "无效的请求参数: "+err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(ctx)

	response, err := h.service.CreateK8sCluster(ctx.Request.Context(), &req, username)
	if err != nil {
		render.InternalServerError(ctx, "创建集群失败: "+err.Error())
		return
	}

	ctx.JSON(http.StatusCreated, render.Response{
		Code: http.StatusCreated,
		Msg:  "success",
		Data: response,
	})
}

// UpdateK8sCluster 更新K8s集群
// @Summary 更新K8s集群
// @Description 更新现有K8s集群信息
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param cluster body service.UpdateK8sClusterRequest true "集群信息"
// @Success 200 {object} service.K8sClusterResponse
// @Failure 400 {object} service.ErrorResponse
// @Failure 404 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters/{id} [put]
func (h *K8sClusterHandler) UpdateK8sCluster(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(ctx, "无效的ID: "+err.Error())
		return
	}

	var req service.UpdateK8sClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		render.BadRequest(ctx, "无效的请求参数: "+err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(ctx)

	response, err := h.service.UpdateK8sCluster(ctx.Request.Context(), id, &req, username)
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, "更新集群失败: "+err.Error())
		return
	}

	render.Success(ctx, response)
}

// DeleteK8sCluster 删除K8s集群
// @Summary 删除K8s集群
// @Description 删除K8s集群
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 204 "No Content"
// @Failure 400 {object} service.ErrorResponse
// @Failure 404 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters/{id} [delete]
func (h *K8sClusterHandler) DeleteK8sCluster(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(ctx, "无效的ID: "+err.Error())
		return
	}

	err = h.service.DeleteK8sCluster(ctx.Request.Context(), id)
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, "删除集群失败: "+err.Error())
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetK8sClusterNodes 获取集群的节点列表
// @Summary 获取集群的节点列表
// @Description 获取指定集群的所有节点
// @Tags K8s集群
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {array} service.K8sNodeResponse
// @Failure 400 {object} service.ErrorResponse
// @Failure 404 {object} service.ErrorResponse
// @Failure 500 {object} service.ErrorResponse
// @Router /fe-v1/k8s-clusters/{id}/nodes [get]
func (h *K8sClusterHandler) GetK8sClusterNodes(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(ctx, "无效的ID: "+err.Error())
		return
	}

	response, err := h.service.GetK8sClusterNodes(ctx.Request.Context(), id)
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, "获取集群节点失败: "+err.Error())
		return
	}

	render.Success(ctx, response)
}

// GetCurrentUsername 获取当前用户名
func GetCurrentUsername(ctx *gin.Context) string {
	// 从上下文中获取用户名，如果没有则返回默认值
	username, exists := ctx.Get("username")
	if !exists {
		return "system"
	}

	usernameStr, ok := username.(string)
	if !ok {
		return "system"
	}

	return usernameStr
}
