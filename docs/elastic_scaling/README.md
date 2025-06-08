# 弹性伸缩系统测试文档

## 📁 目录结构

本目录包含弹性伸缩系统的所有测试相关文档和脚本。

```
docs/elastic_scaling/
├── README.md                                           # 本文件，总览文档
├── README_frontend_tests.md                           # 前端测试文档总览
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
├── email_notification_example.html                    # 邮件通知效果预览（正常情况）
├── email_notification_no_devices_example.html         # 邮件通知效果预览（无设备情况）
├── email_notification_feature.md                      # 邮件通知功能说明
├── elastic_scaling_design.md                          # 弹性伸缩设计文档
└── elastic_scaling_design_updated.md                  # 弹性伸缩设计文档（更新版）
```

## 🎯 测试覆盖范围

### 核心功能测试
- ✅ **策略评估逻辑**：验证不同条件下的策略评估结果
- ✅ **订单生成功能**：验证入池和退池订单的正确生成
- ✅ **设备匹配算法**：验证设备匹配的各种场景
- ✅ **邮件通知功能**：验证邮件正文生成和格式

### 边界条件测试
- ✅ **阈值连续性检查**：验证阈值未连续满足的处理
- ✅ **设备不足处理**：验证无可用设备时的错误处理
- ✅ **部分匹配处理**：验证只能匹配部分设备时的逻辑
- ✅ **失败原因记录**：验证各种失败场景的原因记录

### 前端页面测试
- ✅ **策略管理页面**：策略配置、状态管理、执行历史
- ✅ **资源监控页面**：资源趋势图、阈值显示、告警状态
- ✅ **订单管理页面**：订单列表、详情查看、状态更新
- ✅ **设备管理页面**：设备状态、匹配结果、分配情况

## 🚀 快速开始

### 1. 环境准备
```bash
# 启动后端服务
cd server/portal && go run main.go

# 启动前端服务（新终端）
cd web/navy-fe && npm run dev
```

### 2. 运行测试
```bash
# 基础场景测试
./docs/elastic_scaling/run_frontend_tests.sh 1    # 入池订单生成
./docs/elastic_scaling/run_frontend_tests.sh 2    # 退池订单生成
./docs/elastic_scaling/run_frontend_tests.sh 3    # 不满足条件

# 边界场景测试
./docs/elastic_scaling/run_frontend_tests.sh 4    # 入池无法匹配到设备
./docs/elastic_scaling/run_frontend_tests.sh 5    # 退池无法匹配到设备
./docs/elastic_scaling/run_frontend_tests.sh 6    # 入池只能匹配部分设备
./docs/elastic_scaling/run_frontend_tests.sh 7    # 退池只能匹配部分设备

# 运行所有场景
./docs/elastic_scaling/run_frontend_tests.sh all
```

## 📋 测试场景矩阵

| 场景 | 类型 | 阈值满足 | 设备情况 | 预期结果 | 验证重点 |
|------|------|----------|----------|----------|----------|
| 1 | 入池 | ✅ 连续3天 | 充足可用设备 | 生成完整订单 | 正常流程 |
| 2 | 退池 | ✅ 连续2天 | 充足在池设备 | 生成完整订单 | 正常流程 |
| 3 | 入池 | ❌ 第2天中断 | 充足可用设备 | 不生成订单 | 连续性检查 |
| 4 | 入池 | ✅ 连续3天 | 无可用设备 | 生成提醒订单 | 设备申请提醒 |
| 5 | 退池 | ✅ 连续2天 | 无在池设备 | 生成提醒订单 | 协调处理提醒 |
| 6 | 入池 | ✅ 连续3天 | 部分可用设备 | 生成部分订单 | 部分匹配处理 |
| 7 | 退池 | ✅ 连续2天 | 部分在池设备 | 生成部分订单 | 部分匹配处理 |

## 📊 测试数据特点

### 数据设计原则
- **真实性**：模拟真实的集群资源使用情况
- **时序性**：使用相对时间确保测试的时效性
- **完整性**：包含集群、设备、策略、资源快照等完整数据链
- **一致性**：确保外键关系和数据约束正确

### 数据量统计
- **集群数量**：7个（每个场景1个）
- **设备数量**：30台（每个场景2-6台）
- **策略数量**：7个（每个场景1个）
- **资源快照**：21条（每个场景3-4天历史数据）

## 🔍 验证检查清单

### 功能验证
- [ ] 策略配置正确显示
- [ ] 资源趋势图准确展示
- [ ] 阈值线和告警状态正确
- [ ] 订单生成逻辑正确
- [ ] 设备分配算法正确
- [ ] 执行历史记录完整
- [ ] 邮件通知内容正确

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

## 📖 文档说明

### 核心文档
- **[README_frontend_tests.md](./README_frontend_tests.md)**：前端测试详细说明
- **[frontend_test_cases.md](./frontend_test_cases.md)**：完整测试用例文档
- **[quick_test_guide.md](./quick_test_guide.md)**：快速测试指南
- **[user_manual.md](./user_manual.md)**：弹性伸缩系统用户手册

### 功能文档
- **[email_notification_feature.md](./email_notification_feature.md)**：邮件通知功能说明
- **[elastic_scaling_design.md](./elastic_scaling_design.md)**：系统设计文档

### 工具脚本
- **[run_frontend_tests.sh](./run_frontend_tests.sh)**：自动化测试执行脚本

## 🛠️ 工具使用

### 自动化脚本
```bash
# 查看帮助
./docs/elastic_scaling/run_frontend_tests.sh

# 运行特定场景
./docs/elastic_scaling/run_frontend_tests.sh 1

# 清理测试数据
./docs/elastic_scaling/run_frontend_tests.sh clean
```

### 手动执行
```bash
# 连接数据库
sqlite3 ./data/navy.db

# 执行SQL脚本
.read docs/elastic_scaling/test_data_scenario1_pool_entry.sql
```

### API测试
```bash
# 手动触发策略评估
curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/1/evaluate

# 查看策略列表
curl http://localhost:8080/api/v1/elastic-scaling/strategies

# 查看订单列表
curl http://localhost:8080/api/v1/elastic-scaling/orders
```

## 🐛 故障排除

### 常见问题
1. **数据库文件不存在**：先启动后端服务创建数据库
2. **SQL执行失败**：检查数据库连接和SQL语法
3. **前端页面无数据**：检查后端API是否正常
4. **策略不触发**：使用手动触发API测试

### 调试方法
```bash
# 检查数据库内容
sqlite3 ./data/navy.db "SELECT * FROM elastic_scaling_strategies;"

# 查看后端日志
tail -f logs/app.log

# 检查前端控制台
# 打开浏览器开发者工具查看Network和Console
```

## 📈 扩展测试

### 性能测试
- 大量数据下的页面响应时间
- 并发策略评估的处理能力
- 长时间运行的稳定性

### 集成测试
- 与其他模块的交互
- 权限控制的验证
- 数据迁移的兼容性

### 压力测试
- 高频策略评估的系统负载
- 大量订单生成的处理能力
- 并发用户访问的响应性能

## 📞 技术支持

如果在测试过程中遇到问题，可以：
1. 查看详细的测试用例文档
2. 参考单元测试的实现逻辑
3. 检查后端日志和前端控制台
4. 联系开发团队获取支持

---

**注意**：测试完成后请及时清理测试数据，避免影响其他功能的测试和开发。
