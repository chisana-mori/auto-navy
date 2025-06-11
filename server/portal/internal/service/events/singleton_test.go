package events

import (
	"testing"

	"go.uber.org/zap"
)

func TestGetGlobalEventManager(t *testing.T) {
	// 重置全局状态
	ResetGlobalEventManager()

	// 测试首次获取时自动创建
	em1 := GetGlobalEventManager()
	if em1 == nil {
		t.Fatal("Expected non-nil EventManager")
	}

	// 测试再次获取返回同一实例
	em2 := GetGlobalEventManager()
	if em1 != em2 {
		t.Error("Expected same EventManager instance")
	}

	// 验证初始化状态
	if !IsGlobalEventManagerInitialized() {
		t.Error("Expected EventManager to be initialized")
	}
}

func TestInitGlobalEventManager(t *testing.T) {
	// 重置全局状态
	ResetGlobalEventManager()

	// 创建自定义配置
	logger, _ := zap.NewDevelopment()
	config := &Config{
		Async:      false,
		RetryCount: 5,
	}

	// 初始化全局 EventManager
	em1 := InitGlobalEventManager(logger, config)
	if em1 == nil {
		t.Fatal("Expected non-nil EventManager")
	}

	// 验证配置
	if em1.config.Async != false {
		t.Error("Expected Async to be false")
	}
	if em1.config.RetryCount != 5 {
		t.Error("Expected RetryCount to be 5")
	}

	// 再次初始化应该返回同一实例
	em2 := InitGlobalEventManager(logger, DefaultConfig())
	if em1 != em2 {
		t.Error("Expected same EventManager instance")
	}

	// 配置不应该改变
	if em2.config.Async != false {
		t.Error("Expected Async to remain false")
	}
}

func TestIsGlobalEventManagerInitialized(t *testing.T) {
	// 重置全局状态
	ResetGlobalEventManager()

	// 初始状态应该是未初始化
	if IsGlobalEventManagerInitialized() {
		t.Error("Expected EventManager to be uninitialized")
	}

	// 获取实例后应该是已初始化
	_ = GetGlobalEventManager()
	if !IsGlobalEventManagerInitialized() {
		t.Error("Expected EventManager to be initialized")
	}
}

func TestResetGlobalEventManager(t *testing.T) {
	// 初始化 EventManager
	_ = GetGlobalEventManager()
	if !IsGlobalEventManagerInitialized() {
		t.Error("Expected EventManager to be initialized")
	}

	// 重置
	ResetGlobalEventManager()
	if IsGlobalEventManagerInitialized() {
		t.Error("Expected EventManager to be uninitialized after reset")
	}

	// 重置后应该能够重新初始化
	em := GetGlobalEventManager()
	if em == nil {
		t.Fatal("Expected non-nil EventManager after reset")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// 重置全局状态
	ResetGlobalEventManager()

	// 并发获取 EventManager
	const numGoroutines = 100
	results := make(chan *EventManager, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- GetGlobalEventManager()
		}()
	}

	// 收集结果
	var managers []*EventManager
	for i := 0; i < numGoroutines; i++ {
		managers = append(managers, <-results)
	}

	// 验证所有实例都是同一个
	firstManager := managers[0]
	for i, manager := range managers {
		if manager != firstManager {
			t.Errorf("Goroutine %d got different EventManager instance", i)
		}
	}
}