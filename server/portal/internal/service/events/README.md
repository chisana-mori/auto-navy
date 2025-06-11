# Events Service - 重构后的文件架构

## 架构概览

事件服务已重构为模块化的文件结构，每个文件负责特定的功能模块，提高了代码的可读性和可维护性。

## 文件结构

```
events/
├── interfaces.go          # 所有接口定义
├── manager_core.go        # 事件管理器核心结构和配置
├── generic_events.go      # 泛型事件相关功能
├── converter.go           # 类型转换器功能
├── handlers.go            # 处理器管理功能
├── publisher.go           # 事件发布功能
├── execution.go           # 事件处理执行功能
├── utils.go               # 辅助工具方法
├── types.go               # 具体事件类型定义
└── README.md              # 文档
```

## 模块说明

### 1. interfaces.go - 接口定义模块
**职责：** 定义所有的接口和函数类型

**主要内容：**
- `GenericEventHandler[T]` - 泛型事件处理器接口
- `GenericEventHandlerFunc[T]` - 泛型函数类型处理器
- `Event` - 基础事件接口
- `EventHandler` - 传统事件处理器接口（兼容性）
- `EventHandlerFunc` - 函数类型处理器（兼容性）
- `TypeConverter[T]` - 类型转换器接口

### 2. manager_core.go - 核心管理器模块
**职责：** 事件管理器的核心结构定义和基础配置

**主要内容：**
- `EventManager` - 事件管理器主结构体
- `Config` - 配置结构体
- `PublishRequest`, `GetHandlersRequest`, `ShutdownRequest` - 请求结构体
- `NewEventManager()` - 构造函数
- `DefaultConfig()` - 默认配置

### 3. generic_events.go - 泛型事件模块
**职责：** 泛型事件系统的核心功能

**主要内容：**
- `GenericEvent[T]` - 泛型事件结构体
- `GenericEventRequest[T]` - 泛型事件发布请求
- `GenericConvertRequest[T]` - 带转换器的泛型发布请求
- `RegisterGenericRequest[T]` - 泛型注册请求
- `UnregisterGenericRequest[T]` - 泛型注销请求
- `NewGenericEvent()` - 创建泛型事件
- `PublishGeneric()` - 泛型事件发布
- `PublishGenericWithConverter()` - 带转换器的泛型事件发布

### 4. converter.go - 类型转换器模块
**职责：** 处理不同数据类型之间的转换

**主要内容：**
- `DefaultTypeConverter[T]` - 默认类型转换器实现
- `RegisterConverter()` - 注册自定义转换器
- `GetConverter()` - 获取类型转换器

### 5. handlers.go - 处理器管理模块
**职责：** 事件处理器的注册、注销和管理

**主要内容：**
- `NamedGenericEventHandler[T]` - 带名称的泛型处理器
- `NamedEventHandler` - 带名称的传统处理器（兼容性）
- `RegisterGeneric()` - 注册泛型处理器
- `RegisterGenericFunc()` - 注册泛型函数处理器
- `UnregisterGeneric()` - 注销泛型处理器
- `isGenericHandlerRegistered()` - 检查处理器是否已注册

### 6. publisher.go - 事件发布模块
**职责：** 事件的发布和分发逻辑

**主要内容：**
- `Publish()` - 主发布方法
- `publishGenericEvent()` - 泛型事件发布逻辑

### 7. execution.go - 事件执行模块
**职责：** 事件处理的具体执行逻辑，包括重试和错误处理

**主要内容：**
- `handleGenericEventSync()` - 同步处理泛型事件
- `handleGenericEventAsync()` - 异步处理泛型事件
- `executeGenericHandlerWithRetry()` - 带重试的泛型处理器执行
- `handleEventSync()` - 同步处理传统事件
- `handleEventAsync()` - 异步处理传统事件
- `executeHandlerWithRetry()` - 带重试的传统处理器执行

### 8. utils.go - 辅助工具模块
**职责：** 提供辅助功能和管理器的生命周期管理

**主要内容：**
- `GetHandlers()` - 获取处理器列表
- `GetAllEventTypes()` - 获取所有事件类型
- `Shutdown()` - 优雅关闭事件管理器

### 9. types.go - 事件类型模块
**职责：** 定义具体的业务事件类型（保持原有结构）

**主要内容：**
- 订单相关事件
- 设备操作事件
- 维护事件
- 弹性伸缩事件

## 优势

### 1. 模块化设计
每个文件职责明确，功能内聚，降低了代码的复杂度。

### 2. 易于维护
- 相关功能集中在同一文件中
- 接口定义统一管理
- 核心逻辑与辅助功能分离

### 3. 易于扩展
- 新增事件类型可以独立添加
- 转换器可以独立扩展
- 处理器管理逻辑独立

### 4. 更好的可测试性
- 每个模块可以独立测试
- 依赖关系更清晰

## 使用示例

### 基本使用
```go
// 创建事件管理器
em := NewEventManager(logger, DefaultConfig())

// 注册处理器
RegisterGeneric(em, RegisterGenericRequest[OrderEvent]{
    EventType:   "order.created",
    HandlerName: "order_handler",
    Handler:     &MyOrderHandler{},
})

// 发布事件
err := PublishGeneric(em, GenericEventRequest[OrderEvent]{
    EventType: "order.created",
    Data:      orderEvent,
    Source:    "order_service",
    Context:   ctx,
})
```

### 使用类型转换器
```go
// 注册转换器
RegisterConverter(em, &CustomOrderConverter{})

// 发布带转换的事件
err := PublishGenericWithConverter(em, GenericConvertRequest[OrderEvent]{
    EventType: "order.created",
    RawData:   rawOrderData,
    Source:    "order_service",
    Context:   ctx,
})
```

## 兼容性

重构后的代码完全向后兼容，原有的使用方式仍然有效，同时提供了更现代化的泛型接口。

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