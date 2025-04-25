package resource_report

import (
	"navy-ng/models/portal"
)

// DateFormat defines the standard date format used in the report.
const DateFormat = "2006-01-02"

// snapshotQueryResult is a struct to hold the necessary fields from the join query
type snapshotQueryResult struct {
	portal.ResourceSnapshot        // Embed ResourceSnapshot fields
	ClusterName             string `gorm:"column:cluster_name"` // Explicitly map the joined cluster name
	ResourceType            string // Type of resource (total, intel_common, arm_common, etc.)
	NodeType                string // Type of node
}

// ClusterResourceSummary holds aggregated resource data for a single cluster.
type ClusterResourceSummary struct {
	ClusterName         string
	TotalNodes          int
	TotalCPURequest     float64                  // in cores
	TotalMemoryRequest  float64                  // in GiB
	TotalCPUCapacity    float64                  // Total CPU capacity in cores
	TotalMemoryCapacity float64                  // Total Memory capacity in GiB
	ResourcePools       []ResourcePool           // 添加ResourcePools字段
	ResourcePoolsByType map[string]*ResourcePool // 根据资源池类型快速查找资源池
	// Additional usage percentage fields
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	// Optional fields that may be used by the template but not directly set
	NodesData []NodeResourceDetail
	// Additional fields for physical/virtual nodes
	PhysicalNodes int // 物理节点数量
	VirtualNodes  int // 虚拟节点数量
}

// ResourcePool 资源池详情
type ResourcePool struct {
	ResourceType      string
	NodeType          string
	Nodes             int
	CPUCapacity       float64
	MemoryCapacity    float64
	CPURequest        float64
	MemoryRequest     float64
	BMCount           int
	VMCount           int
	PodCount          int     // 新增Pod数量字段
	PerNodeCpuRequest float64 // 新增节点平均CPU分配
	PerNodeMemRequest float64 // 新增节点平均内存分配
	// 7天资源波动历史数据
	CPUHistory    []float64 // CPU使用率历史数据
	MemoryHistory []float64 // 内存使用率历史数据
	// Additional usage percentage fields
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	// 资源池描述提示信息
	TooltipText string // 资源池类型的tooltip文本，用于前端展示
	// 过去24小时平均CPU和内存最大使用率
	MaxCpuUsageRatio    float64 // 平均CPU最大使用率，存储为小数，如0.36代表百分之36
	MaxMemoryUsageRatio float64 // 平均内存最大使用率，存储为小数，如0.36代表百分之36
}

// NodeResourceDetail holds resource data for a single node.
type NodeResourceDetail struct {
	NodeName          string
	CPURequest        float64
	MemoryRequest     float64
	CPULimit          float64
	MemoryLimit       float64
	CPUUsage          float64
	MemoryUsage       float64
	CPUAllocatable    float64
	MemoryAllocatable float64
}

// ClusterStats 集群统计信息
type ClusterStats struct {
	TotalClusters     int     // 总已巡检集群数
	NormalClusters    int     // 正常集群数
	AbnormalClusters  int     // 异常集群数
	GeneralPodDensity float64 // 通用集群Pod密度
}

// ReportTemplateData structures the fetched data for the HTML template.
type ReportTemplateData struct {
	ReportDate           string
	Clusters             []ClusterResourceSummary
	Stats                ClusterStats // 添加集群统计信息
	HasHighUsageClusters bool         // 是否存在高使用率集群（CPU或内存使用率>=70%）
	Environment          string       // 环境类型："prd" 或 "test"
	// Add any other global data needed for the template
}
