package events

import (
	"context"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventManager 事件管理器
type EventManager struct {
	genericHandlers map[string]map[reflect.Type][]interface{} // 存储泛型处理器
	handlers        map[string][]EventHandler                 // 保留用于兼容
	converters      map[reflect.Type]interface{}              // 存储类型转换器
	mutex           sync.RWMutex
	logger          *zap.Logger
	config          *Config
}

// Config 事件管理器配置
type Config struct {
	Timeout     time.Duration // 事件处理超时时间
	RetryCount  int           // 重试次数
	BufferSize  int           // 事件队列缓冲大小
	Async       bool          // 是否异步处理
	EnableStats bool          // 是否启用统计
}

// PublishRequest 发布事件的请求结构体
type PublishRequest struct {
	Event Event
	Ctx   context.Context
}

// GetHandlersRequest 获取处理器列表的请求结构体
type GetHandlersRequest struct {
	EventType string
}

// ShutdownRequest 关闭的请求结构体
type ShutdownRequest struct {
	Ctx context.Context
}

// RegisterRequest 注册事件处理器的请求结构体
type RegisterRequest struct {
	EventType   string
	HandlerName string
	Handler     EventHandler
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:     30 * time.Second,
		RetryCount:  3,
		BufferSize:  1000,
		Async:       true,
		EnableStats: true,
	}
}

// NewEventManager 创建新的事件管理器
func NewEventManager(logger *zap.Logger, config *Config) *EventManager {
	if config == nil {
		config = DefaultConfig()
	}

	return &EventManager{
		genericHandlers: make(map[string]map[reflect.Type][]interface{}),
		handlers:        make(map[string][]EventHandler),
		converters:      make(map[reflect.Type]interface{}),
		logger:          logger,
		config:          config,
	}
}
