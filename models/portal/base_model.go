/*
Package portal 提供数据模型定义.
*/
package portal

// BaseModel 基础模型.
type BaseModel struct {
	ID        int64    `gorm:"primaryKey;autoIncrement"`        // 主键ID
	CreatedAt NavyTime `gorm:"column:created_at;type:datetime"` // 创建时间
	UpdatedAt NavyTime `gorm:"column:updated_at;type:datetime"` // 更新时间
}
