package service

// ResourcePoolAllocationRateDTO represents the allocation rate response for a specific resource pool
type ResourcePoolAllocationRateDTO struct {
	ClusterName  string  `json:"cluster_name"`  // 集群名称
	ResourcePool string  `json:"resource_pool"` // 资源池名称
	CPURate      float64 `json:"cpu_rate"`      // CPU分配率 (0-100)
	MemoryRate   float64 `json:"memory_rate"`   // 内存分配率 (0-100)
	CPURequest   float64 `json:"cpu_request"`   // CPU请求量 (cores)
	CPUCapacity  float64 `json:"cpu_capacity"`  // CPU容量 (cores)
	MemRequest   float64 `json:"mem_request"`   // 内存请求量 (GiB)
	MemCapacity  float64 `json:"mem_capacity"`  // 内存容量 (GiB)
	QueryDate    string  `json:"query_date"`    // 查询日期 (YYYY-MM-DD)
}