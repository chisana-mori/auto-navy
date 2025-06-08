package portal

// K8sNodeTaint 表示 Kubernetes 节点污点信息.
type K8sNodeTaint struct {
	BaseModel
	Key    string   `gorm:"column:key;type:varchar(191)"`        // 污点key
	Value  string   `gorm:"column:value;type:varchar(191)"`      // 污点value
	Effect string   `gorm:"column:effect;type:varchar(191)"`     // 生效类型
	Status string   `gorm:"column:status;type:varchar(191)"`     // 状态
	NodeID int64    `gorm:"column:node_id;type:bigint unsigned"` // 所属节点ID
	Node   *K8sNode `gorm:"foreignKey:NodeID"`                   // 关联的节点信息
}

// TableName 指定表名.
func (K8sNodeTaint) TableName() string {
	return "k8s_node_taint"
}
