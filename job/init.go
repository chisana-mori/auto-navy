package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// initDB 初始化数据库连接
func initDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/navy?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// initS3Client 初始化S3客户端
func initS3Client(accessKey, secretKey, endPoint string) (*s3.Client, error) {
	// 创建静态凭证
	creds := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		"", // 会话token，通常对于静态凭证为空
	)

	// 加载基础配置（区域、凭证等）
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"), // 设置默认区域，或者根据需要调整
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 创建S3客户端，并在选项中指定自定义 Endpoint 和 PathStyle
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endPoint) // 直接设置 BaseEndpoint
		o.UsePathStyle = true                 // 对非 AWS S3 兼容存储通常需要
	})

	return client, nil
}
