package security_check

import (
	"bufio"
	"context"
	"fmt"
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
}

// NewS3ConfigCollector 创建新的采集器实例
func NewS3ConfigCollector(s3Client *s3.Client, db *gorm.DB, bucket string) *S3ConfigCollector {
	return &S3ConfigCollector{
		s3Client: s3Client,
		db:       db,
		bucket:   bucket,
	}
}

// ListClusters 列出所有集群
func (c *S3ConfigCollector) ListClusters(ctx context.Context) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String("cluster/"),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusters := make([]string, 0)
	for _, prefix := range result.CommonPrefixes {
		clusterName := strings.TrimPrefix(*prefix.Prefix, "cluster/")
		clusterName = strings.TrimSuffix(clusterName, "/")
		clusters = append(clusters, clusterName)
	}

	return clusters, nil
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

	var checks []ConfigCheck
	scanner := bufio.NewScanner(output.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			status := strings.Contains(strings.ToLower(value), "true")

			checks = append(checks, ConfigCheck{
				Name:   name,
				Value:  value,
				Status: status,
			})
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
			}

			if err := tx.Create(checkItem).Error; err != nil {
				return fmt.Errorf("failed to create check item: %w", err)
			}
		}

		return nil
	})
}

// Run 运行配置采集任务
func (c *S3ConfigCollector) Run(ctx context.Context) error {
	clusters, err := c.ListClusters(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusters {
		if err := c.ProcessCluster(ctx, cluster); err != nil {
			return fmt.Errorf("failed to process cluster %s: %w", cluster, err)
		}
	}

	return nil
}
