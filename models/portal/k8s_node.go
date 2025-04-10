package portal

// K8sNode 表示 Kubernetes 节点信息
type K8sNode struct {
	BaseModel
	NodeName              string `gorm:"column:nodename;type:varchar(191);not null" json:"nodename"`                // 节点名
	HostIP                string `gorm:"column:hostip;type:varchar(191)" json:"hostip"`                            // 节点IP地址
	Role                  string `gorm:"column:role;type:varchar(191)" json:"role"`                                // 节点角色
	OSImage               string `gorm:"column:osimage;type:varchar(128)" json:"osimage"`                          // OS镜像
	KernelVersion         string `gorm:"column:kernelversion;type:varchar(64)" json:"kernel_version"`              // 内核版本
	KubeletVersion        string `gorm:"column:kubeletversion;type:varchar(64)" json:"kubelet_version"`            // kubelet版本
	ContainerRuntimeVersion string `gorm:"column:containerruntimeversion;type:varchar(64)" json:"container_runtime_version"` // 运行时版本
	KubeProxyVersion      string `gorm:"column:kubeproxyversion;type:varchar(64)" json:"kube_proxy_version"`      // proxy版本
	CPULogic             string `gorm:"column:cpulogic;type:varchar(191)" json:"cpu_logic"`                       // 物理逻辑核数
	MemLogic             string `gorm:"column:memlogic;type:varchar(191)" json:"mem_logic"`                       // 物理逻辑值
	CPUCapacity          string `gorm:"column:cpucapacity;type:varchar(191)" json:"cpu_capacity"`                 // CPU总容量
	MemCapacity          string `gorm:"column:memcapacity;type:varchar(191)" json:"mem_capacity"`                 // 内存总容量
	CPUAllocatable       string `gorm:"column:cpuallocatable;type:varchar(191)" json:"cpu_allocatable"`          // CPU可分配容量
	MemAllocatable       string `gorm:"column:memallocatable;type:varchar(191)" json:"mem_allocatable"`          // 内存可分配容量
	FSTypeRoot           string `gorm:"column:fstyperoot;type:varchar(191)" json:"fs_type_root"`                 // 特殊分区
	DiskRoot             string `gorm:"column:diskroot;type:varchar(191)" json:"disk_root"`                      // /root磁盘量
	DiskDocker           string `gorm:"column:diskdocker;type:varchar(191)" json:"disk_docker"`                  // /var/lib/docker磁盘量
	DiskKubelet          string `gorm:"column:diskkubelet;type:varchar(191)" json:"disk_kubelet"`                // /var/lib/kubelet磁盘量
	NodeCreated          string `gorm:"column:nodecreated;type:varchar(191)" json:"node_created"`                // 节点创建时间
	Status               string `gorm:"column:status;type:varchar(191)" json:"status"`                           // 状态
	K8sClusterID         int64  `gorm:"column:k8s_cluster_id;type:bigint unsigned" json:"k8s_cluster_id"`       // 所属集群ID
	K8sCluster           *K8sCluster `gorm:"foreignKey:K8sClusterID" json:"k8s_cluster,omitempty"`              // 关联的集群信息
	Labels               []K8sNodeLabel `gorm:"foreignKey:NodeID" json:"labels,omitempty"`                       // 节点的标签列表
	Taints               []K8sNodeTaint `gorm:"foreignKey:NodeID" json:"taints,omitempty"`                      // 节点的污点列表
	GPU                  string `gorm:"column:gpu;type:varchar(64)" json:"gpu"`                                  // node是否含有gpu
	DiskCount            int    `gorm:"column:disk_count;type:int" json:"disk_count"`                           // 硬盘数
	DiskDetail           string `gorm:"column:disk_detail;type:varchar(512)" json:"disk_detail"`                // 硬盘详情
	NetworkSpeed         int    `gorm:"column:network_speed;type:int" json:"network_speed"`                     // 网卡网速
}

// TableName 指定表名
func (K8sNode) TableName() string {
	return "k8s_node"
}
