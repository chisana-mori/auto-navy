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

// Constants for HTTP status codes and default values - moved to constants.go

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

// getF5Info 获取F5信息
// @Summary 获取F5信息
// @Description 根据F5 ID获取F5设备的详细信息
// @Tags F5管理
// @Accept json
// @Produce json
// @Param id path string true "F5设备ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/f5/{id} [get]
func (h *F5InfoHandler) getF5Info(c *gin.Context) {
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidIDFormat)
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

// listF5Info 获取F5信息列表
// @Summary 获取F5信息列表
// @Description 获取F5设备信息列表，支持分页
// @Tags F5管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页大小，默认10"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/f5 [get]
func (h *F5InfoHandler) listF5Infos(c *gin.Context) {
	var query service.F5InfoQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, MsgInvalidQueryParams+err.Error())
		return
	}

	response, err := h.service.ListF5Infos(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToListF5, err.Error()))
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
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidIDFormat)
		return
	}

	var dto service.F5InfoUpdateDTO
	if bindErr := c.ShouldBindJSON(&dto); bindErr != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidRequestBody, bindErr.Error()))
		return
	}

	if err := h.service.UpdateF5Info(c.Request.Context(), id, &dto); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, "f5", id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(MsgFailedToUpdateF5, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, MsgF5UpdateSuccess, nil)
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
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidIDFormat)
		return
	}

	if err := h.service.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, "f5", id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(MsgFailedToDeleteF5, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, MsgF5DeleteSuccess, nil)
}
