package security_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/smtp"
	"sort"
	"strconv"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jinzhu/now"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"navy-ng/job/chore/security_check"
	"navy-ng/models/portal"
)

// 日期格式常量
const (
	DateFormat = "2006-01-02"
)

// 集群状态常量
const (
	ClusterStatusInit    = "init"
	ClusterStatusRunning = "running"
	ClusterStatusUnknown = "unknown"
)

// 节点类型常量
const (
	NodeTypeMaster = "master"
	NodeTypeNode   = "node"
	NodeTypeEtcd   = "etcd"
	NodeTypeWorker = "worker"
)

// 节点状态常量
const (
	NodeStatusOffline = "Offline"
)

// 数据表名常量
const (
	TableK8sCluster  = "k8s_cluster"
	TableK8sNode     = "k8s_node"
	TableK8sEtcdInfo = "k8s_etcd_info"
)

// 集群状态颜色常量
const (
	StatusColorGreen  = "green"
	StatusColorRed    = "red"
	StatusColorYellow = "yellow"
)

// 热力图颜色常量
const (
	HeatLevelHigh = "heat-level-high"
	HeatLevel2    = "heat-level-2"
	HeatLevel1    = "heat-level-1"
)

//go:embed template.html
var templateFS embed.FS

// SecurityReportSender 安全报告发送器
type SecurityReportSender struct {
	db *gorm.DB
	// 邮件配置
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	toEmails     []string
	// 集群状态信息
	clusterStatus map[string]*security_check.ClusterStatus
	// 未巡检节点
	missingNodes []MissingNode
	// 在线集群信息
	onlineClusters map[string]*ClusterNodesInfo
	// 日志记录
	logger *zap.Logger
}

// NewSecurityReportSender 创建安全报告发送器
func NewSecurityReportSender(db *gorm.DB, smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string) *SecurityReportSender {
	logger, err := zap.NewProduction()
	if err != nil {
		// 如果无法创建生产级别的logger，回退到开发级别
		logger, _ = zap.NewDevelopment()
	}

	return &SecurityReportSender{
		db:             db,
		smtpHost:       smtpHost,
		smtpPort:       smtpPort,
		smtpUser:       smtpUser,
		smtpPassword:   smtpPassword,
		fromEmail:      fromEmail,
		toEmails:       toEmails,
		clusterStatus:  make(map[string]*security_check.ClusterStatus),
		missingNodes:   []MissingNode{},
		onlineClusters: make(map[string]*ClusterNodesInfo),
		logger:         logger,
	}
}

// ClusterNodesInfo 存储集群节点信息
type ClusterNodesInfo struct {
	ClusterName    string
	Status         string
	TotalNodes     int
	MasterNodes    []string        // hostip列表
	WorkerNodes    []string        // hostip列表
	EtcdNodes      []string        // instance列表
	etcdSet        map[string]bool // 用于去重etcd实例
	InspectedNodes map[string]bool // 记录已巡检节点，key为nodeKey (type/ip)
}

// getOnlineClustersAndNodes 获取当前在线的集群和节点信息
func (s *SecurityReportSender) getOnlineClustersAndNodes(ctx context.Context) (map[string]*ClusterNodesInfo, error) {
	// 获取今天的开始时间
	today := now.BeginningOfDay()

	// 使用map存储结果
	result := make(map[string]*ClusterNodesInfo)

	// 1. 获取在线集群列表及其节点信息（使用JOIN优化查询）
	type ClusterNodeResult struct {
		ClusterID     uint64
		ClusterName   string
		ClusterStatus string
		NodeRole      string
		NodeHostIP    string
		NodeStatus    string
	}

	var clusterNodeResults []ClusterNodeResult

	// 使用JOIN一次性获取所有在线集群及其节点信息
	nodeQuery := s.db.WithContext(ctx).
		Table(fmt.Sprintf("%s c", TableK8sCluster)).
		Select("c.id as cluster_id, c.clustername as cluster_name, c.status as cluster_status, "+
			"n.role as node_role, n.hostip as node_host_ip, n.status as node_status").
		Joins(fmt.Sprintf("LEFT JOIN %s n ON c.id = n.k8s_cluster_id AND n.created_at >= ? AND n.status != ?", TableK8sNode), today, NodeStatusOffline).
		Where("c.status IN ?", []string{ClusterStatusInit, ClusterStatusRunning})

	if err := nodeQuery.Scan(&clusterNodeResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters and nodes: %w", err)
	}

	// 2. 获取etcd信息（使用JOIN优化查询）
	type ClusterEtcdResult struct {
		ClusterID    uint64
		ClusterName  string
		EtcdInstance string
	}

	var clusterEtcdResults []ClusterEtcdResult

	etcdQuery := s.db.WithContext(ctx).
		Table(fmt.Sprintf("%s c", TableK8sCluster)).
		Select("c.id as cluster_id, c.clustername as cluster_name, e.instance as etcd_instance").
		Joins(fmt.Sprintf("LEFT JOIN %s e ON c.id = e.k8s_cluster_id AND e.created_at >= ?", TableK8sEtcdInfo), today).
		Where("c.status IN ?", []string{ClusterStatusInit, ClusterStatusRunning})

	if err := etcdQuery.Scan(&clusterEtcdResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters and etcd: %w", err)
	}

	// 3. 处理查询结果，将节点信息组织到对应的集群中
	for _, cr := range clusterNodeResults {
		// 如果还没有此集群的记录，创建一个
		if _, exists := result[cr.ClusterName]; !exists {
			result[cr.ClusterName] = &ClusterNodesInfo{
				ClusterName:    cr.ClusterName,
				Status:         cr.ClusterStatus,
				InspectedNodes: make(map[string]bool),
				etcdSet:        make(map[string]bool), // 初始化etcdSet用于去重
			}
		}

		// 只处理有效的节点记录（JOIN可能会返回NULL值）
		if cr.NodeHostIP != "" && cr.NodeRole != "" {
			result[cr.ClusterName].TotalNodes++

			switch cr.NodeRole {
			case NodeTypeMaster:
				result[cr.ClusterName].MasterNodes = append(result[cr.ClusterName].MasterNodes, cr.NodeHostIP)
			case NodeTypeNode, NodeTypeWorker:
				result[cr.ClusterName].WorkerNodes = append(result[cr.ClusterName].WorkerNodes, cr.NodeHostIP)
			}
		}
	}

	// 4. 添加etcd信息
	for _, er := range clusterEtcdResults {
		// 如果还没有此集群的记录（理论上不应该发生，除非etcd信息没有对应的集群）
		if _, exists := result[er.ClusterName]; !exists {
			result[er.ClusterName] = &ClusterNodesInfo{
				ClusterName:    er.ClusterName,
				Status:         ClusterStatusUnknown, // 这种情况下，集群状态未知
				InspectedNodes: make(map[string]bool),
				etcdSet:        make(map[string]bool), // 初始化etcdSet用于去重
			}
		}

		// 只添加有效的etcd实例，并确保不重复
		if er.EtcdInstance != "" && !result[er.ClusterName].etcdSet[er.EtcdInstance] {
			result[er.ClusterName].EtcdNodes = append(result[er.ClusterName].EtcdNodes, er.EtcdInstance)
			result[er.ClusterName].etcdSet[er.EtcdInstance] = true // 标记此实例已添加
		}
	}

	return result, nil
}

// collectData 收集报告数据
func (s *SecurityReportSender) collectData(ctx context.Context, clusterStatus map[string]*security_check.ClusterStatus) ([]SecurityReportData, error) {
	// 获取在线集群和节点信息
	onlineClusters, err := s.getOnlineClustersAndNodes(ctx)
	if err != nil {
		s.logger.Warn("Failed to get online clusters and nodes", zap.Error(err))
		// 继续执行，不阻断流程
	} else {
		// 存储在线集群信息以供后续使用
		s.onlineClusters = onlineClusters
	}

	// 从数据库获取所有集群名称
	var clusters []string
	if err := s.db.Model(&portal.SecurityCheck{}).
		Distinct("cluster_name").
		Pluck("cluster_name", &clusters).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	// 合并数据库中的安全检查集群和在线集群列表
	clusterSet := make(map[string]bool)
	for _, cluster := range clusters {
		clusterSet[cluster] = true
	}

	// 添加在线集群但可能没有安全检查记录的集群
	for clusterName := range onlineClusters {
		if !clusterSet[clusterName] {
			clusters = append(clusters, clusterName)
			clusterSet[clusterName] = true
		}
	}

	// 将集群状态信息保存到全局变量
	s.clusterStatus = clusterStatus

	var reports []SecurityReportData
	for _, cluster := range clusters {
		report, err := s.collectClusterData(ctx, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to collect data for cluster %s: %w", cluster, err)
		}

		// 如果该集群有在线节点信息，标记已检查的节点
		if clusterInfo, ok := onlineClusters[cluster]; ok {
			for _, result := range report.DetailedResults {
				nodeKey := fmt.Sprintf("%s/%s", result.NodeType, result.NodeName)
				clusterInfo.InspectedNodes[nodeKey] = true
			}
		}

		reports = append(reports, report)
	}

	// 不在此处计算未巡检节点，而是在generateEmailData中完成
	s.missingNodes = []MissingNode{} // 仅初始化空切片

	// 返回报告数据
	return reports, nil
}

// collectClusterData 收集单个集群的报告数据
func (s *SecurityReportSender) collectClusterData(ctx context.Context, clusterName string) (SecurityReportData, error) {
	var checks []portal.SecurityCheck
	if err := s.db.WithContext(ctx).
		Where("cluster_name = ? AND created_at >= ?", clusterName, now.BeginningOfDay()).
		Find(&checks).Error; err != nil {
		return SecurityReportData{}, fmt.Errorf("failed to get security checks: %w", err)
	}

	report := SecurityReportData{
		ClusterName: clusterName,
	}

	for _, check := range checks {
		var items []portal.SecurityCheckItem
		if err := s.db.WithContext(ctx).
			Where("security_check_id = ?", check.ID).
			Find(&items).Error; err != nil {
			return SecurityReportData{}, fmt.Errorf("failed to get check items: %w", err)
		}

		for _, item := range items {
			report.TotalChecks++
			if item.Status {
				report.PassedChecks++
			} else {
				report.FailedChecks++
			}

			report.DetailedResults = append(report.DetailedResults, SecurityCheckResult{
				NodeType:      check.NodeType,
				NodeName:      check.NodeName,
				CheckType:     check.CheckType,
				ItemName:      item.ItemName,
				ItemValue:     item.ItemValue,
				Status:        item.Status,
				FixSuggestion: item.FixSuggestion,
			})
		}
	}

	return report, nil
}

// generateEmailData 生成邮件数据
func (s *SecurityReportSender) generateEmailData(reports []SecurityReportData) EmailTemplateData {
	data := EmailTemplateData{
		TotalClusters: len(reports),
		// Initialize slice to avoid nil pointer if no reports
		ClusterHealthSummary:    make([]ClusterHealthInfo, 0, len(reports)),
		CheckItemFailureSummary: make([]CheckItemFailureInfo, 0), // Initialize slice
		MissingNodes:            s.missingNodes,                  // Use the collected missing nodes
	}

	// 从onlineClusters中计算总节点数
	totalNodesFromDB := 0
	for _, clusterInfo := range s.onlineClusters {
		totalNodesFromDB += clusterInfo.TotalNodes
	}
	data.TotalNodes = totalNodesFromDB

	// 用于跟踪所有节点的状态
	allNodeKeys := make(map[string]bool)                      // 跟踪所有在线节点
	checkedNodeMap := make(map[string]bool)                   // 所有被检查过的节点
	healthyNodeMap := make(map[string]bool)                   // 健康节点（无异常检查项）
	abnormalNodeMap := make(map[string]bool)                  // 异常节点（有异常检查项）
	missingNodeMap := make(map[string]bool)                   // 未巡检的节点
	clusterNodeFailureMap := make(map[string]map[string]bool) // cluster -> nodeKey -> hasFailure
	checkItemFailureCounts := make(map[string]int)            // itemName -> totalFailureCount

	// 用于跟踪每个集群的健康节点
	clusterHealthyNodeMap := make(map[string]map[string]bool) // cluster -> nodeKey -> isHealthy

	// --- First pass: Calculate global stats and identify abnormal nodes per cluster ---
	clusterTotalNodes := make(map[string]int) // Track total nodes per cluster

	// 使用已存储的在线集群信息设置每个集群的总节点数
	for clusterName, clusterInfo := range s.onlineClusters {
		if _, ok := clusterTotalNodes[clusterName]; !ok {
			clusterTotalNodes[clusterName] = clusterInfo.TotalNodes
		}

		// 初始化集群健康节点映射
		if _, ok := clusterHealthyNodeMap[clusterName]; !ok {
			clusterHealthyNodeMap[clusterName] = make(map[string]bool)
		}

		// 收集所有节点
		for _, ip := range clusterInfo.MasterNodes {
			nodeKey := fmt.Sprintf("%s/%s/%s", clusterName, NodeTypeMaster, ip)
			allNodeKeys[nodeKey] = true
		}
		for _, ip := range clusterInfo.WorkerNodes {
			nodeKey := fmt.Sprintf("%s/%s/%s", clusterName, NodeTypeNode, ip)
			allNodeKeys[nodeKey] = true
		}
		for _, ip := range clusterInfo.EtcdNodes {
			nodeKey := fmt.Sprintf("%s/%s/%s", clusterName, NodeTypeEtcd, ip)
			allNodeKeys[nodeKey] = true
		}
	}

	for _, report := range reports {
		if _, ok := clusterNodeFailureMap[report.ClusterName]; !ok {
			clusterNodeFailureMap[report.ClusterName] = make(map[string]bool)
		}

		// 初始化集群健康节点映射
		if _, ok := clusterHealthyNodeMap[report.ClusterName]; !ok {
			clusterHealthyNodeMap[report.ClusterName] = make(map[string]bool)
		}

		for _, result := range report.DetailedResults {
			nodeKey := fmt.Sprintf("%s/%s/%s", report.ClusterName, result.NodeType, result.NodeName)

			// 标记节点已被检查
			checkedNodeMap[nodeKey] = true

			// Mark node as having failure if any check fails
			if !result.Status {
				clusterNodeFailureMap[report.ClusterName][nodeKey] = true
				abnormalNodeMap[nodeKey] = true
			} else if _, exists := clusterNodeFailureMap[report.ClusterName][nodeKey]; !exists {
				// Initialize with false if no failure seen yet for this node in this cluster
				clusterNodeFailureMap[report.ClusterName][nodeKey] = false

				// 如果检查通过且此节点还没有被标记为异常，则将其添加到健康节点映射中
				if !abnormalNodeMap[nodeKey] {
					healthyNodeMap[nodeKey] = true
					clusterHealthyNodeMap[report.ClusterName][nodeKey] = true
				}
			}

			// Global check counts
			data.TotalChecks++
			if result.Status {
				data.PassedChecks++
			} else {
				data.FailedChecks++
				checkItemFailureCounts[result.ItemName]++ // Increment failure count for this item name
			}
		}
	}

	// 确定健康节点（被检查过但没有异常）
	for nodeKey := range checkedNodeMap {
		// 只有明确没有异常的节点才被视为健康节点
		if _, isAbnormal := abnormalNodeMap[nodeKey]; !isAbnormal {
			healthyNodeMap[nodeKey] = true

			// 解析nodeKey以获取集群名称
			parts := bytes.SplitN([]byte(nodeKey), []byte("/"), 3)
			if len(parts) == 3 {
				clusterName := string(parts[0])
				if _, ok := clusterHealthyNodeMap[clusterName]; ok {
					clusterHealthyNodeMap[clusterName][nodeKey] = true
				}
			}
		}
	}

	// 确定未巡检节点 (未在checkedNodeMap中出现的allNodeKeys)
	for nodeKey := range allNodeKeys {
		if !checkedNodeMap[nodeKey] {
			missingNodeMap[nodeKey] = true

			// 解析nodeKey获取集群名称、节点类型和节点名称
			parts := bytes.SplitN([]byte(nodeKey), []byte("/"), 3)
			if len(parts) == 3 {
				clusterName := string(parts[0])
				nodeType := string(parts[1])
				nodeName := string(parts[2])

				// 检查是否已经在未巡检节点列表中
				alreadyMissing := false
				for _, missingNode := range s.missingNodes {
					if missingNode.ClusterName == clusterName &&
						missingNode.NodeType == nodeType &&
						missingNode.NodeName == nodeName {
						alreadyMissing = true
						break
					}
				}

				// 如果不在列表中，添加到未巡检节点列表
				if !alreadyMissing {
					s.missingNodes = append(s.missingNodes, MissingNode{
						ClusterName: clusterName,
						NodeType:    nodeType,
						NodeName:    nodeName,
					})
				}
			}
		}
	}

	// 设置健康节点和异常节点的数量
	data.NormalNodes = len(healthyNodeMap) // 使用巡检确认正常的节点数
	data.AbnormalNodes = len(abnormalNodeMap)

	// 更新未巡检节点总数
	// 注意：此处重新使用missingNodeMap计算，而不是直接使用s.missingNodes的长度
	data.MissingNodesCount = len(missingNodeMap)

	// 将未巡检节点按照集群名称排序
	sort.Slice(s.missingNodes, func(i, j int) bool {
		// 先按集群名排序
		if s.missingNodes[i].ClusterName != s.missingNodes[j].ClusterName {
			return s.missingNodes[i].ClusterName < s.missingNodes[j].ClusterName
		}
		// 集群名相同时按节点类型排序
		if s.missingNodes[i].NodeType != s.missingNodes[j].NodeType {
			return s.missingNodes[i].NodeType < s.missingNodes[j].NodeType
		}
		// 节点类型相同时按节点名称排序
		return s.missingNodes[i].NodeName < s.missingNodes[j].NodeName
	})

	// 更新未巡检节点列表到模板数据
	data.MissingNodes = s.missingNodes

	// 对异常节点详情也进行排序
	sort.Slice(data.AbnormalDetails, func(i, j int) bool {
		// 先按集群名排序
		if data.AbnormalDetails[i].ClusterName != data.AbnormalDetails[j].ClusterName {
			return data.AbnormalDetails[i].ClusterName < data.AbnormalDetails[j].ClusterName
		}
		// 集群名相同时按节点类型排序
		if data.AbnormalDetails[i].NodeType != data.AbnormalDetails[j].NodeType {
			return data.AbnormalDetails[i].NodeType < data.AbnormalDetails[j].NodeType
		}
		// 节点类型相同时按节点名称排序
		return data.AbnormalDetails[i].NodeName < data.AbnormalDetails[j].NodeName
	})

	// --- Second pass: Calculate cluster health and populate details ---
	for _, report := range reports {
		clusterAbnormalNodes := 0
		clusterFailedChecks := 0 // Recalculate failed checks per cluster for health status

		// Populate AbnormalDetails and count cluster abnormal nodes
		clusterNodes := clusterNodeFailureMap[report.ClusterName]
		for nodeKey, hasFailure := range clusterNodes {
			if hasFailure {
				clusterAbnormalNodes++
				// Extract node details from nodeKey (assuming format cluster/type/name)
				// This is a bit fragile, consider storing node details differently if possible
				parts := bytes.SplitN([]byte(nodeKey), []byte("/"), 3)
				if len(parts) == 3 {
					nodeType := string(parts[1])
					nodeName := string(parts[2])
					var failedItems []FailedItem
					// Find failed items for this specific node
					for _, detail := range report.DetailedResults {
						if detail.NodeType == nodeType && detail.NodeName == nodeName && !detail.Status {
							failedItems = append(failedItems, FailedItem{
								ItemName:      detail.ItemName,
								ItemValue:     detail.ItemValue,
								FixSuggestion: detail.FixSuggestion,
							})
							clusterFailedChecks++ // Count failed checks for this cluster
						}
					}
					data.AbnormalDetails = append(data.AbnormalDetails, AbnormalDetail{
						ClusterName: report.ClusterName,
						NodeType:    nodeType,
						NodeName:    nodeName,
						FailedItems: failedItems,
					})
				}
			}
		}

		// 获取集群状态信息
		clusterExists := true
		if status, ok := s.clusterStatus[report.ClusterName]; ok {
			clusterExists = status.Exists
		}

		// 检查集群是否有未巡检节点
		hasMissingNodes := false
		clusterMissingNodes := 0 // 计算该集群的未巡检节点数
		for _, missingNode := range s.missingNodes {
			if missingNode.ClusterName == report.ClusterName {
				hasMissingNodes = true
				clusterMissingNodes++
			}
		}

		// 生成锚点ID
		anchorID := report.ClusterName

		// Determine cluster status color
		statusColor := StatusColorGreen
		if !clusterExists {
			statusColor = StatusColorRed // 集群不存在，标记为红色
		} else if hasMissingNodes {
			statusColor = StatusColorRed // 集群有未巡检节点，标记为红色
		} else if clusterAbnormalNodes > 0 {
			statusColor = StatusColorRed // Any abnormal node makes the cluster red
		} else if clusterFailedChecks > 0 {
			statusColor = StatusColorYellow // No abnormal nodes, but some failed checks
		}

		// Add cluster health summary
		data.ClusterHealthSummary = append(data.ClusterHealthSummary, ClusterHealthInfo{
			ClusterName:   report.ClusterName,
			StatusColor:   statusColor,
			AbnormalNodes: clusterAbnormalNodes,
			NormalNodes:   len(clusterHealthyNodeMap[report.ClusterName]), // 使用巡检确认正常的节点数
			TotalNodes:    clusterTotalNodes[report.ClusterName],          // Use counted total nodes
			MissingNodes:  clusterMissingNodes,                            // 添加未巡检节点数
			FailedChecks:  clusterFailedChecks,                            // Use the per-cluster failed count
			Exists:        clusterExists,
			AnchorID:      anchorID,
		})

		// 添加调试日志
		s.logger.Info("集群节点计数",
			zap.String("集群名", report.ClusterName),
			zap.Int("健康节点数", len(clusterHealthyNodeMap[report.ClusterName])),
			zap.Int("异常节点数", clusterAbnormalNodes),
			zap.Int("未巡检节点数", clusterMissingNodes),
			zap.Int("总节点数", clusterTotalNodes[report.ClusterName]),
		)
	}

	// 计算百分比之后、计算正常集群之前，对集群健康状态概览进行排序
	sort.Slice(data.ClusterHealthSummary, func(i, j int) bool {
		return data.ClusterHealthSummary[i].ClusterName < data.ClusterHealthSummary[j].ClusterName
	})

	// --- Third pass: Populate CheckItemFailureSummary (Heatmap data) ---
	for itemName, count := range checkItemFailureCounts {
		heatColor := ""
		if count > 5 {
			heatColor = HeatLevelHigh
		} else if count >= 3 {
			heatColor = HeatLevel2
		} else if count > 0 {
			heatColor = HeatLevel1
		} // count == 0 will have no specific class

		data.CheckItemFailureSummary = append(data.CheckItemFailureSummary, CheckItemFailureInfo{
			ItemName:      itemName,
			TotalFailures: count,
			HeatColor:     heatColor,
		})
	}

	// Sort CheckItemFailureSummary by TotalFailures descending
	sort.Slice(data.CheckItemFailureSummary, func(i, j int) bool {
		return data.CheckItemFailureSummary[i].TotalFailures > data.CheckItemFailureSummary[j].TotalFailures
	})

	// Calculate percentages
	if data.TotalNodes > 0 {
		data.NormalNodesPercent = fmt.Sprintf("%.0f", float64(data.NormalNodes)*100/float64(data.TotalNodes))
		data.AbnormalNodesPercent = fmt.Sprintf("%.0f", float64(data.AbnormalNodes)*100/float64(data.TotalNodes))
		data.MissingNodesPercent = fmt.Sprintf("%.0f", float64(data.MissingNodesCount)*100/float64(data.TotalNodes))
	} else {
		data.NormalNodesPercent = "0"
		data.AbnormalNodesPercent = "0"
		data.MissingNodesPercent = "0"
	}

	// 检查节点数据一致性
	calculatedTotal := data.NormalNodes + data.AbnormalNodes + data.MissingNodesCount
	if calculatedTotal != data.TotalNodes {
		s.logger.Warn("节点数据不一致",
			zap.Int("数据库总节点数", data.TotalNodes),
			zap.Int("计算总节点数", calculatedTotal),
			zap.Int("健康节点数", data.NormalNodes),
			zap.Int("异常节点数", data.AbnormalNodes),
			zap.Int("未巡检节点数", data.MissingNodesCount),
		)

		// 更新总节点数为计算值，确保数据一致性
		data.TotalNodes = calculatedTotal

		// 重新计算百分比
		if data.TotalNodes > 0 {
			data.NormalNodesPercent = fmt.Sprintf("%.0f", float64(data.NormalNodes)*100/float64(data.TotalNodes))
			data.AbnormalNodesPercent = fmt.Sprintf("%.0f", float64(data.AbnormalNodes)*100/float64(data.TotalNodes))
			data.MissingNodesPercent = fmt.Sprintf("%.0f", float64(data.MissingNodesCount)*100/float64(data.TotalNodes))
		}
	}

	// 输出总体节点数据日志
	s.logger.Info("总体节点统计",
		zap.Int("总节点数", data.TotalNodes),
		zap.Int("健康节点数", data.NormalNodes),
		zap.Int("异常节点数", data.AbnormalNodes),
		zap.Int("未巡检节点数", data.MissingNodesCount),
		zap.String("健康节点百分比", data.NormalNodesPercent),
		zap.String("异常节点百分比", data.AbnormalNodesPercent),
		zap.String("未巡检节点百分比", data.MissingNodesPercent),
	)

	// Count normal clusters (those with green status)
	data.NormalClusters = 0
	data.UnscannedClusters = 0
	for _, clusterHealth := range data.ClusterHealthSummary {
		if clusterHealth.StatusColor == StatusColorGreen {
			data.NormalClusters++
		}
		// Count clusters that don't exist in storage as unscanned
		if !clusterHealth.Exists {
			data.UnscannedClusters++
		}
	}

	// 创建一个映射来跟踪有未巡检节点的集群
	missingNodeClusters := make(map[string]bool)
	for _, missingNode := range s.missingNodes {
		missingNodeClusters[missingNode.ClusterName] = true
	}

	return data
}

// generateEmailContent 生成邮件内容
func (s *SecurityReportSender) generateEmailContent(data EmailTemplateData) (string, error) {
	// 扩展函数映射
	funcMap := template.FuncMap{
		"toFloat64": toFloat64,
		"sin": func(a interface{}) float64 {
			return math.Sin(toFloat64(a))
		},
		"cos": func(a interface{}) float64 {
			return math.Cos(toFloat64(a))
		},
		"negate": func(a interface{}) float64 {
			return -toFloat64(a)
		},
		"gt": func(a, b interface{}) bool {
			return toFloat64(a) > toFloat64(b)
		},
		"lt": func(a, b interface{}) bool {
			return toFloat64(a) < toFloat64(b)
		},
		"eq": func(a, b interface{}) bool {
			return toFloat64(a) == toFloat64(b)
		},
		// 添加一个专门用于比较字符串的函数
		"strEq": func(a, b string) bool {
			return a == b
		},
		"printf": func(format string, a ...interface{}) string {
			return fmt.Sprintf(format, a...)
		},
		"safeHTML": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprint(s))
		},
		// 基本算术函数
		"add": func(a, b interface{}) float64 {
			return toFloat64(a) + toFloat64(b)
		},
		"sub": func(a, b interface{}) float64 {
			return toFloat64(a) - toFloat64(b)
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		"div": func(a, b interface{}) float64 {
			bb := toFloat64(b)
			if bb == 0 {
				return 0
			}
			return toFloat64(a) / bb
		},
		// 计算饼图坐标
		"svgArcX": func(angle float64, radius float64) float64 {
			return radius * math.Sin(angle*math.Pi/180)
		},
		"svgArcY": func(angle float64, radius float64) float64 {
			return -radius * math.Cos(angle*math.Pi/180)
		},
		// 日期格式化
		"now": func() time.Time {
			return time.Now()
		},
		"date": func(layout string, t time.Time) string {
			return t.Format(layout)
		},
	}

	// 创建模板，使用自定义函数
	tmpl, err := template.New("template.html").Funcs(sprig.FuncMap()).Funcs(funcMap).ParseFS(templateFS, "template.html")
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// sendEmail 发送邮件
func (s *SecurityReportSender) sendEmail(subject, content string) error {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)

	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = s.fromEmail
	headers["To"] = s.formatToHeader() // 支持多收件人
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	var message bytes.Buffer
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n" + content)

	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)
	if err := smtp.SendMail(addr, auth, s.fromEmail, s.toEmails, message.Bytes()); err != nil {
		// 记录日志，便于排查
		s.logger.Error("邮件发送失败", zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("邮件发送成功", zap.Strings("to", s.toEmails))
	return nil
}

// formatToHeader 格式化 To 字段，支持多收件人
func (s *SecurityReportSender) formatToHeader() string {
	return fmt.Sprintf("%s", s.toEmails)
}

// Run 运行邮件发送任务
func (s *SecurityReportSender) Run(ctx context.Context, clusterStatus map[string]*security_check.ClusterStatus) error {
	// 检查S3数据是否存在今天的目录
	if err := s.validateS3Data(ctx, clusterStatus); err != nil {
		s.logger.Error("安全巡检数据验证失败", zap.Error(err))
		return fmt.Errorf("failed to validate S3 security data: %w", err)
	}

	// 收集数据
	reports, err := s.collectData(ctx, clusterStatus)
	if err != nil {
		s.logger.Error("数据收集失败", zap.Error(err))
		return fmt.Errorf("failed to collect report data: %w", err)
	}

	// 生成邮件数据
	emailData := s.generateEmailData(reports)

	// 生成邮件内容
	content, err := s.generateEmailContent(emailData)
	if err != nil {
		s.logger.Error("邮件内容生成失败", zap.Error(err))
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// 生成邮件主题
	subject := fmt.Sprintf("集群安全巡检报告 - %s", time.Now().Format(DateFormat))

	// 发送邮件
	if err := s.sendEmail(subject, content); err != nil {
		s.logger.Error("邮件发送失败", zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("巡检报告邮件发送流程完成。")
	return nil
}

// validateS3Data 验证S3数据是否包含今天的目录
func (s *SecurityReportSender) validateS3Data(ctx context.Context, clusterStatus map[string]*security_check.ClusterStatus) error {
	// 检查是否有任何集群状态信息
	if len(clusterStatus) == 0 {
		return fmt.Errorf("no cluster status information available")
	}

	// 获取今天的日期字符串 (格式: 2023-01-15)
	todayStr := time.Now().Format(DateFormat)

	// 检查是否有今天的数据
	hasTodayData := false
	for _, status := range clusterStatus {
		if status != nil && status.TodayDataExists {
			hasTodayData = true
			break
		}
	}

	if !hasTodayData {
		// 没有任何集群有今天的数据，认为数据异常
		return fmt.Errorf("no data found for today (%s) in S3 safeconf-check directory", todayStr)
	}

	return nil
}

// toFloat64 converts various types to float64
func toFloat64(a interface{}) float64 {
	switch v := a.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}
