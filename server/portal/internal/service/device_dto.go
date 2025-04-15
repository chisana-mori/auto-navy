package service

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
	ID           int64  `json:"id" example:"1" swagger:"description=设备ID"`
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
