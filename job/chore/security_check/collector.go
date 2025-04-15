package security_check

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gorm.io/gorm"

	"navy-ng/models/portal"
)

// S3ConfigCollector S3配置采集器
type S3ConfigCollector struct {
	s3Client *s3.Client
	db       *gorm.DB
	bucket   string
	// 集群状态跟踪
	clusterStatus map[string]*ClusterStatus
}

// ClusterStatus 集群状态
type ClusterStatus struct {
	Exists       bool     // 集群在S3中是否存在
	HasFailures  bool     // 集群是否有失败项
	FailureNodes []string // 有失败项的节点列表
}

// NewS3ConfigCollector 创建新的采集器实例
func NewS3ConfigCollector(s3Client *s3.Client, db *gorm.DB, bucket string) *S3ConfigCollector {
	return &S3ConfigCollector{
		s3Client:      s3Client,
		db:            db,
		bucket:        bucket,
		clusterStatus: make(map[string]*ClusterStatus),
	}
}

// ListClusters 列出所有集群
func (c *S3ConfigCollector) ListClusters(ctx context.Context) ([]string, error) {
	// 1. 从数据库获取集群列表
	var dbClusters []portal.K8sCluster
	if err := c.db.WithContext(ctx).Where("deleted = ''").Find(&dbClusters).Error; err != nil {
		return nil, fmt.Errorf("failed to get clusters from database: %w", err)
	}

	// 2. 从 S3 获取集群目录
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String("cluster/"),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters from S3: %w", err)
	}

	// 3. 将 S3 目录转换为集群名称集合
	s3Clusters := make(map[string]bool)
	for _, prefix := range result.CommonPrefixes {
		clusterName := strings.TrimPrefix(*prefix.Prefix, "cluster/")
		clusterName = strings.TrimSuffix(clusterName, "/")
		s3Clusters[clusterName] = true
	}

	// 4. 初始化所有数据库集群的状态
	processClusters := make([]string, 0, len(dbClusters))
	for _, cluster := range dbClusters {
		// 初始化集群状态
		c.clusterStatus[cluster.Name] = &ClusterStatus{
			Exists:       s3Clusters[cluster.Name],
			HasFailures:  false,
			FailureNodes: make([]string, 0),
		}

		// 如果集群在 S3 中存在，添加到处理列表
		if s3Clusters[cluster.Name] {
			processClusters = append(processClusters, cluster.Name)
		}
	}

	// 返回需要处理的集群列表
	return processClusters, nil
}

// ProcessCluster 处理单个集群的配置检查
func (c *S3ConfigCollector) ProcessCluster(ctx context.Context, clusterName string) error {
	nodeTypes := []string{"master", "etcd", "node"}

	for _, nodeType := range nodeTypes {
		// 处理k8s配置
		if err := c.processNodeType(ctx, clusterName, nodeType, "k8s"); err != nil {
			return fmt.Errorf("failed to process %s k8s config: %w", nodeType, err)
		}

		// 对于非etcd节点，处理runtime配置
		if nodeType != "etcd" {
			if err := c.processNodeType(ctx, clusterName, nodeType, "runtime"); err != nil {
				return fmt.Errorf("failed to process %s runtime config: %w", nodeType, err)
			}
		}
	}

	return nil
}

// processNodeType 处理特定节点类型的配置
func (c *S3ConfigCollector) processNodeType(ctx context.Context, clusterName, nodeType, checkType string) error {
	prefix := fmt.Sprintf("cluster/%s/%s/%s/", clusterName, nodeType, checkType)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	for _, obj := range result.Contents {
		nodeName := strings.TrimPrefix(*obj.Key, prefix)
		nodeName = strings.TrimSuffix(nodeName, ".txt")

		if err := c.processNodeConfig(ctx, clusterName, nodeType, nodeName, checkType, *obj.Key); err != nil {
			return fmt.Errorf("failed to process node config: %w", err)
		}
	}

	return nil
}

// processNodeConfig 处理节点配置文件
func (c *S3ConfigCollector) processNodeConfig(ctx context.Context, clusterName, nodeType, nodeName, checkType, objectKey string) error {
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer output.Body.Close()

	// 定义一个通用的正则表达式来匹配大多数格式
	// 这个正则表达式可以匹配以下格式：
	// 1. kube.conf文件权限：（ 644 ）,True
	// 2. kube.conf文件权限：（ 777 ）,False.应该修复为644
	// 3. kube.conf文件权限：777,False,建议修改为644
	// 4. 其他类似格式
	checkRegex := regexp.MustCompile(`^([^:]+):\s*(?:\(([^)]+)\)|([^,]+))\s*,\s*(True|False)(?:[.,]\s*(.*))?$`)

	var checks []ConfigCheck
	scanner := bufio.NewScanner(output.Body)
	hasFailures := false

	for scanner.Scan() {
		line := scanner.Text()
		var name, value string
		var status bool
		var fixSuggestion string
		var parsed bool

		// 使用正则表达式解析行
		matches := checkRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			// 提取名称
			name = strings.TrimSpace(matches[1])

			// 提取值（可能在括号中或不在括号中）
			if matches[2] != "" {
				// 括号中的值
				value = strings.TrimSpace(matches[2])
			} else if matches[3] != "" {
				// 非括号的值
				value = strings.TrimSpace(matches[3])
			}

			// 提取状态
			status = strings.EqualFold(matches[4], "true")

			// 提取修复建议（如果有）
			if len(matches) >= 6 && matches[5] != "" {
				fixSuggestion = strings.TrimSpace(matches[5])
			}

			parsed = true
		}

		// 如果正则表达式匹配失败，尝试基本的分割方法
		if !parsed {
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
		}

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

	// 更新集群状态
	if hasFailures {
		clusterStatus := c.clusterStatus[clusterName]
		if clusterStatus != nil {
			clusterStatus.HasFailures = true

			// 生成节点唯一标识
			nodeKey := fmt.Sprintf("%s/%s", nodeType, nodeName)

			// 检查节点是否已经在列表中
			found := false
			for _, node := range clusterStatus.FailureNodes {
				if node == nodeKey {
					found = true
					break
				}
			}

			// 如果节点不在列表中，添加到列表
			if !found {
				clusterStatus.FailureNodes = append(clusterStatus.FailureNodes, nodeKey)
			}
		}
	}

	return c.saveChecks(ctx, clusterName, nodeType, nodeName, checkType, checks)
}

// saveChecks 将检查结果保存到数据库
func (c *S3ConfigCollector) saveChecks(ctx context.Context, clusterName, nodeType, nodeName, checkType string, checks []ConfigCheck) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		// 创建主记录
		securityCheck := &portal.SecurityCheck{
			ClusterName: clusterName,
			NodeType:    nodeType,
			NodeName:    nodeName,
			CheckType:   checkType,
		}

		if err := tx.Create(securityCheck).Error; err != nil {
			return fmt.Errorf("failed to create security check: %w", err)
		}

		// 创建检查项记录
		for _, check := range checks {
			checkItem := &portal.SecurityCheckItem{
				SecurityCheckID: securityCheck.ID,
				ItemName:        check.Name,
				ItemValue:       check.Value,
				Status:          check.Status,
				FixSuggestion:   check.FixSuggestion,
			}

			if err := tx.Create(checkItem).Error; err != nil {
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
			fmt.Printf("[ERROR] Failed to process cluster %s: %v\n", cluster, err)

			// 标记集群处理失败
			if status, ok := c.clusterStatus[cluster]; ok {
				status.HasFailures = true
			}
		}
	}

	return c.clusterStatus, nil
}
