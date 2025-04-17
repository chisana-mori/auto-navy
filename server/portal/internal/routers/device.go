package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Added import for gorm
)

// Constants for DeviceHandler
const (
	msgInvalidDeviceIDFormat     = "invalid device id format"
	msgInvalidDeviceQueryParams  = "invalid query parameters: %s"
	msgFailedToListDevices       = "failed to list devices: %s"
	msgFailedToGetDevice         = "failed to get device: %s"
	msgFailedToExportDevices     = "failed to export devices: %s"
	msgInvalidRoleUpdateRequest  = "invalid role update request: %s"
	msgFailedToUpdateDeviceRole  = "failed to update device role: %s"
	msgDeviceRoleUpdated         = "device role updated successfully"
	msgInvalidGroupUpdateRequest = "invalid group update request: %s"
	msgFailedToUpdateDeviceGroup = "failed to update device group: %s"
	msgDeviceGroupUpdated        = "device group updated successfully"
)

// DeviceHandler handles HTTP requests related to Device.
type DeviceHandler struct {
	service *service.DeviceService // Renamed field for consistency
}

// NewDeviceHandler creates a new DeviceHandler, instantiating the service internally.
func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	deviceService := service.NewDeviceService(db) // Instantiate service here
	return &DeviceHandler{service: deviceService}
}

// RegisterRoutes registers Device routes with the given router group.
func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	const idPath = "/:id" // Define path segment constant
	deviceGroup := r.Group("/device")
	{
		deviceGroup.GET(idPath, h.getDevice)
		deviceGroup.GET("", h.listDevices)
		deviceGroup.GET("/export", h.exportDevices)
		deviceGroup.PATCH(idPath+"/group", h.updateDeviceGroup) // 新增更新用途的路由
	}
}

// @Summary 获取设备详情
// @Description 根据设备ID获取设备详情
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID" example:"1"
// @Success 200 {object} service.DeviceResponse "成功获取设备详情"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "设备不存在"
// @Failure 500 {object} service.ErrorResponse "获取设备详情失败"
// @Router /device/{id} [get]
// getDevice handles GET /device/:id requests.
func (h *DeviceHandler) getDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, msgInvalidDeviceIDFormat)
		return
	}

	device, err := h.service.GetDevice(c.Request.Context(), id)
	if err != nil {
		render.NotFound(c, fmt.Sprintf(msgFailedToGetDevice, err.Error()))
		return
	}

	render.Success(c, device)
}

// @Summary 获取设备列表
// @Description 获取设备列表，支持分页和关键字搜索
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param page query int false "页码" example:"1"
// @Param size query int false "每页数量" example:"10"
// @Param keyword query string false "搜索关键字" example:"192.168"
// @Success 200 {object} service.DeviceListResponse "成功获取设备列表"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取设备列表失败"
// @Router /device [get]
// listDevices handles GET /device requests.
func (h *DeviceHandler) listDevices(c *gin.Context) {
	var query service.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidDeviceQueryParams, err.Error()))
		return
	}

	response, err := h.service.ListDevices(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToListDevices, err.Error()))
		return
	}

	render.Success(c, response)
}

// @Summary 更新设备角色
// @Description 根据设备ID更新设备的集群角色
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID" example:"1"
// @Param data body service.DeviceRoleUpdateRequest true "角色更新信息"
// @Success 200 {object} map[string]string "角色更新成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "设备不存在"
// @Failure 500 {object} service.ErrorResponse "更新角色失败"
// @Router /device/{id}/role [patch]
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
	err = h.service.UpdateDeviceRole(c.Request.Context(), id, &request)
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

// @Summary 导出设备信息
// @Description 导出所有设备信息为CSV文件，包含设备的全部字段
// @Tags 设备管理
// @Accept json
// @Produce text/csv
// @Success 200 {file} file "device_info.csv"
// @Failure 500 {object} service.ErrorResponse "导出设备信息失败"
// @Router /device/export [get]
// exportDevices handles GET /device/export requests.
func (h *DeviceHandler) exportDevices(c *gin.Context) {
	data, err := h.service.ExportDevices(c.Request.Context())
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

// @Summary 更新设备用途
// @Description 根据设备ID更新设备的用途
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID" example:"1"
// @Param data body service.DeviceGroupUpdateRequest true "用途更新信息"
// @Success 200 {object} map[string]string "用途更新成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "设备不存在"
// @Failure 500 {object} service.ErrorResponse "更新用途失败"
// @Router /device/{id}/group [patch]
// updateDeviceGroup handles PATCH /device/:id/group requests.
func (h *DeviceHandler) updateDeviceGroup(c *gin.Context) {
	// 解析设备ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		render.BadRequest(c, msgInvalidDeviceIDFormat)
		return
	}

	// 解析请求体
	var request service.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidGroupUpdateRequest, err.Error()))
		return
	}

	// 更新设备用途
	err = h.service.UpdateDeviceGroup(c.Request.Context(), id, &request)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf(service.ErrDeviceNotFoundMsg, id)) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToUpdateDeviceGroup, err.Error()))
		}
		return
	}

	// 返回成功响应
	render.Success(c, gin.H{"message": msgDeviceGroupUpdated})
}
