package service

import (
	"bytes"
	"context"
	"fmt"
	"navy-ng/models/portal"
	"navy-ng/pkg/redis" // Import the redis package
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Constants for DeviceService
const (
	ErrDeviceNotFoundMsg = "device with ID %d not found"
	// DeviceUpdatesChannel 已移至 pkg/redis/keys.go
)

// DeviceService 设备服务
type DeviceService struct {
	db                 *gorm.DB
	cache              *DeviceCache
	deviceQueryService *DeviceQueryService // 添加 DeviceQueryService 依赖
	redisHandler       *redis.Handler      // 添加 Redis Handler 依赖
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB, cache *DeviceCache, deviceQueryService *DeviceQueryService, redisHandler *redis.Handler) *DeviceService {
	return &DeviceService{
		db:                 db,
		cache:              cache,
		deviceQueryService: deviceQueryService, // 注入 DeviceQueryService
		redisHandler:       redisHandler,       // 注入 Redis Handler
	}
}

// buildDeviceBaseQuery 构建设备查询的基础查询，包含与 k8s_node 等表的关联
func (s *DeviceService) buildDeviceBaseQuery(ctx context.Context) *gorm.DB {
	// 使用注入的 DeviceQueryService 实例
	query := s.deviceQueryService.buildDeviceQuery(ctx)
	return query
}

// processMultilineKeyword 处理多行查询关键字
func processMultilineKeyword(keyword string) []string {
	// 检查是否包含常见的分隔符
	hasNewline := strings.Contains(keyword, "\n")
	hasComma := strings.Contains(keyword, ",")
	hasOr := strings.Contains(strings.ToLower(keyword), " or ")

	// 如果包含分隔符，则分割关键字
	if hasNewline || hasComma || hasOr {
		var lines []string

		// 首先按换行符分割
		if hasNewline {
			lines = strings.Split(keyword, "\n")
		} else if hasComma {
			// 如果没有换行符但有逗号，则按逗号分割
			lines = strings.Split(keyword, ",")
		} else if hasOr {
			// 如果没有换行符和逗号，但有 OR，则按 OR 分割
			lines = strings.Split(strings.ToLower(keyword), " or ")
		}

		// 过滤空行并去除空格
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, line)
			}
		}

		if len(result) > 0 {
			return result
		}
	}

	// 如果没有有效的多行关键字，返回原始关键字作为单个元素的切片
	return []string{keyword}
}

// ListDevices 获取设备列表
func (s *DeviceService) ListDevices(ctx context.Context, query *DeviceQuery) (*DeviceListResponse, error) {
	// 生成查询参数的哈希值
	queryHash := GenerateQueryHash(query)

	// 尝试从缓存获取
	if s.cache != nil {
		if cachedResponse, err := s.cache.GetDeviceList(queryHash); err == nil {
			return cachedResponse, nil
		}
	}

	// 缓存未命中，从数据库查询
	var models []portal.Device
	var total int64

	// 使用基础查询，包含与 k8s_node 等表的关联
	db := s.buildDeviceBaseQuery(ctx)

	// 应用关键字搜索
	if query.Keyword != "" {
		// 处理多行查询
		keywords := processMultilineKeyword(query.Keyword)

		// 检测是否是IP地址列表查询
		isIPList := true
		for _, keyword := range keywords {
			// 简单检查是否是IP地址格式（可以根据需要使用更严格的正则表达式）
			if !strings.Contains(keyword, ".") || strings.Contains(keyword, " ") {
				isIPList = false
				break
			}
		}

		// 如果是IP地址列表查询，则只在IP字段中查询
		if isIPList && len(keywords) > 1 {
			// 构建IP查询条件
			for i, ip := range keywords {
				ip = "%" + ip + "%"

				// 第一个IP使用 Where，后续IP使用 Or
				if i == 0 {
					db = db.Where("device.ip LIKE ?", ip)
				} else {
					db = db.Or("device.ip LIKE ?", ip)
				}
			}
		} else {
			// 常规关键字查询 - 动态构建
			// 定义可用于关键字搜索的字段 (使用 camelCase)
			searchableFields := []string{
				"ciCode", "ip", "archType", "cluster", "role", "idc", "room",
				"cabinet", "cabinetNo", "infraType", "netZone", "appId", "group",
				// 可以根据需要添加更多字段，例如 "osName", "model" 等
			}

			// 构建 WHERE 子句
			var conditions []string
			// var args []interface{} // Removed unused variable
			for _, fieldKey := range searchableFields {
				dbColumn, found := s.deviceQueryService.GetDbColumnForField(fieldKey)
				if found {
					conditions = append(conditions, fmt.Sprintf("%s LIKE ?", dbColumn))
				} else {
					// 如果字段定义未找到，记录警告（可选）
					fmt.Printf("Warning: Searchable field key '%s' not found in definitions.\n", fieldKey)
				}
			}

			if len(conditions) > 0 {
				// 对每个关键字应用 OR 条件
				for i, keyword := range keywords {
					likeKeyword := "%" + keyword + "%"
					keywordArgs := make([]interface{}, len(conditions))
					for j := range conditions {
						keywordArgs[j] = likeKeyword
					}

					// 构建当前关键字的 OR 查询组
					keywordCondition := strings.Join(conditions, " OR ")

					if i == 0 {
						// 第一个关键字使用 Where
						db = db.Where(keywordCondition, keywordArgs...)
					} else {
						// 后续关键字使用 Or
						db = db.Or(keywordCondition, keywordArgs...)
					}
				}
			} else {
				// 如果没有可搜索的字段，可以记录日志或返回错误，或者简单地不应用关键字过滤
				fmt.Println("Warning: No searchable fields found to apply keyword filter.")
			}
		}
	}

	// 如果只显示特殊设备
	if query.OnlySpecial {
		// 使用与 device_query.go 中相同的条件定义
		db = db.Where(SpecialDeviceCondition)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count devices: %w", err)
	}

	// 分页
	page := query.Page
	if page <= 0 {
		page = 1
	}

	size := query.Size
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	// 转换为响应格式
	responses := make([]DeviceResponse, len(models))
	for i, model := range models {
		responses[i] = DeviceResponse{
			ID:             model.ID,
			CICode:         model.CICode,
			IP:             model.IP,
			ArchType:       model.ArchType,
			IDC:            model.IDC,
			Room:           model.Room,
			Cabinet:        model.Cabinet,
			CabinetNO:      model.CabinetNO,
			InfraType:      model.InfraType,
			IsLocalization: model.IsLocalization,
			NetZone:        model.NetZone,
			Group:          model.Group,
			AppID:          model.AppID,
			AppName:        model.AppName,
			OsCreateTime:   model.OsCreateTime,
			CPU:            model.CPU,
			Memory:         model.Memory,
			Model:          model.Model,
			KvmIP:          model.KvmIP,
			OS:             model.OS,
			Company:        model.Company,
			OSName:         model.OSName,
			OSIssue:        model.OSIssue,
			OSKernel:       model.OSKernel,
			Status:         model.Status,
			Role:           model.Role,
			Cluster:        model.Cluster,
			ClusterID:      model.ClusterID,
			AcceptanceTime: model.AcceptanceTime,
			DiskCount:      model.DiskCount,
			DiskDetail:     model.DiskDetail,
			NetworkSpeed:   model.NetworkSpeed,
			IsSpecial:      model.IsSpecial,
			FeatureCount:   model.FeatureCount,
			CreatedAt:      time.Time(model.CreatedAt),
			UpdatedAt:      time.Time(model.UpdatedAt),
		}
	}

	// 构建响应
	response := &DeviceListResponse{
		List:  responses,
		Total: total,
		Page:  page,
		Size:  size,
	}

	// 缓存查询结果
	if s.cache != nil {
		s.cache.SetDeviceList(queryHash, response)

		// 同时缓存单个设备
		for _, deviceResp := range responses {
			s.cache.SetDevice(deviceResp.ID, &deviceResp)
		}
	}

	return response, nil
}

// GetDevice 获取设备详情
// 使用基础查询，包含与 k8s_node 等表的关联，以获取 AppName 字段
func (s *DeviceService) GetDevice(ctx context.Context, id int64) (*DeviceResponse, error) {
	// 尝试从缓存获取
	if s.cache != nil {
		if cachedDevice, err := s.cache.GetDevice(id); err == nil {
			return cachedDevice, nil
		}
	}

	// 缓存未命中，从数据库查询
	var model portal.Device
	// 使用基础查询，包含与 k8s_node 等表的关联
	db := s.buildDeviceBaseQuery(ctx).Where("device.id = ?", id)
	err := db.First(&model).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// 构建响应
	response := &DeviceResponse{
		ID:             model.ID,
		CICode:         model.CICode,
		IP:             model.IP,
		ArchType:       model.ArchType,
		IDC:            model.IDC,
		Room:           model.Room,
		Cabinet:        model.Cabinet,
		CabinetNO:      model.CabinetNO,
		InfraType:      model.InfraType,
		IsLocalization: model.IsLocalization,
		NetZone:        model.NetZone,
		Group:          model.Group,
		AppID:          model.AppID,
		AppName:        model.AppName,
		OsCreateTime:   model.OsCreateTime,
		CPU:            model.CPU,
		Memory:         model.Memory,
		Model:          model.Model,
		KvmIP:          model.KvmIP,
		OS:             model.OS,
		Company:        model.Company,
		OSName:         model.OSName,
		OSIssue:        model.OSIssue,
		OSKernel:       model.OSKernel,
		Status:         model.Status,
		Role:           model.Role,
		Cluster:        model.Cluster,
		ClusterID:      model.ClusterID,
		AcceptanceTime: model.AcceptanceTime,
		DiskCount:      model.DiskCount,
		DiskDetail:     model.DiskDetail,
		NetworkSpeed:   model.NetworkSpeed,
		// 保留特性标记，便于前端显示
		IsSpecial:    model.IsSpecial,
		FeatureCount: model.FeatureCount,
		CreatedAt:    time.Time(model.CreatedAt),
		UpdatedAt:    time.Time(model.UpdatedAt),
	}

	// 缓存结果
	if s.cache != nil {
		s.cache.SetDevice(id, response)
	}

	return response, nil
}

// UpdateDeviceRole 更新设备角色

func (s *DeviceService) UpdateDeviceRole(ctx context.Context, id int64, request *DeviceRoleUpdateRequest) error {
	// 查找设备
	var device portal.Device
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return fmt.Errorf("failed to find device: %w", result.Error)
	}

	// 更新角色字段
	result = s.db.WithContext(ctx).Model(&device).Update("role", request.Role)
	if result.Error != nil {
		return fmt.Errorf("failed to update device role: %w", result.Error)
	}

	// GORM Hook (AfterSave) 会自动调用 publishDeviceChangeEvent
	// s.publishDeviceUpdate(id) // 不再需要手动调用

	return nil
}

// UpdateDeviceGroup 更新设备用途
func (s *DeviceService) UpdateDeviceGroup(ctx context.Context, id int64, request *DeviceGroupUpdateRequest) error {
	// 查找设备
	var device portal.Device
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&device)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf(ErrDeviceNotFoundMsg, id)
		}
		return fmt.Errorf("failed to find device: %w", result.Error)
	}

	// 更新用途字段
	result = s.db.WithContext(ctx).Model(&device).Update("`group`", request.Group)
	if result.Error != nil {
		return fmt.Errorf("failed to update device group: %w", result.Error)
	}

	// GORM Hook (AfterSave) 会自动调用 publishDeviceChangeEvent
	// s.publishDeviceUpdate(id) // 不再需要手动调用

	return nil
}

// publishDeviceChangeEvent 是实际的事件发布函数，符合 models.PublishDeviceChangeEventFunc 签名
// 它将被注册到 models 包中，由 GORM Hooks 调用
func publishDeviceChangeEvent(deviceID int64) {
	// 注意：这个函数现在是包级别的，不再是 DeviceService 的方法
	// 因此，它需要一种方式来访问 Redis Handler。
	// 方案1: 使用全局变量（如果 Redis Handler 是全局单例）
	// 方案2: 在注册时传递 Redis Handler (更推荐，但需要修改注册机制)
	// 方案3: 重新获取 Redis Handler (如下，简单但不高效)

	// 暂时使用重新获取的方式，后续可以优化
	redisHandler := redis.NewRedisHandler("default")
	if redisHandler == nil {
		fmt.Printf("Error: Redis handler is not available, cannot publish change event for device ID %d\n", deviceID)
		return
	}

	// 将 int64 ID 转换为字符串
	message := strconv.FormatInt(deviceID, 10)

	// 发布消息到 Redis Pub/Sub
	err := redisHandler.Pub(redis.DeviceUpdatesChannel, message)
	if err != nil {
		fmt.Printf("Error publishing device change event for ID %d to channel %s: %v\n", deviceID, redis.DeviceUpdatesChannel, err)
	} else {
		fmt.Printf("Published device change event for ID %d to channel %s\n", deviceID, redis.DeviceUpdatesChannel)
	}
}

// ExportDevices 导出所有设备信息为Excel

func (s *DeviceService) ExportDevices(ctx context.Context) ([]byte, error) {
	var devices []portal.Device

	// 使用基础查询，包含与 k8s_node 等表的关联
	db := s.buildDeviceBaseQuery(ctx)

	// 获取所有设备信息
	if err := db.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to get devices for export: %w", err)
	}

	// 创建CSV格式的数据
	buffer := &bytes.Buffer{}

	// 写入表头
	headers := []string{
		"设备编码", "IP地址", "CPU架构", "IDC", "机房",
		"机柜", "机柜编号", "网络类型", "是否国产化", "网络区域",
		"机器类别", "APPID", "操作系统创建时间", "CPU", "内存",
		"型号", "KVM IP", "操作系统", "厂商", "操作系统名称",
		"操作系统版本", "操作系统内核", "状态", "角色", "集群",
		"集群ID", "创建时间", "更新时间",
	}

	// 写入CSV表头
	for i, header := range headers {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(header)
	}
	buffer.WriteString("\n")

	// 写入数据行
	for _, device := range devices {
		// 设备编码
		buffer.WriteString(device.CICode)
		buffer.WriteString(",")

		// IP地址
		buffer.WriteString(device.IP)
		buffer.WriteString(",")

		// CPU架构
		buffer.WriteString(device.ArchType)
		buffer.WriteString(",")

		// IDC
		buffer.WriteString(device.IDC)
		buffer.WriteString(",")

		// 机房
		buffer.WriteString(device.Room)
		buffer.WriteString(",")

		// 机柜
		buffer.WriteString(device.Cabinet)
		buffer.WriteString(",")

		// 机柜编号
		buffer.WriteString(device.CabinetNO)
		buffer.WriteString(",")

		// 网络类型
		buffer.WriteString(device.InfraType)
		buffer.WriteString(",")

		// 是否国产化
		if device.IsLocalization {
			buffer.WriteString("true")
		} else {
			buffer.WriteString("false")
		}
		buffer.WriteString(",")

		// 网络区域
		buffer.WriteString(device.NetZone)
		buffer.WriteString(",")

		// 机器类别
		buffer.WriteString(device.Group)
		buffer.WriteString(",")

		// APPID
		buffer.WriteString(device.AppID)
		buffer.WriteString(",")

		// 操作系统创建时间
		buffer.WriteString(device.OsCreateTime)
		buffer.WriteString(",")

		// CPU
		buffer.WriteString(fmt.Sprintf("%f", device.CPU))
		buffer.WriteString(",")

		// 内存
		buffer.WriteString(fmt.Sprintf("%f", device.Memory))
		buffer.WriteString(",")

		// 型号
		buffer.WriteString(device.Model)
		buffer.WriteString(",")

		// KVM IP
		buffer.WriteString(device.KvmIP)
		buffer.WriteString(",")

		// 操作系统
		buffer.WriteString(device.OS)
		buffer.WriteString(",")

		// 厂商
		buffer.WriteString(device.Company)
		buffer.WriteString(",")

		// 操作系统名称
		buffer.WriteString(device.OSName)
		buffer.WriteString(",")

		// 操作系统版本
		buffer.WriteString(device.OSIssue)
		buffer.WriteString(",")

		// 操作系统内核
		buffer.WriteString(device.OSKernel)
		buffer.WriteString(",")

		// 状态
		buffer.WriteString(device.Status)
		buffer.WriteString(",")

		// 角色
		buffer.WriteString(device.Role)
		buffer.WriteString(",")

		// 集群
		buffer.WriteString(device.Cluster)
		buffer.WriteString(",")

		// 集群ID
		buffer.WriteString(fmt.Sprintf("%d", device.ClusterID))
		buffer.WriteString(",")

		// 创建时间
		buffer.WriteString(time.Time(device.CreatedAt).Format("2006-01-02"))
		buffer.WriteString(",")

		// 更新时间
		buffer.WriteString(time.Time(device.UpdatedAt).Format("2006-01-02"))

		buffer.WriteString("\n")
	}

	return buffer.Bytes(), nil
}
