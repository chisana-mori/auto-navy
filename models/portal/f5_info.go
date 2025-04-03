package portal

import "time"

// F5Info F5负载均衡信息
type F5Info struct {
	BaseModel
	Name          string     `gorm:"column:name;type:varchar(255);not null" json:"name"`              // 名称
	VIP           string     `gorm:"column:vip;type:varchar(15);not null" json:"vip"`                 // 虚拟IP
	Port          string     `gorm:"column:port;type:varchar(10);not null" json:"port"`               // 端口
	AppID         string     `gorm:"column:appid;type:varchar(50)" json:"appid"`                      // 应用ID
	InstanceGroup string     `gorm:"column:instance_group;type:varchar(50)" json:"instance_group"`    // 实例组
	Status        string     `gorm:"column:status;type:varchar(50)" json:"status"`                    // 状态
	PoolName      string     `gorm:"column:pool_name;type:varchar(50)" json:"pool_name"`              // Pool名称
	PoolStatus    string     `gorm:"column:pool_status;type:varchar(50)" json:"pool_status"`          // Pool状态
	PoolMembers   string     `gorm:"column:pool_members;type:text" json:"pool_members"`               // Pool成员
	K8sClusterID  int64      `gorm:"column:k8s_cluster_id;type:bigint" json:"k8s_cluster_id"`         // K8s集群ID
	K8sCluster    K8sCluster `gorm:"foreignKey:K8sClusterID" json:"k8s_cluster,omitempty"`           // K8s集群信息
	Domains       string     `gorm:"column:domains;type:text" json:"domains"`                         // 域名
	GrafanaParams string     `gorm:"column:grafana_params;type:text" json:"grafana_params"`           // Grafana参数
	Ignored       bool       `gorm:"column:ignored;type:boolean;default:false" json:"ignored"`        // 是否忽略
	CreatedAt     time.Time  `gorm:"column:created_at;type:datetime" json:"created_at"`               // 创建时间
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:datetime" json:"updated_at"`               // 更新时间
	Deleted       string     `gorm:"column:deleted;type:varchar(255)" json:"deleted,omitempty"`       // 软删除标记
}

// TableName 指定表名
func (F5Info) TableName() string {
	return "f5_info"
}
