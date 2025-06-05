package service

import (
	"fmt"
	"math"
	"navy-ng/models/portal"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

const DEFAULT_CLUSTER_DESC_MATCHER = "(?i)(.*TOOL.*|.*GEN-BIZ.*|.*DSU.*｜.*APLUS.*|.*SANDBOX.*)"

// ClusterResourceService handles cluster resource calculations
type ClusterResourceService struct {
	db *gorm.DB
}

// NewClusterResourceService creates a new cluster resource service instance
func NewClusterResourceService(db *gorm.DB) *ClusterResourceService {
	return &ClusterResourceService{
		db: db,
	}
}

// CalculateRemainingResources calculates remaining cluster resources based on business logic
// Now includes non-pooled device resources in the Pending field of SecurityZoneResourceDTO
// Accepts optional date parameter and description filter, defaults to today if not provided
func (s *ClusterResourceService) CalculateRemainingResources(queryDate *time.Time, descFilter *string) (*ClusterResourceDTO, error) {
	// Use today if no date specified
	var targetDate time.Time
	if queryDate == nil {
		targetDate = time.Now()
	} else {
		targetDate = *queryDate
	}

	// Step 1: Calculate clustered resources
	clusterAggregated, err := s.calculateClusteredResources(targetDate, descFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate clustered resources: %w", err)
	}

	// Step 2: Calculate non-pooled device resources
	deviceAggregated, err := s.calculateNonPooledDeviceResources()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate non-pooled device resources: %w", err)
	}

	// Step 3: Build response structure with device resources in Pending field
	response := s.buildResponseStructureWithPendingField(clusterAggregated, deviceAggregated)

	return &ClusterResourceDTO{
		Code:    0,
		Message: "Success",
		List:    response,
	}, nil
}

// calculateClusteredResources calculates remaining resources from clustered machines
func (s *ClusterResourceService) calculateClusteredResources(queryDate time.Time, descFilter *string) (map[string]map[string]*AggregatedResourceData, error) {
	// Set up date range for the specified date (start of day to end of day)
	startOfDay := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 0, 0, 0, 0, queryDate.Location())
	endOfDay := time.Date(queryDate.Year(), queryDate.Month(), queryDate.Day(), 23, 59, 59, 999999999, queryDate.Location())

	// Define a struct to hold the joined data
	type SnapshotWithCluster struct {
		// Snapshot fields
		ClusterID      uint    `gorm:"column:cluster_id"`
		MemoryCapacity float64 `gorm:"column:mem_capacity"`
		MemRequest     float64 `gorm:"column:mem_request"`
		ResourcePool   string  `gorm:"column:resource_pool"`
		ResourceType   string  `gorm:"column:resource_type"`

		// Cluster fields
		IDC  string `gorm:"column:idc"`
		Zone string `gorm:"column:zone"`
		Desc string `gorm:"column:desc"`
	}

	// Build the query with optional description filter
	query := s.db.Table("k8s_cluster_resource_snapshot as s").
		Select("s.cluster_id, s.mem_capacity, s.mem_request, s.resource_pool, s.resource_type, c.idc, c.zone, c.desc").
		Joins("INNER JOIN k8s_cluster as c ON s.cluster_id = c.id").
		Where("s.created_at BETWEEN ? AND ? AND s.resource_type = ?", startOfDay, endOfDay, "hg_common")

	// Add description filter if provided (using regex pattern matching)
	if descFilter != nil && *descFilter != "" {
		// Use database-specific regex syntax (MySQL/MariaDB uses REGEXP, PostgreSQL uses ~)
		// For SQLite, we'll use LIKE with wildcards as a fallback
		query = query.Where("c.desc REGEXP ?", *descFilter)
	}

	// Execute the optimized query: JOIN snapshots with clusters in a single query
	var results []SnapshotWithCluster
	err := query.Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resource snapshots with cluster info: %w", err)
	}

	// Calculate remaining resources for each resource pool
	var calculations []ResourcePoolCalculation

	for _, result := range results {
		// Business logic: Total capacity * 0.75 must be greater than request
		capacityThreshold := result.MemoryCapacity * 0.75

		calculation := ResourcePoolCalculation{
			ClusterID:     result.ClusterID,
			IDC:           result.IDC,
			Zone:          s.normalizeClusterZone(result.Zone),
			ResourcePool:  result.ResourcePool,
			TotalCapacity: result.MemoryCapacity,
			Request:       result.MemRequest,
			IsEligible:    capacityThreshold > result.MemRequest,
		}

		// Calculate remaining resource if eligible
		if calculation.IsEligible {
			calculation.RemainingMem = capacityThreshold - result.MemRequest
		}

		calculations = append(calculations, calculation)
	}

	// Aggregate by IDC and Zone
	return s.aggregateResourcesByIDCAndZone(calculations), nil
}

// calculateNonPooledDeviceResources calculates total memory from non-pooled devices for merging into zones
func (s *ClusterResourceService) calculateNonPooledDeviceResources() (map[string]map[string]float64, error) {
	// Query non-pooled devices based on filtering criteria
	var devices []portal.Device
	err := s.db.Where("cluster = ? AND ci_code LIKE ? AND appid IN (?, ?) AND UPPER(arch_type) = ? AND is_localization = ?",
		"", "%EQUHST%", "85004", "85494", "X86", true).
		Find(&devices).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch non-pooled devices: %w", err)
	}

	// Group devices by IDC and Zone, calculate total memory (not * 0.75)
	aggregated := make(map[string]map[string]float64)

	for _, device := range devices {
		normalizedZone := s.normalizeDeviceZone(device.NetZone)

		// Initialize IDC map if not exists
		if aggregated[device.IDC] == nil {
			aggregated[device.IDC] = make(map[string]float64)
		}

		// Add total memory to the aggregated data
		aggregated[device.IDC][normalizedZone] += device.Memory
	}

	return aggregated, nil
}

// classifyOrganization classifies IDC into organization types based on naming patterns
func (s *ClusterResourceService) classifyOrganization(idc string) string {
	// Convert to lowercase for case-insensitive matching
	idcLower := strings.ToLower(idc)

	// Check for 理财子公司 (Wealth Management Subsidiary)
	if strings.Contains(idcLower, "lc") {
		return "理财子公司"
	}

	// Check for 港分 (Hong Kong Branch)
	if strings.Contains(idcLower, "hk") {
		return "港分"
	}

	// Default to 总行 (Head Office)
	return "总行"
}

// normalizeZone converts zone names to fixed zone values using regex patterns
// Supports both cluster and device zones with case-insensitive matching
func (s *ClusterResourceService) normalizeZone(zoneName string) string {
	// Handle empty or nil input
	if zoneName == "" {
		return zoneName
	}

	// Define comprehensive regex patterns for all zone types
	// APP zone patterns: covers cluster (central, testcore, devcore) and device (app, application) patterns
	appPattern := regexp.MustCompile(`(?i)(central|app|application|testcore|devcore|应用区)`)

	// COREDB zone patterns: covers both cluster and device patterns
	coredbPattern := regexp.MustCompile(`(?i)(coredb|core_database|核心数据库区)`)

	// DB zone patterns: covers both exact "db" match and database-related terms
	dbPattern := regexp.MustCompile(`(?i)(^db$|database|数据库区)`)

	// MGT zone patterns: covers management-related terms
	mgtPattern := regexp.MustCompile(`(?i)(mgt|management|管理区)`)

	// Match patterns and return standardized zone (case-insensitive)
	if appPattern.MatchString(zoneName) {
		return "APP"
	}
	if coredbPattern.MatchString(zoneName) {
		return "COREDB"
	}
	if dbPattern.MatchString(zoneName) {
		return "DB"
	}
	if mgtPattern.MatchString(zoneName) {
		return "MGT"
	}

	// Return original value if no pattern matches
	return zoneName
}

// normalizeClusterZone converts cluster.zone to fixed zone values (wrapper for backward compatibility)
func (s *ClusterResourceService) normalizeClusterZone(clusterZone string) string {
	return s.normalizeZone(clusterZone)
}

// normalizeDeviceZone converts device.net_zone to fixed zone values (wrapper for backward compatibility)
func (s *ClusterResourceService) normalizeDeviceZone(deviceZone string) string {
	return s.normalizeZone(deviceZone)
}

// aggregateResourcesByIDCAndZone aggregates remaining resources by IDC and Zone
func (s *ClusterResourceService) aggregateResourcesByIDCAndZone(calculations []ResourcePoolCalculation) map[string]map[string]*AggregatedResourceData {
	// Map structure: IDC -> Zone -> AggregatedResourceData
	aggregated := make(map[string]map[string]*AggregatedResourceData)

	for _, calc := range calculations {
		if !calc.IsEligible {
			continue // Skip ineligible resource pools
		}

		// Initialize IDC map if not exists
		if aggregated[calc.IDC] == nil {
			aggregated[calc.IDC] = make(map[string]*AggregatedResourceData)
		}

		// Initialize Zone data if not exists
		if aggregated[calc.IDC][calc.Zone] == nil {
			aggregated[calc.IDC][calc.Zone] = &AggregatedResourceData{
				IDC:            calc.IDC,
				Zone:           calc.Zone,
				TotalRemaining: 0,
				AvailableCount: 0,
			}
		}

		// Add remaining memory to the aggregated data
		aggregated[calc.IDC][calc.Zone].TotalRemaining += calc.RemainingMem
	}

	// Calculate available count (TotalRemaining / 8)
	for idc := range aggregated {
		for zone := range aggregated[idc] {
			data := aggregated[idc][zone]
			data.AvailableCount = int64(math.Floor(data.TotalRemaining / 8))
		}
	}

	return aggregated
}


// buildResponseStructureWithPendingField builds the final response structure with device resources in Pending field
func (s *ClusterResourceService) buildResponseStructureWithPendingField(
	clusterAggregated map[string]map[string]*AggregatedResourceData,
	deviceAggregated map[string]map[string]float64,
) []OrganizationResourceDTO {
	// Build organization resource map with pending field
	orgResourceMap := s.buildOrganizationResourceMapWithPendingField(clusterAggregated, deviceAggregated)

	// Convert to final result list
	return convertOrgMapToListWithPendingField(orgResourceMap)
}

// buildOrganizationResourceMapWithPendingField builds organization resource map with device resources in Pending field
func (s *ClusterResourceService) buildOrganizationResourceMapWithPendingField(
	clusterAggregated map[string]map[string]*AggregatedResourceData,
	deviceAggregated map[string]map[string]float64,
) map[string]map[string][]SecurityZoneResourceDTO {
	// Initialize organization mapping
	orgMap := make(map[string]map[string][]SecurityZoneResourceDTO)

	// First, process cluster resources
	for idc, zones := range clusterAggregated {
		organization := s.classifyOrganization(idc)

		// Ensure organization mapping is initialized
		if orgMap[organization] == nil {
			orgMap[organization] = make(map[string][]SecurityZoneResourceDTO)
		}

		// Build security zones from cluster aggregated data
		var securityZones []SecurityZoneResourceDTO
		for _, data := range zones {
			securityZone := SecurityZoneResourceDTO{
				SecurityZone:   data.Zone,
				AvailableMem:   fmt.Sprintf("%.2fGiB", data.TotalRemaining),
				AvailableCount: fmt.Sprintf("%d", data.AvailableCount),
				Pending:        "0.00GiB", // Initialize with 0, will be updated if device data exists
			}
			securityZones = append(securityZones, securityZone)
		}

		orgMap[organization][idc] = securityZones
	}

	// Then, add device resources to Pending field
	for idc, zones := range deviceAggregated {
		organization := s.classifyOrganization(idc)

		// Ensure organization mapping is initialized
		if orgMap[organization] == nil {
			orgMap[organization] = make(map[string][]SecurityZoneResourceDTO)
		}

		// Get existing zones for this IDC or create new ones
		existingZones := orgMap[organization][idc]
		zoneMap := make(map[string]*SecurityZoneResourceDTO)

		// Create a map for easy lookup of existing zones
		for i := range existingZones {
			zoneMap[existingZones[i].SecurityZone] = &existingZones[i]
		}

		// Add device memory to Pending field
		for zone, deviceMemory := range zones {
			if existingZone, exists := zoneMap[zone]; exists {
				// Update existing zone's Pending field
				existingZone.Pending = fmt.Sprintf("%.2fGiB", deviceMemory)
			} else {
				// Create new zone with only device resources
				newZone := SecurityZoneResourceDTO{
					SecurityZone:   zone,
					AvailableMem:   "0.00GiB",
					AvailableCount: "0",
					Pending:        fmt.Sprintf("%.2fGiB", deviceMemory),
				}
				existingZones = append(existingZones, newZone)
			}
		}

		orgMap[organization][idc] = existingZones
	}

	return orgMap
}

// convertOrgMapToListWithPendingField converts organization resource map to final DTO list with Pending field
func convertOrgMapToListWithPendingField(
	orgMap map[string]map[string][]SecurityZoneResourceDTO,
) []OrganizationResourceDTO {
	var result []OrganizationResourceDTO

	for organization, idcs := range orgMap {
		var idcGroups []IDCResourceGroupDTO

		for idcName, zones := range idcs {
			// Only add IDCs with data
			if len(zones) > 0 {
				idcGroup := IDCResourceGroupDTO{
					IDCName: idcName,
					Zones:   zones,
				}
				idcGroups = append(idcGroups, idcGroup)
			}
		}

		// Only add organizations with IDCs
		if len(idcGroups) > 0 {
			orgResource := OrganizationResourceDTO{
				Organization: organization,
				IDCs:         idcGroups,
			}
			result = append(result, orgResource)
		}
	}

	return result
}
