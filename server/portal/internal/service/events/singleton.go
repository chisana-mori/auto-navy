package events

import (
	"sync"

	"go.uber.org/zap"
)

// 全局单例变量
var (
	globalEventManager *EventManager
	once               sync.Once
	mu                 sync.RWMutex
)

// GetGlobalEventManager 获取全局唯一的 EventManager 实例
// 如果尚未初始化，将使用默认配置创建一个新实例
func GetGlobalEventManager() *EventManager {
	mu.RLock()
	if globalEventManager != nil {
		defer mu.RUnlock()
		return globalEventManager
	}
	mu.RUnlock()

	// 如果未初始化，使用默认配置创建
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		if globalEventManager == nil {
			// 使用默认 logger 和配置
			logger, _ := zap.NewProduction()
			globalEventManager = NewEventManager(logger, DefaultConfig())
		}
	})

	return globalEventManager
}

// InitGlobalEventManager 初始化全局 EventManager 实例
// 只能在应用启动时调用一次，如果已经初始化则忽略
func InitGlobalEventManager(logger *zap.Logger, config *Config) *EventManager {
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		if globalEventManager == nil {
			globalEventManager = NewEventManager(logger, config)
		}
	})

	return globalEventManager
}

// ResetGlobalEventManager 重置全局 EventManager 实例
// 仅用于测试目的，生产环境不应调用
func ResetGlobalEventManager() {
	mu.Lock()
	defer mu.Unlock()
	globalEventManager = nil
	once = sync.Once{}
}

// IsGlobalEventManagerInitialized 检查全局 EventManager 是否已初始化
func IsGlobalEventManagerInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return globalEventManager != nil
}