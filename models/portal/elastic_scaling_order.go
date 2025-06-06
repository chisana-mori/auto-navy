package portal

// ElasticScalingOrder 弹性伸缩订单表（已废弃）
//
// 废弃说明：此模型已被新的通用订单系统替代
// - 新代码应使用 Order + ElasticScalingOrderDetail 模型
// - 此模型仅保留用于数据迁移和向后兼容
// - 请勿在新功能中使用此模型
//
// Deprecated: 使用 Order + ElasticScalingOrderDetail 替代
type ElasticScalingOrder struct {
	BaseModel
	OrderNumber            string    `gorm:"column:order_number;type:varchar(50);unique" json:"orderNumber"`                  // 唯一订单号
	Name                   string    `gorm:"column:name;type:varchar(255)" json:"name"`                                       // 订单名称
	Description            string    `gorm:"column:description;type:text" json:"description"`                                 // 订单描述
	ClusterID              int64     `gorm:"column:cluster_id;type:bigint" json:"clusterId"`                                  // 关联集群ID
	StrategyID             *int64    `gorm:"column:strategy_id;type:bigint" json:"strategyId"`                                // 关联策略ID(手动订单可为NULL)
	ActionType             string    `gorm:"column:action_type;type:varchar(50)" json:"actionType"`                           // 订单操作类型
	Status                 string    `gorm:"column:status;type:varchar(50)" json:"status"`                                    // 订单状态
	DeviceCount            int       `gorm:"column:device_count;type:int" json:"deviceCount"`                                 // 请求的设备数量
	Executor               string    `gorm:"column:executor;type:varchar(100)" json:"executor"`                               // 执行人
	ExecutionTime          *NavyTime `gorm:"column:execution_time;type:datetime" json:"executionTime"`                        // 执行时间
	CreatedBy              string    `gorm:"column:created_by;type:varchar(100)" json:"createdBy"`                            //
	CompletionTime         *NavyTime `gorm:"column:completion_time;type:datetime" json:"completionTime"`                      // 完成时间
	FailureReason          string    `gorm:"column:failure_reason;type:text" json:"failureReason"`                            // 失败原因
	MaintenanceStartTime   *NavyTime `gorm:"column:maintenance_start_time;type:datetime" json:"maintenanceStartTime"`         // 维护开始时间
	MaintenanceEndTime     *NavyTime `gorm:"column:maintenance_end_time;type:datetime" json:"maintenanceEndTime"`             // 维护结束时间
	ExternalTicketID       string    `gorm:"column:external_ticket_id;type:varchar(100)" json:"externalTicketId"`             // 外部工单号
	StrategyTriggeredValue string    `gorm:"column:strategy_triggered_value;type:varchar(255)" json:"strategyTriggeredValue"` // 策略触发时的具体指标值 (用于延迟记录历史)
	StrategyThresholdValue string    `gorm:"column:strategy_threshold_value;type:varchar(255)" json:"strategyThresholdValue"` // 策略触发时的阈值设定 (用于延迟记录历史)

	// 关联关系 - 通过OrderDevice表建立与设备的多对多关系
	Devices []Device `gorm:"many2many:order_device;" json:"devices,omitempty"` // 关联的设备列表
}

// TableName 指定表名
func (ElasticScalingOrder) TableName() string {
	return "elastic_scaling_order"
}

// OrderDevice 订单设备关联表
type OrderDevice struct {
	BaseModel
	OrderID  int64  `gorm:"column:order_id;type:bigint" json:"orderId"`   // 订单ID
	DeviceID int64  `gorm:"column:device_id;type:bigint" json:"deviceId"` // 设备ID
	Status   string `gorm:"column:status;type:varchar(50)" json:"status"` // 处理状态
}

// TableName 指定表名
func (OrderDevice) TableName() string {
	return "order_device"
}

// StrategyExecutionHistory 策略执行历史表
type StrategyExecutionHistory struct {
	BaseModel
	StrategyID     int64    `gorm:"column:strategy_id;type:bigint" json:"strategyId"`               // 策略ID
	ExecutionTime  NavyTime `gorm:"column:execution_time;type:datetime" json:"executionTime"`       // 执行时间
	TriggeredValue string   `gorm:"column:triggered_value;type:varchar(255)" json:"triggeredValue"` // 触发策略时的具体指标值
	ThresholdValue string   `gorm:"column:threshold_value;type:varchar(255)" json:"thresholdValue"` // 触发策略时的阈值设定
	Result         string   `gorm:"column:result;type:varchar(50)" json:"result"`                   // 执行结果
	OrderID        *int64   `gorm:"column:order_id;type:bigint" json:"orderId"`                     // 关联订单ID
	Reason         string   `gorm:"column:reason;type:text" json:"reason"`                          // 执行结果的原因
}

// TableName 指定表名
func (StrategyExecutionHistory) TableName() string {
	return "strategy_execution_history"
}

// NotificationLog 通知日志表
type NotificationLog struct {
	BaseModel
	OrderID          *int64   `gorm:"column:order_id;type:bigint" json:"orderId"`                        // 关联订单ID(可选)
	StrategyID       *int64   `gorm:"column:strategy_id;type:bigint" json:"strategyId"`                  // 关联策略ID(可选)
	NotificationType string   `gorm:"column:notification_type;type:varchar(50)" json:"notificationType"` // 通知类型
	Recipient        string   `gorm:"column:recipient;type:varchar(255)" json:"recipient"`               // 接收人信息
	Content          string   `gorm:"column:content;type:text" json:"content"`                           // 通知内容
	Status           string   `gorm:"column:status;type:varchar(50)" json:"status"`                      // 发送状态
	SendTime         NavyTime `gorm:"column:send_time;type:datetime" json:"sendTime"`                    // 发送时间
	ErrorMessage     string   `gorm:"column:error_message;type:text" json:"errorMessage"`                // 错误信息
}

// TableName 指定表名
func (NotificationLog) TableName() string {
	return "notification_log"
}

// DutyRoster 值班表
type DutyRoster struct {
	BaseModel
	UserID      string `gorm:"column:user_id;type:varchar(100)" json:"userId"`           // 用户ID/名称
	DutyDate    string `gorm:"column:duty_date;type:date" json:"dutyDate"`               // 值班日期
	StartTime   string `gorm:"column:start_time;type:time" json:"startTime"`             // 开始时间
	EndTime     string `gorm:"column:end_time;type:time" json:"endTime"`                 // 结束时间
	ContactInfo string `gorm:"column:contact_info;type:varchar(255)" json:"contactInfo"` // 联系方式
}

// TableName 指定表名
func (DutyRoster) TableName() string {
	return "duty_roster"
}
