package service

// FilterOptionResponse 筛选选项响应
// swagger:model
type FilterOptionResponse struct {
	ID       string `json:"id" example:"ip" swagger:"description=选项ID"`
	Label    string `json:"label" example:"IP地址" swagger:"description=选项标签"`
	Value    string `json:"value" example:"ip" swagger:"description=选项值"`
	DbColumn string `json:"dbColumn,omitempty" example:"d.ip" swagger:"description=数据库列名"`
}

// DeviceQueryFilterOptionsResponse 设备查询筛选选项响应
// swagger:model
type DeviceQueryFilterOptionsResponse struct {
	DeviceFields []FilterOptionResponse `json:"deviceFields" swagger:"description=设备字段选项列表"`
	NodeLabels   []FilterOptionResponse `json:"nodeLabels" swagger:"description=节点标签选项列表"`
	NodeTaints   []FilterOptionResponse `json:"nodeTaints" swagger:"description=节点污点选项列表"`
}

// DeviceQueryLabelValuesResponse 标签值响应
// swagger:model
type DeviceQueryLabelValuesResponse struct {
	Values []FilterOptionResponse `json:"values" swagger:"description=标签值列表"`
}

// DeviceQueryTaintValuesResponse 污点值响应
// swagger:model
type DeviceQueryTaintValuesResponse struct {
	Values []FilterOptionResponse `json:"values" swagger:"description=污点值列表"`
}

// DeviceQueryResponseDTO 设备查询响应
// swagger:model
type DeviceQueryResponseDTO struct {
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

// DeviceQueryListResponseDTO 设备查询列表响应
// swagger:model
type DeviceQueryListResponseDTO struct {
	List  []DeviceQueryResponseDTO `json:"list" swagger:"description=设备列表"`
	Total int64                    `json:"total" example:"100" swagger:"description=总数"`
	Page  int                      `json:"page" example:"1" swagger:"description=当前页码"`
	Size  int                      `json:"size" example:"10" swagger:"description=每页数量"`
}

// FilterBlock 筛选块
// swagger:model
type FilterBlockRequest struct {
	ID            string          `json:"id" example:"block1" swagger:"description=筛选块ID"`
	Type          FilterType      `json:"type" example:"device" swagger:"description=筛选类型(device/nodeLabel/taint)"`
	ConditionType ConditionType   `json:"conditionType" example:"equal" swagger:"description=条件类型(equal/notEqual/contains/notContains/exists/notExists/in/notIn)"`
	Key           string          `json:"key" example:"ip" swagger:"description=键"`
	Value         string          `json:"value" example:"192.168.1.1" swagger:"description=值"`
	Operator      LogicalOperator `json:"operator" example:"and" swagger:"description=与下一个条件的逻辑关系(and/or)"`
}

// FilterGroup 筛选组
// swagger:model
type FilterGroupRequest struct {
	ID       string               `json:"id" example:"group1" swagger:"description=筛选组ID"`
	Blocks   []FilterBlockRequest `json:"blocks" swagger:"description=筛选块列表"`
	Operator LogicalOperator      `json:"operator" example:"and" swagger:"description=与下一个组的逻辑关系(and/or)"`
}

// DeviceQueryRequest 设备查询请求
// swagger:model
type DeviceQueryRequestDTO struct {
	Groups []FilterGroupRequest `json:"groups" swagger:"description=筛选组列表"`
	Page   int                  `json:"page" example:"1" swagger:"description=页码"`
	Size   int                  `json:"size" example:"10" swagger:"description=每页数量"`
}

// QueryTemplate 查询模板
// swagger:model
type QueryTemplateRequest struct {
	ID          int64                `json:"id,omitempty" example:"1" swagger:"description=模板ID,新增时不需要传"`
	Name        string               `json:"name" example:"生产环境设备" swagger:"description=模板名称"`
	Description string               `json:"description" example:"查询所有生产环境的设备" swagger:"description=模板描述"`
	Groups      []FilterGroupRequest `json:"groups" swagger:"description=筛选组列表"`
}

// QueryTemplate 查询模板响应
// swagger:model
type QueryTemplateResponse struct {
	ID          int64                `json:"id" example:"1" swagger:"description=模板ID"`
	Name        string               `json:"name" example:"生产环境设备" swagger:"description=模板名称"`
	Description string               `json:"description" example:"查询所有生产环境的设备" swagger:"description=模板描述"`
	Groups      []FilterGroupRequest `json:"groups" swagger:"description=筛选组列表"`
	CreatedAt   string               `json:"createdAt" example:"2024-01-01T12:00:00Z" swagger:"description=创建时间"`
	UpdatedAt   string               `json:"updatedAt" example:"2024-01-01T12:30:00Z" swagger:"description=更新时间"`
}

// QueryTemplateListResponse 查询模板列表响应
// swagger:model
type QueryTemplateListResponse struct {
	List  []QueryTemplateResponse `json:"list" swagger:"description=模板列表"`
	Total int64                   `json:"total" example:"10" swagger:"description=总数"`
}
