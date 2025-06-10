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
	"gorm.io/gorm" // Added import for gorm
)

// Constants for OpsJobHandler - moved to constants.go

// OpsJobHandler handles HTTP requests related to OpsJob.
type OpsJobHandler struct {
	service  *service.OpsJobService // Renamed field for consistency
	upgrader websocket.Upgrader
}

// NewOpsJobHandler creates a new OpsJobHandler, instantiating the service internally.
func NewOpsJobHandler(db *gorm.DB) *OpsJobHandler {
	opsService := service.NewOpsJobService(db) // Instantiate service here
	return &OpsJobHandler{
		service: opsService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// TODO: Implement proper origin checking for production
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

// getOpsJob 获取运维任务详情
// @Summary 获取运维任务详情
// @Description 根据任务ID获取运维任务的详细信息
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 404 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/ops-jobs/{id} [get]
func (h *OpsJobHandler) getOpsJob(c *gin.Context) {
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidJobIDFormat)
		return
	}

	job, err := h.service.GetOpsJob(c.Request.Context(), int(id))
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

// listOpsJobs 获取运维任务列表
// @Summary 获取运维任务列表
// @Description 获取运维任务列表，支持分页
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param size query int false "每页大小，默认10"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/ops-jobs [get]
func (h *OpsJobHandler) listOpsJobs(c *gin.Context) {
	var query service.OpsJobQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidJobQueryParams, err.Error()))
		return
	}

	response, err := h.service.ListOpsJobs(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToListJobs, err.Error()))
		return
	}

	render.Success(c, response)
}

// createOpsJob 创建运维任务
// @Summary 创建运维任务
// @Description 创建新的运维任务
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param job body service.OpsJobCreateDTO true "运维任务数据"
// @Success 200 {object} render.Response
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/ops-jobs [post]
func (h *OpsJobHandler) createOpsJob(c *gin.Context) {
	var dto service.OpsJobCreateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, fmt.Sprintf(MsgInvalidJobBody, err.Error()))
		return
	}

	job, err := h.service.CreateOpsJob(c.Request.Context(), &dto)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgFailedToCreateJob, err.Error()))
		return
	}

	render.SuccessWithMessage(c, MsgJobCreatedSuccess, job)
}

// handleWebSocket 处理WebSocket连接
// @Summary 处理WebSocket连接
// @Description 建立WebSocket连接以实时获取运维任务执行状态
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} render.ErrorResponse
// @Failure 500 {object} render.ErrorResponse
// @Router /fe-v1/ops-jobs/{id}/ws [get]
func (h *OpsJobHandler) handleWebSocket(c *gin.Context) {
	idStr := c.Param(ParamID)
	id, err := strconv.ParseInt(idStr, Base10, BitSize64)
	if err != nil {
		render.BadRequest(c, MsgInvalidJobIDFormat)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(MsgWebSocketUpgradeError, err.Error()))
		return
	}

	// Handle WebSocket connection
	go h.handleConnection(conn, int(id))
}

// handleConnection manages the WebSocket connection for a specific job
func (h *OpsJobHandler) handleConnection(conn *websocket.Conn, jobID int) {
	defer conn.Close()

	// Register client for job updates
	err := h.service.RegisterClient(jobID, conn)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	// Handle incoming messages (like start job command)
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred
			h.service.UnregisterClient(jobID, conn)
			break
		}

		// Handle different message types
		if messageType == websocket.TextMessage {
			message := string(p)
			if message == "start" {
				// Start job execution
				if err := h.service.StartJob(jobID, conn); err != nil {
					conn.WriteJSON(map[string]string{"error": err.Error()})
				}
			}
		}
	}
}
