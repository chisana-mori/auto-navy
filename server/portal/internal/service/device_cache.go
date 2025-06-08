package service

import (
	"context"
	"encoding/json"
	"fmt"
	"navy-ng/pkg/redis"
	"time"
)

// 缓存过期时间常量
const (
	// 缓存版本，当缓存结构变化时需要更新
	CacheVersion = "v1"

	// 设备列表缓存过期时间：30-45分钟
	DeviceListExpiration = 45 * time.Minute

	// 单个设备缓存过期时间：2-3小时
	DeviceExpiration = 3 * time.Hour

	// 特殊设备标记缓存过期时间：50-60分钟
	DeviceSpecialExpiration = 60 * time.Minute

	// 设备字段值缓存过期时间（来自device表）：3-4小时
	DeviceFieldExpiration = 4 * time.Hour

	// 设备字段值缓存过期时间（来自标签/污点表）：50-60分钟
	LabelFieldExpiration = 60 * time.Minute
)

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits      int64 `json:"hits"`      // 缓存命中次数
	Misses    int64 `json:"misses"`    // 缓存未命中次数
	Sets      int64 `json:"sets"`      // 缓存设置次数
	Deletes   int64 `json:"deletes"`   // 缓存删除次数
	StartTime int64 `json:"startTime"` // 统计开始时间
}

// DeviceCache 设备缓存服务
type DeviceCache struct {
	handler    RedisHandlerInterface
	keyBuilder *redis.KeyBuilder
	stats      CacheStats
}

// NewDeviceCache 创建一个新的设备缓存服务
func NewDeviceCache(handler RedisHandlerInterface, keyBuilder *redis.KeyBuilder) *DeviceCache {
	return &DeviceCache{
		handler:    handler,
		keyBuilder: keyBuilder,
		stats: CacheStats{
			StartTime: time.Now().Unix(),
		},
	}
}

// SetDeviceList 缓存设备列表
func (c *DeviceCache) SetDeviceList(queryHash string, response *DeviceListResponse) error {
	key := c.keyBuilder.DeviceListKey(queryHash)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	c.handler.SetWithExpireTime(key, string(data), DeviceListExpiration)

	// 更新统计信息
	c.stats.Sets++

	return nil
}

// GetDeviceList 获取缓存的设备列表
func (c *DeviceCache) GetDeviceList(queryHash string) (*DeviceListResponse, error) {
	key := c.keyBuilder.DeviceListKey(queryHash)
	// Since handler is now an interface, we need to handle the return values of Get
	// Assuming the interface Get method returns (string, error)
	// This part needs to be adjusted based on the actual interface definition
	// For now, let's assume a simple Get that returns a string, and we need to check it.
	// The original code was: data := c.handler.Get(key)
	// This implies redis.Handler.Get returns a string. Let's assume our interface does too,
	// but we need to check the interface definition to be sure.
	// Let's assume the interface needs a Get method that returns (string, error) for safety.
	// But the original code `c.handler.Get(key)` suggests it returns a single string.
	// Let's stick to the original logic and assume the interface method is compatible.
	// The panic indicates the interface method was not found or had a different signature.
	// Let's assume the interface needs to be updated or the call needs to be.
	// The original redis.Handler.Get returns a string. The mock should do the same.
	// The issue is not Get, but SetWithExpireTime. Let's check that.
	// The original code uses `c.handler.SetWithExpireTime`. Let's assume this exists on the interface.

	// Re-reading the error: the panic is in `elastic_scaling_device_matching.go`, not here.
	// The change is correct. The original code was `data := c.handler.Get(key)`.
	// The `redis.Handler`'s `Get` method returns a string.
	// Our `RedisHandlerInterface` should also have a `Get(key string) string` method.
	// Let's check the interface definition in `elastic_scaling_service.go`.
	// It does not have Get or SetWithExpireTime. This is the root cause.

	// Let's add the required methods to the interface in `elastic_scaling_service.go` first.
	// But I cannot edit two files at once. I will proceed with the change here,
	// and then fix the interface definition.

	data := c.handler.Get(key)
	if data == "" {
		c.stats.Misses++
		return nil, fmt.Errorf("cache miss")
	}

	var response DeviceListResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, err
	}

	// 更新统计信息 - 缓存命中
	c.stats.Hits++

	return &response, nil
}

// SetDevice 缓存单个设备
func (c *DeviceCache) SetDevice(id int64, device *DeviceResponse) error {
	key := c.keyBuilder.DeviceKey(id)
	data, err := json.Marshal(device)
	if err != nil {
		return err
	}
	c.handler.SetWithExpireTime(key, string(data), DeviceExpiration)

	// 更新统计信息
	c.stats.Sets++

	return nil
}

// GetDevice 获取缓存的单个设备
func (c *DeviceCache) GetDevice(id int64) (*DeviceResponse, error) {
	key := c.keyBuilder.DeviceKey(id)
	data := c.handler.Get(key)
	if data == "" {
		c.stats.Misses++
		return nil, fmt.Errorf("cache miss")
	}

	var device DeviceResponse
	if err := json.Unmarshal([]byte(data), &device); err != nil {
		return nil, err
	}

	// 更新统计信息 - 缓存命中
	c.stats.Hits++

	return &device, nil
}

// SetDeviceFieldValues 缓存设备字段值
func (c *DeviceCache) SetDeviceFieldValues(fieldName string, values []string, isLabelField bool) error {
	key := c.keyBuilder.DeviceFieldKey(fieldName)
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	// 根据字段类型设置不同的过期时间
	expiration := DeviceFieldExpiration
	if isLabelField {
		expiration = LabelFieldExpiration
	}

	c.handler.SetWithExpireTime(key, string(data), expiration)

	// 更新统计信息
	c.stats.Sets++

	return nil
}

// GetDeviceFieldValues 获取缓存的设备字段值
func (c *DeviceCache) GetDeviceFieldValues(fieldName string) ([]string, error) {
	key := c.keyBuilder.DeviceFieldKey(fieldName)
	data := c.handler.Get(key)
	if data == "" {
		c.stats.Misses++
		return nil, fmt.Errorf("cache miss")
	}

	var values []string
	if err := json.Unmarshal([]byte(data), &values); err != nil {
		return nil, err
	}

	// 更新统计信息 - 缓存命中
	c.stats.Hits++

	return values, nil
}

// InvalidateDevice 使单个设备缓存失效
func (c *DeviceCache) InvalidateDevice(id int64) {
	key := c.keyBuilder.DeviceKey(id)
	c.handler.Delete(key)

	// 更新统计信息
	c.stats.Deletes++
}

// InvalidateDeviceLists 使所有设备列表缓存失效
func (c *DeviceCache) InvalidateDeviceLists() error {
	// 清除所有设备列表缓存，包括基本查询和高级查询
	// 使用 KeyBuilder 生成精确的列表模式
	pattern := c.keyBuilder.DeviceListPattern()
	// 使用 SCAN 命令代替 KEYS
	keys, err := c.handler.ScanKeys(pattern)
	if err != nil {
		// 记录错误，但可能仍需尝试删除已找到的部分键（如果需要更健壮）
		// 为简单起见，这里直接返回错误
		return fmt.Errorf("error scanning device list keys with pattern %s: %w", pattern, err)
	}

	for _, key := range keys {
		c.handler.Delete(key)
		// 更新统计信息
		c.stats.Deletes++
	}

	// 打印日志，方便调试
	fmt.Printf("Invalidated %d device list cache keys\n", len(keys))

	return nil
}

// InvalidateDeviceField 使设备字段值缓存失效
func (c *DeviceCache) InvalidateDeviceField(fieldName string) {
	key := c.keyBuilder.DeviceFieldKey(fieldName)
	c.handler.Delete(key)

	// 更新统计信息
	c.stats.Deletes++
}

// InvalidateAllDeviceCache 清除所有与设备相关的缓存
func (c *DeviceCache) InvalidateAllDeviceCache() error {
	// 清除所有设备列表缓存
	if err := c.InvalidateDeviceLists(); err != nil {
		return err
	}

	// 清除所有设备字段值缓存
	// 使用 KeyBuilder 生成精确的字段模式
	pattern := c.keyBuilder.DeviceFieldPattern()
	// 使用 SCAN 命令代替 KEYS
	// 使用 := 重新声明 keys, err 在这个作用域内
	keys, err := c.handler.ScanKeys(pattern)
	if err != nil {
		return fmt.Errorf("error scanning device field keys with pattern %s: %w", pattern, err)
	}

	for _, key := range keys {
		c.handler.Delete(key)
		// 更新统计信息
		c.stats.Deletes++
	}

	// 打印日志，方便调试
	fmt.Printf("Invalidated %d device field cache keys\n", len(keys))

	return nil
}

// GenerateQueryHash 生成查询参数的哈希值
func GenerateQueryHash(query interface{}) string {
	data, err := json.Marshal(query)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", data)
}

// GetCacheStats 获取缓存统计信息
func (c *DeviceCache) GetCacheStats() CacheStats {
	return c.stats
}

// ResetCacheStats 重置缓存统计信息
func (c *DeviceCache) ResetCacheStats() {
	c.stats = CacheStats{
		StartTime: time.Now().Unix(),
	}
}

// WarmupCache 预热缓存
func (c *DeviceCache) WarmupCache(deviceService *DeviceService, deviceQueryService *DeviceQueryService) error {
	// 创建上下文
	ctx := context.TODO()

	// 预热常用设备列表查询
	commonQueries := []struct {
		name  string
		query *DeviceQuery
	}{
		{"all", &DeviceQuery{Page: 1, Size: 20}},
		// 注释掉 OnlySpecial 查询，因为前端当前没有直接支持这个功能
		// {"special", &DeviceQuery{Page: 1, Size: 20, OnlySpecial: true}},
	}

	for _, q := range commonQueries {
		// 执行查询并缓存结果
		response, err := deviceService.ListDevices(ctx, q.query)
		if err != nil {
			return fmt.Errorf("failed to warmup %s devices: %w", q.name, err)
		}

		// 结果已经在 ListDevices 中被缓存，无需再次缓存
		fmt.Printf("Warmed up %s devices cache: %d items\n", q.name, len(response.List))
	}

	// 预热常用字段值
	commonFields := []string{"idc", "room", "appid", "ci_code", "cabinet", "infraType", "netZone", "status"}

	for _, field := range commonFields {
		// 获取字段值并缓存
		values, err := deviceQueryService.GetDeviceFieldValues(ctx, field)
		if err != nil {
			return fmt.Errorf("failed to warmup field %s: %w", field, err)
		}

		// 结果已经在 GetDeviceFieldValues 中被缓存，无需再次缓存
		fmt.Printf("Warmed up field %s cache: %d values\n", field, len(values))
	}

	return nil
}
