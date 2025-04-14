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
	msgFailedToGetFilterOptions = "failed to get filter options: %s"
	msgFailedToGetLabelValues   = "failed to get label values: %s"
	msgFailedToGetTaintValues   = "failed to get taint values: %s"
	msgFailedToQueryDevices     = "failed to query devices: %s"
	msgInvalidQueryRequest      = "invalid query request: %s"
	msgFailedToSaveTemplate     = "failed to save template: %s"
	msgFailedToGetTemplates     = "failed to get templates: %s"
	msgFailedToGetTemplate      = "failed to get template: %s"
	msgFailedToDeleteTemplate   = "failed to delete template: %s"
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
		deviceQueryGroup.POST("/query", h.queryDevices)
		deviceQueryGroup.POST("/templates", h.saveTemplate)
		deviceQueryGroup.GET("/templates", h.getTemplates)
		deviceQueryGroup.GET("/templates/:id", h.getTemplate)
		deviceQueryGroup.DELETE("/templates/:id", h.deleteTemplate)
	}
}

// getFilterOptions handles GET /device-query/filter-options requests.
func (h *DeviceQueryHandler) getFilterOptions(c *gin.Context) {
	options, err := h.service.GetFilterOptions(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetFilterOptions, err.Error()))
		return
	}

	render.Success(c, options)
}

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

// getTemplates handles GET /device-query/templates requests.
func (h *DeviceQueryHandler) getTemplates(c *gin.Context) {
	templates, err := h.service.GetQueryTemplates(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToGetTemplates, err.Error()))
		return
	}

	render.Success(c, templates)
}

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
