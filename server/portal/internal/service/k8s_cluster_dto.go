package service

import (
	"navy-ng/models/portal"
)

// K8sClusterQuery 集群查询参数
type K8sClusterQuery struct {
	Page          int    `form:"page" json:"page" binding:"omitempty,min=1"`
	Size          int    `form:"size" json:"size" binding:"omitempty,min=1,max=100"`
	ClusterName   string `form:"cluster_name" json:"clusterName"`
	ClusterNameCn string `form:"cluster_name_cn" json:"clusterNameCn"`
	Status        string `form:"status" json:"status"`
	ClusterType   string `form:"cluster_type" json:"clusterType"`
	Idc           string `form:"idc" json:"idc"`
	Zone          string `form:"zone" json:"zone"`
}

// K8sClusterResponse 集群响应
type K8sClusterResponse struct {
	ID                int              `json:"id"`
	ClusterID         string             `json:"clusterId"`
	ClusterName       string             `json:"clusterName"`
	ClusterNameCn     string             `json:"clusterNameCn"`
	Alias             string             `json:"alias"`
	ApiServer         string             `json:"apiServer"`
	ApiServerVip      string             `json:"apiServerVip"`
	EtcdServer        string             `json:"etcdServer"`
	EtcdServerVip     string             `json:"etcdServerVip"`
	IngressServername string             `json:"ingressServername"`
	IngressServerVip  string             `json:"ingressServerVip"`
	KubePromVersion   string             `json:"kubePromVersion"`
	PromServer        string             `json:"promServer"`
	ThanosServer      string             `json:"thanosServer"`
	Idc               string             `json:"idc"`
	Zone              string             `json:"zone"`
	Status            string             `json:"status"`
	ClusterType       string             `json:"clusterType"`
	KubeConfig        string             `json:"kubeConfig"`
	Desc              string             `json:"desc"`
	Creator           string             `json:"creator"`
	Group             string             `json:"group"`
	EsServer          string             `json:"esServer"`
	NetType           string             `json:"netType"`
	Architecture      string             `json:"architecture"`
	FlowType          string             `json:"flowType"`
	NovaName          string             `json:"novaName"`
	Priority          int                `json:"priority"`
	ClusterGroup      string             `json:"clusterGroup"`
	PodCidr           string             `json:"podCidr"`
	ServiceCidr       string             `json:"serviceCidr"`
	RrCicode          string             `json:"rrCicode"`
	RrGroup           string             `json:"rrGroup"`
	CreatedAt         string             `json:"createdAt"`
	UpdatedAt         string             `json:"updatedAt"`
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
	ClusterID         string `json:"clusterId" binding:"required"`
	ClusterName       string `json:"clusterName" binding:"required"`
	ClusterNameCn     string `json:"clusterNameCn" binding:"required"`
	Alias             string `json:"alias"`
	ApiServer         string `json:"apiServer" binding:"required"`
	ApiServerVip      string `json:"apiServerVip"`
	EtcdServer        string `json:"etcdServer"`
	EtcdServerVip     string `json:"etcdServerVip"`
	IngressServername string `json:"ingressServername"`
	IngressServerVip  string `json:"ingressServerVip"`
	KubePromVersion   string `json:"kubePromVersion"`
	PromServer        string `json:"promServer"`
	ThanosServer      string `json:"thanosServer"`
	Idc               string `json:"idc" binding:"required"`
	Zone              string `json:"zone" binding:"required"`
	Status            string `json:"status" binding:"required"`
	ClusterType       string `json:"clusterType" binding:"required"`
	KubeConfig        string `json:"kubeConfig"`
	Desc              string `json:"desc"`
	Group             string `json:"group"`
	EsServer          string `json:"esServer"`
	NetType           string `json:"netType"`
	Architecture      string `json:"architecture"`
	FlowType          string `json:"flowType"`
	NovaName          string `json:"novaName"`
	Priority          int    `json:"priority"`
	ClusterGroup      string `json:"clusterGroup"`
	PodCidr           string `json:"podCidr"`
	ServiceCidr       string `json:"serviceCidr"`
	RrCicode          string `json:"rrCicode"`
	RrGroup           string `json:"rrGroup"`
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
	ClusterID         string `json:"clusterId"`
	ClusterName       string `json:"clusterName"`
	ClusterNameCn     string `json:"clusterNameCn"`
	Alias             string `json:"alias"`
	ApiServer         string `json:"apiServer"`
	ApiServerVip      string `json:"apiServerVip"`
	EtcdServer        string `json:"etcdServer"`
	EtcdServerVip     string `json:"etcdServerVip"`
	IngressServername string `json:"ingressServername"`
	IngressServerVip  string `json:"ingressServerVip"`
	KubePromVersion   string `json:"kubePromVersion"`
	PromServer        string `json:"promServer"`
	ThanosServer      string `json:"thanosServer"`
	Idc               string `json:"idc"`
	Zone              string `json:"zone"`
	Status            string `json:"status"`
	ClusterType       string `json:"clusterType"`
	KubeConfig        string `json:"kubeConfig"`
	Desc              string `json:"desc"`
	Group             string `json:"group"`
	EsServer          string `json:"esServer"`
	NetType           string `json:"netType"`
	Architecture      string `json:"architecture"`
	FlowType          string `json:"flowType"`
	NovaName          string `json:"novaName"`
	Priority          *int   `json:"priority"`
	ClusterGroup      string `json:"clusterGroup"`
	PodCidr           string `json:"podCidr"`
	ServiceCidr       string `json:"serviceCidr"`
	RrCicode          string `json:"rrCicode"`
	RrGroup           string `json:"rrGroup"`
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
	ID                      int  `json:"id"`
	NodeName                string `json:"nodeName"`
	HostIP                  string `json:"hostIp"`
	Role                    string `json:"role"`
	OSImage                 string `json:"osImage"`
	KernelVersion           string `json:"kernelVersion"`
	KubeletVersion          string `json:"kubeletVersion"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion"`
	KubeProxyVersion        string `json:"kubeProxyVersion"`
	CPULogic                string    `json:"cpuLogic"`
	MemLogic                string    `json:"memLogic"`
	CPUCapacity             string `json:"cpuCapacity"`
	MemCapacity             string `json:"memCapacity"`
	CPUAllocatable          string `json:"cpuAllocatable"`
	MemAllocatable          string `json:"memAllocatable"`
	FSTypeRoot              string `json:"fsTypeRoot"`
	DiskRoot                string `json:"diskRoot"`
	DiskDocker              string `json:"diskDocker"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
}
