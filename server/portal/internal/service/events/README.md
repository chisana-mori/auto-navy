# Navy-NG 泛型事件系统

## 概述

Navy-NG 泛型事件系统是一个高度抽象和灵活的事件驱动架构，支持类型安全的事件发布和处理，并提供自动类型转换功能。

## 核心特性

### 🔥 泛型支持
- 完全类型安全的事件发布和处理
- 支持任意类型的事件数据
- 编译时类型检查

### 🔄 自动类型转换
- 注册自定义转换器，支持多种数据源
- 自动转换 JSON、Map、现有结构体等格式
- 可扩展的转换逻辑

### ⚡ 高性能
- 异步事件处理
- 带重试机制的可靠投递
- 可配置的超时和缓冲

### 🛡️ 生产就绪
- 完整的错误处理和日志记录
- 支持分布式环境
- 优雅的关闭机制

### 🎯 优雅API
- 参数内聚到请求结构体
- 链式调用支持
- 直观的方法命名

## 快速开始

### 1. 初始化事件管理器

```go
import (
    "navy-ng/server/portal/internal/service/events"
    "go.uber.org/zap"
)

// 创建事件管理器
logger, _ := zap.NewProduction()
eventManager := events.NewEventManager(logger, events.DefaultConfig())

// 初始化泛型事件系统（注册内置转换器）
events.InitializeGenericEventSystem(eventManager)
```

### 2. 定义事件数据结构

```go
// 订单事件数据
type OrderEventData struct {
    OrderID     int    `json:"order_id"`
    OrderType   string `json:"order_type"`
    Status      string `json:"status"`
    Operator    string `json:"operator"`
    Description string `json:"description"`
}

// 设备事件数据
type DeviceEventData struct {
    DeviceID    int    `json:"device_id"`
    OrderID     int    `json:"order_id"`
    Action      string `json:"action"`
    Status      string `json:"status"`
    Result      string `json:"result"`
    ErrorMsg    string `json:"error_msg,omitempty"`
}
```

### 3. 发布事件

#### 方式1：直接发布结构体（推荐）

```go
ctx := context.Background()

orderData := OrderEventData{
    OrderID:     12345,
    OrderType:   "elastic_scaling",
    Status:      "completed",
    Operator:    "system",
    Description: "订单处理完成",
}

// 泛型发布（优雅API）
err := events.PublishGeneric(eventManager, events.GenericEventRequest[OrderEventData]{
    EventType: "order.completed",
    Data:      orderData,
    Source:    "order_service", 
    Context:   ctx,
})
```

#### 方式2：使用便利方法

```go
// 订单事件便利方法
err := events.PublishOrderEventDirect(eventManager, events.OrderEventRequest{
    EventType:   "order.processing",
    OrderID:     12346,
    OrderType:   "maintenance",
    Status:      "processing",
    Operator:    "admin",
    Description: "维护订单处理中",
    Context:     ctx,
})

// 设备事件便利方法
err := events.PublishDeviceEventDirect(eventManager, events.DeviceEventRequest{
    EventType: "device.operation.completed",
    DeviceID:  98765,
    OrderID:   12345,
    Action:    "pool_entry",
    Status:    "success",
    Result:    "设备成功加入资源池",
    Context:   ctx,
})
```

#### 方式3：自动转换发布

```go
// 从 Map 发布（自动转换）
mapData := map[string]interface{}{
    "order_id":    12345,
    "order_type":  "maintenance", 
    "status":      "processing",
    "operator":    "admin",
    "description": "维护订单处理中",
}

err := events.PublishGenericWithConverter[OrderEventData](eventManager, 
    events.GenericConvertRequest[OrderEventData]{
        EventType: "order.processing",
        RawData:   mapData,
        Source:    "order_service",
        Context:   ctx,
    })

// 从 JSON 发布（自动转换）
jsonData := `{"device_id": 98765, "order_id": 12345, "action": "pool_entry", "status": "success"}`

err := events.PublishGenericWithConverter[DeviceEventData](eventManager,
    events.GenericConvertRequest[DeviceEventData]{
        EventType: "device.operation.completed",
        RawData:   jsonData,
        Source:    "device_service",
        Context:   ctx,
    })

// 兼容旧API的便利方法
err := events.PublishOrderEvent(eventManager, ctx, "order.processing", mapData)
err := events.PublishDeviceEvent(eventManager, ctx, "device.operation.completed", jsonData)
```

### 4. 注册事件处理器

#### 泛型方式（推荐）

```go
// 自动类型转换，类型安全
events.RegisterGenericHandler(eventManager, "order.completed", "order_completion_handler",
    func(ctx context.Context, data OrderEventData) error {
        // 直接使用强类型数据，无需转换
        log.Printf("Order %d of type %s completed by %s", 
            data.OrderID, data.OrderType, data.Operator)
        
        // 执行业务逻辑
        return nil
    })

events.RegisterGenericHandler(eventManager, "device.operation.completed", "device_operation_handler", 
    func(ctx context.Context, data DeviceEventData) error {
        log.Printf("Device %d completed action %s with status %s",
            data.DeviceID, data.Action, data.Status)
        
        // 执行业务逻辑
        return nil
    })
```

#### 传统方式

```go
eventManager.RegisterFunc(events.RegisterFuncRequest{
    EventType:   "order.completed",
    HandlerName: "order_completion_handler",
    HandlerFunc: func(ctx context.Context, event events.Event) error {
        // 手动类型转换
        data, ok := event.Data().(OrderEventData)
        if !ok {
            return fmt.Errorf("invalid event data type")
        }
        
        // 处理逻辑
        log.Printf("Order %d completed", data.OrderID)
        return nil
    },
})
```

## 在 ElasticScalingService 中的使用

### 集成示例（优化版）

```go
func (s *ElasticScalingService) UpdateOrderStatus(orderID int, newStatus, executor, reason string) error {
    // 获取旧状态
    oldOrder, err := s.GetOrder(orderID)
    if err != nil {
        return err
    }
    
    // 更新数据库
    err = s.updateOrderInDB(orderID, newStatus, executor, reason)
    if err != nil {
        return err
    }
    
    // 发布事件（优雅API）
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 方式1：使用便利方法
    if s.eventManager != nil {
        err = events.PublishOrderEventDirect(s.eventManager, events.OrderEventRequest{
            EventType:   "order.status.changed",
            OrderID:     orderID,
            OrderType:   oldOrder.OrderType,
            Status:      newStatus,
            Operator:    executor,
            Description: reason,
            Context:     ctx,
        })
        if err != nil {
            s.logger.Error("Failed to publish order status change event", zap.Error(err))
            // 不影响主流程
        }
    }
    
    return nil
}

func (s *ElasticScalingService) NotifyDeviceOperation(deviceID, orderID int, action, status, result string) error {
    if s.eventManager == nil {
        return nil
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 使用便利方法发布设备事件
    return events.PublishDeviceEventDirect(s.eventManager, events.DeviceEventRequest{
        EventType: fmt.Sprintf("device.operation.%s", status),
        DeviceID:  deviceID,
        OrderID:   orderID,
        Action:    action,
        Status:    status,
        Result:    result,
        Context:   ctx,
    })
}
```

## API对比

### 旧API vs 新API

```go
// 旧API（参数分散）
err := events.PublishGeneric(em, ctx, "order.completed", orderData, "order_service")

// 新API（参数内聚，更优雅）
err := events.PublishGeneric(em, events.GenericEventRequest[OrderEventData]{
    EventType: "order.completed",
    Data:      orderData,
    Source:    "order_service",
    Context:   ctx,
})

// 便利方法（最简洁）
err := events.PublishOrderEventDirect(em, events.OrderEventRequest{
    EventType:   "order.completed",
    OrderID:     12345,
    OrderType:   "elastic_scaling",
    Status:      "completed",
    Operator:    "system",
    Description: "订单处理完成",
    Context:     ctx,
})
```

### 优势对比

| 特性 | 旧API | 新API |
|------|--------|---------|
| 参数组织 | 分散的参数列表 | 内聚的请求结构 |
| 类型安全 | ✅ | ✅ |
| 代码可读性 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 扩展性 | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| IDE支持 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 向后兼容 | - | ✅ |

## 最佳实践

### 1. 优先使用便利方法
```go
// 推荐：使用便利方法
events.PublishOrderEventDirect(em, events.OrderEventRequest{
    EventType: "order.completed",
    OrderID:   12345,
    // ... 其他字段
    Context:   ctx,
})

// 而不是：
events.PublishGeneric(em, events.GenericEventRequest[OrderEventData]{...})
```

### 2. 参数验证
```go
func PublishOrderEvent(em *EventManager, req OrderEventRequest) error {
    if req.OrderID <= 0 {
        return fmt.Errorf("invalid order ID: %d", req.OrderID)
    }
    if req.EventType == "" {
        return fmt.Errorf("event type is required")
    }
    if req.Context == nil {
        req.Context = context.Background()
    }
    
    return events.PublishOrderEventDirect(em, req)
}
```

### 3. 上下文传递
```go
// 带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

req := events.OrderEventRequest{
    EventType: "order.timeout",
    Context:   ctx, // 传递带超时的上下文
    // ... 其他字段
}
```

这个优化使得Navy-NG泛型事件系统的API更加优雅、直观和易用，同时保持了强大的功能和向后兼容性。

## 测试与验证

### 单元测试覆盖

本项目包含完整的单元测试套件，覆盖以下核心功能：

#### 🧪 核心功能测试
- **泛型事件注册和发布** - 验证类型安全的事件系统
- **类型转换器** - 测试自动类型转换和自定义转换器
- **错误处理机制** - 验证重试、超时和错误传播
- **异步事件处理** - 测试并发和异步处理能力
- **处理器生命周期** - 测试注册、注销和重复注册

#### 🎯 业务场景测试
测试包含完整的订单状态变更业务场景：

```go
// 场景1: 订单完成流程
// device.operation.completed → order.status.completed

// 场景2: 订单取消流程  
// order.status.cancelled

// 场景3: 订单退回流程
// order.status.returning → device.operation.returning

// 场景4: 并发多订单处理
// 多个订单同时处理不同状态变更
```

#### 📊 性能基准测试
- **ConvertFromMap**: ~23.66 ns/op, 0 allocations
- **ConvertFromJSON**: ~642.3 ns/op, 448 B/10 allocations  
- **PublishGenericEvent**: ~5198 ns/op, 5207 B/44 allocations
- **RegisterGenericHandler**: ~2010 ns/op, 3024 B/23 allocations

### 运行测试

```bash
# 运行所有测试
go test ./server/portal/internal/service/events/ -v

# 运行业务场景测试
go test ./server/portal/internal/service/events/ -run "Test_OrderStatusChangeScenario" -v

# 运行性能基准测试
go test ./server/portal/internal/service/events/ -bench=. -benchmem

# 测试覆盖率
go test ./server/portal/internal/service/events/ -cover
```

### 测试结果摘要

✅ **19个测试全部通过**
- 8个类型转换器测试
- 10个泛型事件系统测试  
- 1个完整业务场景测试（包含4个子场景）

🔥 **核心验证点**：
- 类型安全性和编译时检查
- 事件的正确发布和接收
- 错误处理和重试机制
- 异步处理和并发安全
- 业务场景的端到端流程

## 架构优势

### 与传统方案对比

| 特性 | 传统事件系统 | Navy-NG 泛型事件系统 |
|------|-------------|-------------------|
| 类型安全 | ❌ | ✅ 编译时检查 |
| 性能 | 中等 | ⚡ 高性能 |
| 可维护性 | 低 | 🔧 高度可维护 |
| 扩展性 | 受限 | 🚀 完全可扩展 |
| 代码简洁度 | 冗余 | 🎯 简洁优雅 |
| 测试覆盖 | 不足 | ✅ 全面测试 |

### 解决的核心问题

1. **类型安全问题** - 通过泛型确保编译时类型检查
2. **代码重复问题** - 通过优雅的API设计减少重复代码
3. **性能问题** - 通过异步处理和对象池优化性能
4. **扩展性问题** - 通过可配置的转换器支持任意数据格式
5. **测试难度问题** - 提供完整的测试工具和业务场景验证

---

**NavyNG 泛型事件系统 - 让事件驱动架构更简单、更安全、更高效！** 🚀