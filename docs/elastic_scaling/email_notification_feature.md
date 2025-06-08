# 弹性伸缩订单邮件通知功能

## 功能概述

在弹性伸缩订单生成时，系统会自动生成美观的HTML邮件正文，用于通知当周值班人员处理集群的入池和退池变更工作。

## 功能特性

### 1. 自动邮件生成
- 在 `CreateOrder` 方法中自动触发邮件正文生成
- 支持入池（pool_entry）和退池（pool_exit）操作类型
- 支持维护申请（maintenance_request）和维护解除（maintenance_uncordon）操作类型

### 2. 邮件内容包含
- **订单基本信息**：订单号、创建时间、创建人
- **变更详情**：目标集群、变更类型、设备数量
- **设备信息**：CI编码、IP地址、架构、CPU、内存、状态
- **设备不足提醒**：当无法匹配到设备时的特殊提醒和处理指引
- **操作指引**：详细的处理步骤说明
- **重要提醒**：时间要求和注意事项
- **联系信息**：技术支持联系方式

### 3. 美观的样式设计
- 响应式设计，支持移动端查看
- 渐变色背景，视觉效果美观
- 清晰的信息层次结构
- 表格形式展示设备信息
- 不同操作类型使用不同的颜色主题

## 技术实现

### 核心方法

#### `generateOrderNotificationEmail`
```go
func (s *ElasticScalingService) generateOrderNotificationEmail(orderID int64, dto OrderDTO) string
```
- 主要的邮件生成入口方法
- 获取集群名称和设备信息
- 调用HTML构建方法生成完整邮件

#### `buildEmailHTML`
```go
func (s *ElasticScalingService) buildEmailHTML(subject, actionName, clusterName string, dto OrderDTO, devices []DeviceDTO) string
```
- 构建完整的HTML邮件正文
- 包含头部、正文、设备表格、操作指引等所有部分
- 根据操作类型动态调整颜色和图标
- 当无法匹配到设备时，生成特殊的提醒邮件内容

#### `getActionName`
```go
func (s *ElasticScalingService) getActionName(actionType string) string
```
- 将英文操作类型转换为中文显示名称
- 支持的操作类型：
  - `pool_entry` → "入池"
  - `pool_exit` → "退池"
  - `maintenance_request` → "维护申请"
  - `maintenance_uncordon` → "维护解除"

#### `getDeviceInfoForEmail`
```go
func (s *ElasticScalingService) getDeviceInfoForEmail(deviceIDs []int64) []DeviceDTO
```
- 根据设备ID列表获取设备详细信息
- 用于在邮件中展示设备配置

### 集成位置

邮件生成功能已集成到 `CreateOrder` 方法中：

```go
// 生成邮件正文通知值班人员
emailContent := s.generateOrderNotificationEmail(orderID, dto)
s.logger.Info("Generated order notification email", 
    zap.Int64("orderID", orderID),
    zap.String("emailContent", emailContent))

// TODO: 实现邮件发送功能
// 这里需要用户自定义实现邮件发送逻辑
// 可以集成企业邮件系统、钉钉、企业微信等通知渠道
```

## 邮件发送集成（TODO）

当前实现只生成邮件正文，实际的邮件发送需要用户根据企业环境自定义实现。

### 建议的集成方式

1. **企业邮件系统**
   ```go
   err = s.sendEmail(emailContent, getOnDutyPersons())
   if err != nil {
       s.logger.Error("Failed to send notification email", zap.Error(err))
   }
   ```

2. **钉钉机器人**
   ```go
   err = s.sendDingTalkMessage(emailContent, getDingTalkWebhook())
   ```

3. **企业微信**
   ```go
   err = s.sendWeChatWorkMessage(emailContent, getWeChatWorkWebhook())
   ```

4. **短信通知**
   ```go
   err = s.sendSMSNotification(generateSMSContent(dto), getOnDutyPhones())
   ```

### 值班人员获取

建议实现 `getOnDutyPersons()` 方法来获取当前值班人员信息：

```go
func (s *ElasticScalingService) getOnDutyPersons() []string {
    // 从值班表或配置中获取当前值班人员
    // 可以根据时间、日期等条件动态获取
    return []string{"duty1@company.com", "duty2@company.com"}
}
```

## 邮件样式预览

可以查看以下文件来预览不同场景的邮件效果：
- `email_notification_example.html` - 正常入池订单邮件效果
- `email_notification_no_devices_example.html` - 无可用设备时的提醒邮件效果

## 测试

已提供完整的单元测试：

```bash
# 运行邮件生成相关测试
go test ./internal/service -run "Email|ActionName" -v

# 运行特定测试
go test ./internal/service -run TestGenerateOrderNotificationEmail -v
go test ./internal/service -run TestGetActionName -v
go test ./internal/service -run TestEmailHTMLStructure -v
```

## 配置建议

### 邮件主题模板
```go
const emailSubjectTemplate = "【弹性伸缩】%s变更通知 - 订单号：%s"
```

### 操作类型映射
```go
const (
    actionTypePoolEntry  = "pool_entry"
    actionTypePoolExit   = "pool_exit"
    actionNamePoolEntry  = "入池"
    actionNamePoolExit   = "退池"
)
```

### 颜色主题
- 入池操作：绿色主题 (`#52c41a`)
- 退池操作：橙色主题 (`#ff7a45`)
- 设备不足：警告橙色主题 (`#ff7a45`)
- 默认操作：蓝色主题 (`#1890ff`)

## 扩展建议

1. **多语言支持**：根据用户偏好生成中英文邮件
2. **模板配置**：支持自定义邮件模板
3. **附件支持**：添加操作手册或相关文档
4. **邮件追踪**：记录邮件发送状态和阅读状态
5. **批量通知**：支持同时发送给多个值班人员
6. **紧急通知**：对于重要变更支持短信+邮件双重通知

## 注意事项

1. 邮件内容包含敏感的集群和设备信息，请确保邮件传输安全
2. 建议定期清理邮件日志，避免占用过多存储空间
3. 在生产环境中使用时，请确保邮件服务的高可用性
4. 建议设置邮件发送失败的重试机制和告警
