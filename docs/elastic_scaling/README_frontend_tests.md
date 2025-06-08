# 弹性伸缩策略评估前端测试文档

## 📁 文件结构

```
docs/elastic_scaling/
├── README_frontend_tests.md                           # 本文件，测试文档总览
├── frontend_test_cases.md                             # 详细测试用例说明
├── quick_test_guide.md                                # 快速测试指南
├── run_frontend_tests.sh                              # 自动化测试执行脚本
├── test_data_scenario1_pool_entry.sql                 # 场景1：入池订单生成数据
├── test_data_scenario2_pool_exit.sql                  # 场景2：退池订单生成数据
├── test_data_scenario3_threshold_not_met.sql          # 场景3：不满足条件数据
├── test_data_scenario4_pool_entry_no_devices.sql      # 场景4：入池无法匹配到设备
├── test_data_scenario5_pool_exit_no_devices.sql       # 场景5：退池无法匹配到设备
├── test_data_scenario6_pool_entry_partial_devices.sql # 场景6：入池只能匹配部分设备
├── test_data_scenario7_pool_exit_partial_devices.sql  # 场景7：退池只能匹配部分设备
├── email_notification_example.html                    # 邮件通知效果预览
├── email_notification_feature.md                      # 邮件通知功能说明
├── elastic_scaling_design.md                          # 弹性伸缩设计文档
└── elastic_scaling_design_updated.md                  # 弹性伸缩设计文档（更新版）
```

## 🎯 测试目标

基于现有的单元测试 `elastic_scaling_evaluation_test.go`，为弹性伸缩策略评估功能提供完整的前端页面测试用例，验证以下核心功能：

1. **策略评估逻辑**：验证不同条件下的策略评估结果
2. **订单生成功能**：验证入池和退池订单的正确生成
3. **前端页面展示**：验证策略管理、资源监控、订单管理页面
4. **数据一致性**：验证前后端数据的一致性和准确性

## 🧪 测试场景

### 场景1：入池订单生成测试
- **文件**：`test_data_scenario1_pool_entry.sql`
- **条件**：CPU使用率连续3天超过80%
- **预期**：生成入池订单，包含可用设备
- **验证点**：策略配置、资源趋势、订单生成、设备分配

### 场景2：退池订单生成测试
- **文件**：`test_data_scenario2_pool_exit.sql`
- **条件**：内存分配率连续2天低于20%
- **预期**：生成退池订单，包含在池设备
- **验证点**：退池逻辑、设备选择、订单类型

### 场景3：不满足条件测试
- **文件**：`test_data_scenario3_threshold_not_met.sql`
- **条件**：阈值要求连续3天，但第2天中断
- **预期**：不生成订单，记录失败原因
- **验证点**：连续性检查、失败处理、历史记录

### 场景4：入池无法匹配到设备测试
- **文件**：`test_data_scenario4_pool_entry_no_devices.sql`
- **条件**：满足CPU阈值，但无可用设备
- **预期**：生成提醒订单，提示申请新设备
- **验证点**：设备匹配逻辑、提醒订单生成、邮件通知内容

### 场景5：退池无法匹配到设备测试
- **文件**：`test_data_scenario5_pool_exit_no_devices.sql`
- **条件**：满足内存阈值，但无在池设备
- **预期**：生成提醒订单，提示协调处理
- **验证点**：在池设备检查、提醒订单生成、邮件通知内容

### 场景6：入池只能匹配部分设备测试
- **文件**：`test_data_scenario6_pool_entry_partial_devices.sql`
- **条件**：满足CPU阈值，需要5台但只有2台可用
- **预期**：生成部分订单，包含2台设备
- **验证点**：部分匹配逻辑、订单生成、设备数量处理

### 场景7：退池只能匹配部分设备测试
- **文件**：`test_data_scenario7_pool_exit_partial_devices.sql`
- **条件**：满足内存阈值，需要4台但只有2台在池
- **预期**：生成部分订单，包含2台设备
- **验证点**：部分匹配逻辑、订单生成、在池设备处理

## 🚀 快速开始

### 1. 使用自动化脚本（推荐）

```bash
# 给脚本添加执行权限（首次使用）
chmod +x docs/elastic_scaling/run_frontend_tests.sh

# 运行特定场景
./docs/elastic_scaling/run_frontend_tests.sh 1    # 场景1
./docs/elastic_scaling/run_frontend_tests.sh 2    # 场景2
./docs/elastic_scaling/run_frontend_tests.sh 3    # 场景3
./docs/elastic_scaling/run_frontend_tests.sh 4    # 场景4
./docs/elastic_scaling/run_frontend_tests.sh 5    # 场景5
./docs/elastic_scaling/run_frontend_tests.sh 6    # 场景6
./docs/elastic_scaling/run_frontend_tests.sh 7    # 场景7

# 运行所有场景
./docs/elastic_scaling/run_frontend_tests.sh all

# 清理测试数据
./docs/elastic_scaling/run_frontend_tests.sh clean
```

### 2. 手动执行SQL

```bash
# 连接数据库
sqlite3 ./data/navy.db

# 执行对应场景的SQL
.read docs/elastic_scaling/test_data_scenario1_pool_entry.sql
.read docs/elastic_scaling/test_data_scenario2_pool_exit.sql
.read docs/elastic_scaling/test_data_scenario3_threshold_not_met.sql
.read docs/elastic_scaling/test_data_scenario4_pool_entry_no_devices.sql
.read docs/elastic_scaling/test_data_scenario5_pool_exit_no_devices.sql
.read docs/elastic_scaling/test_data_scenario6_pool_entry_partial_devices.sql
.read docs/elastic_scaling/test_data_scenario7_pool_exit_partial_devices.sql
```

## 📋 测试流程

### 环境准备
1. 启动后端服务：`cd server/portal && go run main.go`
2. 启动前端服务：`cd web/navy-fe && npm run dev`
3. 确保数据库文件存在：`./data/navy.db`

### 执行测试
1. 运行测试脚本初始化数据
2. 访问前端页面验证功能
3. 手动触发策略评估（可选）
4. 验证结果和数据一致性
5. 清理测试数据

### 验证页面
- **策略管理**：`http://localhost:3000/elastic-scaling/strategies`
- **资源监控**：`http://localhost:3000/elastic-scaling/dashboard`  
- **订单管理**：`http://localhost:3000/elastic-scaling/orders`

## 📊 Mock数据说明

### 数据特点
- **真实性**：模拟真实的集群资源使用情况
- **时序性**：使用相对时间确保测试的时效性
- **完整性**：包含集群、设备、策略、资源快照等完整数据链
- **一致性**：确保外键关系和数据约束正确

### 数据量
- **集群**：每个场景1个集群
- **设备**：每个场景2-3个设备
- **策略**：每个场景1个策略
- **资源快照**：每个场景3-4天的历史数据

## 🔍 验证要点

### 功能验证
- [ ] 策略配置正确显示
- [ ] 资源趋势图准确展示
- [ ] 阈值线和告警状态正确
- [ ] 订单生成逻辑正确
- [ ] 设备分配算法正确
- [ ] 执行历史记录完整

### 界面验证
- [ ] 页面加载正常
- [ ] 数据刷新及时
- [ ] 交互操作流畅
- [ ] 错误提示友好
- [ ] 响应式布局适配

### 数据验证
- [ ] 前后端数据一致
- [ ] 计算结果准确
- [ ] 时间处理正确
- [ ] 状态更新及时

## 🐛 故障排除

### 常见问题
1. **数据库文件不存在**
   - 解决：先启动后端服务创建数据库

2. **SQL执行失败**
   - 解决：检查数据库连接和SQL语法

3. **前端页面无数据**
   - 解决：检查后端API是否正常

4. **策略不触发**
   - 解决：使用手动触发API测试

### 调试方法
```bash
# 检查数据库内容
sqlite3 ./data/navy.db "SELECT * FROM elastic_scaling_strategies;"

# 手动触发策略评估
curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/1/evaluate

# 查看后端日志
tail -f logs/app.log
```

## 📈 扩展测试

### 性能测试
- 大量数据下的页面响应时间
- 并发策略评估的处理能力
- 长时间运行的稳定性

### 边界测试
- 极值数据的处理
- 异常情况的容错
- 网络异常的恢复

### 集成测试
- 与其他模块的交互
- 权限控制的验证
- 数据迁移的兼容性

## 📝 测试报告

建议使用 `quick_test_guide.md` 中提供的测试报告模板记录测试结果，包括：
- 测试环境信息
- 各场景执行结果
- 发现的问题
- 改进建议

## 🔗 相关资源

- **单元测试参考**：`server/portal/internal/service/elastic_scaling_evaluation_test.go`
- **邮件功能文档**：`docs/email_notification_feature.md`
- **API文档**：查看后端服务的Swagger文档
- **前端组件**：查看前端项目中的相关组件实现

## 📞 技术支持

如果在测试过程中遇到问题，可以：
1. 查看详细的测试用例文档
2. 参考单元测试的实现逻辑
3. 检查后端日志和前端控制台
4. 联系开发团队获取支持

---

**注意**：测试完成后请及时清理测试数据，避免影响其他功能的测试和开发。
