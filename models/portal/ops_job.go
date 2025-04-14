package portal

import (
	"time"

	"gorm.io/gorm"
)

// OpsJob 运维任务信息.
type OpsJob struct {
	BaseModel
	Name        string    `gorm:"column:name;type:varchar(255);not null" json:"name"`              // 任务名称
	Description string    `gorm:"column:description;type:text" json:"description"`                 // 任务描述
	Status      string    `gorm:"column:status;type:varchar(50);not null" json:"status"`
	// 任务状态：pending, running, completed, failed
	Progress    int       `gorm:"column:progress;type:int;default:0" json:"progress"`              // 任务进度（0-100）
	StartTime   time.Time `gorm:"column:start_time;type:datetime" json:"start_time"`               // 开始时间
	EndTime     time.Time `gorm:"column:end_time;type:datetime" json:"end_time"`                   // 结束时间
	LogContent  string    `gorm:"column:log_content;type:text" json:"log_content"`                 // 日志内容

	Deleted     string    `gorm:"column:deleted;type:varchar(255)" json:"deleted,omitempty"`       // 软删除标记
}

// TableName 指定表名.
func (OpsJob) TableName() string {
	return "ops_job"
}

// BeforeCreate 钩子函数，确保ID为0时使用数据库自增.
func (j *OpsJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == 0 {
		j.ID = 0 // 确保ID为0，让数据库自增
	}
	return nil
}
