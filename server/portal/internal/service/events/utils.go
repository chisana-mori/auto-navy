package events

// GetHandlers 获取指定事件类型的处理器列表
func (em *EventManager) GetHandlers(req GetHandlersRequest) []string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	handlers := em.handlers[req.EventType]
	names := make([]string, len(handlers))
	for i, handler := range handlers {
		names[i] = handler.Name()
	}

	return names
}

// GetAllEventTypes 获取所有已注册的事件类型
func (em *EventManager) GetAllEventTypes() []string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	types := make([]string, 0, len(em.handlers))
	for eventType := range em.handlers {
		types = append(types, eventType)
	}

	return types
}

// Shutdown 优雅关闭事件管理器
func (em *EventManager) Shutdown(req ShutdownRequest) error {
	em.logger.Info("Shutting down event manager")

	// 等待正在处理的事件完成
	// 这里可以添加更复杂的等待逻辑

	em.mutex.Lock()
	defer em.mutex.Unlock()

	// 清空处理器
	em.handlers = make(map[string][]EventHandler)

	em.logger.Info("Event manager shutdown completed")
	return nil
}
