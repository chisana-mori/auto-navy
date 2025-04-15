package routers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Added import for gorm

	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
)

// Constants for DeviceQueryHandler
const (
	msgFailedToGetFilterOptions     = "failed to get filter options: %s"
	msgFailedToGetLabelValues       = "failed to get label values: %s"
	msgFailedToGetTaintValues       = "failed to get taint values: %s"
	msgFailedToGetDeviceFieldValues = "failed to get device field values: %s"
	msgFailedToQueryDevices         = "failed to query devices: %s"
	msgInvalidQueryRequest          = "invalid query request: %s"
	msgFailedToSaveTemplate         = "failed to save template: %s"
	msgFailedToGetTemplates         = "failed to get templates: %s"
	msgFailedToGetTemplate          = "failed to get template: %s"
	msgFailedToDeleteTemplate       = "failed to delete template: %s"
)

// DeviceQueryHandler handles HTTP requests related to DeviceQuery.
type DeviceQueryHandler struct {
	service *service.DeviceQueryService // Renamed field for consistency
}

// NewDeviceQueryHandler creates a new DeviceQueryHandler, instantiating the service internally.
func NewDeviceQueryHandler(db *gorm.DB) *DeviceQueryHandler {
	deviceQueryService := service.NewDeviceQueryService(db) // Instantiate service here
	return &DeviceQueryHandler{service: deviceQueryService}
}

// RegisterRoutes registers DeviceQuery routes with the given router group.
func (h *DeviceQueryHandler) RegisterRoutes(r *gin.RouterGroup) {
	deviceQueryGroup := r.Group("/device-query")
	{
		deviceQueryGroup.GET("/filter-options", h.getFilterOptions)
		deviceQueryGroup.GET("/label-values", h.getLabelValues)
		deviceQueryGroup.GET("/taint-values", h.getTaintValues)
		deviceQueryGroup.GET("/device-field-values", h.getDeviceFieldValues)
		deviceQueryGroup.POST("/query", h.queryDevices)
		deviceQueryGroup.POST("/templates", h.saveTemplate)
		deviceQueryGroup.GET("/templates", h.getTemplates)
		deviceQueryGroup.GET("/templates/:id", h.getTemplate)
		deviceQueryGroup.DELETE("/templates/:id", h.deleteTemplate)
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
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetFilterOptions, err.Error()))
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
func (h *DeviceQueryHandler) getLabelValues(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		render.BadRequest(c, "key is required")
		return
	}

	values, err := h.service.GetLabelValues(c.Request.Context(), key)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetLabelValues, err.Error()))
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
	key := c.Query("key")
	if key == "" {
		render.BadRequest(c, "key is required")
		return
	}

	values, err := h.service.GetTaintValues(c.Request.Context(), key)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetTaintValues, err.Error()))
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
func (h *DeviceQueryHandler) getDeviceFieldValues(c *gin.Context) {
	field := c.Query("field")
	if field == "" {
		render.BadRequest(c, "field is required")
		return
	}

	values, err := h.service.GetDeviceFieldValues(c.Request.Context(), field)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetDeviceFieldValues, err.Error()))
		return
	}

	render.Success(c, values)
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
		render.BadRequest(c, fmt.Sprintf(msgInvalidQueryRequest, err.Error()))
		return
	}

	resp, err := h.service.QueryDevices(c.Request.Context(), &req)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToQueryDevices, err.Error()))
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
		render.BadRequest(c, fmt.Sprintf(msgInvalidQueryRequest, err.Error()))
		return
	}

	err := h.service.SaveQueryTemplate(c.Request.Context(), &template)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToSaveTemplate, err.Error()))
		return
	}

	render.Success(c, gin.H{"message": "Template saved successfully"})
}

// @Summary 获取查询模板列表
// @Description 获取所有设备查询模板列表
// @Tags 设备查询
// @Accept json
// @Produce json
// @Success 200 {array} service.QueryTemplate "成功获取模板列表"
// @Failure 500 {object} service.ErrorResponse "获取模板列表失败"
// @Router /device-query/templates [get]
// getTemplates handles GET /device-query/templates requests.
func (h *DeviceQueryHandler) getTemplates(c *gin.Context) {
	templates, err := h.service.GetQueryTemplates(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetTemplates, err.Error()))
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
	idStr := c.Param("id")
	if idStr == "" {
		render.BadRequest(c, "id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, "invalid id format")
		return
	}

	template, err := h.service.GetQueryTemplate(c.Request.Context(), id)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetTemplate, err.Error()))
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
	idStr := c.Param("id")
	if idStr == "" {
		render.BadRequest(c, "id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, "invalid id format")
		return
	}

	err = h.service.DeleteQueryTemplate(c.Request.Context(), id)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToDeleteTemplate, err.Error()))
		return
	}

	render.Success(c, gin.H{"message": "Template deleted successfully"})
}

// Removed unused import
