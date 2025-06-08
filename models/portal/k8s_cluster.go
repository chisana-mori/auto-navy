package portal

// K8sCluster K8s集群信息.
type K8sCluster struct {
	BaseModel
	ClusterID         string    `gorm:"unique;default:'';size:36;column:cluster_id"`
	ClusterName       string    `gorm:"default:'';size:128;column:clustername"`
	ClusterNameCn     string    `gorm:"default:'';size:256;column:clusternamecn"`
	Alias             string    `gorm:"default:'';size:128;column:alias"`
	ApiServer         string    `gorm:"default:'';size:256;column:apiserver"`
	ApiServerVip      string    `gorm:"default:'';size:256"`
	EtcdServer        string    `gorm:"default:'';size:256;column:etcdserver"`
	EtcdServerVip     string    `gorm:"default:'';size:1024"`
	IngressServername string    `gorm:"default:'';size:256;column:ingressservername"`
	IngressServerVip  string    `gorm:"default:'';size:256;column:ingressservervip"`
	KubePromVersion   string    `gorm:"default:'';size:256;column:kubepromversion"`
	PromServer        string    `gorm:"default:'';size:256;column:promserver"`
	ThanosServer      string    `gorm:"default:'';size:256;column:thanosserver"`
	Idc               string    `gorm:"default:'';size:36;column:idc"`         // gl ft wg qf
	Zone              string    `gorm:"default:'';size:36;column:zone"`        // egt core
	Status            string    `gorm:"default:'';size:36;column:status"`      // pengding running maintaining
	ClusterType       string    `gorm:"default:'';size:36;column:clustertype"` // tool work game cloud
	KubeConfig        string    `gorm:"default:'';size:1024;column:kubeconfig"`
	Desc              string    `gorm:"default:'';column:desc"`
	Creator           string    `gorm:"default:'';size:128;column:creator"`
	Group             string    `gorm:"default:'';size:128;column:group"` // 物理网络、应用网络网络分组管理，wayne使用
	EsServer          string    `gorm:"default:'';size:128;column:esserver"`
	NetType           string    `gorm:"default:'';size:20;column:nettype"`
	Architecture      string    `gorm:"default:'';size:20"`
	FlowType          string    `gorm:"default:'';size:255"`
	NovaName          string    `gorm:"default:'';size:255"`
	Priority          int       `gorm:"default:0;column:level;size:20"`
	ClusterGroup      string    `gorm:"default:'';size:128"` // 同IDC中上的集群分组信息
	Nodes             []K8sNode `gorm:"foreignKey:K8sClusterID"`
	PodCidr           string    `gorm:"default:'';size:1024"`
	ServiceCidr       string    `gorm:"default:'';size:1024"`
	RrCicode          string    `gorm:"default:'';size:1024"`
	RrGroup           string    `gorm:"default:'';size:1024"`
}

// TableName 指定表名.
func (K8sCluster) TableName() string {
	return "k8s_cluster"
}
