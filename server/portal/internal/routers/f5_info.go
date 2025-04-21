// Package routers defines the HTTP routes for the portal module.
package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Added import for gorm
)

// Constants for HTTP status codes and default values
const (
	base10                = 10
	bitSize64             = 64
	routeParamID          = "id" // Route parameter for ID
	msgInvalidIDFormat    = "invalid id format"
	msgInvalidQueryParams = "invalid query parameters: %s"
	msgInvalidRequestBody = "invalid request body: %s"
	msgFailedToList       = "failed to list F5 infos: %s"
	msgFailedToUpdate     = "failed to update F5 info: %s"
	msgFailedToDelete     = "failed to delete F5 info: %s"
	msgSuccessUpdate      = "F5 info updated successfully"
	msgSuccessDelete      = "F5 info deleted successfully"
)

// F5InfoHandler handles HTTP requests related to F5Info.
type F5InfoHandler struct {
	service *service.F5InfoService // Renamed field for consistency
}

// NewF5InfoHandler creates a new F5InfoHandler, instantiating the service internally.
func NewF5InfoHandler(db *gorm.DB) *F5InfoHandler {
	f5Service := service.NewF5InfoService(db) // Instantiate service here
	return &F5InfoHandler{service: f5Service}
}

// RegisterRoutes registers F5Info routes with the given router group.
func (h *F5InfoHandler) RegisterRoutes(r *gin.RouterGroup) {
	const idPath = "/:id" // Define path segment constant
	f5Group := r.Group("/f5")
	{
		f5Group.GET(idPath, h.getF5Info)
		f5Group.GET("", h.listF5Infos)
		f5Group.PUT(idPath, h.updateF5Info)
		f5Group.DELETE(idPath, h.deleteF5Info)
	}
}

// @Summary 获取F5信息详情
// @Description 根据ID获取F5信息的详细信息
// @Tags F5管理
// @Accept json
// @Produce json
// @Param id path int true "F5信息ID" example:"1"
// @Success 200 {object} service.F5InfoResponse "成功获取F5信息详情"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "F5信息不存在"
// @Failure 500 {object} service.ErrorResponse "获取F5信息详情失败"
// @Router /f5/{id} [get]
// getF5Info handles GET /f5/:id requests.
func (h *F5InfoHandler) getF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidIDFormat)
		return
	}

	f5Info, err := h.service.GetF5Info(c.Request.Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, "f5", id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, err.Error())
		}
		return
	}

	render.Success(c, f5Info)
}

// @Summary 获取F5信息列表
// @Description 获取F5信息列表，支持分页和多条件筛选
// @Tags F5管理
// @Accept json
// @Produce json
// @Param page query int true "页码" example:"1"
// @Param size query int true "每页数量" example:"10"
// @Param name query string false "F5名称" example:"f5-test"
// @Param vip query string false "VIP地址" example:"192.168.1.1"
// @Param port query string false "端口" example:"80"
// @Param appid query string false "应用ID" example:"app-001"
// @Param status query string false "状态" example:"active"
// @Success 200 {object} service.F5InfoListResponse "成功获取F5信息列表"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取F5信息列表失败"
// @Router /f5 [get]
// listF5Infos handles GET /f5 requests.
func (h *F5InfoHandler) listF5Infos(c *gin.Context) {
	var query service.F5InfoQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidQueryParams, err.Error()))
		return
	}

	response, err := h.service.ListF5Infos(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToList, err.Error()))
		return
	}

	render.Success(c, response)
}

// @Summary 更新F5信息
// @Description 根据ID更新F5信息的各项属性
// @Tags F5管理
// @Accept json
// @Produce json
// @Param id path int true "F5信息ID" example:"1"
// @Param data body service.F5InfoUpdateDTO true "F5信息更新内容"
// @Success 200 {object} map[string]string "F5信息更新成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "F5信息不存在"
// @Failure 500 {object} service.ErrorResponse "更新F5信息失败"
// @Router /f5/{id} [put]
// updateF5Info handles PUT /f5/:id requests.
func (h *F5InfoHandler) updateF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidIDFormat)
		return
	}

	var dto service.F5InfoUpdateDTO
	if bindErr := c.ShouldBindJSON(&dto); bindErr != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidRequestBody, bindErr.Error()))
		return
	}

	if err := h.service.UpdateF5Info(c.Request.Context(), id, &dto); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, "f5", id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToUpdate, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, msgSuccessUpdate, nil)
}

// @Summary 删除F5信息
// @Description 根据ID删除F5信息，删除后无法恢复
// @Tags F5管理
// @Accept json
// @Produce json
// @Param id path int true "F5信息ID" example:"1"
// @Success 200 {object} map[string]string "F5信息删除成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "F5信息不存在"
// @Failure 500 {object} service.ErrorResponse "删除F5信息失败"
// @Router /f5/{id} [delete]
// deleteF5Info handles DELETE /f5/:id requests.
func (h *F5InfoHandler) deleteF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidIDFormat)
		return
	}

	if err := h.service.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, "f5", id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToDelete, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, msgSuccessDelete, nil)
}
