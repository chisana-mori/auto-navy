package es

// 资源类型常量
const (
	ResourceTypeTotal    = "total"
	ResourceTypeCompute  = "compute"
	ResourceTypeMemory   = "memory"
	ResourceTypeStorage  = "storage"
	ResourceTypeNetwork  = "network"
	ResourceTypeDatabase = "database"
	ResourceTypeGPU      = "gpu"
)

// 策略状态常量
const (
	StrategyStatusEnabled  = "enabled"
	StrategyStatusDisabled = "disabled"
)

// 订单状态常量
const (
	OrderStatusPending                 = "pending"
	OrderStatusProcessing              = "processing"
	OrderStatusCompleted               = "completed"
	OrderStatusFailed                  = "failed"
	OrderStatusCancelled               = "cancelled"
	OrderStatusPendingConfirmation     = "pending_confirmation"
	OrderStatusScheduledForMaintenance = "scheduled_for_maintenance"
	OrderStatusMaintenanceInProgress   = "maintenance_in_progress"
)

// 设备状态常量
const (
	DeviceStatusPending    = "pending"
	DeviceStatusProcessing = "processing"
	DeviceStatusCompleted  = "completed"
	DeviceStatusFailed     = "failed"
	StatusSuccess          = "success"
	StatusExecuting        = "executing"
)

// 阈值类型
const (
	ThresholdTypeUsage     = "usage"
	ThresholdTypeAllocated = "allocated"
)

// 条件逻辑
const (
	ConditionLogicAnd = "AND"
	ConditionLogicOr  = "OR"
)

// 将所有常量和类型定义移到独立文件
const (
	TriggerActionPoolEntry = "pool_entry"
	TriggerActionPoolExit  = "pool_exit"
	// 其他常量...
)

const (
	preloadElasticScalingDetail = "ElasticScalingDetail"
	preloadDevices              = "Devices"
)
