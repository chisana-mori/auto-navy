package portal

// OrderType 订单类型枚举
type OrderType string

const (
	OrderTypeElasticScaling OrderType = "elastic_scaling" // 弹性伸缩
	OrderTypeMaintenance    OrderType = "maintenance"     // 设备维护
	OrderTypeDeployment     OrderType = "deployment"      // 应用部署
	// 可扩展更多类型...
)

// OrderStatus 订单状态枚举
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"    // 待处理
	OrderStatusProcessing OrderStatus = "processing" // 处理中
	OrderStatusCompleted  OrderStatus = "completed"  // 已完成
	OrderStatusFailed     OrderStatus = "failed"     // 失败
	OrderStatusCancelled  OrderStatus = "cancelled"  // 已取消
	OrderStatusIgnored    OrderStatus = "ignored"    // 已忽略
	// 可扩展更多状态...
)

// Order 基础订单模型
type Order struct {
	BaseModel
	OrderNumber    string      `gorm:"column:order_number;type:varchar(50);unique" json:"orderNumber"` // 唯一订单号
	Name           string      `gorm:"column:name;type:varchar(255)" json:"name"`                      // 订单名称
	Description    string      `gorm:"column:description;type:text" json:"description"`                // 订单描述
	Type           OrderType   `gorm:"column:type;type:varchar(50)" json:"type"`                       // 订单类型
	Status         OrderStatus `gorm:"column:status;type:varchar(50)" json:"status"`                   // 订单状态
	Executor       string      `gorm:"column:executor;type:varchar(100)" json:"executor"`              // 执行人
	ExecutionTime  *NavyTime   `gorm:"column:execution_time;type:datetime" json:"executionTime"`       // 执行时间
	CreatedBy      string      `gorm:"column:created_by;type:varchar(100)" json:"createdBy"`           // 创建人
	CompletionTime *NavyTime   `gorm:"column:completion_time;type:datetime" json:"completionTime"`     // 完成时间
	FailureReason  string      `gorm:"column:failure_reason;type:text" json:"failureReason"`           // 失败原因

	// 关联关系
	ElasticScalingDetail *ElasticScalingOrderDetail `gorm:"foreignKey:OrderID" json:"elasticScalingDetail,omitempty"` // 弹性伸缩详情

}

// TableName 指定表名
func (Order) TableName() string {
	return "orders"
}

// ElasticScalingOrderDetail 弹性伸缩订单详情模型
type ElasticScalingOrderDetail struct {
	BaseModel
	OrderID                int64     `gorm:"column:order_id;type:bigint;unique" json:"orderId"`                               // 关联订单ID（外键）
	ClusterID              int64     `gorm:"column:cluster_id;type:bigint" json:"clusterId"`                                  // 关联集群ID
	StrategyID             *int64    `gorm:"column:strategy_id;type:bigint" json:"strategyId"`                                // 关联策略ID（可为NULL）
	ActionType             string    `gorm:"column:action_type;type:varchar(50)" json:"actionType"`                           // 订单操作类型（入池/退池）
	DeviceCount            int       `gorm:"column:device_count;type:int" json:"deviceCount"`                                 // 请求的设备数量
	MaintenanceStartTime   *NavyTime `gorm:"column:maintenance_start_time;type:datetime" json:"maintenanceStartTime"`         // 维护开始时间
	MaintenanceEndTime     *NavyTime `gorm:"column:maintenance_end_time;type:datetime" json:"maintenanceEndTime"`             // 维护结束时间
	ExternalTicketID       string    `gorm:"column:external_ticket_id;type:varchar(100)" json:"externalTicketId"`             // 外部工单号
	StrategyTriggeredValue string    `gorm:"column:strategy_triggered_value;type:varchar(255)" json:"strategyTriggeredValue"` // 策略触发时的具体指标值
	StrategyThresholdValue string    `gorm:"column:strategy_threshold_value;type:varchar(255)" json:"strategyThresholdValue"` // 策略触发时的阈值设定

	// 关联关系
	Order   *Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`        // 关联的基础订单
	Devices []Device `gorm:"many2many:order_device;" json:"devices,omitempty"` // 关联的设备列表
}

// TableName 指定表名
func (ElasticScalingOrderDetail) TableName() string {
	return "elastic_scaling_order_details"
}
