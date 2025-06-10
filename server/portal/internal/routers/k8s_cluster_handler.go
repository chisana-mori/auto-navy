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
	k8sClusterGroup := router.Group(RouteGroupK8sClusters) // 路由组已经在main.go中设置为/fe-v1
	{
		k8sClusterGroup.GET("", h.GetK8sClusters)
		k8sClusterGroup.GET(RouteParamID, h.GetK8sClusterByID)
		k8sClusterGroup.POST("", h.CreateK8sCluster)
		k8sClusterGroup.PUT(RouteParamID, h.UpdateK8sCluster)
		k8sClusterGroup.DELETE(RouteParamID, h.DeleteK8sCluster)
		k8sClusterGroup.GET(RouteParamIDNodes, h.GetK8sClusterNodes)
	}
}

// GetK8sClusters 获取K8s集群列表
// @Summary 获取K8s集群列表
// @Description 获取K8s集群列表，支持分页和过滤
// @Tags K8s集群管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页大小，默认10"
// @Param name query string false "集群名称"
// @Param status query string false "集群状态"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/k8s-clusters [get]
func (h *K8sClusterHandler) GetK8sClusters(ctx *gin.Context) {
	var query service.K8sClusterQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		render.BadRequest(ctx, MsgInvalidQueryParams+err.Error())
		return
	}

	// 设置默认值
	if query.Page <= 0 {
		query.Page = DefaultPageInt
	}
	if query.Size <= 0 {
		query.Size = DefaultSizeInt
	}

	response, err := h.service.GetK8sClusters(ctx.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(ctx, MsgFailedToGetClusters+err.Error())
		return
	}

	render.Success(ctx, response)
}

// GetK8sClusterByID 获取K8s集群详情
// @Summary 获取K8s集群详情
// @Description 根据集群ID获取K8s集群的详细信息
// @Tags K8s集群管理
// @Accept json
// @Produce json
// @Param id path string true "集群ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/k8s-clusters/{id} [get]
func (h *K8sClusterHandler) GetK8sClusterByID(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(ctx, MsgInvalidID+": "+err.Error())
		return
	}

	response, err := h.service.GetK8sClusterByID(ctx.Request.Context(), int(id))
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, MsgFailedToGetCluster+err.Error())
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
		render.BadRequest(ctx, MsgInvalidRequestParams+err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(ctx)

	response, err := h.service.CreateK8sCluster(ctx.Request.Context(), &req, username)
	if err != nil {
		render.InternalServerError(ctx, MsgFailedToCreateCluster+err.Error())
		return
	}

	ctx.JSON(http.StatusCreated, render.Response{
		Code: http.StatusCreated,
		Msg:  MsgSuccess,
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
	id, err := strconv.ParseInt(ctx.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(ctx, MsgInvalidID+": "+err.Error())
		return
	}

	var req service.UpdateK8sClusterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		render.BadRequest(ctx, MsgInvalidRequestParams+err.Error())
		return
	}

	// 获取当前用户
	username := GetCurrentUsername(ctx)

	response, err := h.service.UpdateK8sCluster(ctx.Request.Context(), int(id), &req, username)
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, MsgFailedToUpdateCluster+err.Error())
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
	id, err := strconv.ParseInt(ctx.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(ctx, MsgInvalidID+": "+err.Error())
		return
	}

	err = h.service.DeleteK8sCluster(ctx.Request.Context(), int(id))
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, MsgFailedToDeleteCluster+err.Error())
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
	id, err := strconv.ParseInt(ctx.Param(ParamID), Base10, BitSize64)
	if err != nil {
		render.BadRequest(ctx, MsgInvalidID+": "+err.Error())
		return
	}

	response, err := h.service.GetK8sClusterNodes(ctx.Request.Context(), int(id))
	if err != nil {
		if service.IsNotFound(err) {
			render.NotFound(ctx, err.Error())
			return
		}
		render.InternalServerError(ctx, MsgFailedToGetClusterNodes+err.Error())
		return
	}

	render.Success(ctx, response)
}

// GetCurrentUsername 获取当前用户名
func GetCurrentUsername(ctx *gin.Context) string {
	// 从上下文中获取用户名，如果没有则返回默认值
	username, exists := ctx.Get(UsernameContextKey)
	if !exists {
		return DefaultUsername
	}

	usernameStr, ok := username.(string)
	if !ok {
		return DefaultUsername
	}

	return usernameStr
}
