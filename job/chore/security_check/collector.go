package security_check

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// S3 路径常量
const (
	S3PathPrefix = "safeconf-check/"
	DateFormat   = time.DateOnly
)

// 集群状态常量
const (
	ClusterStatusInit    = "init"
	ClusterStatusRunning = "running"
)

// 节点类型常量
const (
	NodeTypeMaster = "master"
	NodeTypeEtcd   = "etcd"
	NodeTypeWorker = "node"
)

// 检查类型常量
const (
	CheckTypeK8s     = "k8s"
	CheckTypeRuntime = "runtime"
)

// S3ConfigCollector S3配置采集器
type S3ConfigCollector struct {
	s3Client *s3.Client
	db       *gorm.DB
	bucket   string
	// 集群状态跟踪
	clusterStatus map[string]*ClusterStatus
	logger        *zap.Logger
}

// ClusterStatus 集群状态
type ClusterStatus struct {
	Exists          bool     // 集群在S3中是否存在
	HasFailures     bool     // 集群是否有失败项
	FailureNodes    []string // 有失败项的节点列表
	TodayDataExists bool     // 集群是否有今天的数据
}

// NewS3ConfigCollector 创建新的采集器实例
func NewS3ConfigCollector(s3Client *s3.Client, db *gorm.DB, bucket string) *S3ConfigCollector {
	logger, err := zap.NewProduction()
	if err != nil {
		// 如果无法创建生产级别的logger，回退到开发级别
		logger, _ = zap.NewDevelopment()
	}
	return &S3ConfigCollector{
		s3Client:      s3Client,
		db:            db,
		bucket:        bucket,
		clusterStatus: make(map[string]*ClusterStatus),
		logger:        logger,
	}
}

// getFormattedToday 获取格式化的今日日期
func getFormattedToday() string {
	return time.Now().Format(DateFormat)
}

// getTodayClusters 获取今天有数据的集群列表
func (c *S3ConfigCollector) getTodayClusters(ctx context.Context, todayStr string) (map[string]bool, error) {
	todayPrefix := fmt.Sprintf("%s%s/", S3PathPrefix, todayStr)
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(todayPrefix),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err // 错误在上层处理
	}

	todayClusters := make(map[string]bool)
	for _, prefix := range result.CommonPrefixes {
		parts := filterEmptyStrings(strings.Split(strings.TrimPrefix(*prefix.Prefix, todayPrefix), "/"))
		if len(parts) > 0 {
			todayClusters[parts[0]] = true
		}
	}

	return todayClusters, nil
}

// getAllS3Clusters 获取所有日期的集群列表
func (c *S3ConfigCollector) getAllS3Clusters(ctx context.Context) (map[string]bool, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(S3PathPrefix),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters from S3: %w", err)
	}

	s3Clusters := make(map[string]bool)
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	for _, prefix := range result.CommonPrefixes {
		parts := filterEmptyStrings(strings.Split(strings.TrimPrefix(*prefix.Prefix, S3PathPrefix), "/"))
		if len(parts) > 0 && datePattern.MatchString(parts[0]) && len(parts) > 1 {
			s3Clusters[parts[1]] = true
		}
	}

	return s3Clusters, nil
}

// getDBClusters 从数据库中获取集群列表
func (c *S3ConfigCollector) getDBClusters(ctx context.Context) ([]portal.K8sCluster, error) {
	var dbClusters []portal.K8sCluster
	if err := c.db.WithContext(ctx).Where("deleted = ''").Find(&dbClusters).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters from database: %w", err)
	}
	return dbClusters, nil
}

// initClusterStatus 初始化集群状态并确定需要处理的集群
func (c *S3ConfigCollector) initClusterStatus(dbClusters []portal.K8sCluster, s3Clusters, todayClusters map[string]bool) []string {
	processClusters := make([]string, 0, len(dbClusters))

	for _, cluster := range dbClusters {
		exists := s3Clusters[cluster.ClusterName]
		hasTodayData := todayClusters[cluster.ClusterName]

		c.clusterStatus[cluster.ClusterName] = &ClusterStatus{
			Exists:          exists,
			HasFailures:     false,
			FailureNodes:    make([]string, 0),
			TodayDataExists: hasTodayData,
		}

		if exists {
			processClusters = append(processClusters, cluster.ClusterName)
		}
	}

	return processClusters
}

// ListClusters 列出所有集群
func (c *S3ConfigCollector) ListClusters(ctx context.Context) ([]string, error) {
	// 1. 从数据库获取集群列表
	dbClusters, err := c.getDBClusters(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 获取今天的日期字符串
	todayStr := getFormattedToday()

	// 3. 获取今天有数据的集群
	todayClusters, _ := c.getTodayClusters(ctx, todayStr)
	// 即使出错也继续，因为可能今天的数据不存在

	// 4. 获取所有日期的集群
	s3Clusters, err := c.getAllS3Clusters(ctx)
	if err != nil {
		return nil, err
	}

	// 5. 初始化集群状态并确定要处理的集群
	return c.initClusterStatus(dbClusters, s3Clusters, todayClusters), nil
}

// filterEmptyStrings 过滤掉空字符串
func filterEmptyStrings(strs []string) []string {
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// ProcessCluster 处理单个集群的配置检查
func (c *S3ConfigCollector) ProcessCluster(ctx context.Context, clusterName string) error {
	nodeTypes := []string{NodeTypeMaster, NodeTypeEtcd, NodeTypeWorker}

	for _, nodeType := range nodeTypes {
		// 处理k8s配置
		if err := c.processNodeType(ctx, clusterName, nodeType, CheckTypeK8s); err != nil {
			return fmt.Errorf("failed to process %s k8s config: %w", nodeType, err)
		}

		// 对于非etcd节点，处理runtime配置
		if nodeType != NodeTypeEtcd {
			if err := c.processNodeType(ctx, clusterName, nodeType, CheckTypeRuntime); err != nil {
				return fmt.Errorf("failed to process %s runtime config: %w", nodeType, err)
			}
		}
	}

	return nil
}

// listNodeConfigs 列出节点配置文件
func (c *S3ConfigCollector) listNodeConfigs(ctx context.Context, prefix string) ([]string, []string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list objects: %w", err)
	}

	objectKeys := make([]string, 0, len(result.Contents))
	nodeNames := make([]string, 0, len(result.Contents))

	for _, obj := range result.Contents {
		objectKeys = append(objectKeys, *obj.Key)
		nodeName := strings.TrimPrefix(*obj.Key, prefix)
		nodeName = strings.TrimSuffix(nodeName, ".txt")
		nodeNames = append(nodeNames, nodeName)
	}

	return objectKeys, nodeNames, nil
}

// processNodeType 处理特定节点类型的配置
func (c *S3ConfigCollector) processNodeType(ctx context.Context, clusterName, nodeType, checkType string) error {
	// 获取当天日期字符串
	todayStr := getFormattedToday()

	// 更新前缀格式，包含日期
	prefix := fmt.Sprintf("%s%s/%s/%s/%s/", S3PathPrefix, todayStr, clusterName, nodeType, checkType)

	objectKeys, nodeNames, err := c.listNodeConfigs(ctx, prefix)
	if err != nil {
		return err
	}

	for i, objectKey := range objectKeys {
		if err := c.processNodeConfig(ctx, clusterName, nodeType, nodeNames[i], checkType, objectKey); err != nil {
			return fmt.Errorf("failed to process node config: %w", err)
		}
	}

	return nil
}

// parseConfigLine 解析配置行
func (c *S3ConfigCollector) parseConfigLine(line string) (name, value string, status bool, fixSuggestion string) {
	// 定义正则表达式匹配模式
	checkRegex := regexp.MustCompile(`^([^:]+):\s*(?:\(([^)]+)\)|([^,]+))\s*,\s*(True|False)(?:[.,]\s*(.*))?$`)

	// 尝试正则表达式匹配
	if matches := checkRegex.FindStringSubmatch(line); len(matches) >= 5 {
		name = strings.TrimSpace(matches[1])

		// 提取值（可能在括号中或不在括号中）
		if matches[2] != "" {
			value = strings.TrimSpace(matches[2])
		} else if matches[3] != "" {
			value = strings.TrimSpace(matches[3])
		}

		// 提取状态
		status = strings.EqualFold(matches[4], "true")

		// 提取修复建议（如果有）
		if len(matches) >= 6 && matches[5] != "" {
			fixSuggestion = strings.TrimSpace(matches[5])
		}

		return
	}

	// 如果正则表达式匹配失败，尝试基本的分割方法
	parts := strings.Split(line, ":")
	if len(parts) >= 2 {
		name = strings.TrimSpace(parts[0])
		valuePart := strings.TrimSpace(parts[1])

		// 检查是否有逗号分隔的状态和修复建议
		valueParts := strings.Split(valuePart, ",")
		if len(valueParts) >= 2 {
			value = strings.TrimSpace(valueParts[0])
			status = strings.Contains(strings.ToLower(valueParts[1]), "true")

			if len(valueParts) >= 3 {
				fixSuggestion = strings.TrimSpace(valueParts[2])
			}
		} else {
			// 如果没有逗号，就只有值
			value = valuePart
			status = strings.Contains(strings.ToLower(value), "true")
		}
	}

	return
}

// scanConfigFile 扫描配置文件并提取检查项
func (c *S3ConfigCollector) scanConfigFile(reader *bufio.Scanner) ([]ConfigCheck, bool) {
	var checks []ConfigCheck
	hasFailures := false

	for reader.Scan() {
		line := reader.Text()
		name, value, status, fixSuggestion := c.parseConfigLine(line)

		// 检查是否有失败项
		if !status {
			hasFailures = true
		}

		// 只有当解析到有效的名称和值时才添加检查项
		if name != "" && value != "" {
			checks = append(checks, ConfigCheck{
				Name:          name,
				Value:         value,
				Status:        status,
				FixSuggestion: fixSuggestion,
			})
		}
	}

	return checks, hasFailures
}

// updateClusterStatus 更新集群状态
func (c *S3ConfigCollector) updateClusterStatus(clusterName, nodeType, nodeName string, hasFailures bool) {
	if !hasFailures {
		return
	}

	clusterStatus := c.clusterStatus[clusterName]
	if clusterStatus == nil {
		return
	}

	clusterStatus.HasFailures = true

	// 生成节点唯一标识
	nodeKey := fmt.Sprintf("%s/%s", nodeType, nodeName)

	// 检查节点是否已经在列表中
	for _, node := range clusterStatus.FailureNodes {
		if node == nodeKey {
			return
		}
	}

	// 如果节点不在列表中，添加到列表
	clusterStatus.FailureNodes = append(clusterStatus.FailureNodes, nodeKey)
}

// processNodeConfig 处理节点配置文件
func (c *S3ConfigCollector) processNodeConfig(ctx context.Context, clusterName, nodeType, nodeName, checkType, objectKey string) error {
	// 获取S3对象
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer output.Body.Close()

	// 扫描配置文件并提取检查项
	scanner := bufio.NewScanner(output.Body)
	checks, hasFailures := c.scanConfigFile(scanner)

	// 更新集群状态
	c.updateClusterStatus(clusterName, nodeType, nodeName, hasFailures)

	// 保存检查结果到数据库
	return c.saveChecks(ctx, clusterName, nodeType, nodeName, checkType, checks)
}

// saveChecks 将检查结果保存到数据库（Upsert逻辑）
func (c *S3ConfigCollector) saveChecks(ctx context.Context, clusterName, nodeType, nodeName, checkType string, checks []ConfigCheck) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		var existingCheck portal.SecurityCheck
		todayStart := time.Now().Truncate(24 * time.Hour)
		todayEnd := todayStart.Add(24 * time.Hour)

		// 查询当天是否已存在记录
		err := tx.WithContext(ctx).Where("cluster_name = ? AND node_type = ? AND node_name = ? AND check_type = ? AND created_at >= ? AND created_at < ?",
			clusterName, nodeType, nodeName, checkType, todayStart, todayEnd).First(&existingCheck).Error

		var securityCheckID uint

		if err == nil {
			// 记录已存在，执行更新逻辑
			securityCheckID = uint(existingCheck.ID)

			// 1. 删除旧的检查项
			if err := tx.WithContext(ctx).Where("security_check_id = ?", securityCheckID).Delete(&portal.SecurityCheckItem{}).Error; err != nil {
				return fmt.Errorf("failed to delete old check items: %w", err)
			}

			// 2. 更新主记录的时间戳 (可选，根据业务需求决定是否需要更新时间)
			// existingCheck.UpdatedAt = time.Now() // GORM 会自动更新 UpdatedAt
			// if err := tx.Save(&existingCheck).Error; err != nil {
			// 	return fmt.Errorf("failed to update security check timestamp: %w", err)
			// }

		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在，执行创建逻辑
			securityCheck := &portal.SecurityCheck{
				ClusterName: clusterName,
				NodeType:    nodeType,
				NodeName:    nodeName,
				CheckType:   checkType,
				// CreatedAt 和 UpdatedAt 由 GORM 自动处理
			}

			if err := tx.WithContext(ctx).Create(securityCheck).Error; err != nil {
				return fmt.Errorf("failed to create security check: %w", err)
			}
			securityCheckID = uint(securityCheck.ID)
		} else {
			// 查询时发生其他错误
			return fmt.Errorf("failed to query existing security check: %w", err)
		}

		// 插入新的检查项记录
		for _, check := range checks {
			checkItem := &portal.SecurityCheckItem{
				SecurityCheckID: int64(securityCheckID),
				ItemName:        check.Name,
				ItemValue:       check.Value,
				Status:          check.Status,
				FixSuggestion:   check.FixSuggestion,
			}

			if err := tx.WithContext(ctx).Create(checkItem).Error; err != nil {
				// 注意：如果这里失败，事务会回滚，之前的删除/创建操作也会被撤销
				return fmt.Errorf("failed to create check item: %w", err)
			}
		}

		return nil
	})
}

// Run 运行配置采集任务
func (c *S3ConfigCollector) Run(ctx context.Context) (map[string]*ClusterStatus, error) {
	clusters, err := c.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusters {
		if err := c.ProcessCluster(ctx, cluster); err != nil {
			// 处理单个集群失败不应该导致整个任务失败
			// 记录错误并继续处理其他集群
			c.logger.Error("Failed to process cluster", zap.String("cluster", cluster), zap.Error(err))

			// 标记集群处理失败
			if status, ok := c.clusterStatus[cluster]; ok {
				status.HasFailures = true
			}
		}
	}

	return c.clusterStatus, nil
}
