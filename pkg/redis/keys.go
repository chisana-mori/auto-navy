package redis

import (
	"fmt"
	"strings"
)

// 全局前缀，用于区分不同环境或应用
const (
	// 可以根据环境变量设置不同的前缀
	GlobalPrefix = "navy"
)

// 模块前缀
const (
	DeviceModule = "device"
	K8sModule    = "k8s"
	F5Module     = "f5"
	// 其他模块...
)

// 设备相关键模板
const (
	// 设备列表缓存键模板
	DeviceListKeyTpl = "%s:%s:%s:list:%s" // {global}:{module}:{version}:list:{query_hash}

	// 单个设备缓存键模板
	DeviceKeyTpl = "%s:%s:%s:id:%d" // {global}:{module}:{version}:id:{device_id}

	// 特殊设备集合键
	DeviceSpecialKey = "%s:%s:%s:special" // {global}:{module}:{version}:special

	// 设备字段值缓存键模板
	DeviceFieldKeyTpl = "%s:%s:%s:field:%s" // {global}:{module}:{version}:field:{field_name}
)

// Pub/Sub 通道名称
const (
	// DeviceUpdatesChannel 定义用于设备更新通知的 Redis Pub/Sub 通道名称
	DeviceUpdatesChannel = "navy:device:updates"
)

// KeyBuilder 提供构建Redis键的方法
type KeyBuilder struct {
	globalPrefix string
	version      string
}

// NewKeyBuilder 创建一个新的KeyBuilder实例
func NewKeyBuilder(globalPrefix string, version string) *KeyBuilder {
	if globalPrefix == "" {
		globalPrefix = GlobalPrefix
	}
	if version == "" {
		version = "v1" // 默认版本
	}
	return &KeyBuilder{globalPrefix: globalPrefix, version: version}
}

// DeviceListKey 构建设备列表缓存键
func (kb *KeyBuilder) DeviceListKey(queryHash string) string {
	return fmt.Sprintf(DeviceListKeyTpl, kb.globalPrefix, DeviceModule, kb.version, queryHash)
}

// DeviceKey 构建单个设备缓存键
func (kb *KeyBuilder) DeviceKey(id int) string {
	return fmt.Sprintf(DeviceKeyTpl, kb.globalPrefix, DeviceModule, kb.version, id)
}

// DeviceSpecialKey 构建特殊设备集合键
func (kb *KeyBuilder) DeviceSpecialKey() string {
	return fmt.Sprintf(DeviceSpecialKey, kb.globalPrefix, DeviceModule, kb.version)
}

// DeviceFieldKey 构建设备字段值缓存键
func (kb *KeyBuilder) DeviceFieldKey(fieldName string) string {
	return fmt.Sprintf(DeviceFieldKeyTpl, kb.globalPrefix, DeviceModule, kb.version, fieldName)
}

// GetModuleFromKey 从键中提取模块名
func (kb *KeyBuilder) GetModuleFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// GetKeyPattern 获取特定模块的通用键模式，可能不适用于所有场景
// 建议使用更具体的模式生成方法，如 DeviceListPattern
func (kb *KeyBuilder) GetKeyPattern(module string) string {
	return fmt.Sprintf("%s:%s:%s:*", kb.globalPrefix, module, kb.version)
}

// DeviceListPattern 生成用于扫描设备列表缓存的模式
func (kb *KeyBuilder) DeviceListPattern() string {
	// 模式应为 {global}:{module}:{version}:list:*
	return fmt.Sprintf("%s:%s:%s:list:*", kb.globalPrefix, DeviceModule, kb.version)
}

// DeviceFieldPattern 生成用于扫描设备字段缓存的模式
func (kb *KeyBuilder) DeviceFieldPattern() string {
	// 模式应为 {global}:{module}:{version}:field:*
	return fmt.Sprintf("%s:%s:%s:field:*", kb.globalPrefix, DeviceModule, kb.version)
}
