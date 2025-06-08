package portal

// QueryTemplate 查询模板.
type QueryTemplate struct {
	BaseModel
	Name        string `gorm:"column:name;type:varchar(255);not null"` // 模板名称
	Description string `gorm:"column:description;type:text"`           // 模板描述
	Groups      string `gorm:"column:groups;type:text;not null"`       // 筛选组列表，JSON格式
	CreatedBy   string `gorm:"column:created_by;type:varchar(255)"`    // 创建者
	UpdatedBy   string `gorm:"column:updated_by;type:varchar(255)"`    // 更新者
}

// TableName 返回表名.
func (QueryTemplate) TableName() string {
	return "query_template"
}
