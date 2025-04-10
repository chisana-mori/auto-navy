package portal

// K8sNodeLabel 表示 Kubernetes 节点标签信息
type K8sNodeLabel struct {
	BaseModel
	Key     string `gorm:"column:key;type:varchar(191)" json:"key"`         // 标签key
	Value   string `gorm:"column:value;type:varchar(191)" json:"value"`     // 标签value
	Status  string `gorm:"column:status;type:varchar(191)" json:"status"`   // 状态
	NodeID  int64  `gorm:"column:node_id;type:bigint unsigned" json:"node_id"` // 所属节点ID
	Node    *K8sNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`      // 关联的节点信息
}

// TableName 指定表名
func (K8sNodeLabel) TableName() string {
	return "k8s_node_label"
}