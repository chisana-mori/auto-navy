package resource_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"regexp"
	"sort"
	"strconv" // Add strconv import for custom funcs
	"strings"
	"time"

	"navy-ng/models/portal"

	"github.com/Masterminds/sprig/v3"
	"github.com/jinzhu/now"
	"go.uber.org/zap" // Add zap import
	"gorm.io/gorm"
)

//go:embed template.html
var templateFS embed.FS

// List of general purpose clusters
var defaultGeneralClusters = []string{
	"cluster1",
	"cluster2",
	"cluster3",
}

// 在函数外部定义全局正则表达式
var groupRegex = regexp.MustCompile(`-([^-]+)-`)

// NewResourceReportSender creates a new ResourceReportSender with the given parameters
func NewResourceReportSender(db *gorm.DB, smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string, generalClusterList []string, environment string, logger *zap.Logger) *ResourceReportSender { // Add environment and logger parameters
	// Ensure logger is not nil
	if logger == nil {
		// Fallback to a no-op logger if none provided, or handle error
		logger = zap.NewNop()
	}
	// Validate environment parameter
	if environment != "prd" && environment != "test" {
		logger.Warn("Invalid environment specified, defaulting to 'prd'", zap.String("environment", environment))
		environment = "prd" // Default to production if invalid
	}
	return &ResourceReportSender{
		DB:              db,
		SMTPHost:        smtpHost,
		SMTPPort:        smtpPort,
		SMTPUsername:    smtpUser,
		SMTPPassword:    smtpPassword,
		FromEmail:       fromEmail,
		ToEmails:        toEmails,
		generalClusters: generalClusterList,
		Environment:     environment,
		Logger:          logger, // Assign logger
	}
}

// ResourceReportSender handles the generation and delivery of resource usage reports.
type ResourceReportSender struct {
	DB              *gorm.DB
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	FromEmail       string
	ToEmails        []string
	CCEmails        []string
	DryRun          bool
	TemplatePath    string // Optional custom template path
	ReportDate      time.Time
	ClusterFilter   string      // Optional cluster name filter
	generalClusters []string    // List of general purpose clusters
	Logger          *zap.Logger // Add Logger field
	Environment     string      // Environment: "prd" or "test"
}

// Run executes the resource report email generation and sending process.
func (s *ResourceReportSender) Run(ctx context.Context) error {
	s.Logger.Info("Starting resource report generation")

	// Initialize parameters
	s.initializeParameters()

	// Fetch resource data
	clusters, stats, err := s.fetchClusterResourceData(ctx)
	if err != nil {
		s.Logger.Error("Failed to fetch resource data", zap.Error(err))
		return fmt.Errorf("failed to fetch resource data: %w", err)
	}

	// Process data for the template
	templateData := s.prepareTemplateData(clusters, stats)

	// Generate email content
	emailContent, err := s.generateEmailContent(templateData)
	if err != nil {
		s.Logger.Error("Failed to generate email content", zap.Error(err))
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// Send email or handle dry run
	return s.sendEmailOrHandleDryRun(emailContent)
}

// initializeParameters sets default values for report parameters if not specified.
func (s *ResourceReportSender) initializeParameters() {
	// Set default report date to yesterday if not specified
	if s.ReportDate.IsZero() {
		s.ReportDate = time.Now().AddDate(0, 0, 0)
	}

	// Initialize general clusters if not set
	if len(s.generalClusters) == 0 {
		s.generalClusters = defaultGeneralClusters
	}
}

// prepareTemplateData creates a ReportTemplateData structure from cluster data.
func (s *ResourceReportSender) prepareTemplateData(clusters []ClusterResourceSummary, stats ClusterStats) ReportTemplateData {
	templateData := ReportTemplateData{
		ReportDate:           s.ReportDate.Format(DateFormat),
		Clusters:             clusters,
		Stats:                stats,
		Environment:          s.Environment, // 设置环境类型
		ShowResourcePoolDesc: false,         // 默认不显示资源池描述，因为当前ResourcePool类型没有Desc字段
	}

	// Determine if any clusters have abnormal usage
	templateData.HasHighUsageClusters = s.detectAbnormalUsageClusters(clusters)

	return templateData
}

// detectAbnormalUsageClusters checks if any clusters have abnormal resource usage.
func (s *ResourceReportSender) detectAbnormalUsageClusters(clusters []ClusterResourceSummary) bool {
	for _, cluster := range clusters {
		for _, pool := range cluster.ResourcePools {
			if s.isResourcePoolAbnormal(pool) {
				return true
			}
		}
	}
	return false
}

// sendEmailOrHandleDryRun sends the email or handles dry run mode.
func (s *ResourceReportSender) sendEmailOrHandleDryRun(emailContent string) error {
	if s.DryRun {
		s.Logger.Info("Dry run mode - email content generated but not sent")
		return nil
	}

	// Create the subject line
	subject := fmt.Sprintf("Kubernetes集群资源使用报告 - %s", s.ReportDate.Format(DateFormat))

	return s.sendEmail(subject, emailContent)
}

// fetchClusterResourceData retrieves and processes resource usage data from the database.
func (s *ResourceReportSender) fetchClusterResourceData(ctx context.Context) ([]ClusterResourceSummary, ClusterStats, error) {
	s.Logger.Info("Fetching resource data",
		zap.String("reportDate", s.ReportDate.Format(DateFormat)),
		zap.String("clusterFilter", s.ClusterFilter))

	// Get clusters from database
	clusters, err := s.fetchClusters(ctx)
	if err != nil {
		return nil, ClusterStats{}, err
	}

	if len(clusters) == 0 {
		s.Logger.Warn("No matching clusters found", zap.String("filter", s.ClusterFilter))
		return []ClusterResourceSummary{}, ClusterStats{}, nil
	}

	// Fetch snapshots for the period
	allSnapshots, err := s.fetchSnapshots(ctx)
	if err != nil {
		return nil, ClusterStats{}, err
	}

	// Process data into summaries
	return s.processData(allSnapshots)
}

// fetchClusters retrieves clusters from the database.
func (s *ResourceReportSender) fetchClusters(ctx context.Context) ([]portal.K8sCluster, error) {
	var clusters []portal.K8sCluster

	query := s.DB.WithContext(ctx).Model(&portal.K8sCluster{})
	if s.ClusterFilter != "" {
		query = query.Where("name LIKE ?", "%"+s.ClusterFilter+"%")
	}

	// Only include online clusters with production labels
	query = query.Where("status <> 'Offline'").
		Where("created_at <= ?", s.ReportDate)

	if err := query.Find(&clusters).Error; err != nil {
		s.Logger.Error("Failed to query clusters", zap.Error(err))
		return nil, fmt.Errorf("failed to query clusters: %w", err)
	}

	s.Logger.Info("Processing clusters", zap.Int("count", len(clusters)))

	return clusters, nil
}

// fetchSnapshots retrieves snapshot data for the reporting period.
func (s *ResourceReportSender) fetchSnapshots(ctx context.Context) ([]snapshotQueryResult, error) {
	endDate := now.EndOfDay()
	startDate := s.ReportDate.AddDate(0, 0, -7) // 获取7天前的数据

	s.Logger.Info("Fetching snapshots",
		zap.Time("startDate", startDate),
		zap.Time("endDate", endDate))

	var allSnapshots []snapshotQueryResult

	// Join query to get snapshots with their cluster names for the 7-day period
	query := s.DB.WithContext(ctx).Table("k8s_cluster_resource_snapshot").
		Select("k8s_cluster_resource_snapshot.*, k8s_clusters.clustername AS cluster_name").
		Joins("JOIN k8s_clusters ON k8s_cluster_resource_snapshot.cluster_id = k8s_clusters.id").
		Where("k8s_cluster_resource_snapshot.created_at BETWEEN ? AND ?", startDate, endDate).
		Where("k8s_clusters.status <> 'Offline'")

	if s.ClusterFilter != "" {
		query = query.Where("k8s_clusters.name LIKE ?", "%"+s.ClusterFilter+"%")
	}

	if err := query.Order("k8s_cluster_resource_snapshot.created_at ASC").Find(&allSnapshots).Error; err != nil {
		s.Logger.Error("Failed to query snapshots", zap.Error(err))
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}

	s.Logger.Info("Fetched snapshots for period", zap.Int("count", len(allSnapshots)))

	return allSnapshots, nil
}

// processData processes raw database data into structured summaries.
func (s *ResourceReportSender) processData(
	allSnapshots []snapshotQueryResult,
) ([]ClusterResourceSummary, ClusterStats, error) {
	// Filter snapshots for the specific report date
	latestSnapshots := filterSnapshotsForDate(allSnapshots, s.ReportDate)

	s.Logger.Info("Processing snapshots for report date",
		zap.Int("count", len(latestSnapshots)),
		zap.String("reportDate", s.ReportDate.Format(DateFormat)))

	// Group snapshots for the report date by cluster
	clusterSummaries, stats := s.processSnapshotsToSummaries(latestSnapshots)

	// Fetch and process historical data using all snapshots from the 7-day period
	if err := s.fetchHistoricalData(clusterSummaries, allSnapshots); err != nil {
		s.Logger.Error("Failed to process historical data", zap.Error(err))
		// Continue without historical data, but the report might be incomplete
	}

	// 根据集群名称或标签分配组别和排序优先级
	clusters := s.assignGroupsAndOrder(clusterSummaries)

	return clusters, stats, nil
}

// filterSnapshotsForDate filters a list of snapshots to include only those from a specific date.
func filterSnapshotsForDate(snapshots []snapshotQueryResult, targetDate time.Time) []snapshotQueryResult {
	var filtered []snapshotQueryResult
	startOfDay := targetDate.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24*time.Hour - time.Nanosecond)

	for _, snap := range snapshots {
		snapTime := time.Time(snap.CreatedAt)
		if !snapTime.Before(startOfDay) && !snapTime.After(endOfDay) {
			filtered = append(filtered, snap)
		}
	}
	return filtered
}

// fetchHistoricalData processes 7-day historical snapshot data for each cluster summary.
// It now accepts the full list of snapshots fetched for the period.
func (s *ResourceReportSender) fetchHistoricalData(summaries []ClusterResourceSummary, historicalSnapshots []snapshotQueryResult) error {
	s.Logger.Info("Processing historical data", zap.Int("snapshotCount", len(historicalSnapshots)))

	// Group snapshots by ClusterName and then by Date
	snapshotsByClusterDate := s.groupSnapshotsByClusterAndDate(historicalSnapshots)

	// Iterate through summaries and populate history
	s.populateHistoricalData(summaries, snapshotsByClusterDate)

	return nil
}

// groupSnapshotsByClusterAndDate groups snapshots by cluster name and date.
func (s *ResourceReportSender) groupSnapshotsByClusterAndDate(snapshots []snapshotQueryResult) map[string]map[string]snapshotQueryResult {
	result := make(map[string]map[string]snapshotQueryResult)

	for _, snap := range snapshots {
		snapTime := time.Time(snap.CreatedAt)
		dateStr := snapTime.Format(DateFormat) // Use YYYY-MM-DD format as the key

		if _, ok := result[snap.ClusterName]; !ok {
			result[snap.ClusterName] = make(map[string]snapshotQueryResult)
		}

		// If we already have a snapshot for this cluster on this date, keep the later one
		if existingSnap, ok := result[snap.ClusterName][dateStr]; ok {
			if snapTime.After(time.Time(existingSnap.CreatedAt)) {
				result[snap.ClusterName][dateStr] = snap
			}
		} else {
			result[snap.ClusterName][dateStr] = snap
		}
	}

	return result
}

// populateHistoricalData fills in historical data for each cluster and resource pool.
func (s *ResourceReportSender) populateHistoricalData(summaries []ClusterResourceSummary, snapshotsByClusterDate map[string]map[string]snapshotQueryResult) {
	reportDate := s.ReportDate.Truncate(24 * time.Hour) // Ensure we compare dates correctly

	for i := range summaries {
		clusterName := summaries[i].ClusterName
		dailySnapshots, clusterFound := snapshotsByClusterDate[clusterName]

		for j := range summaries[i].ResourcePools {
			// Initialize history arrays (size 7 for 7 days)
			summaries[i].ResourcePools[j].CPUHistory = make([]float64, 7)
			summaries[i].ResourcePools[j].MemoryHistory = make([]float64, 7)

			// Calculate usage for each day in the 7-day period
			s.calculateDailyUsageForPool(
				&summaries[i].ResourcePools[j],
				dailySnapshots,
				clusterFound,
				reportDate,
			)
		}
	}
}

// calculateDailyUsageForPool computes daily CPU and memory usage for a resource pool.
func (s *ResourceReportSender) calculateDailyUsageForPool(
	pool *ResourcePool,
	dailySnapshots map[string]snapshotQueryResult,
	clusterFound bool,
	reportDate time.Time,
) {
	for dayIndex := 0; dayIndex < 7; dayIndex++ {
		// Calculate the date for the current history index
		currentDate := reportDate.AddDate(0, 0, -(6 - dayIndex)) // dayIndex 0 -> -6 days, dayIndex 6 -> 0 days
		currentDateStr := currentDate.Format(DateFormat)

		cpuUsage := 0.0
		memUsage := 0.0

		if clusterFound {
			if snap, dateFound := dailySnapshots[currentDateStr]; dateFound {
				// Calculate usage based on the snapshot for this day
				cpuUsage = s.calculateCpuUsageFromSnapshot(snap)
				memUsage = s.calculateMemoryUsageFromSnapshot(snap)
			}
		}

		pool.CPUHistory[dayIndex] = cpuUsage
		pool.MemoryHistory[dayIndex] = memUsage
	}
}

// calculateCpuUsageFromSnapshot calculates CPU usage percentage from a snapshot.
func (s *ResourceReportSender) calculateCpuUsageFromSnapshot(snap snapshotQueryResult) float64 {
	if snap.CpuCapacity > 0 {
		return (snap.CpuRequest / snap.CpuCapacity) * 100
	}
	return 0.0
}

// calculateMemoryUsageFromSnapshot calculates memory usage percentage from a snapshot.
func (s *ResourceReportSender) calculateMemoryUsageFromSnapshot(snap snapshotQueryResult) float64 {
	if snap.MemoryCapacity > 0 {
		// Convert MemoryCapacity and MemRequest to the same unit (GiB) before calculating
		memCapacityGiB := float64(snap.MemoryCapacity) / (1024 * 1024 * 1024)
		memRequestGiB := float64(snap.MemRequest) / (1024 * 1024 * 1024)
		if memCapacityGiB > 0 {
			return (memRequestGiB / memCapacityGiB) * 100
		}
	}
	return 0.0
}

// processSnapshotsToSummaries converts raw snapshot data into structured summaries for each cluster.
func (s *ResourceReportSender) processSnapshotsToSummaries(
	snapshots []snapshotQueryResult,
) ([]ClusterResourceSummary, ClusterStats) {
	// Get the latest snapshot for each cluster
	latestSnapshotsMap := s.findLatestSnapshotsPerCluster(snapshots)

	// Process snapshots into cluster summaries
	clusterMap := s.buildClusterSummariesFromSnapshots(latestSnapshotsMap)

	// Calculate usage percentages for all summaries
	s.calculateUsagePercentages(clusterMap)

	// Convert the map to a sorted slice
	clusterSummaries := s.convertToSortedSummaries(clusterMap)

	// Calculate global statistics
	stats := s.calculateClusterStats(clusterSummaries)

	return clusterSummaries, stats
}

// findLatestSnapshotsPerCluster identifies the most recent snapshot for each cluster.
func (s *ResourceReportSender) findLatestSnapshotsPerCluster(snapshots []snapshotQueryResult) map[string][]snapshotQueryResult {
	latestSnapshotsMap := make(map[string][]snapshotQueryResult)

	beginOfToday := now.BeginningOfDay()
	for _, snap := range snapshots {
		snapTime := time.Time(snap.CreatedAt)
		if _, ok := latestSnapshotsMap[snap.ClusterName]; ok {
			if snapTime.After(beginOfToday) {
				latestSnapshotsMap[snap.ClusterName] = append(latestSnapshotsMap[snap.ClusterName], snap)
			}
		} else {
			latestSnapshotsMap[snap.ClusterName] = []snapshotQueryResult{snap}
		}
	}

	return latestSnapshotsMap
}

// buildClusterSummariesFromSnapshots creates cluster summaries from latest snapshots.
func (s *ResourceReportSender) buildClusterSummariesFromSnapshots(latestSnapshotsMap map[string][]snapshotQueryResult) map[string]*ClusterResourceSummary {
	clusterMap := make(map[string]*ClusterResourceSummary)

	for _, records := range latestSnapshotsMap {
		for _, snap := range records {
			if _, ok := clusterMap[snap.ClusterName]; !ok {
				// Initialize a new cluster summary
				clusterMap[snap.ClusterName] = &ClusterResourceSummary{
					ClusterName:   snap.ClusterName,
					ResourcePools: []ResourcePool{},
				}
			}

			summary := clusterMap[snap.ClusterName]
			// Create or update the resource pool for this snapshot
			s.addResourcePoolToSummary(summary, snap)
		}

	}

	return clusterMap
}

// addResourcePoolToSummary adds or updates a resource pool in a cluster summary.
func (s *ResourceReportSender) addResourcePoolToSummary(summary *ClusterResourceSummary, snap snapshotQueryResult) {
	poolKey := snap.ResourceType
	var pool *ResourcePool

	// Find existing pool if any
	for i := range summary.ResourcePools {
		if summary.ResourcePools[i].ResourceType == poolKey {
			pool = &summary.ResourcePools[i]
			break
		}
	}

	if pool == nil {
		// Create a new resource pool
		newPool := ResourcePool{
			ResourceType:        snap.ResourceType,
			NodeType:            s.getSimplifiedNodeType(snap.ResourceType), // 使用简化的NodeType
			Nodes:               int(snap.NodeCount),
			CPUCapacity:         snap.CpuCapacity,
			MemoryCapacity:      snap.MemoryCapacity, // 数据库中已经是GB单位，不需要转换
			CPURequest:          snap.CpuRequest,
			MemoryRequest:       snap.MemRequest, // 数据库中已经是GB单位，不需要转换
			BMCount:             int(snap.BMCount),
			VMCount:             int(snap.VMCount),
			PodCount:            int(snap.PodCount),
			PerNodeCpuRequest:   snap.PerNodeCpuRequest,
			PerNodeMemRequest:   snap.PerNodeMemRequest, // 数据库中已经是GB单位，不需要转换
			CPUHistory:          make([]float64, 0),
			MemoryHistory:       make([]float64, 0),
			TooltipText:         s.getResourcePoolTooltip(snap.ResourceType),
			MaxCpuUsageRatio:    snap.MaxCpuUsageRatio,
			MaxMemoryUsageRatio: snap.MaxMemoryUsageRatio,
		}
		summary.ResourcePools = append(summary.ResourcePools, newPool)
	}
}

// getSimplifiedNodeType returns a simplified Chinese display name for the resource type
func (s *ResourceReportSender) getSimplifiedNodeType(resourceType string) string {
	switch resourceType {
	case "total":
		return "总资源"
	case "total_intel":
		return "Intel总资源"
	case "intel_common":
		return "Intel通用"
	case "intel_gpu":
		return "Intel GPU"
	case "intel_taint":
		return "Intel污点"
	case "intel_non_gpu":
		return "Intel非GPU"
	case "total_arm":
		return "ARM总资源"
	case "arm_common":
		return "ARM通用"
	case "arm_gpu":
		return "ARM GPU"
	case "arm_taint":
		return "ARM污点"
	case "total_hg":
		return "海光总资源"
	case "hg_common":
		return "海光通用"
	case "hg_taint":
		return "海光污点"
	case "total_taint":
		return "总污点资源"
	case "total_common":
		return "总通用资源"
	case "total_gpu":
		return "总GPU资源"
	case "aplus_total":
		return "A+总资源"
	case "aplus_intel":
		return "A+Intel"
	case "aplus_arm":
		return "A+ARM"
	case "aplus_hg":
		return "A+海光"
	case "dplus_total":
		return "D+总资源"
	case "dplus_intel":
		return "D+Intel"
	case "dplus_arm":
		return "D+ARM"
	case "dplus_hg":
		return "D+海光"
	default:
		return resourceType
	}
}

// getResourcePoolTooltip returns a descriptive tooltip text for each resource pool type
func (s *ResourceReportSender) getResourcePoolTooltip(resourceType string) string {
	switch resourceType {
	case "total":
		return "集群所有物理机资源总和，包含集群中所有类型的节点。"
	case "total_intel":
		return "Intel架构物理机节点资源，使用Intel CPU的所有节点。"
	case "intel_common":
		return "Intel物理机通用应用节点资源，没有特殊标记或污点的Intel节点。"
	case "intel_gpu":
		return "Intel架构GPU物理机节点，配备了GPU的Intel节点。"
	case "intel_taint":
		return "Intel架构带污点物理机节点，带有特殊污点标记的Intel节点。"
	case "intel_non_gpu":
		return "Intel架构无GPU物理机节点，不包含GPU的Intel节点。"
	case "total_arm":
		return "ARM架构物理机节点资源，使用ARM CPU的所有节点。"
	case "arm_common":
		return "ARM物理机节点通用应用资源，没有特殊标记或污点的ARM节点。"
	case "arm_gpu":
		return "ARM架构GPU物理机节点，配备了GPU的ARM节点。"
	case "arm_taint":
		return "ARM架构带污点物理机节点，带有特殊污点标记的ARM节点。"
	case "total_hg":
		return "海光架构物理机节点资源，使用海光CPU的所有节点。"
	case "hg_common":
		return "海光物理机通用应用节点资源，没有特殊标记或污点的海光节点。"
	case "hg_taint":
		return "海光架构带污点物理机节点，带有特殊污点标记的海光节点。"
	case "total_taint":
		return "带污点的物理机节点资源，所有带有特殊污点标记的节点。"
	case "total_common":
		return "物理机节点通用应用资源总和，所有没有特殊标记或污点的普通节点。"
	case "total_gpu":
		return "包含GPU的物理机节点资源，所有配备了GPU的节点。"
	case "aplus_total":
		return "A+物理机资源总和，所有高性能计算节点。"
	case "aplus_intel":
		return "A+Intel架构物理机节点，高性能计算的Intel节点。"
	case "aplus_arm":
		return "A+ARM架构物理机节点，高性能计算的ARM节点。"
	case "aplus_hg":
		return "A+海光架构物理机节点，高性能计算的海光节点。"
	case "dplus_total":
		return "D+物理机资源总和，所有高存储容量节点。"
	case "dplus_intel":
		return "D+Intel架构物理机节点，高存储容量的Intel节点。"
	case "dplus_arm":
		return "D+ARM架构物理机节点，高存储容量的ARM节点。"
	case "dplus_hg":
		return "D+海光架构物理机节点，高存储容量的海光节点。"
	default:
		return resourceType + "资源池"
	}
}

// populateResourcePoolsMap 填充资源池映射，便于根据类型快速查找资源池
func (s *ResourceReportSender) populateResourcePoolsMap(summary *ClusterResourceSummary) {
	// 初始化映射
	summary.ResourcePoolsByType = make(map[string]*ResourcePool)

	// 填充映射
	for i := range summary.ResourcePools {
		pool := &summary.ResourcePools[i]
		summary.ResourcePoolsByType[pool.ResourceType] = pool
	}
}

// calculateUsagePercentages calculates CPU and memory usage percentages for all clusters and pools.
func (s *ResourceReportSender) calculateUsagePercentages(clusterMap map[string]*ClusterResourceSummary) {
	for _, summary := range clusterMap {

		// Calculate resource pool level percentages
		for i := range summary.ResourcePools {
			if summary.ResourcePools[i].CPUCapacity > 0 {
				summary.ResourcePools[i].CPUUsagePercent = (summary.ResourcePools[i].CPURequest / summary.ResourcePools[i].CPUCapacity) * 100
			}
			if summary.ResourcePools[i].MemoryCapacity > 0 {
				summary.ResourcePools[i].MemoryUsagePercent = (summary.ResourcePools[i].MemoryRequest / summary.ResourcePools[i].MemoryCapacity) * 100
			}
		}
	}
}

// convertToSortedSummaries converts a map of summaries to a sorted slice.
func (s *ResourceReportSender) convertToSortedSummaries(clusterMap map[string]*ClusterResourceSummary) []ClusterResourceSummary {
	var clusterSummaries []ClusterResourceSummary
	for _, summary := range clusterMap {
		// 填充ResourcePoolsByType映射
		s.populateResourcePoolsMap(summary)
		clusterSummaries = append(clusterSummaries, *summary)
	}

	// 按 GroupOrder 和 Desc 排序
	sort.Slice(clusterSummaries, func(i, j int) bool {
		if clusterSummaries[i].GroupOrder != clusterSummaries[j].GroupOrder {
			return clusterSummaries[i].GroupOrder < clusterSummaries[j].GroupOrder // 主要按 GroupOrder 排序
		}
		// 次要直接按 Desc 字符串排序
		return clusterSummaries[i].Desc < clusterSummaries[j].Desc
	})

	return clusterSummaries
}

// calculateClusterStats computes aggregate statistics across all clusters.
func (s *ResourceReportSender) calculateClusterStats(clusters []ClusterResourceSummary) ClusterStats {
	totalClusters := len(clusters)
	normalClusters := 0
	abnormalClusters := 0

	// Compute normal/abnormal based on resource usage
	for _, cluster := range clusters {
		if s.isClusterAbnormal(cluster) {
			abnormalClusters++
		} else {
			normalClusters++
		}
	}

	// Calculate pod density
	generalPodDensity := s.calculateGeneralClusterPodDensity(clusters)

	return ClusterStats{
		TotalClusters:     totalClusters,
		NormalClusters:    normalClusters,
		AbnormalClusters:  abnormalClusters,
		GeneralPodDensity: generalPodDensity,
	}
}

// isClusterAbnormal determines if a cluster has abnormally high resource usage
// based on CPU and memory usage thresholds.
func (s *ResourceReportSender) isClusterAbnormal(cluster ClusterResourceSummary) bool {
	// 根据环境类型使用不同的阈值
	cpuThreshold := 90.0 // 默认生产环境阈值
	memThreshold := 90.0

	if s.Environment == "test" {
		cpuThreshold = 80.0 // 测试环境使用较低的阈值
		memThreshold = 80.0
	}

	// 直接检查CPU和内存使用率是否超过阈值
	if cluster.CPUUsagePercent >= cpuThreshold || cluster.MemoryUsagePercent >= memThreshold {
		return true
	}

	// 检查是否有异常的资源池
	return s.hasAbnormalResourcePool(cluster)
}

// hasAbnormalResourcePool checks if any of the cluster's resource pools have abnormal usage.
func (s *ResourceReportSender) hasAbnormalResourcePool(cluster ClusterResourceSummary) bool {
	for _, pool := range cluster.ResourcePools {
		// 检查所有资源池，不再限制只检查total和total_common
		if s.isResourcePoolAbnormal(pool) {
			return true
		}
	}
	return false
}

// isResourcePoolAbnormal determines if a resource pool's usage is outside acceptable thresholds.
func (s *ResourceReportSender) isResourcePoolAbnormal(pool ResourcePool) bool {
	// 根据环境类型应用不同的规则
	if s.Environment == "test" {
		// 测试环境：低利用率不计算为异常，高利用率阈值上调5%
		if pool.BMCount > 150 {
			// 大型集群 (>150 物理节点)
			// 测试环境大型集群阈值：85% (80% + 5%)
			return pool.CPUUsagePercent >= 85.0 || pool.MemoryUsagePercent >= 85.0
		} else {
			// 小型/中型集群 (≤150 物理节点)
			// 测试环境小型/中型集群阈值：75% (70% + 5%)
			return pool.CPUUsagePercent >= 75.0 || pool.MemoryUsagePercent >= 75.0
		}
	} else {
		// 生产环境：使用当前规则
		if pool.BMCount > 150 {
			// 大型集群 (>150 物理节点)
			return pool.CPUUsagePercent >= 80.0 || pool.MemoryUsagePercent >= 80.0 ||
				pool.CPUUsagePercent < 55.0 || pool.MemoryUsagePercent < 55.0
		} else {
			// 小型/中型集群 (≤150 物理节点)
			return pool.CPUUsagePercent >= 70.0 || pool.MemoryUsagePercent >= 70.0 ||
				pool.CPUUsagePercent < 55.0 || pool.MemoryUsagePercent < 55.0
		}
	}
}

// hasAbnormalClusterMetrics checks if the cluster's overall metrics are abnormal.
func (s *ResourceReportSender) hasAbnormalClusterMetrics(cluster ClusterResourceSummary) bool {
	isLargeCluster := s.isLargeCluster(cluster)

	cpuUsage := cluster.CPUUsagePercent
	memUsage := cluster.MemoryUsagePercent

	// 根据环境类型应用不同的规则
	if s.Environment == "test" {
		// 测试环境：低利用率不计算为异常，高利用率阈值上调5%
		if isLargeCluster {
			// 大型集群标准：85% (80% + 5%)
			return cpuUsage >= 85.0 || memUsage >= 85.0
		} else {
			// 小型/中型集群标准：75% (70% + 5%)
			return cpuUsage >= 75.0 || memUsage >= 75.0
		}
	} else {
		// 生产环境：使用当前规则
		if isLargeCluster {
			// 大型集群标准
			return cpuUsage >= 80.0 || memUsage >= 80.0 ||
				cpuUsage < 55.0 || memUsage < 55.0
		} else {
			// 小型/中型集群标准
			return cpuUsage >= 70.0 || memUsage >= 70.0 ||
				cpuUsage < 55.0 || memUsage < 55.0
		}
	}
}

// isLargeCluster determines if a cluster is considered large (>150 physical nodes).
func (s *ResourceReportSender) isLargeCluster(cluster ClusterResourceSummary) bool {
	for _, pool := range cluster.ResourcePools {
		if pool.ResourceType == "total" && pool.BMCount > 150 {
			return true
		}
	}
	return false
}

// calculateGeneralClusterPodDensity calculates the Pod density for general clusters.
func (s *ResourceReportSender) calculateGeneralClusterPodDensity(clusters []ClusterResourceSummary) float64 {
	// If no general clusters are specified, return 0
	if len(s.generalClusters) == 0 {
		return 0
	}

	// Create a map of general cluster names for quick lookup
	generalClusterMap := s.createGeneralClusterMap()

	// Count total pods and physical nodes across all general clusters
	totalPods, totalBMs := s.countPodsAndNodesInGeneralClusters(clusters, generalClusterMap)

	// Calculate pod density (pods per physical node)
	if totalBMs > 0 {
		return float64(totalPods) / float64(totalBMs)
	}

	return 0
}

// createGeneralClusterMap creates a lookup map of general clusters.
func (s *ResourceReportSender) createGeneralClusterMap() map[string]bool {
	generalClusterMap := make(map[string]bool)
	for _, name := range s.generalClusters {
		generalClusterMap[name] = true
	}
	return generalClusterMap
}

// countPodsAndNodesInGeneralClusters counts pods and physical nodes in general clusters.
func (s *ResourceReportSender) countPodsAndNodesInGeneralClusters(
	clusters []ClusterResourceSummary,
	generalClusterMap map[string]bool,
) (int, int) {
	totalPods := 0
	totalBMs := 0

	for _, cluster := range clusters {
		// Skip clusters that aren't in the general clusters list
		if _, ok := generalClusterMap[cluster.ClusterName]; !ok {
			continue
		}

		// Count pods and physical nodes from all resource pools in this cluster
		for _, pool := range cluster.ResourcePools {
			totalPods += pool.PodCount
			totalBMs += pool.BMCount
		}
	}

	return totalPods, totalBMs
}

// --- Add custom template functions (copied from preview_generator.go) ---

// 辅助函数，用于将接口转换为浮点数
func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// 添加自定义模板函数
func customTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toFloat": func(val interface{}) float64 {
			return toFloat(val)
		},
		"mul": func(a, b interface{}) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			return af * bf
		},
		"div": func(a, b interface{}) float64 {
			af := toFloat(a)
			bf := toFloat(b)
			if bf == 0 {
				return 0
			}
			return af / bf
		},
		"ge": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af >= bf
		},
		"gt": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af > bf
		},
		"lt": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af < bf
		},
		"eq": func(a, b interface{}) bool {
			af := toFloat(a)
			bf := toFloat(b)
			return af == bf
		},
		"add": func(a, b interface{}) int {
			af := int(toFloat(a))
			bf := int(toFloat(b))
			return af + bf
		},
		"sub": func(a, b interface{}) int {
			af := int(toFloat(a))
			bf := int(toFloat(b))
			return af - bf
		},
		"len": func(a interface{}) int {
			switch v := a.(type) {
			// case []interface{}: // Avoid interface{} if possible
			// 	return len(v)
			case []string:
				return len(v)
			case []float64:
				return len(v)
			case []int:
				return len(v)
			case string:
				return len(v)
			default:
				// Add specific types used in the template if needed
				return 0
			}
		},
		"formatFloat": func(f float64, precision int) string {
			format := "%." + strconv.Itoa(precision) + "f"
			return fmt.Sprintf(format, f)
		},
		"formatBytes": func(bytes float64) string {
			// 输入已经是GB单位，直接格式化输出
			if bytes >= 1024 {
				return fmt.Sprintf("%.2f TB", bytes/1024)
			} else {
				return fmt.Sprintf("%.2f GB", bytes)
			}
		},
		// 获取CPU颜色类的函数
		"getCpuColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}

			isLargePool := pool.BMCount > 150
			cpuUsage := pool.CPUUsagePercent

			if environment == "test" {
				// 测试环境规则
				if isLargePool {
					if cpuUsage >= 95.0 {
						return "emergency"
					} else if cpuUsage >= 90.0 {
						return "critical"
					} else if cpuUsage >= 85.0 {
						return "warning"
					}
					return "normal"
				} else {
					if cpuUsage >= 90.0 {
						return "emergency"
					} else if cpuUsage >= 80.0 {
						return "critical"
					} else if cpuUsage >= 75.0 {
						return "warning"
					}
					return "normal"
				}
			} else {
				// 生产环境规则
				if isLargePool {
					if cpuUsage >= 95.0 {
						return "emergency"
					} else if cpuUsage >= 85.0 {
						return "critical"
					} else if cpuUsage >= 80.0 {
						return "warning"
					} else if cpuUsage < 55.0 {
						return "underutilized"
					}
					return "normal"
				} else {
					if cpuUsage >= 90.0 {
						return "emergency"
					} else if cpuUsage >= 75.0 {
						return "critical"
					} else if cpuUsage >= 70.0 {
						return "warning"
					} else if cpuUsage < 55.0 {
						return "underutilized"
					}
					return "normal"
				}
			}
		},
		// 获取内存颜色类的函数
		"getMemColorClass": func(pool *ResourcePool, environment string) string {
			if pool == nil {
				return ""
			}

			isLargePool := pool.BMCount > 150
			memUsage := pool.MemoryUsagePercent

			if environment == "test" {
				// 测试环境规则
				if isLargePool {
					if memUsage >= 95.0 {
						return "emergency"
					} else if memUsage >= 90.0 {
						return "critical"
					} else if memUsage >= 85.0 {
						return "warning"
					}
					return "normal"
				} else {
					if memUsage >= 90.0 {
						return "emergency"
					} else if memUsage >= 80.0 {
						return "critical"
					} else if memUsage >= 75.0 {
						return "warning"
					}
					return "normal"
				}
			} else {
				// 生产环境规则
				if isLargePool {
					if memUsage >= 95.0 {
						return "emergency"
					} else if memUsage >= 85.0 {
						return "critical"
					} else if memUsage >= 80.0 {
						return "warning"
					} else if memUsage < 55.0 {
						return "underutilized"
					}
					return "normal"
				} else {
					if memUsage >= 90.0 {
						return "emergency"
					} else if memUsage >= 75.0 {
						return "critical"
					} else if memUsage >= 70.0 {
						return "warning"
					} else if memUsage < 55.0 {
						return "underutilized"
					}
					return "normal"
				}
			}
		},
		// 新增函数，用于判断资源池是否需要显示
		"shouldShowPool": func(cpuUsage, memoryUsage float64, bmCount int, environment string) bool {
			// 根据环境类型应用不同的规则
			if environment == "test" {
				// 测试环境：低利用率不计算为异常，高利用率阈值上调5%
				if bmCount > 150 {
					// 大型集群（物理机节点数 > 150）
					return cpuUsage >= 85.0 || memoryUsage >= 85.0
				} else {
					// 小型集群（物理机节点数 <= 150）
					return cpuUsage >= 75.0 || memoryUsage >= 75.0
				}
			} else {
				// 生产环境：使用当前规则
				if bmCount > 150 {
					// 大型集群（物理机节点数 > 150）
					return cpuUsage >= 80.0 || memoryUsage >= 80.0 || cpuUsage < 55.0 || memoryUsage < 55.0
				} else {
					// 小型集群（物理机节点数 <= 150）
					return cpuUsage >= 70.0 || memoryUsage >= 70.0 || cpuUsage < 55.0 || memoryUsage < 55.0
				}
			}
		},
		// 新增函数，用于获取资源池类型的颜色
		"getPoolTypeColor": func(poolType string) string {
			switch poolType {
			case "total":
				return "#00188F" // 深蓝色
			case "intel_common":
				return "#0078D7" // 蓝色
			case "intel_gpu":
				return "#2B579A" // 深蓝色
			case "amd_common": // Added based on preview funcs
				return "#D83B01" // 红色
			case "arm_common":
				return "#107C10" // 绿色
			case "hg_common":
				return "#5C2D91" // 紫色
			default:
				return "#000000" // 黑色
			}
		},
	}
}

// generateEmailContent renders the HTML email body using the template and data.
func (s *ResourceReportSender) generateEmailContent(data ReportTemplateData) (string, error) {
	// Prepare template functions
	htmlFuncMap := s.prepareTemplateFunctions()

	// Parse the template
	tmpl, err := s.parseEmailTemplate(htmlFuncMap)
	if err != nil {
		return "", err
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "template.html", data); err != nil {
		s.Logger.Error("Failed to execute email template", zap.Error(err))
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// prepareTemplateFunctions creates a FuncMap with both sprig and custom functions.
func (s *ResourceReportSender) prepareTemplateFunctions() template.FuncMap {
	// Use HtmlFuncMap for HTML safety with Sprig funcs
	htmlFuncMap := sprig.HtmlFuncMap()

	// Add custom functions
	for k, v := range customTemplateFuncs() {
		htmlFuncMap[k] = v
	}

	return htmlFuncMap
}

// parseEmailTemplate loads and parses the HTML template.
func (s *ResourceReportSender) parseEmailTemplate(funcMap template.FuncMap) (*template.Template, error) {
	tmpl, err := template.New("resourceReport").Funcs(funcMap).ParseFS(templateFS, "template.html")
	if err != nil {
		s.Logger.Error("Failed to parse email template", zap.Error(err))
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}
	return tmpl, nil
}

// sendEmail sends the generated report via SMTP.
func (s *ResourceReportSender) sendEmail(subject, body string) error {
	auth := smtp.PlainAuth("", s.SMTPUsername, s.SMTPPassword, s.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)

	// Construct the email message
	msg := "From: " + s.FromEmail + "\r\n" +
		"To: " + s.ToEmails[0] // Simplistic; handle multiple recipients better if needed
	for i := 1; i < len(s.ToEmails); i++ {
		msg += "," + s.ToEmails[i]
	}
	msg += "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg += body

	s.Logger.Info("Sending email", zap.Strings("to", s.ToEmails), zap.String("subject", subject))
	err := smtp.SendMail(addr, auth, s.FromEmail, s.ToEmails, []byte(msg))
	if err != nil {
		s.Logger.Error("Failed to send email via SMTP", zap.Error(err))
		return fmt.Errorf("smtp.SendMail failed: %w", err)
	}
	s.Logger.Info("Email sent successfully")
	return nil
}

// 根据集群名称或标签分配组别和排序优先级
func (s *ResourceReportSender) assignGroupsAndOrder(summaries []ClusterResourceSummary) []ClusterResourceSummary {
	for i := range summaries {
		clusterName := summaries[i].ClusterName

		// 提取集群名称中的组标识符
		matches := groupRegex.FindStringSubmatch(clusterName)
		group := ""
		if len(matches) > 1 {
			group = matches[1]
		}

		// 设置默认值
		summaries[i].Desc = group
		summaries[i].GroupOrder = 1000 // 默认优先级最低

		// 根据组名称设置描述和排序优先级
		if s.containsAny(group, []string{"通用"}) || s.containsAny(clusterName, []string{"通用"}) {
			summaries[i].GroupOrder = 100 // 最高优先级
		} else if strings.Contains(group, "+") || strings.Contains(group, "plus") {
			summaries[i].GroupOrder = 200 // 第二优先级
		} else if s.containsAny(group, []string{"工具"}) || s.containsAny(clusterName, []string{"工具"}) {
			summaries[i].GroupOrder = 300 // 第三优先级
		} else if s.containsAny(group, []string{"赛飞", "推理", "大数据"}) || s.containsAny(clusterName, []string{"赛飞", "推理", "大数据"}) {
			summaries[i].GroupOrder = 400 // 第四优先级
		}

		// 如果描述为空，使用默认描述
		if summaries[i].Desc == "" {
			summaries[i].Desc = "其他"
		}
	}

	// 按 GroupOrder 和 Desc 排序
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].GroupOrder != summaries[j].GroupOrder {
			return summaries[i].GroupOrder < summaries[j].GroupOrder // 主要按 GroupOrder 排序
		}
		// 次要直接按 Desc 字符串排序
		return summaries[i].Desc < summaries[j].Desc
	})

	return summaries
}

// 检查字符串是否包含给定子串列表中的任何一个
func (s *ResourceReportSender) containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}
