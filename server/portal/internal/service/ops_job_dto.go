// Package service provides the business logic for the portal module.
package service

import (
	"time"

	"navy-ng/models/portal"
)

// OpsJobQuery defines the query parameters for listing OpsJobs.
type OpsJobQuery struct {
	Page   int    `form:"page" json:"page" example:"1" swagger:"description=页码"`
	Size   int    `form:"size" json:"size" example:"10" swagger:"description=每页数量"`
	Name   string `form:"name" json:"name" example:"deploy-app" swagger:"description=任务名称"`
	Status string `form:"status" json:"status" example:"running" swagger:"description=任务状态"`
}

// OpsJobCreateDTO defines the data transfer object for creating OpsJob.
type OpsJobCreateDTO struct {
	Name        string `json:"name" binding:"required" example:"deploy-app" swagger:"description=任务名称"`
	Description string `json:"description" example:"部署应用到生产环境" swagger:"description=任务描述"`
}

// OpsJobResponse defines the response structure for a single OpsJob.
type OpsJobResponse struct {
	ID          int64  `json:"id" example:"1" swagger:"description=ID"`
	Name        string `json:"name" example:"deploy-app" swagger:"description=任务名称"`
	Description string `json:"description" example:"部署应用到生产环境" swagger:"description=任务描述"`
	Status      string `json:"status" example:"running" swagger:"description=任务状态"`
	Progress    int    `json:"progress" example:"50" swagger:"description=任务进度"`
	StartTime   string `json:"start_time" example:"2024-01-01T12:00:00Z" swagger:"description=开始时间"`
	EndTime     string `json:"end_time" example:"2024-01-01T12:30:00Z" swagger:"description=结束时间"`
	LogContent  string `json:"log_content,omitempty" example:"Starting deployment..." swagger:"description=日志内容"`
	CreatedAt   string `json:"created_at" example:"2024-01-01T12:00:00Z" swagger:"description=创建时间"`
	UpdatedAt   string `json:"updated_at" example:"2024-01-01T12:30:00Z" swagger:"description=更新时间"`
}

// OpsJobListResponse defines the response structure for a list of OpsJobs.
type OpsJobListResponse struct {
	List  []*OpsJobResponse `json:"list" swagger:"description=运维任务列表"`
	Page  int               `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int               `json:"size" example:"10" swagger:"description=每页数量"`
	Total int64             `json:"total" example:"100" swagger:"description=总记录数"`
}

// OpsJobStatusUpdate defines the structure for WebSocket status updates.
type OpsJobStatusUpdate struct {
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	LogLine  string `json:"log_line,omitempty"`
}

// Internal model for data transformation
type internalOpsJob struct {
	ID          int64
	Name        string
	Description string
	Status      string
	Progress    int
	StartTime   time.Time
	EndTime     time.Time
	LogContent  string
	Deleted     string
}

// ToModel 转换为数据库模型
func (i *internalOpsJob) ToModel() *portal.OpsJob {
	return &portal.OpsJob{
		BaseModel: portal.BaseModel{
			ID: i.ID,
		},
		Name:        i.Name,
		Description: i.Description,
		Status:      i.Status,
		Progress:    i.Progress,
		StartTime:   i.StartTime,
		EndTime:     i.EndTime,
		LogContent:  i.LogContent,
		Deleted:     i.Deleted,
	}
}

// FromModel 从数据库模型转换
func (i *internalOpsJob) FromModel(m *portal.OpsJob) {
	i.ID = m.ID
	i.Name = m.Name
	i.Description = m.Description
	i.Status = m.Status
	i.Progress = m.Progress
	i.StartTime = m.StartTime
	i.EndTime = m.EndTime
	i.LogContent = m.LogContent
	i.Deleted = m.Deleted
}

// ToResponse 转换为响应DTO
func (i *internalOpsJob) ToResponse(m *portal.OpsJob) *OpsJobResponse {
	return &OpsJobResponse{
		ID:          i.ID,
		Name:        i.Name,
		Description: i.Description,
		Status:      i.Status,
		Progress:    i.Progress,
		StartTime:   m.StartTime.Format(time.RFC3339),
		EndTime:     m.EndTime.Format(time.RFC3339),
		LogContent:  i.LogContent,
		CreatedAt:   time.Time(m.CreatedAt).Format(time.RFC3339),
		UpdatedAt:   time.Time(m.UpdatedAt).Format(time.RFC3339),
	}
}

// Helper function to convert model to response
func toOpsJobResponse(model *portal.OpsJob) *OpsJobResponse {
	if model == nil {
		return nil
	}
	internal := &internalOpsJob{}
	internal.FromModel(model)

	// 确保ID被正确设置
	response := &OpsJobResponse{
		ID:          model.ID,
		Name:        internal.Name,
		Description: internal.Description,
		Status:      internal.Status,
		Progress:    internal.Progress,
		StartTime:   model.StartTime.Format(time.RFC3339),
		EndTime:     model.EndTime.Format(time.RFC3339),
		LogContent:  internal.LogContent,
		CreatedAt:   time.Time(model.CreatedAt).Format(time.RFC3339),
		UpdatedAt:   time.Time(model.UpdatedAt).Format(time.RFC3339),
	}

	return response
}
