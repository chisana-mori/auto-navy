package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/chisana-mori/auto-navy/models/portal"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 配置信息
const (
	dbUser     = "root"
	dbPassword = "password"
	dbHost     = "localhost"
	dbPort     = "3306"
	dbName     = "navy"
	mockCount  = 100 // 生成的模拟数据数量
)

// 模拟数据选项
var (
	archTypes      = []string{"x86_64", "aarch64", "mips64", "loongarch64", "riscv64"}
	idcs           = []string{"BJ", "SH", "GZ", "SZ", "CD", "HZ"}
	rooms          = []string{"A01", "B02", "C03", "D04", "E05"}
	infraTypes     = []string{"公有云", "私有云", "混合云", "物理机", "虚拟机"}
	netZones       = []string{"生产网", "测试网", "开发网", "管理网", "DMZ"}
	groups         = []string{"计算节点", "存储节点", "网络节点", "控制节点", "边缘节点"}
	osTypes        = []string{"CentOS", "Ubuntu", "Debian", "OpenEuler", "UOS", "Kylin"}
	osVersions     = []string{"7.9", "8.0", "20.04", "22.04", "10", "11", "V10"}
	kernels        = []string{"3.10.0", "4.18.0", "5.4.0", "5.10.0", "6.1.0"}
	companies      = []string{"华为", "阿里云", "腾讯云", "浪潮", "联想", "戴尔", "惠普"}
	models         = []string{"TaiShan 200", "FusionServer", "ThinkSystem", "PowerEdge", "ProLiant"}
	statuses       = []string{"运行中", "已停止", "维护中", "故障", "待分配"}
	roles          = []string{"master", "worker", "etcd", "ingress", "storage", "monitor", "gateway"}
	clusters       = []string{"prod-cluster", "test-cluster", "dev-cluster", "staging-cluster", "edge-cluster"}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 生成模拟数据
	devices := generateMockDevices(mockCount)

	// 保存到数据库
	result := db.Create(&devices)
	if result.Error != nil {
		fmt.Printf("保存数据失败: %v\n", result.Error)
		os.Exit(1)
	}

	fmt.Printf("成功生成 %d 条设备模拟数据\n", mockCount)
}

// 生成模拟设备数据
func generateMockDevices(count int) []portal.Device {
	devices := make([]portal.Device, count)

	for i := 0; i < count; i++ {
		// 生成随机IP
		ip := fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255))
		
		// 生成随机CI编码
		ciCode := fmt.Sprintf("DEV-%06d", i+1)
		
		// 随机选择IDC和机房
		idc := idcs[rand.Intn(len(idcs))]
		room := rooms[rand.Intn(len(rooms))]
		
		// 生成机柜信息
		cabinet := fmt.Sprintf("%s-%s-CAB%02d", idc, room, rand.Intn(50)+1)
		cabinetNo := fmt.Sprintf("%02d", rand.Intn(42)+1)
		
		// 随机选择架构和操作系统
		archType := archTypes[rand.Intn(len(archTypes))]
		osType := osTypes[rand.Intn(len(osTypes))]
		osVersion := osVersions[rand.Intn(len(osVersions))]
		osName := osType
		osIssue := osVersion
		kernel := kernels[rand.Intn(len(kernels))]
		
		// 随机选择集群和角色
		cluster := clusters[rand.Intn(len(clusters))]
		role := roles[rand.Intn(len(roles))]
		
		// 生成创建时间
		createdAt := time.Now().AddDate(0, -rand.Intn(12), -rand.Intn(30))
		osCreateTime := createdAt.Format("2006-01-02")
		
		// 随机生成CPU和内存
		cpu := float64(rand.Intn(32) + 4)
		memory := float64(rand.Intn(128) + 16)
		
		// 随机选择其他属性
		infraType := infraTypes[rand.Intn(len(infraTypes))]
		netZone := netZones[rand.Intn(len(netZones))]
		group := groups[rand.Intn(len(groups))]
		company := companies[rand.Intn(len(companies))]
		model := models[rand.Intn(len(models))]
		status := statuses[rand.Intn(len(statuses))]
		
		// 随机生成是否国产化
		isLocalization := rand.Intn(2) == 1
		
		// 生成KVM IP (与主IP相似但最后一段不同)
		kvmIP := fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255))
		
		// 生成AppID
		appID := fmt.Sprintf("APP%05d", rand.Intn(10000))
		
		// 生成集群ID
		clusterID := rand.Intn(10) + 1
		
		// 创建设备对象
		devices[i] = portal.Device{
			BaseModel: portal.BaseModel{
				CreatedAt: portal.JSONTime(createdAt),
				UpdatedAt: portal.JSONTime(time.Now()),
			},
			CICode:        ciCode,
			IP:            ip,
			ArchType:      archType,
			IDC:           idc,
			Room:          room,
			Cabinet:       cabinet,
			CabinetNO:     cabinetNo,
			InfraType:     infraType,
			IsLocalization: isLocalization,
			NetZone:       netZone,
			Group:         group,
			AppID:         appID,
			OsCreateTime:  osCreateTime,
			CPU:           cpu,
			Memory:        memory,
			Model:         model,
			KvmIP:         kvmIP,
			OS:            fmt.Sprintf("%s %s", osType, osVersion),
			Company:       company,
			OSName:        osName,
			OSIssue:       osIssue,
			OSKernel:      fmt.Sprintf("%s-%s", kernel, strings.ToLower(osType)),
			Status:        status,
			Role:          role,
			Cluster:       cluster,
			ClusterID:     clusterID,
			Deleted:       "",
		}
	}

	return devices
}
