package security_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"sort"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm"

	"navy-ng/job/chore/security_check"
	"navy-ng/models/portal"
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
}

// NewSecurityReportSender 创建安全报告发送器
func NewSecurityReportSender(db *gorm.DB, smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string) *SecurityReportSender {
	return &SecurityReportSender{
		db:            db,
		smtpHost:      smtpHost,
		smtpPort:      smtpPort,
		smtpUser:      smtpUser,
		smtpPassword:  smtpPassword,
		fromEmail:     fromEmail,
		toEmails:      toEmails,
		clusterStatus: make(map[string]*security_check.ClusterStatus),
	}
}

// collectData 收集报告数据
func (s *SecurityReportSender) collectData(ctx context.Context, clusterStatus map[string]*security_check.ClusterStatus) ([]SecurityReportData, error) {
	// 从数据库获取所有集群名称
	var clusters []string
	if err := s.db.Model(&portal.SecurityCheck{}).
		Distinct("cluster_name").
		Pluck("cluster_name", &clusters).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	// 将集群状态信息保存到全局变量
	s.clusterStatus = clusterStatus

	var reports []SecurityReportData
	for _, cluster := range clusters {
		report, err := s.collectClusterData(ctx, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to collect data for cluster %s: %w", cluster, err)
		}
		reports = append(reports, report)
	}

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
	}

	nodeMap := make(map[string]bool)                          // 用于统计全局唯一节点
	clusterNodeFailureMap := make(map[string]map[string]bool) // cluster -> nodeKey -> hasFailure
	checkItemFailureCounts := make(map[string]int)            // itemName -> totalFailureCount

	// --- First pass: Calculate global stats and identify abnormal nodes per cluster ---
	for _, report := range reports {
		if _, ok := clusterNodeFailureMap[report.ClusterName]; !ok {
			clusterNodeFailureMap[report.ClusterName] = make(map[string]bool)
		}

		for _, result := range report.DetailedResults {
			nodeKey := fmt.Sprintf("%s/%s/%s", report.ClusterName, result.NodeType, result.NodeName)

			// Global node count
			if !nodeMap[nodeKey] {
				nodeMap[nodeKey] = true
				data.TotalNodes++
			}

			// Mark node as having failure if any check fails
			if !result.Status {
				clusterNodeFailureMap[report.ClusterName][nodeKey] = true
			} else if _, exists := clusterNodeFailureMap[report.ClusterName][nodeKey]; !exists {
				// Initialize with false if no failure seen yet for this node in this cluster
				clusterNodeFailureMap[report.ClusterName][nodeKey] = false
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
		data.AbnormalNodes += clusterAbnormalNodes              // Add to global abnormal count
		data.NormalNodes = data.TotalNodes - data.AbnormalNodes // Calculate global normal count

		// 获取集群状态信息
		clusterExists := true
		if status, ok := s.clusterStatus[report.ClusterName]; ok {
			clusterExists = status.Exists
		}

		// 生成锚点ID
		anchorID := fmt.Sprintf("cluster-%s", report.ClusterName)

		// Determine cluster status color
		statusColor := "green"
		if !clusterExists {
			statusColor = "red" // 集群不存在，标记为红色
		} else if clusterAbnormalNodes > 0 {
			statusColor = "red" // Any abnormal node makes the cluster red
		} else if clusterFailedChecks > 0 {
			statusColor = "yellow" // No abnormal nodes, but some failed checks
		}

		// Add cluster health summary
		data.ClusterHealthSummary = append(data.ClusterHealthSummary, ClusterHealthInfo{
			ClusterName:   report.ClusterName,
			StatusColor:   statusColor,
			AbnormalNodes: clusterAbnormalNodes,
			FailedChecks:  clusterFailedChecks, // Use the per-cluster failed count
			Exists:        clusterExists,
			AnchorID:      anchorID,
		})

		// Note: MissingNodes logic needs to be handled separately if required by template
		// The current logic focuses on AbnormalDetails and ClusterHealthSummary
	}

	// --- Third pass: Populate CheckItemFailureSummary (Heatmap data) ---
	for itemName, count := range checkItemFailureCounts {
		heatColor := ""
		if count > 5 {
			heatColor = "heat-level-high"
		} else if count >= 3 {
			heatColor = "heat-level-2"
		} else if count > 0 {
			heatColor = "heat-level-1"
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

	return data
}

// generateEmailContent 生成邮件内容
func (s *SecurityReportSender) generateEmailContent(data EmailTemplateData) (string, error) {
	tmpl, err := template.ParseFS(templateFS, "template.html")
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
		fmt.Printf("[ERROR] 邮件发送失败: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("[INFO] 邮件发送成功，收件人: %v\n", s.toEmails)
	return nil
}

// formatToHeader 格式化 To 字段，支持多收件人
func (s *SecurityReportSender) formatToHeader() string {
	return fmt.Sprintf("%s", s.toEmails)
}

// Run 运行邮件发送任务
func (s *SecurityReportSender) Run(ctx context.Context, clusterStatus map[string]*security_check.ClusterStatus) error {
	// 收集数据
	reports, err := s.collectData(ctx, clusterStatus)
	if err != nil {
		fmt.Printf("[ERROR] 数据收集失败: %v\n", err)
		return fmt.Errorf("failed to collect report data: %w", err)
	}

	// 生成邮件数据
	emailData := s.generateEmailData(reports)

	// 生成邮件内容
	content, err := s.generateEmailContent(emailData)
	if err != nil {
		fmt.Printf("[ERROR] 邮件内容生成失败: %v\n", err)
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// 生成邮件主题
	subject := fmt.Sprintf("集群安全巡检报告 - %s", time.Now().Format("2006-01-02"))

	// 发送邮件
	if err := s.sendEmail(subject, content); err != nil {
		fmt.Printf("[ERROR] 邮件发送失败: %v\n", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("[INFO] 巡检报告邮件发送流程完成。")
	return nil
}
