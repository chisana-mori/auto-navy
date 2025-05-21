package portal

// ResourcePoolDeviceMatchingPolicy 资源池设备匹配策略.
type ResourcePoolDeviceMatchingPolicy struct {
	BaseModel
	Name             string `gorm:"column:name;type:varchar(255);not null" json:"name"`                           // 策略名称
	Description      string `gorm:"column:description;type:text" json:"description"`                              // 策略描述
	ResourcePoolType string `gorm:"column:resource_pool_type;type:varchar(255);not null" json:"resourcePoolType"` // 资源池类型
	ActionType       string `gorm:"column:action_type;type:varchar(50);not null" json:"actionType"`               // 动作类型：pool_entry 或 pool_exit
	QueryTemplateID  uint   `gorm:"column:query_template_id;not null" json:"queryTemplateId"`                     // 关联的查询模板ID
	Status           string `gorm:"column:status;type:varchar(50);not null;default:'enabled'" json:"status"`      // 状态：enabled 或 disabled
	AdditionConds    string `gorm:"column:addition_conds;type:text" json:"additionConds"`                         // 额外动态条件，JSON格式存储
	CreatedBy        string `gorm:"column:created_by;type:varchar(255)" json:"createdBy"`                         // 创建者
	UpdatedBy        string `gorm:"column:updated_by;type:varchar(255)" json:"updatedBy"`                         // 更新者

	// 关联查询模板（非数据库字段）
	QueryTemplate *QueryTemplate `gorm:"foreignKey:QueryTemplateID" json:"queryTemplate,omitempty"` // 关联的查询模板
}

// TableName 返回表名.
func (ResourcePoolDeviceMatchingPolicy) TableName() string {
	return "resource_pool_device_matching_policy"
}
