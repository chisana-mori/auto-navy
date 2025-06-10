# 弹性伸缩（Elastic Scaling）模块说明

## 1. 模块概述
弹性伸缩系统自动化管理集群资源，基于策略动态调整设备池，实现资源平衡与成本优化。支持策略驱动、设备匹配、订单化管理、异常处理和邮件通知。

## 2. 系统架构与核心功能
- **策略引擎**：定义、存储、执行伸缩策略（CPU/内存阈值、持续时间、冷却期、设备数量等）。
- **设备匹配器**：基于查询模板筛选目标设备，支持灵活配置。
- **订单生成与管理**：所有伸缩操作均生成订单，便于追踪、审计和人工介入。
- **监控与调度**：定时任务自动触发策略评估，支持分布式锁。
- **通知系统**：订单生成或状态变更时自动发送HTML邮件，支持自定义集成。

详细架构、数据模型与流程请参见[系统设计文档](./elastic_scaling_design.md)。

## 3. 快速开始
### 环境准备
```bash
cd server/portal && go run main.go
cd web/navy-fe && npm run dev
```
### 运行测试
```bash
./docs/elastic_scaling/run_frontend_tests.sh 1    # 场景1：正常入池
./docs/elastic_scaling/run_frontend_tests.sh 4    # 场景4：入池无设备
./docs/elastic_scaling/run_frontend_tests.sh all  # 运行所有7个场景
```

## 4. 测试说明与用例
### 4.1 测试目标
- 验证策略评估、订单生成、前端展示、数据一致性等核心功能。

### 4.2 场景总览
| 场景 | 条件 | 预期结果 |
|------|------|----------|
| 1 | CPU连续3天>80% | 生成入池订单，含可用设备 |
| 2 | 内存连续2天<20% | 生成退池订单，含在池设备 |
| 3 | 阈值未持续 | 不生成订单，记录失败 |
| 4 | 满足阈值无可用设备 | 生成提醒订单，提示申请设备 |
| 5 | 满足阈值无在池设备 | 生成提醒订单，提示协调处理 |
| 6 | 只匹配部分设备 | 生成订单，含部分设备 |
| 7 | 退池只匹配部分设备 | 生成订单，含部分设备 |

### 4.3 快速测试指南
```bash
chmod +x docs/elastic_scaling/run_frontend_tests.sh
./docs/elastic_scaling/run_frontend_tests.sh 1    # 场景1
./docs/elastic_scaling/run_frontend_tests.sh all  # 所有场景
./docs/elastic_scaling/run_frontend_tests.sh clean # 清理测试数据
```
如脚本不可用，可手动执行SQL：
```bash
sqlite3 ./data/navy.db
.read docs/elastic_scaling/test_data_scenario1_pool_entry.sql
```

### 4.4 关键验证点
- 策略管理、资源监控、订单管理页面数据与流程正确
- 订单详情、设备分配、邮件通知内容完整
- API接口可手动触发与验证

## 5. 邮件通知功能
- 订单生成自动生成美观HTML邮件，内容含订单、设备、操作指引等
- 支持自定义集成企业邮箱、钉钉、企业微信等
- 详细说明见[邮件通知功能](./email_notification_feature.md)

## 6. 用户手册
- 策略配置、设备匹配、手动订单、订单流转等详见[用户手册](./user_manual.md)

## 7. 设计与变更
- 架构、数据模型、核心流程详见[系统设计文档](./elastic_scaling_design.md)
- 变更历史见[CHANGELOG](./CHANGELOG.md)

## 8. 其他资源
- 邮件模板示例：[入池](./email_pool_entry_example.html)、[退池](./email_pool_exit_example.html)、[无设备](./email_no_devices_example.html)
- 测试数据脚本：`test_data_scenario*.sql`、`create_test_schema.sql`
- 测试工具：`run_frontend_tests.sh`、`validate_test_data.sh`

---
如需详细流程、数据结构、API等，请查阅各子文档。

## 📋 测试指南与用例

弹性伸缩模块的测试文档已整合优化，提供了一个全面的测试参考，详见：
- [弹性伸缩端到端测试文档](./test_case.md)

该文档涵盖了测试环境准备、七大核心测试场景的详细描述（包括测试目标、SQL数据、关键特征、前置条件、测试步骤及预期结果）、Mock数据示例、测试执行指南（包括环境准备、脚本与手动执行方式、关键验证点）以及测试报告模板，便于端到端验证弹性伸缩功能。
