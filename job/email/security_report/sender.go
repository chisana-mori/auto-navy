package security_report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"gorm.io/gorm"

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
}

// NewSecurityReportSender 创建安全报告发送器
func NewSecurityReportSender(db *gorm.DB, smtpHost string, smtpPort int, smtpUser, smtpPassword, fromEmail string, toEmails []string) *SecurityReportSender {
	return &SecurityReportSender{
		db:           db,
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUser:     smtpUser,
		smtpPassword: smtpPassword,
		fromEmail:    fromEmail,
		toEmails:     toEmails,
	}
}

// collectData 收集报告数据
func (s *SecurityReportSender) collectData(ctx context.Context) ([]SecurityReportData, error) {
	var clusters []string
	if err := s.db.Model(&portal.SecurityCheck{}).
		Distinct("cluster_name").
		Pluck("cluster_name", &clusters).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

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
		Where("cluster_name = ? AND created_at >= ?", clusterName, time.Now().AddDate(0, 0, -1)).
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
				NodeType:  check.NodeType,
				NodeName:  check.NodeName,
				CheckType: check.CheckType,
				ItemName:  item.ItemName,
				ItemValue: item.ItemValue,
				Status:    item.Status,
			})
		}
	}

	return report, nil
}

// generateEmailData 生成邮件数据
func (s *SecurityReportSender) generateEmailData(reports []SecurityReportData) EmailTemplateData {
	data := EmailTemplateData{
		TotalClusters: len(reports),
	}

	nodeMap := make(map[string]bool) // 用于统计唯一节点
	for _, report := range reports {
		// 统计节点数量
		for _, result := range report.DetailedResults {
			nodeKey := fmt.Sprintf("%s/%s/%s", report.ClusterName, result.NodeType, result.NodeName)
			if !nodeMap[nodeKey] {
				nodeMap[nodeKey] = true
				data.TotalNodes++

				// 检查节点是否有失败项
				hasFailure := false
				var failedItems []FailedItem
				for _, detail := range report.DetailedResults {
					if detail.NodeType == result.NodeType && detail.NodeName == result.NodeName && !detail.Status {
						hasFailure = true
						failedItems = append(failedItems, FailedItem{
							ItemName:  detail.ItemName,
							ItemValue: detail.ItemValue,
						})
					}
				}

				if hasFailure {
					data.AbnormalNodes++
					data.AbnormalDetails = append(data.AbnormalDetails, AbnormalDetail{
						ClusterName: report.ClusterName,
						NodeType:    result.NodeType,
						NodeName:    result.NodeName,
						FailedItems: failedItems,
					})
				} else {
					data.NormalNodes++
				}
			}
		}

		// 统计检查项数量
		data.TotalChecks += report.TotalChecks
		data.PassedChecks += report.PassedChecks
		data.FailedChecks += report.FailedChecks
	}

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
func (s *SecurityReportSender) Run(ctx context.Context) error {
	// 收集数据
	reports, err := s.collectData(ctx)
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
