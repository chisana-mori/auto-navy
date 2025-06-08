package portal

// MaintenanceOrderDetail 设备维护订单详情模型
type MaintenanceOrderDetail struct {
	BaseModel
	OrderID              int64     `gorm:"column:order_id;type:bigint;unique"`          // 关联订单ID（外键）
	ClusterID            int64     `gorm:"column:cluster_id;type:bigint"`               // 关联集群ID
	MaintenanceStartTime *NavyTime `gorm:"column:maintenance_start_time;type:datetime"` // 维护开始时间
	MaintenanceEndTime   *NavyTime `gorm:"column:maintenance_end_time;type:datetime"`   // 维护结束时间
	ExternalTicketID     string    `gorm:"column:external_ticket_id;type:varchar(100)"` // 外部工单号
	MaintenanceType      string    `gorm:"column:maintenance_type;type:varchar(50)"`    // 维护类型（cordon/uncordon/general）
	Priority             string    `gorm:"column:priority;type:varchar(20)"`            // 优先级（high/medium/low）
	Reason               string    `gorm:"column:reason;type:text"`                     // 维护原因
	Comments             string    `gorm:"column:comments;type:text"`                   // 附加说明

	// 关联关系
	Order *Order `gorm:"foreignKey:OrderID"` // 关联的基础订单
}

// TableName 指定表名
func (MaintenanceOrderDetail) TableName() string {
	return "maintenance_order_details"
}

// MaintenanceOrderStatus 维护订单状态枚举
type MaintenanceOrderStatus string

const (
	MaintenanceStatusPendingConfirmation MaintenanceOrderStatus = "pending_confirmation"      // 待确认
	MaintenanceStatusScheduled           MaintenanceOrderStatus = "scheduled_for_maintenance" // 已安排维护
	MaintenanceStatusInProgress          MaintenanceOrderStatus = "maintenance_in_progress"   // 维护中
	MaintenanceStatusCompleted           MaintenanceOrderStatus = "completed"                 // 已完成
	MaintenanceStatusCancelled           MaintenanceOrderStatus = "cancelled"                 // 已取消
	MaintenanceStatusFailed              MaintenanceOrderStatus = "failed"                    // 失败
)

// MaintenanceType 维护类型枚举
type MaintenanceType string

const (
	MaintenanceTypeCordon   MaintenanceType = "cordon"   // 节点封锁
	MaintenanceTypeUncordon MaintenanceType = "uncordon" // 节点解封
	MaintenanceTypeGeneral  MaintenanceType = "general"  // 一般维护
)
