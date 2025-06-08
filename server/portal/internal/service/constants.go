package service

// 通用常量
const (
	// 空字符串常量
	EmptyString = ""

	// 分页相关常量
	DefaultPage = 1
	DefaultSize = 10
	MaxSize     = 100

	// 状态相关常量
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// GORM 查询相关常量
const (
	OrderByCreatedAtDesc = "created_at DESC"
)

// 资源类型
const (
	ResourceOpsJob = "Operation job"
	ResourceF5     = "F5 info"
	ResourceDevice = "Device"
)
