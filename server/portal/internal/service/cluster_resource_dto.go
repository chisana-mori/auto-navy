package service

// ClusterResourceDTO represents the response structure for cluster resource calculations
type ClusterResourceDTO struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	List    []OrganizationResourceDTO `json:"list"`
}

// OrganizationResourceDTO represents resources grouped by organization (总行/理财子公司/港分)
type OrganizationResourceDTO struct {
	Organization string                `json:"organization"`
	IDCs         []IDCResourceGroupDTO `json:"idcs"`
}

// IDCResourceGroupDTO represents resources grouped by IDC with dynamic IDC name
type IDCResourceGroupDTO struct {
	IDCName string                    `json:"idc_name"`
	Zones   []SecurityZoneResourceDTO `json:"zones"`
}

// SecurityZoneResourceDTO represents resources for a specific security zone
type SecurityZoneResourceDTO struct {
	SecurityZone   string `json:"security_zone"`
	AvailableMem   string `json:"available_mem"`
	AvailableCount string `json:"available_count"`

	Pending string `json:"pending"`
}

// PendingResourceDTO represents pending (non-pooled) device resources
type PendingResourceDTO struct {
	SecurityZone string `json:"security_zone"`
	TotalMemory  string `json:"total_memory"`
}

// AggregatedResourceData represents aggregated resource data for internal calculations
type AggregatedResourceData struct {
	IDC            string
	Zone           string
	TotalRemaining float64
	AvailableCount int64
}

// ResourcePoolCalculation represents calculation data for a single resource pool
type ResourcePoolCalculation struct {
	ClusterID     uint
	IDC           string
	Zone          string
	ResourcePool  string
	TotalCapacity float64
	Request       float64
	RemainingMem  float64
	IsEligible    bool
}
