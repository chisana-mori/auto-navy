package portal

const (
	StrategyStatusEnabled  = "enabled"
	StrategyStatusDisabled = "disabled"
)

// ElasticScalingStrategy 弹性伸缩策略
type ElasticScalingStrategy struct {
	BaseModel
	Name                   string  `gorm:"column:name;size:128;not null"`
	Description            string  `gorm:"column:description;size:500"`
	ThresholdTriggerAction string  `gorm:"column:threshold_trigger_action;size:20;not null"` // pool_entry 或 pool_exit
	CPUThresholdValue      float64 `gorm:"column:cpu_threshold_value;default:0"`             // CPU使用率阈值
	CPUThresholdType       string  `gorm:"column:cpu_threshold_type;size:20"`                // usage 或 allocated
	CPUTargetValue         float64 `gorm:"column:cpu_target_value;default:0"`                // 动作执行后CPU目标使用率
	MemoryThresholdValue   float64 `gorm:"column:memory_threshold_value;default:0"`          // 内存使用率阈值
	MemoryThresholdType    string  `gorm:"column:memory_threshold_type;size:20"`             // usage 或 allocated
	MemoryTargetValue      float64 `gorm:"column:memory_target_value;default:0"`             // 动作执行后内存目标使用率
	ConditionLogic         string  `gorm:"column:condition_logic;size:10;default:'OR'"`      // AND 或 OR
	DurationMinutes        int     `gorm:"column:duration_minutes;not null"`                 // 持续时间（分钟）
	CooldownMinutes        int     `gorm:"column:cooldown_minutes;not null"`                 // 冷却时间（分钟）
	ResourceTypes          string  `gorm:"column:resource_types;size:255"`                   // 资源类型列表，逗号分隔（计算、存储、网络等）
	Status                 string  `gorm:"column:status;size:20;not null"`                   // enabled 或 disabled
	CreatedBy              string  `gorm:"column:created_by;size:50;not null"`
}

// TableName 指定表名
func (ElasticScalingStrategy) TableName() string {
	return "ng_elastic_scaling_strategy"
}

// StrategyClusterAssociation 策略集群关联表
type StrategyClusterAssociation struct {
	StrategyID int64 `gorm:"primaryKey;column:strategy_id"` // 策略ID
	ClusterID  int64 `gorm:"primaryKey;column:cluster_id"`  // 集群ID
}

// TableName 指定表名
func (StrategyClusterAssociation) TableName() string {
	return "ng_strategy_cluster_association"
}
