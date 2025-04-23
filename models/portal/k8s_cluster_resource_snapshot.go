package portal

type ResourceType string

const (
	Total ResourceType = "total" // 总资源
	HG    ResourceType = "total_hg" // 总海光
	Intel ResourceType = "total_intel" // 总intel机器
	ARM   ResourceType = "total_arm" // 总arm

	WithTaint      ResourceType = "total_taint" //带污点机器
	IntelWithTaint ResourceType = "intel_taint" // intel污点机器
	ArmWithTaint   ResourceType = "arm_taint" // arm污点机器
	HgWithTaint    ResourceType = "hg_taint" // 海光污点机器

	Common      ResourceType = "total_common" // 总通用
	IntelCommon ResourceType = "intel_common" // intel通用
	ArmCommon   ResourceType = "arm_common" // arm通用
	HgCommon    ResourceType = "hg_common" // 海光通用

	GPU         ResourceType = "total_gpu" // 总gpu节点
	IntelGPU    ResourceType = "intel_gpu" // intelGPU节点
	ArmGPU      ResourceType = "arm_gpu" // armGPU节点
	IntelNonGPU ResourceType = "intel_non_gpu" // intel且无gpu

	Aplus      ResourceType = "aplus_total" // A+总资源
	AplusHg    ResourceType = "aplus_hg"  // A+海光
	AplusIntel ResourceType = "aplus_intel" // A+Intel
	AplusArm   ResourceType = "aplus_arm" // A+arm

	Dplus      ResourceType = "dplus_total" // D+总资源
	DplusHg    ResourceType = "dplus_hg" //D+海光
	DplusIntel ResourceType = "dplus_intel" //D+Intel
	DplusArm   ResourceType = "dplus_arm" // D+arm
)

type ResourceSnapshot struct {
	BaseModel
	CpuCapacity         float64 `gorm:"column:cpu_capacity"`
	MemoryCapacity      float64 `gorm:"column:mem_capacity"`
	CpuRequest          float64 `gorm:"column:cpu_request"`
	MemRequest          float64 `gorm:"column:mem_request"`
	NodeCount           int64   `gorm:"column:node_count"`
	BMCount             int64   `gorm:"column:bm_count"`
	VMCount             int64   `gorm:"column:vm_count"`
	MaxCpuUsageRatio    float64 `gorm:"column:max_cpu"`
	MaxMemoryUsageRatio float64 `gorm:"column:max_memory"`
	ClusterID           uint    `gorm:"column:cluster_id"`
	PerNodeCpuRequest   float64 `gorm:"column:per_node_cpu_req"`
	PerNodeMemRequest   float64 `gorm:"column:per_node_mem_req"`
	ResourceType        string  `gorm:"column:resource_type"`
}

func (ResourceSnapshot) TableName() string {
	return "k8s_cluster_resource_snapshot"
}
