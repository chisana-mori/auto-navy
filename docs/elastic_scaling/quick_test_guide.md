# 弹性伸缩策略评估前端测试快速指南

## 🚀 快速开始

### 1. 环境准备

```bash
# 启动后端服务
cd server/portal
go run main.go

# 启动前端服务（新终端）
cd web/navy-fe
npm run dev
```

### 2. 运行测试

```bash
# 在项目根目录执行
./docs/elastic_scaling/run_frontend_tests.sh 1    # 场景1：入池订单生成
./docs/elastic_scaling/run_frontend_tests.sh 2    # 场景2：退池订单生成
./docs/elastic_scaling/run_frontend_tests.sh 3    # 场景3：不满足条件
./docs/elastic_scaling/run_frontend_tests.sh 4    # 场景4：入池无法匹配到设备
./docs/elastic_scaling/run_frontend_tests.sh 5    # 场景5：退池无法匹配到设备
./docs/elastic_scaling/run_frontend_tests.sh 6    # 场景6：入池只能匹配部分设备
./docs/elastic_scaling/run_frontend_tests.sh 7    # 场景7：退池只能匹配部分设备
./docs/elastic_scaling/run_frontend_tests.sh all  # 运行所有场景
```

### 3. 手动执行SQL（可选）

如果脚本无法执行，可以手动运行SQL：

```bash
# 连接数据库
sqlite3 ./data/navy.db

# 执行对应场景的SQL文件
.read docs/elastic_scaling/test_data_scenario1_pool_entry.sql
.read docs/elastic_scaling/test_data_scenario2_pool_exit.sql
.read docs/elastic_scaling/test_data_scenario3_threshold_not_met.sql
.read docs/elastic_scaling/test_data_scenario4_pool_entry_no_devices.sql
.read docs/elastic_scaling/test_data_scenario5_pool_exit_no_devices.sql
.read docs/elastic_scaling/test_data_scenario6_pool_entry_partial_devices.sql
.read docs/elastic_scaling/test_data_scenario7_pool_exit_partial_devices.sql
```

## 📋 测试场景概览

### 场景1：入池订单生成 ✅
- **条件**：CPU使用率连续3天超过80%
- **预期**：生成入池订单，包含可用设备
- **集群**：production-cluster
- **验证页面**：策略管理 → 资源监控 → 订单管理

### 场景2：退池订单生成 ⬇️
- **条件**：内存分配率连续2天低于20%
- **预期**：生成退池订单，包含在池设备
- **集群**：staging-cluster
- **验证页面**：策略管理 → 资源监控 → 订单管理

### 场景3：不满足条件 ❌
- **条件**：CPU阈值要求连续3天，但第2天中断
- **预期**：不生成订单，执行历史显示失败原因
- **集群**：test-cluster
- **验证页面**：策略管理 → 策略执行历史

### 场景4：入池无法匹配到设备 📢
- **条件**：满足CPU阈值，但无可用设备
- **预期**：生成提醒订单，提示申请新设备
- **集群**：no-devices-cluster
- **验证页面**：设备管理 → 订单管理 → 邮件通知

### 场景5：退池无法匹配到设备 📢
- **条件**：满足内存阈值，但无在池设备
- **预期**：生成提醒订单，提示协调处理
- **集群**：no-pool-devices-cluster
- **验证页面**：设备管理 → 订单管理 → 邮件通知

### 场景6：入池只能匹配部分设备 ⚠️
- **条件**：满足CPU阈值，需要5台但只有2台可用
- **预期**：生成部分订单，包含2台设备
- **集群**：partial-devices-cluster
- **验证页面**：策略管理 → 订单管理 → 设备详情

### 场景7：退池只能匹配部分设备 ⚠️
- **条件**：满足内存阈值，需要4台但只有2台在池
- **预期**：生成部分订单，包含2台设备
- **集群**：partial-pool-devices-cluster
- **验证页面**：策略管理 → 订单管理 → 设备详情

## 🔍 关键验证点

### 前端页面验证

1. **策略管理页面** (`/elastic-scaling/strategies`)
   - [ ] 策略状态显示正确
   - [ ] 策略配置参数正确
   - [ ] 策略执行历史可查看

2. **资源监控页面** (`/elastic-scaling/dashboard`)
   - [ ] 集群选择正确
   - [ ] 资源趋势图显示正确
   - [ ] 阈值线标识清晰

3. **订单管理页面** (`/elastic-scaling/orders`)
   - [ ] 订单生成正确（场景1、2）
   - [ ] 无订单生成（场景3）
   - [ ] 订单详情信息完整

### API验证

```bash
# 手动触发策略评估
curl -X POST http://localhost:8080/api/v1/elastic-scaling/strategies/1/evaluate

# 查看策略列表
curl http://localhost:8080/api/v1/elastic-scaling/strategies

# 查看订单列表
curl http://localhost:8080/api/v1/elastic-scaling/orders
```

## 📊 测试数据说明

### 场景1数据特点
- 3个可用设备（in_stock状态）
- 连续3天CPU使用率：85% → 90% → 88%
- 策略阈值：80%，持续3天
- 预期结果：生成入池订单

### 场景2数据特点  
- 3个在池设备（in_pool状态）
- 连续2天内存分配率：15% → 18%
- 策略阈值：20%，持续2天
- 预期结果：生成退池订单

### 场景3数据特点
- 4天数据：85% → 90% → 70% → 88%
- 第2天低于80%阈值，中断连续性
- 策略要求：连续3天超过80%
- 预期结果：不生成订单

### 场景4数据特点
- 4个设备，全部为非可用状态（in_pool、maintenance、offline、reserved）
- 连续3天CPU使用率：85% → 90% → 88%
- 策略阈值：80%，持续3天，要求2台设备
- 预期结果：生成提醒订单，设备数量为0

### 场景5数据特点
- 4个设备，全部为非在池状态（in_stock、maintenance、offline、reserved）
- 连续2天内存分配率：15% → 18%
- 策略阈值：20%，持续2天，要求1台设备
- 预期结果：生成提醒订单，设备数量为0

### 场景6数据特点
- 6个设备，只有2台可用（in_stock），其余为不可用状态
- 连续3天CPU使用率：85% → 90% → 88%
- 策略阈值：80%，持续3天，要求5台设备
- 预期结果：生成部分订单，包含2台设备

### 场景7数据特点
- 6个设备，只有2台在池（in_pool），其余为非在池状态
- 连续2天内存分配率：15% → 18%
- 策略阈值：20%，持续2天，要求4台设备
- 预期结果：生成部分订单，包含2台设备

## 🧹 清理测试数据

```bash
# 使用脚本清理
./docs/run_frontend_tests.sh clean

# 或手动清理
sqlite3 ./data/navy.db "DELETE FROM strategy_execution_history; DELETE FROM orders; DELETE FROM resource_snapshots; DELETE FROM elastic_scaling_strategies; DELETE FROM devices; DELETE FROM k8s_clusters;"
```

## 🐛 常见问题

### 1. 数据库文件不存在
**解决方案**：先启动后端服务，系统会自动创建数据库文件

### 2. 前端页面无数据显示
**解决方案**：检查后端服务是否正常运行，API是否可访问

### 3. 策略评估不触发
**解决方案**：使用手动触发API或检查策略状态是否为启用

### 4. 订单未生成
**解决方案**：检查策略执行历史，查看失败原因

### 5. 权限问题
**解决方案**：确保测试用户具有相应的页面访问权限

## 📝 测试报告模板

```markdown
## 测试执行报告

### 测试环境
- 后端服务：✅ 正常
- 前端服务：✅ 正常  
- 数据库：✅ 连接正常

### 场景1：入池订单生成
- [ ] 策略配置正确
- [ ] 资源数据显示正确
- [ ] 订单生成成功
- [ ] 设备分配正确
- [ ] 邮件通知生成

### 场景2：退池订单生成  
- [ ] 策略配置正确
- [ ] 资源数据显示正确
- [ ] 订单生成成功
- [ ] 设备选择正确

### 场景3：不满足条件
- [ ] 策略配置正确
- [ ] 资源数据显示正确
- [ ] 无订单生成
- [ ] 执行历史记录失败原因

### 场景4：入池无法匹配到设备
- [ ] 策略配置正确
- [ ] 无可用设备
- [ ] 生成提醒订单（设备数量为0）
- [ ] 订单详情显示协调处理提醒
- [ ] 邮件通知包含设备申请提醒

### 场景5：退池无法匹配到设备
- [ ] 策略配置正确
- [ ] 无在池设备
- [ ] 生成提醒订单（设备数量为0）
- [ ] 订单详情显示协调处理提醒
- [ ] 邮件通知包含协调处理提醒

### 场景6：入池只能匹配部分设备
- [ ] 策略配置正确
- [ ] 部分可用设备
- [ ] 生成部分订单
- [ ] 订单包含实际匹配的设备
- [ ] 执行历史记录部分匹配情况

### 场景7：退池只能匹配部分设备
- [ ] 策略配置正确
- [ ] 部分在池设备
- [ ] 生成部分订单
- [ ] 订单包含实际匹配的设备
- [ ] 执行历史记录部分匹配情况

### 发现的问题
1. 
2. 
3. 

### 改进建议
1.
2. 
3.
```

## 🔗 相关文档

- [详细测试用例](./frontend_test_cases.md)
- [邮件通知功能说明](./email_notification_feature.md)
- [单元测试参考](../server/portal/internal/service/elastic_scaling_evaluation_test.go)
