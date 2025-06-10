package portal

// OrderType 订单类型枚举
type OrderType string

const (
	OrderTypeElasticScaling OrderType = "elastic_scaling" // 弹性伸缩
	OrderTypeMaintenance    OrderType = "maintenance"     // 设备维护
	OrderTypeDeployment     OrderType = "deployment"      // 应用部署
	OrderTypeGeneral        OrderType = "general"         // 通用订单
	// 可扩展更多类型...
)

// OrderStatus 订单状态枚举
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "pending"          // 待处理
	OrderStatusProcessing      OrderStatus = "processing"       // 处理中
	OrderStatusReturning       OrderStatus = "returning"        // 归还中（退池订单专用）
	OrderStatusReturnCompleted OrderStatus = "return_completed" // 归还完成（退池订单专用）
	OrderStatusNoReturn        OrderStatus = "no_return"        // 无需归还（退池订单专用）
	OrderStatusCompleted       OrderStatus = "completed"        // 已完成
	OrderStatusFailed          OrderStatus = "failed"           // 失败
	OrderStatusCancelled       OrderStatus = "cancelled"        // 已取消
	OrderStatusIgnored         OrderStatus = "ignored"          // 已忽略
	// 可扩展更多状态...
)

// Order 基础订单模型
type Order struct {
	BaseModel
	OrderNumber    string      `gorm:"column:order_number;type:varchar(50);unique"` // 唯一订单号
	Name           string      `gorm:"column:name;type:varchar(255)"`               // 订单名称
	Description    string      `gorm:"column:description;type:text"`                // 订单描述
	Type           OrderType   `gorm:"column:type;type:varchar(50)"`                // 订单类型
	Status         OrderStatus `gorm:"column:status;type:varchar(50)"`              // 订单状态
	Executor       string      `gorm:"column:executor;type:varchar(100)"`           // 执行人
	ExecutionTime  *NavyTime   `gorm:"column:execution_time;type:datetime"`         // 执行时间
	CreatedBy      string      `gorm:"column:created_by;type:varchar(100)"`         // 创建人
	CompletionTime *NavyTime   `gorm:"column:completion_time;type:datetime"`        // 完成时间
	FailureReason  string      `gorm:"column:failure_reason;type:text"`             // 失败原因

	// 关联关系
	ElasticScalingDetail *ElasticScalingOrderDetail `gorm:"foreignKey:OrderID"` // 弹性伸缩详情
	MaintenanceDetail    *MaintenanceOrderDetail    `gorm:"foreignKey:OrderID"` // 设备维护详情

}

// TableName 指定表名
func (Order) TableName() string {
	return "ng_orders"
}

// ElasticScalingOrderDetail 弹性伸缩订单详情模型
type ElasticScalingOrderDetail struct {
	BaseModel
	OrderID                int  `gorm:"column:order_id;type:bigint;unique"`                // 关联订单ID（外键）
	ClusterID              int  `gorm:"column:cluster_id;type:bigint"`                     // 关联集群ID
	StrategyID             *int `gorm:"column:strategy_id;type:bigint"`                    // 关联策略ID（可为NULL）
	ActionType             string `gorm:"column:action_type;type:varchar(50)"`               // 订单操作类型（入池/退池）
	ResourcePoolType       string `gorm:"column:resource_pool_type;type:varchar(50)"`        // 资源池类型
	DeviceCount            int    `gorm:"column:device_count;type:int"`                      // 请求的设备数量
	StrategyTriggeredValue string `gorm:"column:strategy_triggered_value;type:varchar(255)"` // 策略触发时的具体指标值
	StrategyThresholdValue string `gorm:"column:strategy_threshold_value;type:varchar(255)"` // 策略触发时的阈值设定

	// 关联关系
	Order *Order `gorm:"foreignKey:OrderID"` // 关联的基础订单
	// The Devices field is removed to prevent GORM from creating an implicit join table.
	// The relationship is now explicitly managed through the OrderDevice model.
}

// TableName 指定表名
func (ElasticScalingOrderDetail) TableName() string {
	return "ng_elastic_scaling_order_details"
}
