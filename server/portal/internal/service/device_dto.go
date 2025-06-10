package service

import (
	"time"
)

// DeviceQuery 设备查询参数
// swagger:model
type DeviceQueryDTO struct {
	Page    int    `form:"page" json:"page" example:"1" swagger:"description=页码"`
	Size    int    `form:"size" json:"size" example:"10" swagger:"description=每页数量"`
	Keyword string `form:"keyword" json:"keyword" example:"192.168" swagger:"description=搜索关键字"`
}

// DeviceResponse 设备响应
// swagger:model
type DeviceResponseDTO struct {
	ID           int  `json:"id" example:"1" swagger:"description=设备ID"`
	DeviceID     string `json:"deviceId" example:"SYSOPS00409045" swagger:"description=设备ID"`
	IP           string `json:"ip" example:"29.19.50.124" swagger:"description=IP地址"`
	MachineType  string `json:"machineType" example:"qf-core601-flannel-2" swagger:"description=机器类型"`
	Cluster      string `json:"cluster" example:"work" swagger:"description=所属集群"`
	Role         string `json:"role" example:"x86" swagger:"description=集群角色"`
	Arch         string `json:"arch" example:"qf" swagger:"description=架构"`
	IDC          string `json:"idc" example:"601" swagger:"description=IDC"`
	Room         string `json:"room" example:"OF601-02P" swagger:"description=Room"`
	Datacenter   string `json:"datacenter" example:"private_cloud" swagger:"description=机房"`
	Cabinet      string `json:"cabinet" example:"central" swagger:"description=机柜号"`
	Network      string `json:"network" example:"10703" swagger:"description=网络区域"`
	AppID        string `json:"appId" example:"" swagger:"description=APPID"`
	ResourcePool string `json:"resourcePool" example:"" swagger:"description=资源池/产品"`
	CreatedAt    string `json:"createdAt" example:"2024-01-01T12:00:00Z" swagger:"description=创建时间"`
	UpdatedAt    string `json:"updatedAt" example:"2024-01-01T12:30:00Z" swagger:"description=更新时间"`
}

// DeviceListResponseDTO 设备列表响应
// swagger:model
type DeviceListResponseDTO struct {
	List  []DeviceResponseDTO `json:"list" swagger:"description=设备列表"`
	Total int64               `json:"total" example:"100" swagger:"description=总数"`
	Page  int                 `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int                 `json:"size" example:"10" swagger:"description=每页数量"`
}

// DeviceRoleUpdateRequestDTO 设备角色更新请求
// swagger:model
type DeviceRoleUpdateRequestDTO struct {
	Role string `json:"role" example:"master" swagger:"description=新的集群角色"`
}

// DeviceListResponse 设备列表响应
// swagger:model
type DeviceListResponse struct {
	List  []DeviceResponse `json:"list" swagger:"description=设备列表"`
	Total int64            `json:"total" example:"100" swagger:"description=总数"`
	Page  int              `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int              `json:"size" example:"10" swagger:"description=每页数量"`
}

// --- Request/Query Structs ---

// DeviceQuery (Simplified version, if needed for basic keyword search, otherwise remove)
// Consider removing if ListDevices in device.go is removed.
type DeviceQuery struct {
	Page        int    `form:"page" json:"page"`               // 页码
	Size        int    `form:"size" json:"size"`               // 每页数量
	Keyword     string `form:"keyword" json:"keyword"`         // 搜索关键字 (For simple search)
	OnlySpecial bool   `form:"onlySpecial" json:"onlySpecial"` // 仅显示特殊设备
}

// DeviceRoleUpdateRequest Request to update device role
type DeviceRoleUpdateRequest struct {
	Role string `json:"role" binding:"required"` // 新的角色值
}

// DeviceGroupUpdateRequest Request to update device group/category
type DeviceGroupUpdateRequest struct {
	Group string `json:"group"` // 新的用途值
}

// --- Response Structs ---

// DeviceResponse Standard response for a single device or item in a list
type DeviceResponse struct {
	ID             int     `json:"id"`             // ID
	DeviceID       int     `json:"deviceId"`       // 设备ID (Ensure this is populated)
	CICode         string    `json:"ciCode"`         // 设备编码
	IP             string    `json:"ip"`             // IP地址
	ArchType       string    `json:"archType"`       // CPU架构
	IDC            string    `json:"idc"`            // IDC
	Room           string    `json:"room"`           // 机房
	Cabinet        string    `json:"cabinet"`        // 所属机柜
	CabinetNO      string    `json:"cabinetNo"`      // 机柜编号
	InfraType      string    `json:"infraType"`      // 网络类型
	IsLocalization bool      `json:"isLocalization"` // 是否国产化
	NetZone        string    `json:"netZone"`        // 网络区域
	Group          string    `json:"group"`          // 机器类别
	AppID          string    `json:"appId"`          // APPID
	AppName        string    `json:"appName"`        // 应用名称（来自 device_app 表）
	OsCreateTime   string    `json:"osCreateTime"`   // 操作系统创建时间
	CPU            float64   `json:"cpu"`            // CPU数量
	Memory         float64   `json:"memory"`         // 内存大小
	Model          string    `json:"model"`          // 型号
	KvmIP          string    `json:"kvmIp"`          // KVM IP
	OS             string    `json:"os"`             // 操作系统
	Company        string    `json:"company"`        // 厂商
	OSName         string    `json:"osName"`         // 操作系统名称
	OSIssue        string    `json:"osIssue"`        // 操作系统版本
	OSKernel       string    `json:"osKernel"`       // 操作系统内核
	Status         string    `json:"status"`         // 状态
	AcceptanceTime string    `json:"acceptanceTime"` // 验收时间
	CreatedAt      time.Time `json:"createdAt"`      // 创建时间
	UpdatedAt      time.Time `json:"updatedAt"`      // 更新时间
	IsSpecial      bool      `json:"isSpecial"`      // 是否为特殊设备 (用于前端高亮)

	// Fields from K8s relations and device table
	Role         string `json:"role,omitempty"`         // 最终角色 (来自 k8s_node 或 k8s_etcd 或 device)
	Cluster      string `json:"cluster,omitempty"`      // 最终集群名称 (来自 k8s_node 或 k8s_etcd 或 device)
	ClusterID    int    `json:"clusterId,omitempty"`    // 集群ID (来自 k8s_node 或 k8s_etcd 或 device)
	DiskCount    int    `json:"diskCount,omitempty"`    // 磁盘数量 (来自 device 表)
	DiskDetail   string `json:"diskDetail,omitempty"`   // 磁盘详情 (来自 device 表)
	NetworkSpeed string `json:"networkSpeed,omitempty"` // 网络速度 (来自 device 表)
	FeatureCount int    `json:"featureCount,omitempty"` // 特性数量 (用于前端显示)
}

// DeviceExportRequest 设备导出请求
// swagger:model
type DeviceExportRequest struct {
	// 导出格式，支持 csv, excel
	Format string `json:"format" example:"csv" swagger:"description=导出格式"`
	// 是否包含详细信息
	IncludeDetails bool `json:"include_details" example:"true" swagger:"description=是否包含详细信息"`
}
