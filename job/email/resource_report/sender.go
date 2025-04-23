package resource_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"navy-ng/models/portal"
	"net/smtp"
	"sort"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/jinzhu/now"
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
	TotalCPURequest     float64 // in cores
	TotalMemoryRequest  float64 // in GiB
	TotalCPUCapacity    float64 // Total CPU capacity in cores
	TotalMemoryCapacity float64 // Total Memory capacity in GiB
	// Optional fields that may be used by the template but not directly set
	NodesData []NodeResourceDetail
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

	// 4. Send the email
	location, _ := time.LoadLocation("Asia/Shanghai") // Ensure correct timezone
	currentDate := time.Now().In(location).Format(DateFormat)
	subject := fmt.Sprintf("每日 Kubernetes 集群资源报告 - %s", currentDate)

	err = s.sendEmail(subject, emailBody)
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
				ClusterName: snap.ClusterName, // Use the name from the JOIN
				// Initialize other fields as needed
			}
		}

		// Aggregate data into clusterMap[snap.ClusterName]
		summary := clusterMap[snap.ClusterName]
		summary.TotalNodes = int(snap.NodeCount)                                  // Use correct field NodeCount, convert int64 to int
		summary.TotalCPURequest += snap.CpuRequest                                // Use correct field CpuRequest
		summary.TotalMemoryRequest += snap.MemRequest / (1024 * 1024 * 1024)      // Use correct field MemRequest, Convert bytes to GiB
		summary.TotalCPUCapacity += snap.CpuCapacity                              // Use correct field CpuCapacity
		summary.TotalMemoryCapacity += snap.MemoryCapacity / (1024 * 1024 * 1024) // Use correct field MemoryCapacity, Convert bytes to GiB

		// Note: NodesData population would require additional logic and perhaps additional queries
		// Future enhancement: Query individual node data and populate NodesData slice
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
