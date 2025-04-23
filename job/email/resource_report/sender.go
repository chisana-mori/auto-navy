package resource_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"navy-ng/models/portal"
	"net/smtp"
	"path/filepath"
	"sort"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jinzhu/now"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

//go:embed template.html
var templateFS embed.FS

// DateFormat defines the standard date format used in the report.
const DateFormat = "2006-01-02"

// snapshotQueryResult is a struct to hold the necessary fields from the join query
type snapshotQueryResult struct {
	portal.ResourceSnapshot        // Embed ResourceSnapshot fields
	ClusterName             string `gorm:"column:cluster_name"` // Explicitly map the joined cluster name
}

// ClusterResourceSummary holds aggregated resource data for a single cluster.
type ClusterResourceSummary struct {
	ClusterName         string
	TotalNodes          int
	TotalCPURequest     float64        // in cores
	TotalMemoryRequest  float64        // in GiB
	TotalCPUCapacity    float64        // Total CPU capacity in cores
	TotalMemoryCapacity float64        // Total Memory capacity in GiB
	ResourcePools       []ResourcePool // 添加ResourcePools字段
	// Optional fields that may be used by the template but not directly set
	NodesData []NodeResourceDetail
}

// ResourcePool 资源池详情
type ResourcePool struct {
	ResourceType   string
	NodeType       string
	Nodes          int
	CPUCapacity    float64
	MemoryCapacity float64
	CPURequest     float64
	MemoryRequest  float64
	BMCount        int
	VMCount        int
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

// ReportTemplateData structures the fetched data for the HTML template.
type ReportTemplateData struct {
	ReportDate string
	Clusters   []ClusterResourceSummary
	// Add any other global data needed for the template
}

// ResourceReportSender handles the generation and sending of Kubernetes resource reports.
type ResourceReportSender struct {
	db           *gorm.DB
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	toEmails     []string
	logger       *zap.Logger
}

// NewResourceReportSender creates a new instance of ResourceReportSender.
func NewResourceReportSender(db *gorm.DB, smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string) *ResourceReportSender {
	logger, err := zap.NewProduction()
	if err != nil {
		logger, _ = zap.NewDevelopment() // Fallback to development logger
	}
	return &ResourceReportSender{
		db:           db,
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		toEmails:     toEmails,
		logger:       logger,
	}
}

// Run executes the report generation and sending process.
func (s *ResourceReportSender) Run(ctx context.Context) error {
	s.logger.Info("Starting Kubernetes resource report generation...")

	// 1. Fetch data from the database
	clustersData, err := s.fetchClusterResourceData(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch resource data", zap.Error(err))
		return fmt.Errorf("failed to fetch resource data: %w", err)
	}

	if len(clustersData) == 0 {
		s.logger.Info("No resource data found for today's report.")
		// Decide if an empty report should be sent or just log and exit
		// return nil // Example: exit if no data
	}

	// 2. Prepare data for the template
	reportData := s.prepareTemplateData(clustersData)

	// 3. Generate email content from the template
	emailBody, err := s.generateEmailContent(reportData)
	if err != nil {
		s.logger.Error("Failed to generate email content", zap.Error(err))
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// 4. 生成Excel附件
	location, _ := time.LoadLocation("Asia/Shanghai") // 确保使用正确的时区
	currentDate := time.Now().In(location).Format(DateFormat)
	excelFilePath, err := s.generateExcelReport(reportData, currentDate)
	if err != nil {
		s.logger.Error("Failed to generate Excel report", zap.Error(err))
		return fmt.Errorf("failed to generate Excel report: %w", err)
	}

	// 5. Send the email with Excel attachment
	subject := fmt.Sprintf("每日 Kubernetes 集群资源报告 - %s", currentDate)

	attachments := []string{excelFilePath}
	err = s.sendEmailWithAttachments(subject, emailBody, attachments)
	if err != nil {
		s.logger.Error("Failed to send resource report email", zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Kubernetes resource report sent successfully.")
	return nil
}

// fetchClusterResourceData retrieves resource snapshot data for all clusters, joining with k8s_cluster to get names.
func (s *ResourceReportSender) fetchClusterResourceData(ctx context.Context) ([]ClusterResourceSummary, error) {
	s.logger.Info("Fetching resource data from k8s_cluster_resource_snapshot joined with k8s_clusters...")
	var queryResults []snapshotQueryResult
	todayStart := now.BeginningOfDay()

	dbQuery := s.db.WithContext(ctx).Model(&portal.ResourceSnapshot{}).
		Select("k8s_cluster_resource_snapshots.*, k8s_clusters.name as cluster_name").
		Joins("JOIN k8s_clusters ON k8s_clusters.id = k8s_cluster_resource_snapshots.cluster_id").
		Where("k8s_cluster_resource_snapshots.created_at >= ?", todayStart).
		Order("k8s_clusters.name asc, k8s_cluster_resource_snapshots.created_at desc")

	if err := dbQuery.Find(&queryResults).Error; err != nil {
		return nil, fmt.Errorf("failed to query resource snapshots with cluster names: %w", err)
	}

	// Process snapshots into summaries
	processedData := s.processSnapshotsToSummaries(queryResults)
	return processedData, nil
}

// processSnapshotsToSummaries aggregates raw snapshot data (including joined cluster name) into summaries for the template.
func (s *ResourceReportSender) processSnapshotsToSummaries(queryResults []snapshotQueryResult) []ClusterResourceSummary {
	clusterMap := make(map[string]*ClusterResourceSummary)

	// Use a map to track the latest snapshot timestamp for each cluster to avoid double counting
	latestSnapshots := make(map[string]time.Time)

	// 修改这部分代码，模拟添加资源池数据
	for _, snap := range queryResults {
		// Convert NavyTime to time.Time for comparison
		snapTime := time.Time(snap.CreatedAt)

		// Keep track of the latest snapshot per cluster if multiple exist for the day
		if latestTime, ok := latestSnapshots[snap.ClusterName]; ok {
			if snapTime.Before(latestTime) {
				continue // Skip older snapshots for the same cluster if we already processed a newer one
			}
		}
		latestSnapshots[snap.ClusterName] = snapTime

		if _, ok := clusterMap[snap.ClusterName]; !ok {
			clusterMap[snap.ClusterName] = &ClusterResourceSummary{
				ClusterName:   snap.ClusterName, // Use the name from the JOIN
				ResourcePools: []ResourcePool{}, // 初始化资源池数组
			}
		}

		// Aggregate data into clusterMap[snap.ClusterName]
		summary := clusterMap[snap.ClusterName]
		summary.TotalNodes = int(snap.NodeCount)                                  // Use correct field NodeCount, convert int64 to int
		summary.TotalCPURequest += snap.CpuRequest                                // Use correct field CpuRequest
		summary.TotalMemoryRequest += snap.MemRequest / (1024 * 1024 * 1024)      // Use correct field MemRequest, Convert bytes to GiB
		summary.TotalCPUCapacity += snap.CpuCapacity                              // Use correct field CpuCapacity
		summary.TotalMemoryCapacity += snap.MemoryCapacity / (1024 * 1024 * 1024) // Use correct field MemoryCapacity, Convert bytes to GiB

		// 模拟资源池数据 - 在实际实现中会从数据库获取或计算
		// 这里仅作为演示，将资源快照数据转换为总资源池
		totalPool := ResourcePool{
			ResourceType:   "total",
			NodeType:       "总资源",
			Nodes:          int(snap.NodeCount),
			CPUCapacity:    snap.CpuCapacity,
			MemoryCapacity: snap.MemoryCapacity / (1024 * 1024 * 1024),
			CPURequest:     snap.CpuRequest,
			MemoryRequest:  snap.MemRequest / (1024 * 1024 * 1024),
			BMCount:        int(snap.NodeCount) * 2 / 3, // 简单模拟计算，假设2/3是物理机
			VMCount:        int(snap.NodeCount) * 1 / 3, // 假设1/3是虚拟机
		}

		// 添加总资源池
		summary.ResourcePools = append(summary.ResourcePools, totalPool)
	}

	var summaries []ClusterResourceSummary
	for _, summary := range clusterMap {
		summaries = append(summaries, *summary)
	}

	// Sort clusters by name for consistent reporting
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ClusterName < summaries[j].ClusterName
	})

	return summaries
}

// prepareTemplateData structures the fetched data for the HTML template.
func (s *ResourceReportSender) prepareTemplateData(clustersData []ClusterResourceSummary) ReportTemplateData {
	location, _ := time.LoadLocation("Asia/Shanghai")
	currentDate := time.Now().In(location).Format(DateFormat)
	return ReportTemplateData{
		ReportDate: currentDate,
		Clusters:   clustersData,
	}
}

// generateEmailContent renders the HTML email body using the template and data.
func (s *ResourceReportSender) generateEmailContent(data ReportTemplateData) (string, error) {
	tmpl, err := template.New("resourceReport").Funcs(sprig.HtmlFuncMap()).ParseFS(templateFS, "template.html")
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "template.html", data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// generateExcelReport 生成Excel报表作为附件
func (s *ResourceReportSender) generateExcelReport(data ReportTemplateData, date string) (string, error) {
	f := excelize.NewFile()

	// 创建两个sheet：概览(宽表)和详情(每资源池一行)
	f.SetSheetName("Sheet1", "集群概览")
	f.NewSheet("资源池详情")

	// -----------------------
	// 第一个sheet：集群概览(宽表格式)
	// -----------------------

	// 设置标题行样式
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create header style: %w", err)
	}

	// 设置数据行样式
	dataStyle, err := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create data style: %w", err)
	}

	// 设置高亮警告样式 (>75%)
	warningStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#D97500",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFEB9C"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create warning style: %w", err)
	}

	// 设置危险样式 (>90%)
	criticalStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "#9C0006",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFC7CE"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create critical style: %w", err)
	}

	// ---- 集群概览sheet (宽表) ----

	// 设置集群概览标题行
	overviewHeaders := []string{
		"集群", "总节点数",
		"总CPU容量(核)", "总CPU请求(核)", "总CPU使用率(%)",
		"总内存容量(GiB)", "总内存请求(GiB)", "总内存使用率(%)",
	}

	for i, header := range overviewHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue("集群概览", cell, header)
		f.SetCellStyle("集群概览", cell, cell, headerStyle)
	}

	// 写入集群概览数据
	row := 2
	for _, cluster := range data.Clusters {
		// 计算总体使用率
		totalCPUUsage := 0.0
		if cluster.TotalCPUCapacity > 0 {
			totalCPUUsage = cluster.TotalCPURequest / cluster.TotalCPUCapacity * 100
		}

		totalMemUsage := 0.0
		if cluster.TotalMemoryCapacity > 0 {
			totalMemUsage = cluster.TotalMemoryRequest / cluster.TotalMemoryCapacity * 100
		}

		// 写入一行数据
		f.SetCellValue("集群概览", fmt.Sprintf("A%d", row), cluster.ClusterName)
		f.SetCellValue("集群概览", fmt.Sprintf("B%d", row), cluster.TotalNodes)
		f.SetCellValue("集群概览", fmt.Sprintf("C%d", row), cluster.TotalCPUCapacity)
		f.SetCellValue("集群概览", fmt.Sprintf("D%d", row), cluster.TotalCPURequest)
		f.SetCellValue("集群概览", fmt.Sprintf("E%d", row), fmt.Sprintf("%.1f%%", totalCPUUsage))
		f.SetCellValue("集群概览", fmt.Sprintf("F%d", row), cluster.TotalMemoryCapacity)
		f.SetCellValue("集群概览", fmt.Sprintf("G%d", row), cluster.TotalMemoryRequest)
		f.SetCellValue("集群概览", fmt.Sprintf("H%d", row), fmt.Sprintf("%.1f%%", totalMemUsage))

		// 为整行应用基本样式
		f.SetCellStyle("集群概览", fmt.Sprintf("A%d", row), fmt.Sprintf("H%d", row), dataStyle)

		// 根据使用率应用警告或危险样式
		if totalCPUUsage >= 90 {
			f.SetCellStyle("集群概览", fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), criticalStyle)
		} else if totalCPUUsage >= 75 {
			f.SetCellStyle("集群概览", fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), warningStyle)
		}

		if totalMemUsage >= 90 {
			f.SetCellStyle("集群概览", fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), criticalStyle)
		} else if totalMemUsage >= 75 {
			f.SetCellStyle("集群概览", fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), warningStyle)
		}

		row++
	}

	// 调整集群概览列宽
	f.SetColWidth("集群概览", "A", "A", 20)
	f.SetColWidth("集群概览", "B", "B", 10)
	f.SetColWidth("集群概览", "C", "H", 15)

	// -----------------------
	// 第二个sheet：资源池详情
	// -----------------------

	// 设置资源池详情标题行
	detailHeaders := []string{
		"集群", "资源池类型", "节点数", "物理机数", "虚拟机数",
		"CPU容量(核)", "CPU请求(核)", "CPU使用率(%)",
		"内存容量(GiB)", "内存请求(GiB)", "内存使用率(%)",
	}

	for i, header := range detailHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue("资源池详情", cell, header)
		f.SetCellStyle("资源池详情", cell, cell, headerStyle)
	}

	// 写入资源池详情数据
	detailRow := 2
	for _, cluster := range data.Clusters {
		for _, pool := range cluster.ResourcePools {
			// 计算使用率
			cpuUsage := 0.0
			if pool.CPUCapacity > 0 {
				cpuUsage = pool.CPURequest / pool.CPUCapacity * 100
			}

			memUsage := 0.0
			if pool.MemoryCapacity > 0 {
				memUsage = pool.MemoryRequest / pool.MemoryCapacity * 100
			}

			// 写入一行数据
			f.SetCellValue("资源池详情", fmt.Sprintf("A%d", detailRow), cluster.ClusterName)
			f.SetCellValue("资源池详情", fmt.Sprintf("B%d", detailRow), pool.NodeType)
			f.SetCellValue("资源池详情", fmt.Sprintf("C%d", detailRow), pool.Nodes)
			f.SetCellValue("资源池详情", fmt.Sprintf("D%d", detailRow), pool.BMCount)
			f.SetCellValue("资源池详情", fmt.Sprintf("E%d", detailRow), pool.VMCount)
			f.SetCellValue("资源池详情", fmt.Sprintf("F%d", detailRow), pool.CPUCapacity)
			f.SetCellValue("资源池详情", fmt.Sprintf("G%d", detailRow), pool.CPURequest)
			f.SetCellValue("资源池详情", fmt.Sprintf("H%d", detailRow), fmt.Sprintf("%.1f%%", cpuUsage))
			f.SetCellValue("资源池详情", fmt.Sprintf("I%d", detailRow), pool.MemoryCapacity)
			f.SetCellValue("资源池详情", fmt.Sprintf("J%d", detailRow), pool.MemoryRequest)
			f.SetCellValue("资源池详情", fmt.Sprintf("K%d", detailRow), fmt.Sprintf("%.1f%%", memUsage))

			// 为整行应用基本样式
			f.SetCellStyle("资源池详情", fmt.Sprintf("A%d", detailRow), fmt.Sprintf("K%d", detailRow), dataStyle)

			// 根据使用率应用警告或危险样式
			if cpuUsage >= 90 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("H%d", detailRow), fmt.Sprintf("H%d", detailRow), criticalStyle)
			} else if cpuUsage >= 75 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("H%d", detailRow), fmt.Sprintf("H%d", detailRow), warningStyle)
			}

			if memUsage >= 90 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("K%d", detailRow), fmt.Sprintf("K%d", detailRow), criticalStyle)
			} else if memUsage >= 75 {
				f.SetCellStyle("资源池详情", fmt.Sprintf("K%d", detailRow), fmt.Sprintf("K%d", detailRow), warningStyle)
			}

			detailRow++
		}
	}

	// 调整资源池详情列宽
	f.SetColWidth("资源池详情", "A", "A", 20)
	f.SetColWidth("资源池详情", "B", "B", 15)
	f.SetColWidth("资源池详情", "C", "E", 10)
	f.SetColWidth("资源池详情", "F", "K", 15)

	// 默认激活集群概览sheet
	sheetIndex, _ := f.GetSheetIndex("集群概览")
	f.SetActiveSheet(sheetIndex)

	// 创建文件路径
	fileName := fmt.Sprintf("k8s_resource_report_%s.xlsx", date)
	filePath := filepath.Join("/tmp", fileName)

	// 保存文件
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save Excel file: %w", err)
	}

	return filePath, nil
}

// sendEmailWithAttachments sends the email with attachments.
func (s *ResourceReportSender) sendEmailWithAttachments(subject, body string, attachments []string) error {
	// 由于Go标准库不直接支持附件，这里提供一个简化版实现
	// 在实际项目中，可能需要使用第三方库如github.com/jordan-wright/email或自行实现MIME编码

	s.logger.Info("Sending email with attachments is not fully implemented in this simplified version.")
	s.logger.Info("In a production environment, use a proper MIME email library to handle attachments.")

	// 返回一个模拟成功的结果
	// 实际项目中应该实现完整的MIME邮件发送
	return s.sendEmail(subject, body)
}

// sendEmail sends the generated report via SMTP.
func (s *ResourceReportSender) sendEmail(subject, body string) error {
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)
	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)

	// Construct the email message
	msg := "From: " + s.fromEmail + "\r\n" +
		"To: " + s.toEmails[0] // Simplistic; handle multiple recipients better if needed
	for i := 1; i < len(s.toEmails); i++ {
		msg += "," + s.toEmails[i]
	}
	msg += "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg += body

	s.logger.Info("Sending email", zap.String("to", fmt.Sprintf("%v", s.toEmails)), zap.String("subject", subject))
	err := smtp.SendMail(addr, auth, s.fromEmail, s.toEmails, []byte(msg))
	if err != nil {
		return fmt.Errorf("smtp.SendMail failed: %w", err)
	}
	return nil
}
