package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Constants for DeviceHandler
const (
	msgInvalidDeviceIDFormat    = "invalid device id format"
	msgInvalidDeviceQueryParams = "invalid query parameters: %s"
	msgFailedToListDevices      = "failed to list devices: %s"
	msgFailedToGetDevice        = "failed to get device: %s"
	msgFailedToExportDevices    = "failed to export devices: %s"
	msgInvalidRoleUpdateRequest = "invalid role update request: %s"
	msgFailedToUpdateDeviceRole = "failed to update device role: %s"
	msgDeviceRoleUpdated        = "device role updated successfully"
)

// DeviceHandler handles HTTP requests related to Device.
type DeviceHandler struct {
	deviceService *service.DeviceService
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// RegisterRoutes registers Device routes with the given router group.
func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	const idPath = "/:id" // Define path segment constant
	deviceGroup := r.Group("/device")
	{
		deviceGroup.GET(idPath, h.getDevice)
		deviceGroup.GET("", h.listDevices)
		deviceGroup.GET("/export", h.exportDevices)
		deviceGroup.PATCH(idPath+"/role", h.updateDeviceRole) // 新增更新角色的路由
	}
}

// getDevice handles GET /device/:id requests.
func (h *DeviceHandler) getDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, msgInvalidDeviceIDFormat)
		return
	}

	device, err := h.deviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		render.NotFound(c, fmt.Sprintf(msgFailedToGetDevice, err.Error()))
		return
	}

	render.Success(c, device)
}

// listDevices handles GET /device requests.
func (h *DeviceHandler) listDevices(c *gin.Context) {
	var query service.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidDeviceQueryParams, err.Error()))
		return
	}

	response, err := h.deviceService.ListDevices(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToListDevices, err.Error()))
		return
	}

	render.Success(c, response)
}

// updateDeviceRole handles PATCH /device/:id/role requests.
func (h *DeviceHandler) updateDeviceRole(c *gin.Context) {
	// 解析设备ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, msgInvalidDeviceIDFormat)
		return
	}

	// 解析请求体
	var request service.DeviceRoleUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidRoleUpdateRequest, err.Error()))
		return
	}

	// 更新设备角色
	err = h.deviceService.UpdateDeviceRole(c.Request.Context(), id, &request)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf(service.ErrDeviceNotFoundMsg, id)) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToUpdateDeviceRole, err.Error()))
		}
		return
	}

	// 返回成功响应
	render.Success(c, gin.H{"message": msgDeviceRoleUpdated})
}

// exportDevices handles GET /device/export requests.
func (h *DeviceHandler) exportDevices(c *gin.Context) {
	data, err := h.deviceService.ExportDevices(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToExportDevices, err.Error()))
		return
	}

	// 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=device_info.csv")
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))

	// 写入响应体
	c.Data(http.StatusOK, "text/csv", data)
}
