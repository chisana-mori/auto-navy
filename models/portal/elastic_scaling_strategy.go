package portal

// ElasticScalingStrategy 弹性伸缩策略
type ElasticScalingStrategy struct {
	BaseModel
	Name                   string  `gorm:"column:name;size:128;not null" json:"name"`
	Description            string  `gorm:"column:description;size:500" json:"description"`
	ThresholdTriggerAction string  `gorm:"column:threshold_trigger_action;size:20;not null" json:"thresholdTriggerAction"` // pool_entry 或 pool_exit
	CPUThresholdValue      float64 `gorm:"column:cpu_threshold_value;default:0" json:"cpuThresholdValue"`                  // CPU使用率阈值
	CPUThresholdType       string  `gorm:"column:cpu_threshold_type;size:20" json:"cpuThresholdType"`                      // usage 或 allocated
	CPUTargetValue         float64 `gorm:"column:cpu_target_value;default:0" json:"cpuTargetValue"`                        // 动作执行后CPU目标使用率
	MemoryThresholdValue   float64 `gorm:"column:memory_threshold_value;default:0" json:"memoryThresholdValue"`            // 内存使用率阈值
	MemoryThresholdType    string  `gorm:"column:memory_threshold_type;size:20" json:"memoryThresholdType"`                // usage 或 allocated
	MemoryTargetValue      float64 `gorm:"column:memory_target_value;default:0" json:"memoryTargetValue"`                  // 动作执行后内存目标使用率
	ConditionLogic         string  `gorm:"column:condition_logic;size:10;default:'OR'" json:"conditionLogic"`              // AND 或 OR
	DurationMinutes        int     `gorm:"column:duration_minutes;not null" json:"durationMinutes"`                        // 持续时间（分钟）
	CooldownMinutes        int     `gorm:"column:cooldown_minutes;not null" json:"cooldownMinutes"`                        // 冷却时间（分钟）
	DeviceCount            int     `gorm:"column:device_count;not null" json:"deviceCount"`                                // 设备数量
	NodeSelector           string  `gorm:"column:node_selector;size:255" json:"nodeSelector"`                              // 节点选择器
	ResourceTypes          string  `gorm:"column:resource_types;size:255" json:"resourceTypes"`                            // 资源类型列表，逗号分隔（计算、存储、网络等）
	Status                 string  `gorm:"column:status;size:20;not null" json:"status"`                                   // enabled 或 disabled
	CreatedBy              string  `gorm:"column:created_by;size:50;not null" json:"createdBy"`
	EntryQueryTemplateID   int64   `gorm:"column:entry_query_template_id" json:"entryQueryTemplateId"` // 入池查询模板ID
	ExitQueryTemplateID    int64   `gorm:"column:exit_query_template_id" json:"exitQueryTemplateId"`   // 退池查询模板ID
}

// TableName 指定表名
func (ElasticScalingStrategy) TableName() string {
	return "elastic_scaling_strategy"
}

// StrategyClusterAssociation 策略集群关联表
type StrategyClusterAssociation struct {
	StrategyID int64 `gorm:"primaryKey;column:strategy_id" json:"strategyId"` // 策略ID
	ClusterID  int64 `gorm:"primaryKey;column:cluster_id" json:"clusterId"`   // 集群ID
}

// TableName 指定表名
func (StrategyClusterAssociation) TableName() string {
	return "strategy_cluster_association"
}
