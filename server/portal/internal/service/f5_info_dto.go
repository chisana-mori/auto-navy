// Package service provides the business logic for the portal module.
package service

import (
	"time"

	"navy-ng/models/portal"
)

// F5InfoQuery defines query parameters for listing F5Info.
type F5InfoQuery struct {
	Page           int    `json:"page" form:"page" binding:"required,min=1" example:"1" swagger:"description=页码"`
	Size           int    `json:"size" form:"size" binding:"required,min=1,max=100" example:"10" swagger:"description=每页数量"`
	Name           string `json:"name" form:"name" swagger:"description=F5名称，模糊查询"`
	VIP            string `json:"vip" form:"vip" swagger:"description=VIP地址，模糊查询"`
	Port           string `json:"port" form:"port" swagger:"description=端口，模糊查询"`
	AppID          string `json:"appid" form:"appid" swagger:"description=应用ID，模糊查询"`
	InstanceGroup  string `json:"instance_group" form:"instance_group" swagger:"description=实例组，模糊查询"`
	Status         string `json:"status" form:"status" swagger:"description=状态，模糊查询"`
	PoolName       string `json:"pool_name" form:"pool_name" swagger:"description=池名称，模糊查询"`
	K8sClusterName string `json:"k8s_cluster_name" form:"k8s_cluster_name" swagger:"description=K8s集群名称，模糊查询"`
}

// F5InfoUpdateDTO defines the data transfer object for updating F5Info.
type F5InfoUpdateDTO struct {
	Name          string `json:"name" binding:"required" example:"f5-test" swagger:"description=F5名称"`
	VIP           string `json:"vip" binding:"required" example:"192.168.1.1" swagger:"description=VIP地址"`
	Port          string `json:"port" binding:"required" example:"80" swagger:"description=端口"`
	AppID         string `json:"appid" binding:"required" example:"app-001" swagger:"description=应用ID"`
	InstanceGroup string `json:"instance_group" example:"group-1" swagger:"description=实例组"`
	Status        string `json:"status" example:"active" swagger:"description=状态"`
	PoolName      string `json:"pool_name" example:"pool-1" swagger:"description=池名称"`
	PoolStatus    string `json:"pool_status" example:"active" swagger:"description=池状态"`
	PoolMembers   string `json:"pool_members" example:"192.168.1.10:80,192.168.1.11:80" swagger:"description=池成员列表,逗号分隔"`
	K8sClusterID  int64  `json:"k8s_cluster_id" example:"1" swagger:"description=K8s集群ID"`
	Domains       string `json:"domains" example:"example.com,test.com" swagger:"description=域名列表,逗号分隔"`
	GrafanaParams string `json:"grafana_params" example:"http://grafana.example.com" swagger:"description=Grafana监控参数"`
	Ignored       bool   `json:"ignored" example:"false" swagger:"description=是否忽略"`
}

// F5InfoResponse defines the response structure for a single F5Info.
type F5InfoResponse struct {
	ID             int64  `json:"id" example:"1" swagger:"description=ID"`
	Name           string `json:"name" example:"f5-test" swagger:"description=F5名称"`
	VIP            string `json:"vip" example:"192.168.1.1" swagger:"description=VIP地址"`
	Port           string `json:"port" example:"80" swagger:"description=端口"`
	AppID          string `json:"appid" example:"app-001" swagger:"description=应用ID"`
	InstanceGroup  string `json:"instance_group" example:"group-1" swagger:"description=实例组"`
	Status         string `json:"status" example:"active" swagger:"description=状态"`
	PoolName       string `json:"pool_name" example:"pool-1" swagger:"description=池名称"`
	PoolStatus     string `json:"pool_status" example:"active" swagger:"description=池状态"`
	PoolMembers    string `json:"pool_members" example:"192.168.1.10:80,192.168.1.11:80" swagger:"description=池成员列表"`
	K8sClusterID   int64  `json:"k8s_cluster_id" example:"1" swagger:"description=K8s集群ID"`
	K8sClusterName string `json:"k8s_cluster_name" example:"生产集群" swagger:"description=K8s集群名称"`
	Domains        string `json:"domains" example:"example.com,test.com" swagger:"description=域名列表"`
	GrafanaParams  string `json:"grafana_params" example:"http://grafana.example.com" swagger:"description=Grafana监控参数"`
	Ignored        bool   `json:"ignored" example:"false" swagger:"description=是否忽略"`
	CreatedAt      string `json:"created_at" example:"2024-01-01T12:00:00Z" swagger:"description=创建时间"`
	UpdatedAt      string `json:"updated_at" example:"2024-01-02T12:00:00Z" swagger:"description=更新时间"`
}

// F5InfoListResponse defines the response structure for a list of F5Info.
type F5InfoListResponse struct {
	List  []*F5InfoResponse `json:"list" swagger:"description=F5信息列表"`
	Page  int               `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int               `json:"size" example:"10" swagger:"description=每页数量"`
	Total int64             `json:"total" example:"100" swagger:"description=总记录数"`
}

// internalF5Info 内部使用的F5Info结构体.
type internalF5Info struct {
	ID            int64
	Name          string
	VIP           string
	Port          string
	AppID         string
	InstanceGroup string
	Status        string
	PoolName      string
	PoolStatus    string
	PoolMembers   string
	K8sClusterID  int64
	Domains       string
	GrafanaParams string
	Ignored       bool
	Deleted       string
}

// ToModel 转换为数据库模型.
func (i *internalF5Info) ToModel() *portal.F5Info {
	return &portal.F5Info{
		BaseModel: portal.BaseModel{
			ID: i.ID,
		},
		Name:          i.Name,
		VIP:           i.VIP,
		Port:          i.Port,
		AppID:         i.AppID,
		InstanceGroup: i.InstanceGroup,
		Status:        i.Status,
		PoolName:      i.PoolName,
		PoolStatus:    i.PoolStatus,
		PoolMembers:   i.PoolMembers,
		K8sClusterID:  i.K8sClusterID,
		Domains:       i.Domains,
		GrafanaParams: i.GrafanaParams,
		Ignored:       i.Ignored,
		Deleted:       i.Deleted,
	}
}

// FromModel 从数据库模型转换.
func (i *internalF5Info) FromModel(m *portal.F5Info) {
	i.ID = m.ID
	i.Name = m.Name
	i.VIP = m.VIP
	i.Port = m.Port
	i.AppID = m.AppID
	i.InstanceGroup = m.InstanceGroup
	i.Status = m.Status
	i.PoolName = m.PoolName
	i.PoolStatus = m.PoolStatus
	i.PoolMembers = m.PoolMembers
	i.K8sClusterID = m.K8sClusterID
	i.Domains = m.Domains
	i.GrafanaParams = m.GrafanaParams
	i.Ignored = m.Ignored
	i.Deleted = m.Deleted
}

// FromUpdateDTO 从更新DTO转换.
func (i *internalF5Info) FromUpdateDTO(dto *F5InfoUpdateDTO) {
	i.Name = dto.Name
	i.VIP = dto.VIP
	i.Port = dto.Port
	i.AppID = dto.AppID
	i.InstanceGroup = dto.InstanceGroup
	i.Status = dto.Status
	i.PoolName = dto.PoolName
	i.PoolStatus = dto.PoolStatus
	i.PoolMembers = dto.PoolMembers
	i.K8sClusterID = dto.K8sClusterID
	i.Domains = dto.Domains
	i.GrafanaParams = dto.GrafanaParams
	i.Ignored = dto.Ignored
}

// ToResponse 转换为响应DTO.
func (i *internalF5Info) ToResponse(m *portal.F5Info) *F5InfoResponse {
	resp := &F5InfoResponse{
		ID:            i.ID,
		Name:          i.Name,
		VIP:           i.VIP,
		Port:          i.Port,
		AppID:         i.AppID,
		InstanceGroup: i.InstanceGroup,
		Status:        i.Status,
		PoolName:      i.PoolName,
		PoolStatus:    i.PoolStatus,
		PoolMembers:   i.PoolMembers,
		K8sClusterID:  i.K8sClusterID,
		Domains:       i.Domains,
		GrafanaParams: i.GrafanaParams,
		Ignored:       i.Ignored,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
	}

	// 如果K8s集群关联信息存在，获取集群名称
	if m.K8sCluster.ID > 0 {
		resp.K8sClusterName = m.K8sCluster.Name
	}

	return resp
}
