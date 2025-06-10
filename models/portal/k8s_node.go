package portal

// K8sNode 表示 Kubernetes 节点信息.
type K8sNode struct {
	BaseModel
	NodeName string `gorm:"column:nodename;type:varchar(191);not null"` // 节点名
	HostIP   string `gorm:"column:hostip;type:varchar(191)"`
	// 节点IP地址
	Role           string `gorm:"column:role;type:varchar(191)"`         // 节点角色
	OSImage        string `gorm:"column:osimage;type:varchar(128)"`      // OS镜像
	KernelVersion  string `gorm:"column:kernelversion;type:varchar(64)"` // 内核版本
	KubeletVersion string `gorm:"column:kubeletversion;type:varchar(64)"`
	// kubelet版本
	ContainerRuntimeVersion string `gorm:"column:containerruntimeversion;type:varchar(64)"`
	// 运行时版本
	KubeProxyVersion string `gorm:"column:kubeproxyversion;type:varchar(64)"`
	// proxy版本
	CPULogic       string `gorm:"column:cpulogic;type:varchar(191)"`    // 物理逻辑核数
	MemLogic       string `gorm:"column:memlogic;type:varchar(191)"`    // 物理逻辑值
	CPUCapacity    string `gorm:"column:cpucapacity;type:varchar(191)"` // CPU总容量
	MemCapacity    string `gorm:"column:memcapacity;type:varchar(191)"` // 内存总容量
	CPUAllocatable string `gorm:"column:cpuallocatable;type:varchar(191)"`
	// CPU可分配容量
	MemAllocatable string `gorm:"column:memallocatable;type:varchar(191)"` // 内存可分配容量
	FSTypeRoot     string `gorm:"column:fstyperoot;type:varchar(191)"`     // 特殊分区
	DiskRoot       string `gorm:"column:diskroot;type:varchar(191)"`
	// /root磁盘量
	DiskDocker string `gorm:"column:diskdocker;type:varchar(191)"`
	// /var/lib/docker磁盘量
	DiskKubelet string `gorm:"column:diskkubelet;type:varchar(191)"`
	// /var/lib/kubelet磁盘量
	NodeCreated  string         `gorm:"column:nodecreated;type:varchar(191)"`       // 节点创建时间
	Status       string         `gorm:"column:status;type:varchar(191)"`            // 状态
	K8sClusterID int          `gorm:"column:k8s_cluster_id;type:bigint unsigned"` // 所属集群ID
	K8sCluster   *K8sCluster    `gorm:"foreignKey:K8sClusterID"`                    // 关联的集群信息
	Labels       []K8sNodeLabel `gorm:"foreignKey:NodeID"`                          // 节点的标签列表
	Taints       []K8sNodeTaint `gorm:"foreignKey:NodeID"`                          // 节点的污点列表
	GPU          string         `gorm:"column:gpu;type:varchar(64)"`
	// node是否含有gpu
	DiskCount    int    `gorm:"column:disk_count;type:int"`           // 硬盘数
	DiskDetail   string `gorm:"column:disk_detail;type:varchar(512)"` // 硬盘详情
	NetworkSpeed int    `gorm:"column:network_speed;type:int"`        // 网卡网速
}

// TableName 指定表名.
func (K8sNode) TableName() string {
	return "k8s_node"
}
