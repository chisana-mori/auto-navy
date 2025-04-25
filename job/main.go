package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap" // Import zap

	"navy-ng/job/chore/security_check"
	"navy-ng/job/email/resource_report" // 新增：导入资源报告包
	"navy-ng/job/email/security_report"
	// Assuming initDB and initS3Client are defined in job/init.go or another package
)

// Removed placeholder initDB and initS3Client functions as they likely exist elsewhere

var (
	rootCmd = &cobra.Command{
		Use:   "job",
		Short: "Navy job runner",
		Long:  `Navy job runner is a CLI tool for running various jobs including email notifications and data collection tasks.`,
	}

	// 全局标志
	// 数据库连接参数
	mysqlDSN string

	// S3相关参数
	s3Bucket    string
	s3AccessKey string
	s3SecretKey string
	s3Endpoint  string

	// 其他控制标志
	sendEmail bool // 是否发送邮件
)

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&mysqlDSN, "mysql-dsn", "", "MySQL connection string (default: root:root@tcp(127.0.0.1:3306)/navy?charset=utf8mb4&parseTime=True&loc=Local)")
	rootCmd.PersistentFlags().StringVar(&s3Bucket, "s3-bucket", "", "S3 bucket name")
	rootCmd.PersistentFlags().StringVar(&s3AccessKey, "s3-access-key", "", "S3 access key")
	rootCmd.PersistentFlags().StringVar(&s3SecretKey, "s3-secret-key", "", "S3 secret key")
	rootCmd.PersistentFlags().StringVar(&s3Endpoint, "s3-endpoint", "", "S3 endpoint URL")
	rootCmd.PersistentFlags().BoolVar(&sendEmail, "send-email", false, "Send email report after security check")

	// 添加子命令
	rootCmd.AddCommand(choreCmd)
	rootCmd.AddCommand(emailCmd)
}

// chore 命令
var choreCmd = &cobra.Command{
	Use:   "chore",
	Short: "Run chore jobs",
	Long:  `Run chore jobs for data collection and processing.`,
}

// email 命令
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Run email notification jobs",
	Long:  `Run email notification jobs for various alerts and reports.`,
}

// security-check 命令
var securityCheckCmd = &cobra.Command{
	Use:   "security-check",
	Short: "Run security check collection",
	Long:  `Collect security check results from S3 and store them in the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 初始化数据库连接
		db, err := initDB(mysqlDSN)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		// 初始化S3客户端
		s3Client, err := initS3Client(s3AccessKey, s3SecretKey, s3Endpoint)
		if err != nil {
			return fmt.Errorf("failed to initialize S3 client: %w", err)
		}

		if s3Bucket == "" {
			return fmt.Errorf("S3_BUCKET is required")
		}

		// 创建并运行采集器
		collector := security_check.NewS3ConfigCollector(s3Client, db, s3Bucket)
		clusterStatus, err := collector.Run(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to run security check collection: %w", err)
		}

		// 如果设置了邮件参数，则发送报告邮件
		if sendEmail && smtpHost != "" && fromEmail != "" && toEmails != "" {
			// 解析收件人列表
			recipients := strings.Split(toEmails, ",")
			if len(recipients) == 0 {
				return fmt.Errorf("at least one recipient email is required")
			}

			// 创建并运行邮件发送器
			sender := security_report.NewSecurityReportSender(
				db,
				smtpHost,
				smtpPort,
				smtpUser,
				smtpPassword,
				fromEmail,
				recipients,
			)
			if err := sender.Run(cmd.Context(), clusterStatus); err != nil {
				return fmt.Errorf("failed to send security report: %w", err)
			}
		}

		return nil
	},
}

// security-report 命令
var (
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	toEmails     string

	securityReportCmd = &cobra.Command{
		Use:   "security-report",
		Short: "Send security check report email",
		Long:  `Generate and send security check report email to specified recipients.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化数据库连接
			db, err := initDB(mysqlDSN)
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}

			// 解析收件人列表
			recipients := strings.Split(toEmails, ",")
			if len(recipients) == 0 {
				return fmt.Errorf("at least one recipient email is required")
			}

			// 创建并运行邮件发送器
			sender := security_report.NewSecurityReportSender(
				db,
				smtpHost,
				smtpPort,
				smtpUser,
				smtpPassword,
				fromEmail,
				recipients,
			)
			// 创建空的集群状态映射，因为单独运行邮件发送器时没有集群状态信息
			emptyClusterStatus := make(map[string]*security_check.ClusterStatus)
			if err := sender.Run(cmd.Context(), emptyClusterStatus); err != nil {
				return fmt.Errorf("failed to send security report: %w", err)
			}

			return nil
		},
	}
)

// -------- 新增 K8s 资源报告命令 ---------
var (
	// 资源报告相关参数
	generalClusters string // 通用集群列表，用于计算Pod密度，使用逗号分隔
	environment     string // 环境类型："prd" 或 "test"

	resourceReportCmd = &cobra.Command{
		Use:   "resource-report",
		Short: "Send Kubernetes resource report email",
		Long:  `Generates and sends a daily Kubernetes cluster resource usage report via email.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logger
			logger := zap.NewExample() // Or zap.NewProduction()
			defer logger.Sync()        // flushes buffer, if any

			// 初始化数据库连接
			db, err := initDB(mysqlDSN)
			if err != nil {
				// Log error before returning
				logger.Error("Failed to initialize database", zap.String("dsn", mysqlDSN), zap.Error(err))
				return fmt.Errorf("failed to initialize database: %w", err)
			}

			// 解析收件人列表
			recipients := strings.Split(toEmails, ",")
			if len(recipients) == 0 {
				logger.Error("No recipient emails provided", zap.String("toEmails", toEmails))
				return fmt.Errorf("at least one recipient email is required")
			}

			// 解析通用集群列表
			var generalClusterList []string
			if generalClusters != "" {
				generalClusterList = strings.Split(generalClusters, ",")
			}

			// 创建并运行资源报告发送器
			sender := resource_report.NewResourceReportSender(
				db,
				smtpHost,
				smtpPort,
				smtpUser,
				smtpPassword,
				fromEmail,
				recipients,
				generalClusterList,
				environment, // 传入环境类型
				logger,      // Pass the logger instance
			)
			if err := sender.Run(cmd.Context()); err != nil {
				logger.Error("Failed to send resource report", zap.Error(err))
				return fmt.Errorf("failed to send resource report: %w", err)
			}

			logger.Info("Resource report sent successfully")
			return nil
		},
	}
)

// ----------------------------------------

func init() {
	// 将security-check命令添加到chore命令下
	choreCmd.AddCommand(securityCheckCmd)

	// TODO: 添加更多的chore子命令
	// choreCmd.AddCommand(otherChoreCmd)

	// 将security-report命令添加到email命令下
	emailCmd.AddCommand(securityReportCmd)

	// TODO: 添加更多的email子命令
	// emailCmd.AddCommand(otherEmailCmd)

	// 添加security-check命令的邮件相关标志
	securityCheckCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server host")
	securityCheckCmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	securityCheckCmd.Flags().StringVar(&smtpUser, "smtp-user", "", "SMTP username")
	securityCheckCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "SMTP password")
	securityCheckCmd.Flags().StringVar(&fromEmail, "from", "", "Sender email address")
	securityCheckCmd.Flags().StringVar(&toEmails, "to", "", "Comma-separated list of recipient email addresses")

	// 添加security-report命令的标志
	securityReportCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server host")
	securityReportCmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	securityReportCmd.Flags().StringVar(&smtpUser, "smtp-user", "", "SMTP username")
	securityReportCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "SMTP password")
	securityReportCmd.Flags().StringVar(&fromEmail, "from", "", "Sender email address")
	securityReportCmd.Flags().StringVar(&toEmails, "to", "", "Comma-separated list of recipient email addresses")

	// 标记必需的标志
	securityReportCmd.MarkFlagRequired("smtp-host")
	securityReportCmd.MarkFlagRequired("smtp-user")
	securityReportCmd.MarkFlagRequired("smtp-password")
	securityReportCmd.MarkFlagRequired("from")
	securityReportCmd.MarkFlagRequired("to")

	// --------- 新增 K8s 资源报告相关标志 ---------
	resourceReportCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP server host")
	resourceReportCmd.Flags().IntVar(&smtpPort, "smtp-port", 587, "SMTP server port")
	resourceReportCmd.Flags().StringVar(&smtpUser, "smtp-user", "", "SMTP username")
	resourceReportCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "SMTP password")
	resourceReportCmd.Flags().StringVar(&fromEmail, "from", "", "Sender email address")
	resourceReportCmd.Flags().StringVar(&toEmails, "to", "", "Comma-separated list of recipient email addresses")
	resourceReportCmd.Flags().StringVar(&generalClusters, "general-clusters", "", "Comma-separated list of general purpose clusters for POD density calculation")
	resourceReportCmd.Flags().StringVar(&environment, "env", "prd", "Environment type: 'prd' or 'test'. Production environment uses standard thresholds, test environment ignores low utilization and increases high utilization thresholds by 5%")

	// 标记资源报告命令必需的标志
	resourceReportCmd.MarkFlagRequired("smtp-host")
	resourceReportCmd.MarkFlagRequired("smtp-user")
	resourceReportCmd.MarkFlagRequired("smtp-password")
	resourceReportCmd.MarkFlagRequired("from")
	resourceReportCmd.MarkFlagRequired("to")

	// 将 resource-report 命令添加到 email 命令下
	emailCmd.AddCommand(resourceReportCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Use standard log here as zap might not be initialized if Execute fails early
		log.Printf("Error executing root command: %v\n", err)
		os.Exit(1)
	}
}
