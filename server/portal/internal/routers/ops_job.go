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

// @Summary 获取运维任务详情
// @Description 根据任务ID获取运维任务的详细信息
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param id path int true "任务ID" example:"1"
// @Success 200 {object} service.OpsJobResponse "成功获取任务详情"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 404 {object} service.ErrorResponse "任务不存在"
// @Failure 500 {object} service.ErrorResponse "获取任务详情失败"
// @Router /ops/job/{id} [get]
// getOpsJob handles GET /ops/job/:id requests.
func (h *OpsJobHandler) getOpsJob(c *gin.Context) {
	idStr := c.Param(routeParamID)
	id, err := strconv.ParseInt(idStr, base10, bitSize64)
	if err != nil {
		render.BadRequest(c, msgInvalidJobIDFormat)
		return
	}

	job, err := h.service.GetOpsJob(c.Request.Context(), id)
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

// @Summary 获取运维任务列表
// @Description 获取运维任务列表，支持分页和条件筛选
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param page query int false "页码" example:"1"
// @Param size query int false "每页数量" example:"10"
// @Param name query string false "任务名称" example:"deploy-app"
// @Param status query string false "任务状态" example:"running"
// @Success 200 {object} service.OpsJobListResponse "成功获取任务列表"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "获取任务列表失败"
// @Router /ops/job [get]
// listOpsJobs handles GET /ops/job requests.
func (h *OpsJobHandler) listOpsJobs(c *gin.Context) {
	var query service.OpsJobQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidJobQueryParams, err.Error()))
		return
	}

	response, err := h.service.ListOpsJobs(c.Request.Context(), &query)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToListJobs, err.Error()))
		return
	}

	render.Success(c, response)
}

// @Summary 创建运维任务
// @Description 创建新的运维任务，并返回创建的任务信息
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param data body service.OpsJobCreateDTO true "任务信息"
// @Success 200 {object} service.OpsJobResponse "任务创建成功"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "创建任务失败"
// @Router /ops/job [post]
// createOpsJob handles POST /ops/job requests.
func (h *OpsJobHandler) createOpsJob(c *gin.Context) {
	var dto service.OpsJobCreateDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		render.BadRequest(c, fmt.Sprintf(msgInvalidJobBody, err.Error()))
		return
	}

	job, err := h.service.CreateOpsJob(c.Request.Context(), &dto)
	if err != nil {
		render.InternalServerError(c, fmt.Sprintf(msgFailedToCreateJob, err.Error()))
		return
	}

	render.SuccessWithMessage(c, msgJobCreatedSuccess, job)
}

// @Summary 运维任务WebSocket连接
// @Description 建立WebSocket连接以实时获取运维任务的状态更新和日志
// @Tags 运维任务
// @Accept json
// @Produce json
// @Param id path int true "任务ID" example:"1"
// @Success 101 {string} string "升级为WebSocket协议"
// @Failure 400 {object} service.ErrorResponse "参数错误"
// @Failure 500 {object} service.ErrorResponse "WebSocket连接失败"
// @Router /ops/job/{id}/ws [get]
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
