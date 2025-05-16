package portal

// ResourcePoolDeviceMatchingPolicy 资源池设备匹配策略.
type ResourcePoolDeviceMatchingPolicy struct {
	BaseModel
	Name             string `gorm:"column:name;type:varchar(255);not null" json:"name"`                           // 策略名称
	Description      string `gorm:"column:description;type:text" json:"description"`                              // 策略描述
	ResourcePoolType string `gorm:"column:resource_pool_type;type:varchar(255);not null" json:"resourcePoolType"` // 资源池类型
	ActionType       string `gorm:"column:action_type;type:varchar(50);not null" json:"actionType"`               // 动作类型：pool_entry 或 pool_exit
	QueryGroups      string `gorm:"column:query_groups;type:text;not null" json:"queryGroups"`                    // 查询条件组，JSON格式
	Status           string `gorm:"column:status;type:varchar(50);not null;default:'enabled'" json:"status"`      // 状态：enabled 或 disabled
	CreatedBy        string `gorm:"column:created_by;type:varchar(255)" json:"createdBy"`                         // 创建者
	UpdatedBy        string `gorm:"column:updated_by;type:varchar(255)" json:"updatedBy"`                         // 更新者
}

// TableName 返回表名.
func (ResourcePoolDeviceMatchingPolicy) TableName() string {
	return "resource_pool_device_matching_policy"
}
