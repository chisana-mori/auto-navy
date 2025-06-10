package routers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Added import for gorm

	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/service"
)

// Constants for DeviceQueryHandler - moved to constants.go

// DeviceQueryHandler handles HTTP requests related to DeviceQuery.
type DeviceQueryHandler struct {
	service *service.DeviceQueryService // Renamed field for consistency
}

// NewDeviceQueryHandler creates a new DeviceQueryHandler, instantiating the service internally.
func NewDeviceQueryHandler(db *gorm.DB) *DeviceQueryHandler {
	// 创建 Redis 客户端和键构建器
	redisHandler := redis.NewRedisHandler(RedisDefault)
	keyBuilder := redis.NewKeyBuilder(RedisNamespace, service.CacheVersion)

	// 创建设备缓存
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	// 创建设备查询服务
	deviceQueryService := service.NewDeviceQueryService(db, deviceCache)

	return &DeviceQueryHandler{service: deviceQueryService}
}

// RegisterRoutes registers DeviceQuery routes with the given router group.
func (h *DeviceQueryHandler) RegisterRoutes(r *gin.RouterGroup) {
	deviceQueryGroup := r.Group(RouteGroupDeviceQuery)
	{
		deviceQueryGroup.GET(SubRouteFilterOptions, h.getFilterOptions)
		deviceQueryGroup.GET(SubRouteLabelValues, h.getLabelValues)
		deviceQueryGroup.GET(SubRouteTaintValues, h.getTaintValues)
		deviceQueryGroup.GET(SubRouteDeviceFieldValues, h.getDeviceFieldValues)
		deviceQueryGroup.GET(SubRouteDeviceFeatureDetails, h.getDeviceFeatureDetails)
		deviceQueryGroup.POST(SubRouteQuery, h.queryDevices)
		deviceQueryGroup.POST(SubRouteTemplates, h.saveTemplate)
		deviceQueryGroup.GET(SubRouteTemplates, h.getTemplates)
		deviceQueryGroup.GET(SubRouteTemplates+RouteParamID, h.getTemplate)
		deviceQueryGroup.DELETE(SubRouteTemplates+RouteParamID, h.deleteTemplate)
	}
}

// @Summary 获取设备筛选项
// @Description 获取设备筛选项，包括设备字段、节点标签和节点污点
// @Tags 设备查询
// @Accept json
// @Produce json
// @Success 200 {object} map[string][]service.FilterOptionResponse "成功获取筛选项"
// @Failure 500 {object} service.ErrorResponse "获取筛选项失败"
// @Router /device-query/filter-options [get]
// getFilterOptions handles GET /device-query/filter-options requests.
func (h *DeviceQueryHandler) getFilterOptions(c *gin.Context) {
	options, err := h.service.GetFilterOptions(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetFilterOptions, err.Error()))
		return
	}

	render.Success(c, options)
}

// @Summary 获取节点标签可选值
// @Description 根据标签键获取节点标签的可选值列表
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param key query string true "标签键" example="env"
// @Success 200 {array} service.FilterOptionResponse "成功获取标签值"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取标签值失败"
// @Router /device-query/label-values [get]
// getLabelValues handles GET /device-query/label-values requests.
type KeyQuery struct {
	Key string `form:"key" binding:"required"`
}

func (h *DeviceQueryHandler) getLabelValues(c *gin.Context) {
	var query KeyQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	values, err := h.service.GetLabelValues(c.Request.Context(), query.Key)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetLabelValues, err.Error()))
		return
	}

	render.Success(c, values)
}

// @Summary 获取节点污点可选值
// @Description 根据污点键获取节点污点的可选值列表
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param key query string true "污点键" example="node.kubernetes.io/unschedulable"
// @Success 200 {array} service.FilterOptionResponse "成功获取污点值"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取污点值失败"
// @Router /device-query/taint-values [get]
// getTaintValues handles GET /device-query/taint-values requests.
func (h *DeviceQueryHandler) getTaintValues(c *gin.Context) {
	var query KeyQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	values, err := h.service.GetTaintValues(c.Request.Context(), query.Key)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetTaintValues, err.Error()))
		return
	}

	render.Success(c, values)
}

// @Summary 获取设备字段可选值
// @Description 根据设备字段名获取该字段的可选值列表
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param field query string true "字段名" example:"idc"
// @Success 200 {array} string "成功获取字段值"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取字段值失败"
// @Router /device-query/device-field-values [get]
// getDeviceFieldValues handles GET /device-query/device-field-values requests.
type FieldQuery struct {
	Field string `form:"field" binding:"required"`
}

func (h *DeviceQueryHandler) getDeviceFieldValues(c *gin.Context) {
	var query FieldQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	values, err := h.service.GetDeviceFieldValues(c.Request.Context(), query.Field)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetDeviceFieldValues, err.Error()))
		return
	}

	render.Success(c, values)
}

// @Summary 获取设备特性详情
// @Description 获取设备的标签和污点详情
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param ci_code query string true "设备编码" example:"node-1"
// @Success 200 {object} service.DeviceFeatureDetails "成功获取设备特性详情"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取设备特性详情失败"
// @Router /device-query/device-feature-details [get]
// getDeviceFeatureDetails handles GET /device-query/device-feature-details requests.
type CiCodeQuery struct {
	CiCode string `form:"ci_code" binding:"required"`
}

func (h *DeviceQueryHandler) getDeviceFeatureDetails(c *gin.Context) {
	var query CiCodeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidParams+err.Error())
		return
	}

	details, err := h.service.GetDeviceFeatureDetails(c.Request.Context(), query.CiCode)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf("Failed to get device feature details: %s", err.Error()))
		return
	}

	render.Success(c, details)
}

// @Summary 查询设备
// @Description 根据复杂条件查询设备，支持设备字段、节点标签和节点污点筛选
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param data body service.DeviceQueryRequest true "查询条件"
// @Success 200 {object} service.DeviceListResponse "成功查询设备"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "查询设备失败"
// @Router /device-query/query [post]
// queryDevices handles POST /device-query/query requests.
func (h *DeviceQueryHandler) queryDevices(c *gin.Context) {
	var req service.DeviceQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidQueryRequest, err.Error()))
		return
	}

	resp, err := h.service.QueryDevices(c.Request.Context(), &req)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToQueryDevices, err.Error()))
		return
	}

	render.Success(c, resp)
}

// @Summary 保存查询模板
// @Description 保存设备查询模板，方便后续复用
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param data body service.QueryTemplate true "模板信息"
// @Success 200 {object} map[string]string "模板保存成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "保存模板失败"
// @Router /device-query/templates [post]
// saveTemplate handles POST /device-query/templates requests.
func (h *DeviceQueryHandler) saveTemplate(c *gin.Context) {
	var template service.QueryTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidQueryRequest, err.Error()))
		return
	}

	err := h.service.SaveQueryTemplate(c.Request.Context(), &template)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToSaveTemplate, err.Error()))
		return
	}

	render.Success(c, gin.H{"message": "Template saved successfully"})
}

// @Summary 获取查询模板列表
// @Description 获取所有设备查询模板列表，支持分页
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param page query int false "页码，默认为1" example:"1"
// @Param size query int false "每页数量，默认为10，最大为100" example:"10"
// @Success 200 {object} service.QueryTemplateListResponse "成功获取模板列表"
// @Failure 500 {object} service.ErrorResponse "获取模板列表失败"
// @Router /device-query/templates [get]
// getTemplates handles GET /device-query/templates requests.
func (h *DeviceQueryHandler) getTemplates(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery(ParamPage, DefaultPageValue)
	sizeStr := c.DefaultQuery(ParamSize, DefaultSizeValue)

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = service.DefaultPage
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		size = service.DefaultSize
	}

	// 使用带分页的方法查询模板
	templates, err := h.service.GetQueryTemplatesWithPagination(c.Request.Context(), page, size)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetTemplates, err.Error()))
		return
	}

	render.Success(c, templates)
}

// @Summary 获取查询模板详情
// @Description 根据模板ID获取设备查询模板详情
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param id path int true "模板ID" example:"1"
// @Success 200 {object} service.QueryTemplate "成功获取模板详情"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取模板详情失败"
// @Router /device-query/templates/{id} [get]
// getTemplate handles GET /device-query/templates/:id requests.
func (h *DeviceQueryHandler) getTemplate(c *gin.Context) {
	idStr := c.Param(ParamID)
	if idStr == "" {
		render.BadRequest(c, "id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidIDFormat)
		return
	}

	template, err := h.service.GetQueryTemplate(c.Request.Context(), int(id))
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToGetTemplate, err.Error()))
		return
	}

	render.Success(c, template)
}

// @Summary 删除查询模板
// @Description 根据模板ID删除设备查询模板
// @Tags 设备查询
// @Accept json
// @Produce json
// @Param id path int true "模板ID" example:"1"
// @Success 200 {object} map[string]string "模板删除成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "删除模板失败"
// @Router /device-query/templates/{id} [delete]
// deleteTemplate handles DELETE /device-query/templates/:id requests.
func (h *DeviceQueryHandler) deleteTemplate(c *gin.Context) {
	idStr := c.Param(ParamID)
	if idStr == "" {
		render.BadRequest(c, "id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidIDFormat)
		return
	}

	err = h.service.DeleteQueryTemplate(c.Request.Context(), int(id))
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToDeleteTemplate, err.Error()))
		return
	}

	render.Success(c, gin.H{"message": "Template deleted successfully"})
}

// Removed unused import
