package routers

import (
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MaintenanceHandler 设备维护处理器
type MaintenanceHandler struct {
	service *service.MaintenanceServiceV2
}

// NewMaintenanceHandler 创建设备维护处理器
func NewMaintenanceHandler(db *gorm.DB) *MaintenanceHandler {
	logger, _ := zap.NewProduction()

	return &MaintenanceHandler{
		service: service.NewMaintenanceServiceV2(db, logger),
	}
}

// RegisterRoutes 注册路由
func (h *MaintenanceHandler) RegisterRoutes(api *gin.RouterGroup) {
	maintenanceGroup := api.Group("/device-maintenance")

	// 设备维护请求接口
	maintenanceGroup.POST("/request", h.RequestMaintenance)
	maintenanceGroup.POST("/callback", h.MaintenanceCallback)

	// 运维操作接口
	maintenanceGroup.GET("/requests", h.GetPendingMaintenanceRequests)
	maintenanceGroup.GET("/uncordon-requests", h.GetPendingUncordonRequests)
	maintenanceGroup.POST("/confirm/:id", h.ConfirmMaintenance)
	maintenanceGroup.POST("/start/:id", h.StartMaintenance)
	maintenanceGroup.POST("/uncordon/:id", h.ExecuteUncordon)
}

// RequestMaintenance 处理来自上游系统的设备维护请求
// @Summary 处理设备维护请求
// @Description 接收并处理来自上游系统的设备维护请求
// @Tags 设备维护
// @Accept json
// @Produce json
// @Param request body service.MaintenanceRequestDTO true "维护请求信息"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/request [post]
func (h *MaintenanceHandler) RequestMaintenance(c *gin.Context) {
	var request service.MaintenanceRequestDTO
	if err := c.ShouldBindJSON(&request); err != nil {
		render.BadRequest(c, MsgInvalidMaintenanceRequest+err.Error())
		return
	}

	// 验证必填字段
	if request.DeviceID == 0 && request.CICode == "" {
		render.BadRequest(c, MsgDeviceIDOrCICodeRequired)
		return
	}

	if request.ExternalTicketID == "" {
		render.BadRequest(c, "外部工单号不能为空")
		return
	}

	// 如果提供了CI编码但没有设备ID，需要根据CI编码查找设备ID
	// 此功能在MaintenanceService中实现

	response, err := h.service.RequestMaintenance(&request)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "处理维护请求失败: "+err.Error())
		return
	}

	render.Success(c, response)
}

// MaintenanceCallback 处理上游系统的维护完成回调
// @Summary 处理维护完成回调
// @Description 接收并处理来自上游系统的维护完成通知
// @Tags 设备维护
// @Accept json
// @Produce json
// @Param callback body service.MaintenanceCallbackDTO true "维护回调信息"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/callback [post]
func (h *MaintenanceHandler) MaintenanceCallback(c *gin.Context) {
	var callback service.MaintenanceCallbackDTO
	if err := c.ShouldBindJSON(&callback); err != nil {
		render.BadRequest(c, "无效的回调格式: "+err.Error())
		return
	}

	// 验证必填字段
	if callback.ExternalTicketID == "" {
		render.BadRequest(c, "外部工单号不能为空")
		return
	}

	// 根据回调状态执行相应操作
	switch callback.Status {
	case "completed":
		response, err := h.service.CompleteMaintenance(callback.ExternalTicketID, callback.Message)
		if err != nil {
			render.Fail(c, http.StatusInternalServerError, "处理维护完成回调失败: "+err.Error())
			return
		}
		render.Success(c, response)

	// 可以根据需要扩展其他状态处理，如cancelled, delayed等
	default:
		render.BadRequest(c, "不支持的维护状态: "+callback.Status)
	}
}

// getMaintenance 获取维护任务详情
// @Summary 获取维护任务详情
// @Description 根据维护任务ID获取维护任务的详细信息
// @Tags 维护管理
// @Accept json
// @Produce json
// @Param id path string true "维护任务ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/maintenance/{id} [get]
func (h *MaintenanceHandler) getMaintenance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		render.BadRequest(c, "维护任务ID不能为空")
		return
	}

	// TODO: 实现获取维护任务详情的逻辑
	render.Success(c, gin.H{"message": "获取维护任务详情功能待实现"})
}

// listMaintenances 获取维护任务列表
// @Summary 获取维护任务列表
// @Description 获取维护任务列表，支持分页和过滤
// @Tags 维护管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页大小，默认10"
// @Param status query string false "维护状态"
// @Param type query string false "维护类型"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/maintenance [get]
func (h *MaintenanceHandler) listMaintenances(c *gin.Context) {
	// TODO: 实现获取维护任务列表的逻辑
	render.Success(c, gin.H{"message": "获取维护任务列表功能待实现"})
}

// createMaintenance 创建维护任务
// @Summary 创建维护任务
// @Description 创建新的维护任务
// @Tags 维护管理
// @Accept json
// @Produce json
// @Param request body service.CreateMaintenanceRequest true "创建维护任务请求"
// @Success 201 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/maintenance [post]
func (h *MaintenanceHandler) createMaintenance(c *gin.Context) {
	// TODO: 实现创建维护任务的逻辑
	render.Success(c, gin.H{"message": "创建维护任务功能待实现"})
}

// updateMaintenance 更新维护任务
// @Summary 更新维护任务
// @Description 更新维护任务信息
// @Tags 维护管理
// @Accept json
// @Produce json
// @Param id path string true "维护任务ID"
// @Param request body service.UpdateMaintenanceRequest true "更新维护任务请求"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/maintenance/{id} [put]
func (h *MaintenanceHandler) updateMaintenance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		render.BadRequest(c, "维护任务ID不能为空")
		return
	}

	// TODO: 实现更新维护任务的逻辑
	render.Success(c, gin.H{"message": "更新维护任务功能待实现"})
}

// deleteMaintenance 删除维护任务
// @Summary 删除维护任务
// @Description 删除指定的维护任务
// @Tags 维护管理
// @Accept json
// @Produce json
// @Param id path string true "维护任务ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/maintenance/{id} [delete]
func (h *MaintenanceHandler) deleteMaintenance(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		render.BadRequest(c, "维护任务ID不能为空")
		return
	}

	// TODO: 实现删除维护任务的逻辑
	render.Success(c, gin.H{"message": "删除维护任务功能待实现"})
}

func (h *MaintenanceHandler) GetPendingMaintenanceRequests(c *gin.Context) {
	requests, err := h.service.GetPendingMaintenanceRequests()
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "获取待处理维护请求失败: "+err.Error())
		return
	}

	render.Success(c, requests)
}

// GetPendingUncordonRequests 获取待处理的Uncordon请求
// @Summary 获取待处理的Uncordon请求
// @Description 获取所有待执行的节点Uncordon请求
// @Tags 设备维护
// @Accept json
// @Produce json
// @Success 200 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/uncordon-requests [get]
func (h *MaintenanceHandler) GetPendingUncordonRequests(c *gin.Context) {
	requests, err := h.service.GetPendingUncordonRequests()
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "获取待处理Uncordon请求失败: "+err.Error())
		return
	}

	render.Success(c, requests)
}

// ConfirmMaintenance 确认维护请求
// @Summary 确认维护请求
// @Description 确认接受维护请求，更改状态为已确认待维护
// @Tags 设备维护
// @Accept json
// @Produce json
// @Param id path int true "维护订单ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/confirm/{id} [post]
func (h *MaintenanceHandler) ConfirmMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	// 获取操作人信息，实际环境中应从认证信息获取
	operatorID := c.GetString("userId")
	if operatorID == "" {
		operatorID = "admin"
	}

	err = h.service.ConfirmMaintenance(id, operatorID)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "确认维护请求失败: "+err.Error())
		return
	}

	render.Success(c, gin.H{"message": "维护请求已确认"})
}

// StartMaintenance 开始维护，执行Cordon操作
// @Summary 开始设备维护
// @Description 执行节点Cordon操作，准备设备维护
// @Tags 设备维护
// @Accept json
// @Produce json
// @Param id path int true "维护订单ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/start/{id} [post]
func (h *MaintenanceHandler) StartMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	// 获取操作人信息，实际环境中应从认证信息获取
	operatorID := c.GetString("userId")
	if operatorID == "" {
		operatorID = "admin"
	}

	err = h.service.StartMaintenance(id, operatorID)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "执行Cordon操作失败: "+err.Error())
		return
	}

	render.Success(c, gin.H{"message": "节点Cordon操作已执行，设备维护已开始"})
}

// ExecuteUncordon 执行Uncordon操作
// @Summary 执行节点Uncordon操作
// @Description 执行节点Uncordon操作，恢复节点服务
// @Tags 设备维护
// @Accept json
// @Produce json
// @Param id path int true "Uncordon订单ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.Response
// @Failure 500 {object} render.Response
// @Router /fe-v1/device-maintenance/uncordon/{id} [post]
func (h *MaintenanceHandler) ExecuteUncordon(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.BadRequest(c, "无效的订单ID")
		return
	}

	// 获取操作人信息，实际环境中应从认证信息获取
	operatorID := c.GetString("userId")
	if operatorID == "" {
		operatorID = "admin"
	}

	err = h.service.ExecuteUncordon(id, operatorID)
	if err != nil {
		render.Fail(c, http.StatusInternalServerError, "执行Uncordon操作失败: "+err.Error())
		return
	}

	render.Success(c, gin.H{"message": "节点Uncordon操作已执行，节点已恢复服务"})
}
