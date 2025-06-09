package portal

// OrderDevice 订单设备关联表
type OrderDevice struct {
	BaseModel
	OrderID  int64  `gorm:"primaryKey;column:order_id"`
	DeviceID int64  `gorm:"primaryKey;column:device_id"`
	Status   string `gorm:"type:varchar(50);default:'pending'"`
}

// TableName 指定表名
func (OrderDevice) TableName() string {
	return "ng_order_device"
}

// StrategyExecutionHistory 策略执行历史表
type StrategyExecutionHistory struct {
	BaseModel
	StrategyID     int64    `gorm:"column:strategy_id;type:bigint"`           // 策略ID
	ClusterID      int64    `gorm:"column:cluster_id;type:bigint"`            // 集群ID
	ResourceType   string   `gorm:"column:resource_type;type:varchar(100)"`   // 资源池名称
	ExecutionTime  NavyTime `gorm:"column:execution_time;type:datetime"`      // 执行时间
	TriggeredValue string   `gorm:"column:triggered_value;type:varchar(255)"` // 触发策略时的具体指标值
	ThresholdValue string   `gorm:"column:threshold_value;type:varchar(255)"` // 触发策略时的阈值设定
	Result         string   `gorm:"column:result;type:varchar(50)"`           // 执行结果
	OrderID        *int64   `gorm:"column:order_id;type:bigint"`              // 关联订单ID
	Reason         string   `gorm:"column:reason;type:text"`                  // 执行结果的原因
}

// TableName 指定表名
func (StrategyExecutionHistory) TableName() string {
	return "ng_strategy_execution_history"
}

// NotificationLog 通知日志表
type NotificationLog struct {
	BaseModel
	OrderID          *int64   `gorm:"column:order_id;type:bigint"`               // 关联订单ID(可选)
	StrategyID       *int64   `gorm:"column:strategy_id;type:bigint"`            // 关联策略ID(可选)
	NotificationType string   `gorm:"column:notification_type;type:varchar(50)"` // 通知类型
	Recipient        string   `gorm:"column:recipient;type:varchar(255)"`        // 接收人信息
	Content          string   `gorm:"column:content;type:text"`                  // 通知内容
	Status           string   `gorm:"column:status;type:varchar(50)"`            // 发送状态
	SendTime         NavyTime `gorm:"column:send_time;type:datetime"`            // 发送时间
	ErrorMessage     string   `gorm:"column:error_message;type:text"`            // 错误信息
}

// TableName 指定表名
func (NotificationLog) TableName() string {
	return "notification_log"
}

// DutyRoster 值班表
type DutyRoster struct {
	BaseModel
	UserID      string `gorm:"column:user_id;type:varchar(100)"`      // 用户ID/名称
	DutyDate    string `gorm:"column:duty_date;type:date"`            // 值班日期
	StartTime   string `gorm:"column:start_time;type:time"`           // 开始时间
	EndTime     string `gorm:"column:end_time;type:time"`             // 结束时间
	ContactInfo string `gorm:"column:contact_info;type:varchar(255)"` // 联系方式
}

// TableName 指定表名
func (DutyRoster) TableName() string {
	return "duty_roster"
}
