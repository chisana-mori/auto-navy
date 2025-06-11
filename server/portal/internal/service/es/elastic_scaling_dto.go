package es

import (
	"time"
)

// 构建DTO (Data Transfer Object) 用于服务层与控制器层的数据交换

// StrategyDTO 弹性伸缩策略DTO
type StrategyDTO struct {
	ID                     int      `json:"id,omitempty"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	ThresholdTriggerAction string   `json:"thresholdTriggerAction"` // pool_entry 或 pool_exit
	CPUThresholdValue      *float64 `json:"cpuThresholdValue"`
	CPUThresholdType       *string  `json:"cpuThresholdType"` // usage 或 allocated
	CPUTargetValue         *float64 `json:"cpuTargetValue"`   // 动作执行后CPU目标使用率
	MemoryThresholdValue   *float64 `json:"memoryThresholdValue"`
	MemoryThresholdType    *string  `json:"memoryThresholdType"` // usage 或 allocated
	MemoryTargetValue      *float64 `json:"memoryTargetValue"`   // 动作执行后内存目标使用率
	ConditionLogic         string   `json:"conditionLogic"`      // AND 或 OR
	DurationMinutes        int      `json:"durationMinutes"`
	CooldownMinutes        int      `json:"cooldownMinutes"`

	ResourceTypes string    `json:"resourceTypes"` // 资源类型列表，逗号分隔
	Status        string    `json:"status"`        // enabled 或 disabled
	CreatedBy     string    `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty"`
	ClusterIDs    []int     `json:"clusterIds"` // 关联的集群ID列表
}

// StrategyListItemDTO 策略列表项
type StrategyListItemDTO struct {
	ID                     int       `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	ThresholdTriggerAction string    `json:"thresholdTriggerAction"`
	CPUThresholdValue      *float64  `json:"cpuThresholdValue"`
	CPUThresholdType       *string   `json:"cpuThresholdType"`
	CPUTargetValue         *float64  `json:"cpuTargetValue"`
	MemoryThresholdValue   *float64  `json:"memoryThresholdValue"`
	MemoryThresholdType    *string   `json:"memoryThresholdType"`
	MemoryTargetValue      *float64  `json:"memoryTargetValue"`
	ConditionLogic         string    `json:"conditionLogic"`
	ResourceTypes          string    `json:"resourceTypes"`
	DurationMinutes        int       `json:"durationMinutes"` // 持续时间（分钟）
	CooldownMinutes        int       `json:"cooldownMinutes"`
	Status                 string    `json:"status"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	Clusters               []string  `json:"clusters"` // 关联的集群名称列表
}

// StrategyDetailDTO 策略详情
type StrategyDetailDTO struct {
	StrategyDTO
	ExecutionHistory []StrategyExecutionHistoryDTO `json:"executionHistory"`
	RelatedOrders    []OrderListItemDTO            `json:"relatedOrders"`
}

// StrategyExecutionHistoryDTO 策略执行历史
type StrategyExecutionHistoryDTO struct {
	ID             int       `json:"id"`
	ClusterID      int       `json:"clusterId"`
	ResourceType   string    `json:"resourceType"`
	ExecutionTime  time.Time `json:"executionTime"`
	TriggeredValue string    `json:"triggeredValue"`
	ThresholdValue string    `json:"thresholdValue"`
	Result         string    `json:"result"`
	OrderID        *int      `json:"orderId"`
	Reason         string    `json:"reason"`
}

// StrategyExecutionHistoryDetailDTO 策略执行历史详情（包含策略名和集群名）
type StrategyExecutionHistoryDetailDTO struct {
	ID             int       `json:"id"`
	StrategyID     int       `json:"strategyId"`
	StrategyName   string    `json:"strategyName"`
	ClusterID      int       `json:"clusterId"`
	ClusterName    string    `json:"clusterName"`
	ResourceType   string    `json:"resourceType"`
	ExecutionTime  time.Time `json:"executionTime"`
	TriggeredValue string    `json:"triggeredValue"`
	ThresholdValue string    `json:"thresholdValue"`
	Result         string    `json:"result"`
	OrderID        *int      `json:"orderId"`
	HasOrder       bool      `json:"hasOrder"`
	Reason         string    `json:"reason"`
}

// OrderDTO 弹性伸缩订单DTO
type OrderDTO struct {
	ID               int    `json:"id,omitempty"`
	OrderNumber      string `json:"orderNumber"`
	Name             string `json:"name"`        // 订单名称
	Description      string `json:"description"` // 订单描述
	ClusterID        int    `json:"clusterId"`
	ClusterName      string `json:"clusterName,omitempty"`
	StrategyID       *int   `json:"strategyId"`
	StrategyName     string `json:"strategyName,omitempty"`
	ActionType       string `json:"actionType"`       // pool_entry, pool_exit, maintenance_request, maintenance_uncordon
	ResourcePoolType string `json:"resourcePoolType"` // 资源池类型
	Status           string `json:"status"`
	DeviceCount      int    `json:"deviceCount"`
	// DeviceID字段已移除，使用Devices列表和OrderDevice关联表
	DeviceInfo             *DeviceDTO             `json:"deviceInfo,omitempty"`
	Executor               string                 `json:"executor"`
	ExecutionTime          *time.Time             `json:"executionTime"`
	CreatedBy              string                 `json:"createdBy"`
	CreatedAt              time.Time              `json:"createdAt"`
	CompletionTime         *time.Time             `json:"completionTime"`
	FailureReason          string                 `json:"failureReason"`
	MaintenanceStartTime   *time.Time             `json:"maintenanceStartTime,omitempty"`
	MaintenanceEndTime     *time.Time             `json:"maintenanceEndTime,omitempty"`
	ExternalTicketID       string                 `json:"externalTicketId,omitempty"`
	Devices                []int                  `json:"devices,omitempty"`   // 设备ID列表
	ExtraInfo              map[string]interface{} `json:"extraInfo,omitempty"` // 额外信息，用于存储维护原因等
	StrategyTriggeredValue string                 `json:"strategyTriggeredValue,omitempty"`
	StrategyThresholdValue string                 `json:"strategyThresholdValue,omitempty"`
}

// OrderListItemDTO 订单列表项
type OrderListItemDTO struct {
	ID               int       `json:"id"`
	OrderNumber      string    `json:"orderNumber"`
	Name             string    `json:"name"`        // 订单名称
	Description      string    `json:"description"` // 订单描述
	ClusterID        int       `json:"clusterId"`
	ClusterName      string    `json:"clusterName"`
	StrategyID       *int      `json:"strategyId"`
	StrategyName     string    `json:"strategyName"`
	ActionType       string    `json:"actionType"`
	ResourcePoolType string    `json:"resourcePoolType"` // 资源池类型
	Status           string    `json:"status"`
	DeviceCount      int       `json:"deviceCount"`
	CreatedBy        string    `json:"createdBy"`
	CreatedAt        time.Time `json:"createdAt"`
}

// OrderDetailDTO 订单详情
type OrderDetailDTO struct {
	OrderDTO
	Devices []DeviceDTO `json:"devices"` // 涉及设备的详细信息
}

// DeviceDTO 设备DTO
type DeviceDTO struct {
	ID           int     `json:"id"`
	CICode       string  `json:"ciCode"`
	IP           string  `json:"ip"`
	ArchType     string  `json:"archType"`
	CPU          float64 `json:"cpu"`
	Memory       float64 `json:"memory"`
	Status       string  `json:"status"`
	Role         string  `json:"role"`
	Cluster      string  `json:"cluster"`
	ClusterID    int     `json:"clusterId"`
	IsSpecial    bool    `json:"isSpecial"`
	FeatureCount int     `json:"featureCount"`
	OrderStatus  string  `json:"orderStatus,omitempty"` // 在订单中的状态
}

// DashboardStatsDTO 工作台概览统计
type DashboardStatsDTO struct {
	StrategyCount              int `json:"strategyCount"`              // 策略总数
	TriggeredTodayCount        int `json:"triggeredTodayCount"`        // 今日已触发策略数
	EnabledStrategyCount       int `json:"enabledStrategyCount"`       // 已启用策略数
	ClusterCount               int `json:"clusterCount"`               // 集群总数
	AbnormalClusterCount       int `json:"abnormalClusterCount"`       // 异常集群数
	PendingOrderCount          int `json:"pendingOrderCount"`          // 待处理订单数
	DeviceCount                int `json:"deviceCount"`                // 设备总数
	AvailableDeviceCount       int `json:"availableDeviceCount"`       // 可用设备数
	InPoolDeviceCount          int `json:"inPoolDeviceCount"`          // 池内设备数
	TargetResourcePoolCount    int `json:"targetResourcePoolCount"`    // 目标巡检资源池数
	InspectedResourcePoolCount int `json:"inspectedResourcePoolCount"` // 已巡检资源池数
}

// 资源类型数据DTO
type ResourceTypeDataDTO struct {
	Timestamps         []time.Time `json:"timestamps"`         // 时间点
	CPUUsageRatio      []float64   `json:"cpuUsageRatio"`      // CPU使用率
	CPUAllocationRatio []float64   `json:"cpuAllocationRatio"` // CPU分配率
	MemUsageRatio      []float64   `json:"memUsageRatio"`      // 内存使用率
	MemAllocationRatio []float64   `json:"memAllocationRatio"` // 内存分配率
}

// ResourceAllocationTrendDTO 资源分配趋势
type ResourceAllocationTrendDTO struct {
	Timestamps         []time.Time                     `json:"timestamps"`         // 时间点
	CPUUsageRatio      []float64                       `json:"cpuUsageRatio"`      // CPU使用率
	CPUAllocationRatio []float64                       `json:"cpuAllocationRatio"` // CPU分配率
	MemUsageRatio      []float64                       `json:"memUsageRatio"`      // 内存使用率
	MemAllocationRatio []float64                       `json:"memAllocationRatio"` // 内存分配率
	ResourceTypes      []string                        `json:"resourceTypes"`      // 查询的资源类型列表
	ResourceTypeData   map[string]*ResourceTypeDataDTO `json:"resourceTypeData"`   // 每种资源类型的数据
}

// OrderStatsDTO 订单统计
type OrderStatsDTO struct {
	TotalCount      int `json:"totalCount"`      // 总订单数
	PendingCount    int `json:"pendingCount"`    // 待处理订单数
	ProcessingCount int `json:"processingCount"` // 处理中订单数
	CompletedCount  int `json:"completedCount"`  // 已完成订单数
	FailedCount     int `json:"failedCount"`     // 失败订单数
	CancelledCount  int `json:"cancelledCount"`  // 已取消订单数
}
