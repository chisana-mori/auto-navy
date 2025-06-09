# 弹性伸缩 (Elastic Scaling) 模块说明

## 1. 模块概述

弹性伸缩系统是一个自动化的资源管理解决方案，能够根据预设策略自动调整集群资源，支持设备的入池和退池操作。系统通过监控资源使用情况，在满足触发条件时自动生成相应的操作订单，以实现资源的动态平衡和成本优化。

## 2. 核心功能

- **策略驱动的自动伸缩**: 支持基于CPU、内存等资源的阈值，配置自动化的入池/退池策略。
- **灵活的设备匹配**: 可通过查询模板自定义复杂的设备筛选逻辑，确保精确选择目标设备。
- **订单化管理**: 所有伸缩操作均生成订单，便于追踪和审计。
- **异常情况处理**: 对无可用设备、部分匹配等情况有明确的处理逻辑，生成提醒订单通知人工介入。
- **邮件通知系统**: 在订单生成或状态变更时，自动发送内容详尽的HTML邮件通知。

## 3. 快速开始

### 环境准备

```bash
# 启动后端服务
cd server/portal
go run main.go

# 启动前端服务（新终端）
cd web/navy-fe
npm run dev
```

### 运行测试

使用自动化脚本可以快速初始化特定场景的测试数据。

```bash
# 在项目根目录执行
./docs/elastic_scaling/run_frontend_tests.sh 1    # 场景1：正常入池
./docs/elastic_scaling/run_frontend_tests.sh 4    # 场景4：入池无设备
./docs/elastic_scaling/run_frontend_tests.sh all  # 运行所有7个场景
```

## 4. 文档导航

本模块的详细文档分布在以下文件中，请根据需要查阅：

### 📖 设计与实现
- **[系统设计文档](./elastic_scaling_design.md)**: 深入了解系统架构、数据模型、核心工作流和API设计。
- **[邮件通知功能](./email_notification_feature.md)**: 查看邮件通知的实现细节、模板和集成方式。
    - [入池邮件示例](./email_pool_entry_example.html)
    - [退池邮件示例](./email_pool_exit_example.html)
    - [无设备邮件示例](./email_no_devices_example.html)

### 👨‍💻 使用与操作
- **[用户手册](./user_manual.md)**: 学习如何配置监控策略、手动创建订单以及理解订单的流转状态。
- **[快速测试指南](./quick_test_guide.md)**: 提供开箱即用的测试步骤和关键验证点。

### 🧪 测试用例与数据
- **[前端测试总览](./README_frontend_tests.md)**: 前端测试的总体说明。
- **[详细测试用例](./frontend_test_cases.md)**: 包含所有7个核心场景的详细测试步骤和预期结果。
- **[测试场景概览](./test_scenarios_overview.md)**: 对7个测试场景SQL数据文件的关键特征说明。
- **测试数据脚本**:
    - `test_data_scenario*.sql`: 各场景的SQL数据脚本。
    - `create_test_schema.sql`: 用于创建测试的数据库表结构。
- **测试工具**:
    - `run_frontend_tests.sh`: 自动化测试执行脚本。
    - `validate_test_data.sh`: 测试数据验证脚本。

### 🔄 变更记录
- **[更新日志 (CHANGELOG)](./CHANGELOG.md)**: 查看模块的版本迭代和重要变更历史。

## 5. 系统架构概览

系统采用前后端分离架构，核心组件包括：

- **策略引擎**: 负责策略的定义、存储和执行。
- **设备匹配器**: 根据策略要求匹配合适的设备。
- **订单生成器**: 创建弹性伸缩操作订单。
- **监控模块**: 定时任务，负责触发策略评估。
- **通知系统**: 自动发送邮件通知相关人员。

### 数据模型

核心数据表包括 `elastic_scaling_strategies` (策略), `elastic_scaling_executions` (执行历史), 和 `resource_pool_device_matching_policies` (设备匹配策略)。详细表结构请参考[系统设计文档](./elastic_scaling_design.md)。
