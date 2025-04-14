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
	base10                    = 10
	bitSize64                 = 64
	routeParamID              = "id"           // Route parameter for ID
	msgInvalidIDFormat        = "invalid id format"
	msgInvalidQueryParams     = "invalid query parameters: %s"
	msgInvalidRequestBody     = "invalid request body: %s"
	msgFailedToList           = "failed to list F5 infos: %s"
	msgFailedToUpdate         = "failed to update F5 info: %s"
	msgFailedToDelete         = "failed to delete F5 info: %s"
	msgSuccessUpdate          = "F5 info updated successfully"
	msgSuccessDelete          = "F5 info deleted successfully"
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
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, err.Error())
		}
		return
	}

	render.Success(c, f5Info)
}

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
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToUpdate, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, msgSuccessUpdate, nil)
}

// deleteF5Info handles DELETE /f5/:id requests.
func (h *F5InfoHandler) deleteF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidIDFormat)
		return
	}

	if err := h.service.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, fmt.Sprintf(msgFailedToDelete, err.Error()))
		}
		return
	}

	render.SuccessWithMessage(c, msgSuccessDelete, nil)
}
