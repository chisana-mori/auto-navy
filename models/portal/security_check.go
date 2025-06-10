package portal

import (
	"time"
)

// SecurityCheck 安全检查主表
type SecurityCheck struct {
	ID          int     `gorm:"primaryKey;autoIncrement"`
	ClusterName string    `gorm:"type:varchar(255);not null;index:idx_cluster"`
	NodeType    string    `gorm:"type:enum('master','etcd','node');not null"`
	NodeName    string    `gorm:"type:varchar(255);not null"`
	CheckType   string    `gorm:"type:enum('k8s','runtime');not null;index:idx_check_type"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	Deleted     string    `gorm:"type:varchar(255);default:''"`
}

// SecurityCheckItem 安全检查项表
type SecurityCheckItem struct {
	ID              int     `gorm:"primaryKey;autoIncrement"`
	SecurityCheckID int     `gorm:"not null;index:idx_check_id"`
	ItemName        string    `gorm:"type:varchar(255);not null"`
	ItemValue       string    `gorm:"type:text"`
	Status          bool      `gorm:"not null"`
	FixSuggestion   string    `gorm:"type:text"` // 修复建议
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
	Deleted         string    `gorm:"type:varchar(255);default:''"`
}

func (SecurityCheck) TableName() string {
	return "security_check"
}

func (SecurityCheckItem) TableName() string {
	return "security_check_item"
}
