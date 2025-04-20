package portal

// K8sCluster K8s集群信息.
type K8sCluster struct {
	BaseModel
	ClusterID         string    `json:"clusterUUID" gorm:"unique;default:'';size:36;column:cluster_id"`
	ClusterName       string    `json:"clusterName" gorm:"default:'';size:128;column:clustername"`
	ClusterNameCn     string    `json:"clusterNameCn" gorm:"default:'';size:256;column:clusternamecn"`
	Alias             string    `json:"alias" gorm:"default:'';size:128;column:alias"`
	ApiServer         string    `json:"apiServer" gorm:"default:'';size:256;column:apiserver"`
	ApiServerVip      string    `json:"apiServerVip" gorm:"default:'';size:256"`
	EtcdServer        string    `json:"etcdServer" gorm:"default:'';size:256;column:etcdserver"`
	EtcdServerVip     string    `json:"etcdServerVip" gorm:"default:'';size:1024"`
	IngressServername string    `json:"ingressServername" gorm:"default:'';size:256;column:ingressservername"`
	IngressServerVip  string    `json:"ingressServerVip" gorm:"default:'';size:256;column:ingressservervip"`
	KubePromVersion   string    `json:"kubePromVersion" gorm:"default:'';size:256;column:kubepromversion"`
	PromServer        string    `json:"promServer" gorm:"default:'';size:256;column:promserver"`
	ThanosServer      string    `json:"thanosServer" gorm:"default:'';size:256;column:thanosserver"`
	Idc               string    `json:"idc" gorm:"default:'';size:36;column:idc"`                 // gl ft wg qf
	Zone              string    `json:"zone" gorm:"default:'';size:36;column:zone"`               // egt core
	Status            string    `json:"status" gorm:"default:'';size:36;column:status"`           // pengding running maintaining
	ClusterType       string    `json:"clusterType" gorm:"default:'';size:36;column:clustertype"` // tool work game cloud
	KubeConfig        string    `json:"kubeConfig" gorm:"default:'';size:1024;column:kubeconfig"`
	Desc              string    `json:"desc" gorm:"default:'';column:desc"`
	Creator           string    `json:"creator" gorm:"default:'';size:128;column:creator"`
	Group             string    `json:"group" gorm:"default:'';size:128;column:group"` // 物理网络、应用网络网络分组管理，wayne使用
	EsServer          string    `json:"esServer" gorm:"default:'';size:128;column:esserver"`
	NetType           string    `json:"netType" gorm:"default:'';size:20;column:nettype"`
	Architecture      string    `json:"architecture" gorm:"default:'';size:20"`
	FlowType          string    `json:"flowType" gorm:"default:'';size:255"`
	NovaName          string    `json:"novaName" gorm:"default:'';size:255"`
	Priority          int       `json:"level" gorm:"default:0;column:level;size:20"`
	ClusterGroup      string    `json:"clusterGroup" gorm:"default:'';size:128"` // 同IDC中上的集群分组信息
	Nodes             []K8sNode `gorm:"foreignKey:K8sClusterID" json:"nodes,omitempty"`
	PodCidr           string    `json:"podCidr" gorm:"default:'';size:1024"`
	ServiceCidr       string    `json:"serviceCidr" gorm:"default:'';size:1024"`
	RrCicode          string    `json:"rrCicode" gorm:"default:'';size:1024"`
	RrGroup           string    `json:"rrGroup" gorm:"default:'';size:1024"`
}

// TableName 指定表名.
func (K8sCluster) TableName() string {
	return "k8s_cluster"
}
