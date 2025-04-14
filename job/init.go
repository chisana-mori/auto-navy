package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
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
func initS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}
