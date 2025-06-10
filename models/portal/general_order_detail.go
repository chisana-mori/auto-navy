package portal

// GeneralOrderDetail 通用订单详情模型
type GeneralOrderDetail struct {
	BaseModel
	OrderID int  `gorm:"column:order_id;type:bigint;unique;not null"` // 关联订单ID（外键）
	Summary string `gorm:"column:summary;type:varchar(255)"`            // 一个简单的摘要字段作为示例
	// 可以根据需要添加更多通用订单的特定字段
	// ...

	// 关联关系
	Order *Order `gorm:"foreignKey:OrderID"` // 关联的基础订单
}

// TableName 指定表名
func (GeneralOrderDetail) TableName() string {
	return "general_order_details"
}
