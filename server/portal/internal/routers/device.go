package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/pkg/redis"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Added import for gorm
)

// Constants for DeviceHandler - moved to constants.go

// DeviceHandler handles HTTP requests related to Device.
type DeviceHandler struct {
	service *service.DeviceService // Renamed field for consistency
}

// NewDeviceHandler creates a new DeviceHandler, instantiating the service internally.
func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	// 创建 Redis 客户端和键构建器
	redisHandler := redis.NewRedisHandler(RedisDefault)
	keyBuilder := redis.NewKeyBuilder("", service.CacheVersion)

	// 创建设备缓存
	deviceCache := service.NewDeviceCache(redisHandler, keyBuilder)

	// 创建设备查询服务（需要先创建，因为它现在是 DeviceService 的依赖）
	deviceQueryService := service.NewDeviceQueryService(db, deviceCache)

	// 创建设备服务, 注入 deviceQueryService 和 redisHandler
	deviceService := service.NewDeviceService(db, deviceCache, deviceQueryService, redisHandler)

	// 初始化缓存服务
	service.InitCacheService(deviceService, deviceQueryService, deviceCache)

	return &DeviceHandler{service: deviceService}
}

// RegisterRoutes registers Device routes with the given router group.
func (h *DeviceHandler) RegisterRoutes(r *gin.RouterGroup) {
	deviceGroup := r.Group(RouteGroupDevices)
	{
		deviceGroup.GET(RouteParamID, h.getDevice)
		deviceGroup.GET("", h.listDevices)
		deviceGroup.GET(SubRouteExport, h.exportDevices)
		deviceGroup.PATCH(RouteParamIDGroup, h.updateDeviceGroup) // 新增更新用途的路由
	}
}

// getDevice 获取设备详情
// @Summary 获取设备详情
// @Description 根据设备ID获取设备的详细信息
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/devices/{id} [get]
func (h *DeviceHandler) getDevice(c *gin.Context) {
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidDeviceIDFormat)
		return
	}

	device, err := h.service.GetDevice(c.Request.Context(), int(id))
	if err != nil {
		render.NotFound(c, fmt.Sprintf(MsgFailedToGetDevice, err.Error()))
		return
	}

	render.Success(c, device)
}

// listDevices 获取设备列表
// @Summary 获取设备列表
// @Description 获取设备列表，支持分页
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页大小，默认10"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/devices [get]
func (h *DeviceHandler) listDevices(c *gin.Context) {
	var query service.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidQueryParams+err.Error())
		return
	}

	response, err := h.service.ListDevices(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToListDevices, err.Error()))
		return
	}

	render.Success(c, response)
}

// exportDevices 导出设备数据
// @Summary 导出设备数据
// @Description 导出设备数据为Excel文件
// @Tags 设备管理
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body service.DeviceExportRequest true "导出请求参数"
// @Success 200 {file} file "Excel文件"
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/devices/export [post]
func (h *DeviceHandler) exportDevices(c *gin.Context) {
	data, err := h.service.ExportDevices(c.Request.Context())
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToExportDevices, err.Error()))
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

// updateDeviceGroup 更新设备分组
// @Summary 更新设备分组
// @Description 批量更新设备的分组信息
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param request body service.DeviceGroupUpdateRequest true "更新分组请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/devices/group [put]
func (h *DeviceHandler) updateDeviceGroup(c *gin.Context) {
	// 解析设备ID
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, 0)
	if err != nil {
		render.BadRequest(c, MsgInvalidDeviceIDFormat)
		return
	}

	// 解析请求体
	var request service.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidGroupUpdateRequest, err.Error()))
		return
	}

	// 更新设备用途
	err = h.service.UpdateDeviceGroup(c.Request.Context(), int(id), &request)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf(service.ErrDeviceNotFoundMsg, id)) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(MsgFailedToUpdateDeviceGroup, err.Error()))
		}
		return
	}

	// 返回成功响应
	render.Success(c, gin.H{"message": MsgDeviceGroupUpdated})
}
