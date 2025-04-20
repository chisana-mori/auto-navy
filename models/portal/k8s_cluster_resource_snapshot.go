package portal

type ResourceType string

const (
	Total ResourceType = "total"
	HG    ResourceType = "total_hg"
	Intel ResourceType = "total_intel"
	ARM   ResourceType = "total_arm"

	WithTaint      ResourceType = "total_taint"
	IntelWithTaint ResourceType = "intel_taint"
	ArmWithTaint   ResourceType = "arm_taint"
	HgWithTaint    ResourceType = "hg_taint"

	Common      ResourceType = "total_common"
	IntelCommon ResourceType = "intel_common"
	ArmCommon   ResourceType = "arm_common"
	HgCommon    ResourceType = "hg_common"

	GPU         ResourceType = "total_gpu"
	IntelGPU    ResourceType = "intel_gpu"
	ArmGPU      ResourceType = "arm_gpu"
	IntelNonGPU ResourceType = "intel_non_gpu"

	Aplus      ResourceType = "aplus_total"
	AplusHg    ResourceType = "aplus_hg"
	AplusIntel ResourceType = "aplus_intel"
	AplusArm   ResourceType = "aplus_arm"

	Dplus      ResourceType = "dplus_total"
	DplusHg    ResourceType = "dplus_hg"
	DplusIntel ResourceType = "dplus_intel"
	DplusArm   ResourceType = "dplus_arm"
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
