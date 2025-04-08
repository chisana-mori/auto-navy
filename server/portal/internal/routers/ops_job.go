// Package routers defines the HTTP routes for the portal module.
package routers

import (
	"fmt"
	"navy-ng/pkg/middleware/render"
	"navy-ng/server/portal/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Constants for OpsJobHandler
const (
	msgInvalidJobIDFormat    = "invalid job id format"
	msgInvalidJobQueryParams = "invalid query parameters: %s"
	msgInvalidJobBody        = "invalid request body: %s"
	msgFailedToListJobs      = "failed to list operation jobs: %s"
	msgFailedToCreateJob     = "failed to create operation job: %s"
	msgJobCreatedSuccess     = "Operation job created successfully"
	msgWebSocketUpgradeError = "failed to upgrade to websocket: %s"
)

// OpsJobHandler handles HTTP requests related to OpsJob.
type OpsJobHandler struct {
	opsService *service.OpsJobService
	upgrader   websocket.Upgrader
}

// NewOpsJobHandler creates a new OpsJobHandler.
func NewOpsJobHandler(opsService *service.OpsJobService) *OpsJobHandler {
	return &OpsJobHandler{
		opsService: opsService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				return true
			},
		},
	}
}

// RegisterRoutes registers OpsJob routes with the given router group.
func (h *OpsJobHandler) RegisterRoutes(r *gin.RouterGroup) {
	const idPath = "/:id" // Define path segment constant
	opsGroup := r.Group("/ops")
	{
		jobGroup := opsGroup.Group("/job")
		{
			jobGroup.GET(idPath, h.getOpsJob)
			jobGroup.GET("", h.listOpsJobs)
			jobGroup.POST("", h.createOpsJob)
			jobGroup.GET(idPath+"/ws", h.handleWebSocket)
		}
	}
}

// getOpsJob handles GET /ops/job/:id requests.
func (h *OpsJobHandler) getOpsJob(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidJobIDFormat)
		return
	}

	job, err := h.opsService.GetOpsJob(c.Request.Context(), id)
	if err != nil {
		if err.Error() == fmt.Sprintf(service.ErrOpsJobNotFoundMsg, id) {
			render.NotFound(c, err.Error())
		} else {
			render.InternalServerError(c, err.Error())
		}
		return
	}

	render.Success(c, job)
}

// listOpsJobs handles GET /ops/job requests.
func (h *OpsJobHandler) listOpsJobs(c *gin.Context) {
	var query service.OpsJobQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidJobQueryParams, err.Error()))
		return
	}

	response, err := h.opsService.ListOpsJobs(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToListJobs, err.Error()))
		return
	}

	render.Success(c, response)
}

// createOpsJob handles POST /ops/job requests.
func (h *OpsJobHandler) createOpsJob(c *gin.Context) {
	var dto service.OpsJobCreateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidJobBody, err.Error()))
		return
	}

	job, err := h.opsService.CreateOpsJob(c.Request.Context(), &dto)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToCreateJob, err.Error()))
		return
	}

	render.SuccessWithMessage(c, msgJobCreatedSuccess, job)
}

// handleWebSocket handles WebSocket connections for job status updates.
func (h *OpsJobHandler) handleWebSocket(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidJobIDFormat)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgWebSocketUpgradeError, err.Error()))
		return
	}

	// Handle WebSocket connection
	go h.handleConnection(conn, id)
}

// handleConnection manages the WebSocket connection for a specific job
func (h *OpsJobHandler) handleConnection(conn *websocket.Conn, jobID int64) {
	defer conn.Close()

	// Register client for job updates
	err := h.opsService.RegisterClient(jobID, conn)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	// Handle incoming messages (like start job command)
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred
			h.opsService.UnregisterClient(jobID, conn)
			break
		}

		// Handle different message types
		if messageType == websocket.TextMessage {
			message := string(p)
			if message == "start" {
				// Start job execution
				if err := h.opsService.StartJob(jobID, conn); err != nil {
					conn.WriteJSON(map[string]string{"error": err.Error()})
				}
			}
		}
	}
}
