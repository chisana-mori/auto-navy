// Package routers defines the HTTP routes for the portal module.
package routers

import (
	"fmt"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Constants for HTTP status codes and default values
const (
	StatusBadRequest          = http.StatusBadRequest
	StatusInternalServerError = http.StatusInternalServerError
	StatusOK                    = http.StatusOK
	StatusNotFound              = http.StatusNotFound
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
	f5Service *service.F5InfoService
}

// NewF5InfoHandler creates a new F5InfoHandler.
func NewF5InfoHandler(f5Service *service.F5InfoService) *F5InfoHandler {
	return &F5InfoHandler{f5Service: f5Service}
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
		c.JSON(StatusBadRequest, service.ErrorResponse{Error: msgInvalidIDFormat})
		return
	}

	f5Info, err := h.f5Service.GetF5Info(c.Request.Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			c.JSON(StatusNotFound, service.ErrorResponse{Error: err.Error()})
		} else {
			c.JSON(StatusInternalServerError, service.ErrorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(StatusOK, f5Info)
}

// listF5Infos handles GET /f5 requests.
func (h *F5InfoHandler) listF5Infos(c *gin.Context) {
	var query service.F5InfoQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(StatusBadRequest, service.ErrorResponse{Error: fmt.Sprintf(msgInvalidQueryParams, err.Error())})
		return
	}

	response, err := h.f5Service.ListF5Infos(c.Request.Context(), &query)
	if err != nil {
		c.JSON(StatusInternalServerError, service.ErrorResponse{Error: fmt.Sprintf(msgFailedToList, err.Error())})
		return
	}

	c.JSON(StatusOK, response)
}

// updateF5Info handles PUT /f5/:id requests.
func (h *F5InfoHandler) updateF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		c.JSON(StatusBadRequest, service.ErrorResponse{Error: msgInvalidIDFormat})
		return
	}

	var dto service.F5InfoUpdateDTO
	if bindErr := c.ShouldBindJSON(&dto); bindErr != nil {
		c.JSON(StatusBadRequest, service.ErrorResponse{Error: fmt.Sprintf(msgInvalidRequestBody, bindErr.Error())})
		return
	}

	if err := h.f5Service.UpdateF5Info(c.Request.Context(), id, &dto); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			c.JSON(StatusNotFound, service.ErrorResponse{Error: err.Error()})
		} else {
			c.JSON(StatusInternalServerError, service.ErrorResponse{Error: fmt.Sprintf(msgFailedToUpdate, err.Error())})
		}
		return
	}

	c.JSON(StatusOK, service.SuccessResponse{Message: msgSuccessUpdate})
}

// deleteF5Info handles DELETE /f5/:id requests.
func (h *F5InfoHandler) deleteF5Info(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		c.JSON(StatusBadRequest, service.ErrorResponse{Error: msgInvalidIDFormat})
		return
	}

	if err := h.f5Service.DeleteF5Info(c.Request.Context(), id); err != nil {
		if err.Error() == fmt.Sprintf(service.ErrRecordNotFoundMsg, id) {
			c.JSON(StatusNotFound, service.ErrorResponse{Error: err.Error()})
		} else {
			c.JSON(StatusInternalServerError, service.ErrorResponse{Error: fmt.Sprintf(msgFailedToDelete, err.Error())})
		}
		return
	}

	c.JSON(StatusOK, service.SuccessResponse{Message: msgSuccessDelete})
}
