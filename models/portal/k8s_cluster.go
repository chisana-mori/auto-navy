package portal

import "time"

// K8sCluster K8s集群信息
type K8sCluster struct {
	BaseModel
	Name      string    `gorm:"column:name;type:varchar(255);not null" json:"name"`       // 集群名称
	Region    string    `gorm:"column:region;type:varchar(50)" json:"region"`             // 区域
	Endpoint  string    `gorm:"column:endpoint;type:varchar(255)" json:"endpoint"`        // 集群API端点
	Status    string    `gorm:"column:status;type:varchar(50)" json:"status"`             // 集群状态
	CreatedAt time.Time `gorm:"column:created_at;type:datetime" json:"created_at"`        // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`        // 更新时间
	Deleted   string    `gorm:"column:deleted;type:varchar(255)" json:"deleted,omitempty"`// 软删除标记
	Nodes     []K8sNode `gorm:"foreignKey:K8sClusterID" json:"nodes,omitempty"`          // 关联的节点列表
}

// TableName 指定表名
func (K8sCluster) TableName() string {
	return "k8s_cluster"
} 