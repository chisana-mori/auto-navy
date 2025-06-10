# 弹性伸缩端到端测试文档

本文档整合了弹性伸缩模块相关的测试场景、用例、数据和执行指南，旨在提供一个全面的测试参考。

## 1. 测试场景与用例

本节描述了7个核心的弹性扩缩容测试场景，覆盖了入池、退池、阈值判断、设备匹配等关键逻辑。每个场景均包含测试目标、对应的SQL数据文件名、关键特征、前置条件、测试步骤和预期结果。

### 场景1：入池订单生成 - CPU使用率持续超过阈值 ✅
*   **SQL文件**: `test_data_scenario1_pool_entry.sql`
*   **测试目标**: 验证当CPU使用率连续3天超过80%时，系统能够正确生成入池订单。
*   **关键特征**: 连续3天CPU使用率超阈值（例如85%, 90%, 88%），有可用设备（例如2台），策略要求2台。
*   **前置条件**: 策略状态为启用，CPU阈值设置为80%，有可用的设备用于匹配。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario1_pool_entry.sql`)。
    2.  访问策略管理页面，验证策略状态和配置。
    3.  访问资源监控页面，查看对应集群的资源快照数据，确认CPU使用率符合条件。
    4.  等待系统自动评估或手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了新的入池订单。
    6.  查看订单详情，验证设备分配是否正确，邮件通知（如果配置）是否符合预期。
*   **预期结果**: 成功生成入池订单，订单状态为“待处理”，订单包含2台匹配的设备，策略执行历史记录为“order_created”。

### 场景2：退池订单生成 - CPU使用率持续低于阈值 ✅
*   **SQL文件**: `test_data_scenario2_pool_exit.sql`
*   **测试目标**: 验证当CPU使用率连续3天低于30%时，系统能够正确生成退池订单。
*   **关键特征**: 连续3天CPU使用率低于阈值（例如25%, 20%, 28%），有2台运行中设备，策略要求2台。
*   **前置条件**: 策略状态为启用，CPU阈值设置为30%（或内存分配率低于20%等，根据实际策略配置调整），有在池设备可用于退池。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario2_pool_exit.sql`)。
    2.  访问策略管理页面，验证策略配置。
    3.  访问资源监控页面，查看相应指标趋势。
    4.  等待或手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了新的退池订单。
    6.  查看订单详情，确认需要退池的设备列表是否正确。
*   **预期结果**: 成功生成退池订单，订单类型为“退池”，包含2台需要退池的设备，策略执行历史记录为“order_created”。

### 场景3：阈值未达到 - CPU使用率未持续达标 ✅
*   **SQL文件**: `test_data_scenario3_threshold_not_met.sql`
*   **测试目标**: 验证当CPU使用率未连续达到阈值时（例如连续3天的要求，但中间一天中断），系统不会生成订单。
*   **关键特征**: CPU使用率例如为85%, 75%, 88%（中间一天75%未超过80%阈值），有可用设备。
*   **前置条件**: 策略状态为启用，设置CPU阈值为80%，要求连续3天。资源快照数据在连续性上不满足条件。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario3_threshold_not_met.sql`)。
    2.  访问策略管理页面，验证策略配置。
    3.  访问资源监控页面，查看资源使用趋势，确认数据不满足连续性要求。
    4.  等待或手动触发策略评估。
    5.  访问订单管理页面，验证无新订单生成。
    6.  查看策略执行历史，验证是否有记录以及失败原因是否为“failure_threshold_not_met”。
*   **预期结果**: 不生成订单，策略执行历史记录为“failure_threshold_not_met”，历史记录中应包含详细的失败原因。

### 场景4：入池无可用设备 ✅
*   **SQL文件**: `test_data_scenario4_pool_entry_no_devices.sql`
*   **测试目标**: 验证当触发入池条件但无可用设备时，系统仍会生成订单作为告警。
*   **关键特征**: 连续3天CPU使用率超过80%，但无任何状态为 'in_stock' 的设备。
*   **前置条件**: 策略状态为启用，要求N台设备。CPU阈值满足触发条件。所有设备均为非可用状态（例如 'in_pool', 'maintenance', 'offline', 'reserved'）。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario4_pool_entry_no_devices.sql`)。
    2.  访问设备管理页面，确认无可用设备。
    3.  访问策略管理页面，验证策略配置。
    4.  手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了提醒订单。
    6.  查看订单详情，确认设备数量为0，并有设备不足的提醒信息。
    7.  查看邮件通知内容（如果配置），确认包含设备申请或协调处理的提醒。
*   **预期结果**: 生成告警（提醒）订单，订单状态为“待处理”，设备数量为0，订单详情显示“找不到要处理的设备，请自行协调处理”，策略执行历史记录为“order_created_no_devices”。

### 场景5：退池无运行中设备 ✅
*   **SQL文件**: `test_data_scenario5_pool_exit_no_devices.sql`
*   **测试目标**: 验证当触发退池条件但无可退池设备（例如无 'running' 状态设备）时，系统仍会生成订单作为告警。
*   **关键特征**: 连续3天CPU使用率低于30%，但无任何状态为 'running' 或 'in_pool' 的设备可供退池。
*   **前置条件**: 策略状态为启用，要求退N台设备。资源利用率满足退池触发条件。所有设备均为非在池状态（例如 'in_stock', 'maintenance', 'offline', 'reserved'）。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario5_pool_exit_no_devices.sql`)。
    2.  访问设备管理页面，确认无在池设备可供退池。
    3.  访问策略管理页面，验证策略配置。
    4.  手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了提醒订单。
    6.  查看订单详情，确认设备数量为0，并有相应提醒信息。
*   **预期结果**: 生成告警（提醒）订单，订单状态为“待处理”，设备数量为0，订单详情显示相应提示，策略执行历史记录为“order_created_no_devices”。

### 场景6：入池部分设备匹配 ✅
*   **SQL文件**: `test_data_scenario6_pool_entry_partial_devices.sql`
*   **测试目标**: 验证当触发入池条件但只有部分设备可用时，系统生成订单并匹配所有可用的设备。
*   **关键特征**: 连续3天CPU使用率超过80%，策略要求3台设备，但只有2台 'in_stock' 设备可用。
*   **前置条件**: 策略状态为启用，要求N台设备。CPU阈值满足触发条件。可用设备数量少于策略要求的数量（例如，策略要求5台，只有2台可用）。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario6_pool_entry_partial_devices.sql`)。
    2.  访问设备管理页面，确认只有部分设备可用。
    3.  访问策略管理页面，验证策略配置。
    4.  手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了订单。
    6.  查看订单详情，确认订单中只包含了实际可用的设备数量。
*   **预期结果**: 生成订单，订单中包含实际匹配到的2台可用设备，策略执行历史记录为“order_created_partial”，历史记录中应说明部分匹配的情况。

### 场景7：退池部分设备匹配 ✅
*   **SQL文件**: `test_data_scenario7_pool_exit_partial_devices.sql`
*   **测试目标**: 验证当触发退池条件但只有部分设备可退池时，系统生成订单并匹配所有可退的设备。
*   **关键特征**: 连续3天CPU使用率低于30%，策略要求3台设备，但只有2台 'running' 设备可供退池。
*   **前置条件**: 策略状态为启用，要求退N台设备。资源利用率满足退池触发条件。可退池设备数量少于策略要求的数量（例如，策略要求4台，只有2台在池）。
*   **测试步骤**: 
    1.  执行Mock数据SQL (`test_data_scenario7_pool_exit_partial_devices.sql`)。
    2.  访问设备管理页面，确认只有部分设备可退池。
    3.  访问策略管理页面，验证策略配置。
    4.  手动触发策略评估。
    5.  访问订单管理页面，验证是否生成了订单。
    6.  查看订单详情，确认订单中只包含了实际可退的设备数量。
*   **预期结果**: 生成订单，订单中包含实际匹配到的2台可退设备，策略执行历史记录为“order_created_partial”，历史记录中应说明部分匹配的情况。

### 测试数据说明
*   **SQL文件**: 每个场景均有对应的 `.sql` 文件 (例如 `test_data_scenario1_pool_entry.sql`)，用于初始化数据库状态。
*   **数据结构**: SQL文件通常包含：
    *   **基础数据**: `k8s_clusters`, `devices` (不同状态), `query_templates`, `elastic_scaling_strategies`, `strategy_cluster_association`。
    *   **触发数据**: `k8s_cluster_resource_snapshot` (模拟连续几天的CPU/内存使用率)。
    *   **结果验证**: SQL文件中通常也包含用于验证结果的查询语句，例如统计订单、设备、历史记录等。
*   **Mock数据示例 (场景1 - 核心插入)**:
    ```sql
    -- (部分示例，完整脚本见各SQL文件)
    INSERT INTO k8s_clusters (id, cluster_name) VALUES (1, 'production-cluster');
    INSERT INTO devices (id, ci_code, status, cluster_id) VALUES (101, 'DEV001', 'in_stock', 1), (102, 'DEV002', 'in_stock', 1);
    INSERT INTO elastic_scaling_strategies (id, name, threshold_trigger_action, cpu_threshold_value, duration_minutes, device_count, status, entry_query_template_id) 
    VALUES (1, 'CPU High Scale Out', 'pool_entry', 80.0, 3, 2, 'enabled', 1);
    INSERT INTO k8s_cluster_resource_snapshot (cluster_id, max_cpu_usage_ratio, created_at) VALUES 
    (1, 85.0, datetime('now', '-3 days')),
    (1, 90.0, datetime('now', '-2 days')),
    (1, 88.0, datetime('now', '-1 days'));
    ```

## 2. 测试执行指南

### 环境准备
1.  **后端服务**: 启动Go后端服务 (`cd server/portal && go run main.go`)。
2.  **前端服务**: 启动React前端服务 (`cd web/navy-fe && bun start`)。
3.  **数据库**: 确保SQLite数据库 (`./data/navy.db`) 可访问，并且表结构已根据 `create_test_schema.sql` 创建。

### 执行测试
*   **通过脚本 (推荐)**:
    ```bash
    # 进入项目根目录
    # 执行指定场景的测试数据SQL并触发评估 (示例)
    ./docs/elastic_scaling/run_frontend_tests.sh 1 # 测试场景1
    ./docs/elastic_scaling/run_frontend_tests.sh all # 执行所有场景的SQL
    ```
    *注意: `run_frontend_tests.sh` 脚本主要负责加载SQL数据。策略评估可能需要手动触发或等待系统调度。*
*   **手动执行SQL**:
    ```bash
    sqlite3 ./data/navy.db < ./docs/elastic_scaling/test_data_scenario1_pool_entry.sql
    # (对其他场景重复此操作)
    ```
*   **手动触发评估 (API)**:
    ```bash
    # 触发所有策略评估
    curl -X POST http://localhost:8080/api/v1/elastic-scaling/evaluate 
    # 触发特定策略评估 (假设策略ID为1)
    curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/1/evaluate
    ```

### 关键验证点
*   **前端页面**:
    *   **策略管理** (`/elastic-scaling/strategies`): 策略状态是否正确（启用/禁用），配置参数是否与预期一致。
    *   **资源监控** (`/elastic-scaling/dashboard`): 对应集群的资源利用率图表是否能正确反映Mock数据。
    *   **订单管理** (`/elastic-scaling/orders`): 是否按预期生成了订单，订单类型、设备数量、状态是否正确。
    *   **订单详情**: 订单关联的设备信息是否准确。
    *   **策略执行历史** (`/elastic-scaling/strategies/{id}/history`): 是否记录了正确的执行结果 (e.g., `order_created`, `failure_threshold_not_met`, `order_created_no_devices`, `order_created_partial`) 和相关信息。
*   **数据库验证**: 直接查询相关表 (`orders`, `elastic_scaling_order_details`, `order_device`, `strategy_execution_history`) 确认数据是否符合预期。
*   **API响应**: （可选）直接调用相关API获取订单、策略等信息进行验证。
    ```bash
    curl "http://localhost:8081/fe-v1/elastic-scaling/orders/1" | jq .
    curl "http://localhost:8081/fe-v1/elastic-scaling/orders/1/devices" | jq .
    ```

### 注意事项
*   **时间依赖**: SQL脚本中的时间戳（例如 `datetime('now', '-1 days')`）是相对于执行时间的，确保测试环境时间设置正确。
*   **数据清理**: 测试后可能需要清理数据，可以使用 `./docs/elastic_scaling/run_frontend_tests.sh clean` 或手动执行SQL删除语句。
*   **邮件通知**: 如果测试邮件通知功能，需确保邮件服务已配置或检查应用日志中的邮件内容。

## 3. 测试报告模板 (参考)

```markdown
## 弹性伸缩模块测试报告

**测试日期**: YYYY-MM-DD
**测试版本**: vX.Y.Z
**测试人员**: [Your Name]

**环境信息**:
*   后端服务: [状态 OK/Fail]
*   前端服务: [状态 OK/Fail]
*   数据库: [状态 OK/Fail, SQLite]

**测试场景执行结果**:

| 场景 ID | 场景描述                      | SQL 文件                                      | 状态 (Pass/Fail) | 备注/JIRA链接                                     |
| :------ | :---------------------------- | :-------------------------------------------- | :--------------- | :------------------------------------------------ |
| 1       | 入池订单生成                  | `test_data_scenario1_pool_entry.sql`          | Pass             | 订单及设备匹配符合预期                            |
| 2       | 退池订单生成                  | `test_data_scenario2_pool_exit.sql`           | Pass             |                                                   |
| 3       | 阈值未达到                    | `test_data_scenario3_threshold_not_met.sql`   | Pass             | 执行历史为 'failure_threshold_not_met'          |
| 4       | 入池无可用设备              | `test_data_scenario4_pool_entry_no_devices.sql` | Pass             | 生成提醒订单，设备数0                             |
| 5       | 退池无运行中设备            | `test_data_scenario5_pool_exit_no_devices.sql`  | Pass             | 生成提醒订单，设备数0                             |
| 6       | 入池部分设备匹配              | `test_data_scenario6_pool_entry_partial_devices.sql`| Pass             | 订单包含实际可用设备                            |
| 7       | 退池部分设备匹配              | `test_data_scenario7_pool_exit_partial_devices.sql`| Pass             | 订单包含实际可退设备                            |

**总结与问题**:
*   [记录测试过程中发现的主要问题、缺陷或不一致性]
*   [对模块功能或测试流程的改进建议]

```

This document provides a consolidated guide for end-to-end testing of the elastic scaling module. Refer to individual SQL files for complete data setup details when necessary.