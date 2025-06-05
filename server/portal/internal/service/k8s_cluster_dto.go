package service

import (
	"navy-ng/models/portal"
)

// K8sClusterQuery 集群查询参数
type K8sClusterQuery struct {
	Page          int    `form:"page" json:"page" binding:"required,min=1"`
	Size          int    `form:"size" json:"size" binding:"required,min=1,max=100"`
	ClusterName   string `form:"cluster_name" json:"cluster_name"`
	ClusterNameCn string `form:"cluster_name_cn" json:"cluster_name_cn"`
	Status        string `form:"status" json:"status"`
	ClusterType   string `form:"cluster_type" json:"cluster_type"`
	Idc           string `form:"idc" json:"idc"`
	Zone          string `form:"zone" json:"zone"`
}

// K8sClusterResponse 集群响应
type K8sClusterResponse struct {
	ID                int64              `json:"id"`
	ClusterID         string             `json:"cluster_id"`
	ClusterName       string             `json:"cluster_name"`
	ClusterNameCn     string             `json:"cluster_name_cn"`
	Alias             string             `json:"alias"`
	ApiServer         string             `json:"api_server"`
	ApiServerVip      string             `json:"api_server_vip"`
	EtcdServer        string             `json:"etcd_server"`
	EtcdServerVip     string             `json:"etcd_server_vip"`
	IngressServername string             `json:"ingress_servername"`
	IngressServerVip  string             `json:"ingress_server_vip"`
	KubePromVersion   string             `json:"kube_prom_version"`
	PromServer        string             `json:"prom_server"`
	ThanosServer      string             `json:"thanos_server"`
	Idc               string             `json:"idc"`
	Zone              string             `json:"zone"`
	Status            string             `json:"status"`
	ClusterType       string             `json:"cluster_type"`
	KubeConfig        string             `json:"kube_config"`
	Desc              string             `json:"desc"`
	Creator           string             `json:"creator"`
	Group             string             `json:"group"`
	EsServer          string             `json:"es_server"`
	NetType           string             `json:"net_type"`
	Architecture      string             `json:"architecture"`
	FlowType          string             `json:"flow_type"`
	NovaName          string             `json:"nova_name"`
	Priority          int                `json:"priority"`
	ClusterGroup      string             `json:"cluster_group"`
	PodCidr           string             `json:"pod_cidr"`
	ServiceCidr       string             `json:"service_cidr"`
	RrCicode          string             `json:"rr_cicode"`
	RrGroup           string             `json:"rr_group"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
	Nodes             []*K8sNodeResponse `json:"nodes,omitempty"`
}

// K8sClusterListResponse 集群列表响应
type K8sClusterListResponse struct {
	List  []*K8sClusterResponse `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// CreateK8sClusterRequest 创建集群请求
type CreateK8sClusterRequest struct {
	ClusterID         string `json:"cluster_id" binding:"required"`
	ClusterName       string `json:"cluster_name" binding:"required"`
	ClusterNameCn     string `json:"cluster_name_cn" binding:"required"`
	Alias             string `json:"alias"`
	ApiServer         string `json:"api_server" binding:"required"`
	ApiServerVip      string `json:"api_server_vip"`
	EtcdServer        string `json:"etcd_server"`
	EtcdServerVip     string `json:"etcd_server_vip"`
	IngressServername string `json:"ingress_servername"`
	IngressServerVip  string `json:"ingress_server_vip"`
	KubePromVersion   string `json:"kube_prom_version"`
	PromServer        string `json:"prom_server"`
	ThanosServer      string `json:"thanos_server"`
	Idc               string `json:"idc" binding:"required"`
	Zone              string `json:"zone" binding:"required"`
	Status            string `json:"status" binding:"required"`
	ClusterType       string `json:"cluster_type" binding:"required"`
	KubeConfig        string `json:"kube_config"`
	Desc              string `json:"desc"`
	Group             string `json:"group"`
	EsServer          string `json:"es_server"`
	NetType           string `json:"net_type"`
	Architecture      string `json:"architecture"`
	FlowType          string `json:"flow_type"`
	NovaName          string `json:"nova_name"`
	Priority          int    `json:"priority"`
	ClusterGroup      string `json:"cluster_group"`
	PodCidr           string `json:"pod_cidr"`
	ServiceCidr       string `json:"service_cidr"`
	RrCicode          string `json:"rr_cicode"`
	RrGroup           string `json:"rr_group"`
}

// ToModel 转换为数据模型
func (r *CreateK8sClusterRequest) ToModel(creator string) portal.K8sCluster {
	return portal.K8sCluster{
		ClusterID:         r.ClusterID,
		ClusterName:       r.ClusterName,
		ClusterNameCn:     r.ClusterNameCn,
		Alias:             r.Alias,
		ApiServer:         r.ApiServer,
		ApiServerVip:      r.ApiServerVip,
		EtcdServer:        r.EtcdServer,
		EtcdServerVip:     r.EtcdServerVip,
		IngressServername: r.IngressServername,
		IngressServerVip:  r.IngressServerVip,
		KubePromVersion:   r.KubePromVersion,
		PromServer:        r.PromServer,
		ThanosServer:      r.ThanosServer,
		Idc:               r.Idc,
		Zone:              r.Zone,
		Status:            r.Status,
		ClusterType:       r.ClusterType,
		KubeConfig:        r.KubeConfig,
		Desc:              r.Desc,
		Creator:           creator,
		Group:             r.Group,
		EsServer:          r.EsServer,
		NetType:           r.NetType,
		Architecture:      r.Architecture,
		FlowType:          r.FlowType,
		NovaName:          r.NovaName,
		Priority:          r.Priority,
		ClusterGroup:      r.ClusterGroup,
		PodCidr:           r.PodCidr,
		ServiceCidr:       r.ServiceCidr,
		RrCicode:          r.RrCicode,
		RrGroup:           r.RrGroup,
	}
}

// UpdateK8sClusterRequest 更新集群请求
type UpdateK8sClusterRequest struct {
	ClusterID         string `json:"cluster_id"`
	ClusterName       string `json:"cluster_name"`
	ClusterNameCn     string `json:"cluster_name_cn"`
	Alias             string `json:"alias"`
	ApiServer         string `json:"api_server"`
	ApiServerVip      string `json:"api_server_vip"`
	EtcdServer        string `json:"etcd_server"`
	EtcdServerVip     string `json:"etcd_server_vip"`
	IngressServername string `json:"ingress_servername"`
	IngressServerVip  string `json:"ingress_server_vip"`
	KubePromVersion   string `json:"kube_prom_version"`
	PromServer        string `json:"prom_server"`
	ThanosServer      string `json:"thanos_server"`
	Idc               string `json:"idc"`
	Zone              string `json:"zone"`
	Status            string `json:"status"`
	ClusterType       string `json:"cluster_type"`
	KubeConfig        string `json:"kube_config"`
	Desc              string `json:"desc"`
	Group             string `json:"group"`
	EsServer          string `json:"es_server"`
	NetType           string `json:"net_type"`
	Architecture      string `json:"architecture"`
	FlowType          string `json:"flow_type"`
	NovaName          string `json:"nova_name"`
	Priority          *int   `json:"priority"`
	ClusterGroup      string `json:"cluster_group"`
	PodCidr           string `json:"pod_cidr"`
	ServiceCidr       string `json:"service_cidr"`
	RrCicode          string `json:"rr_cicode"`
	RrGroup           string `json:"rr_group"`
}

// ToUpdateMap 转换为更新数据映射
func (r *UpdateK8sClusterRequest) ToUpdateMap(updater string) map[string]interface{} {
	updateMap := make(map[string]interface{})

	if r.ClusterID != "" {
		updateMap["cluster_id"] = r.ClusterID
	}
	if r.ClusterName != "" {
		updateMap["clustername"] = r.ClusterName
	}
	if r.ClusterNameCn != "" {
		updateMap["clusternamecn"] = r.ClusterNameCn
	}
	if r.Alias != "" {
		updateMap["alias"] = r.Alias
	}
	if r.ApiServer != "" {
		updateMap["api_server"] = r.ApiServer
	}
	if r.ApiServerVip != "" {
		updateMap["api_server_vip"] = r.ApiServerVip
	}
	if r.EtcdServer != "" {
		updateMap["etcd_server"] = r.EtcdServer
	}
	if r.EtcdServerVip != "" {
		updateMap["etcd_server_vip"] = r.EtcdServerVip
	}
	if r.IngressServername != "" {
		updateMap["ingress_servername"] = r.IngressServername
	}
	if r.IngressServerVip != "" {
		updateMap["ingress_server_vip"] = r.IngressServerVip
	}
	if r.KubePromVersion != "" {
		updateMap["kube_prom_version"] = r.KubePromVersion
	}
	if r.PromServer != "" {
		updateMap["prom_server"] = r.PromServer
	}
	if r.ThanosServer != "" {
		updateMap["thanos_server"] = r.ThanosServer
	}
	if r.Idc != "" {
		updateMap["idc"] = r.Idc
	}
	if r.Zone != "" {
		updateMap["zone"] = r.Zone
	}
	if r.Status != "" {
		updateMap["status"] = r.Status
	}
	if r.ClusterType != "" {
		updateMap["clustertype"] = r.ClusterType
	}
	if r.KubeConfig != "" {
		updateMap["kube_config"] = r.KubeConfig
	}
	if r.Desc != "" {
		updateMap["desc"] = r.Desc
	}
	if r.Group != "" {
		updateMap["group"] = r.Group
	}
	if r.EsServer != "" {
		updateMap["es_server"] = r.EsServer
	}
	if r.NetType != "" {
		updateMap["net_type"] = r.NetType
	}
	if r.Architecture != "" {
		updateMap["architecture"] = r.Architecture
	}
	if r.FlowType != "" {
		updateMap["flow_type"] = r.FlowType
	}
	if r.NovaName != "" {
		updateMap["nova_name"] = r.NovaName
	}
	if r.Priority != nil {
		updateMap["priority"] = *r.Priority
	}
	if r.ClusterGroup != "" {
		updateMap["cluster_group"] = r.ClusterGroup
	}
	if r.PodCidr != "" {
		updateMap["pod_cidr"] = r.PodCidr
	}
	if r.ServiceCidr != "" {
		updateMap["service_cidr"] = r.ServiceCidr
	}
	if r.RrCicode != "" {
		updateMap["rr_cicode"] = r.RrCicode
	}
	if r.RrGroup != "" {
		updateMap["rr_group"] = r.RrGroup
	}

	// 更新修改人和修改时间
	updateMap["updater"] = updater

	return updateMap
}

// K8sNodeResponse 节点响应
type K8sNodeResponse struct {
	ID                      int64  `json:"id"`
	NodeName                string `json:"node_name"`
	HostIP                  string `json:"host_ip"`
	Role                    string `json:"role"`
	OSImage                 string `json:"os_image"`
	KernelVersion           string `json:"kernel_version"`
	KubeletVersion          string `json:"kubelet_version"`
	ContainerRuntimeVersion string `json:"container_runtime_version"`
	KubeProxyVersion        string `json:"kube_proxy_version"`
	CPULogic                string    `json:"cpu_logic"`
	MemLogic                string    `json:"mem_logic"`
	CPUCapacity             string `json:"cpu_capacity"`
	MemCapacity             string `json:"mem_capacity"`
	CPUAllocatable          string `json:"cpu_allocatable"`
	MemAllocatable          string `json:"mem_allocatable"`
	FSTypeRoot              string `json:"fs_type_root"`
	DiskRoot                string `json:"disk_root"`
	DiskDocker              string `json:"disk_docker"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
}
