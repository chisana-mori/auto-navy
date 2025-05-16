# K8s 集群资源弹性伸缩管理系统 - 需求文档

## 1. 概述

本文档详细描述了 K8s 集群资源弹性伸缩管理系统的需求，基于 `scaling_management_demo.html` 的原型设计。该系统旨在提供一个可视化的工作台，用于监控集群资源使用情况、管理弹性伸缩策略，并处理由此产生的伸缩订单。系统将支持自动化监控、策略执行、订单处理以及提供必要的运维管理界面。

## 2. 通用布局与导航

### 2.1. 主应用布局
系统应采用单页应用 (SPA) 布局，包含以下主要区域：
-   **侧边导航栏 (Sider)**: 位于左侧，可收起/展开。
-   **顶部应用栏 (Header)**: 位于页面顶部，固定。
-   **主内容区 (Content)**: 显示当前视图的主要内容。
-   **页脚 (Footer)**: 位于页面底部。

### 2.2. 侧边导航栏 (Sider)
-   **可伸缩性**: 用户可以点击按钮展开或收起侧边栏。
    -   收起状态下，仅显示图标。
    -   展开状态下，显示图标和文字。
-   **Logo/标题**:
    -   展开状态下，显示系统图标 (CloudServerOutlined) 和标题 "K8s资源弹性管理"。
    -   收起状态下，仅显示系统图标。
-   **菜单项**:
    -   **工作台**:
        -   图标: `DashboardOutlined`
        -   文本: "工作台"
        -   默认选中项。
    -   (未来可扩展其他菜单，如：集群管理, 设备管理, 值班管理等)

### 2.3. 顶部应用栏 (Header)
-   **侧边栏控制**: 包含一个按钮，用于切换侧边栏的展开/收起状态 (`MenuUnfoldOutlined` / `MenuFoldOutlined`)。
-   **页面标题**: 显示当前模块标题，例如 "资源弹性管理工作台"。
-   **用户操作区**: 位于右侧，包含：
    -   **通知**:
        -   图标: `BellOutlined`
        -   功能: 显示未读通知数量的徽章 (Badge)。
        -   交互: 点击应能查看通知列表 (通知内容和类型见 `notification_log`)。
    -   **用户头像/名称下拉菜单**:
        -   显示用户头像 (`Avatar` with `UserOutlined`) 和用户名称 (例如 "管理员")。
        -   下拉箭头 (`DownOutlined`).
        -   点击触发下拉菜单，包含以下选项：
            -   个人信息 (图标: `UserOutlined`)
            -   系统设置 (图标: `SettingOutlined`) (可链接到系统配置管理页面)
            -   分割线
            -   退出登录 (图标: `LogoutOutlined`)

### 2.4. 页脚 (Footer)
-   显示版权信息，例如 "K8s集群资源弹性伸缩管理系统 ©{CurrentYear}"。
-   文本居中显示。

## 3. 工作台视图 (Dashboard)

工作台是用户登录后看到的主界面，包含多个信息模块。数据应通过后端API获取 (如 `GET /api/dashboard/overview-stats`, `GET /api/resources/allocation-trend`)。

### 3.1. 统计卡片区 (Stats Cards)
以卡片形式展示关键摘要信息，网格布局。数据来源于概览统计API。
-   **卡片 1: 策略巡检进度/状态**
    -   标题: "策略总数 / 今日已触发" (或类似)
    -   值: 例如 "20 / 8"
    -   可视化: 可用进度条表示已启用策略占比。
-   **卡片 2: 集群健康状态**
    -   标题: "集群总数 / 异常集群"
    -   值: 例如 "15 / 3"
    -   附加信息: 如果存在异常，显示警告信息。
-   **卡片 3: 待处理任务**
    -   标题: "待处理伸缩订单"
    -   值: 待处理订单数量 (来自 `elastic_scaling_order` 表 `status='pending'`)。
    -   附加信息: 任务描述。
-   **卡片 4: 设备资源池状态**
    -   标题: "设备总数 / 可用设备 / 池内设备"
    -   值: 例如 "200 / 150 / 30"

### 3.2. 待处理弹性伸缩订单区 (Pending Orders)
集中展示当前状态为"待处理"的弹性伸缩订单。通过订单管理API获取 (`GET /api/orders/list` with `status=pending`)。
-   **区域标题**: "待处理弹性伸缩订单"。
-   **筛选与操作**:
    -   **订单类型筛选**: 下拉选择框，选项包括 "入池" (`pool_entry`)、"退池" (`pool_exit`)。可清空。
    -   **集群筛选**: (若适用) 下拉选择框，选择特定集群。
    -   **搜索按钮**: 图标 `SearchOutlined`。可按订单号、策略名称搜索。
-   **订单列表**:
    -   若无待处理订单，显示空状态提示 ("暂无待处理订单")。
    -   订单以卡片形式展示，网格布局。
    -   **订单卡片内容**:
        -   **头部**:
            -   图标: `CloudUploadOutlined` (入池) 或 `CloudDownloadOutlined` (退池)，颜色区分。
            -   标题: "订单 #{OrderNumber} - {关联策略名称}"。
            -   状态标签: `Tag` 显示订单状态 (如 "待处理")，颜色区分。
        -   **主体**:
            -   元数据:
                -   类型: `action_type` ("入池" / "退池")。
                -   关联集群: `cluster_id` 对应的集群名称。
                -   触发时间: `created_at`。
                -   请求设备数: `device_count`。
            -   **资源利用率信息 (此部分动态获取或从快照获取)**:
                -   显示订单关联集群当前的CPU/内存使用率概览，辅助决策。
                -   提示信息 (`Alert`): 根据订单类型和当前资源状态显示相关提示。
        -   **底部**:
            -   **查看详情按钮**: 图标 `EyeOutlined`。点击打开订单详情抽屉。
            -   **执行/审批按钮**: (根据订单状态和审批流程配置)
                -   例如 "批准", "执行", "拒绝"。
                -   交互: 点击后调用相应API (如 `PUT /api/orders/{id}/approve`)。

### 3.3. 资源用量趋势区 (Resource Usage Trends)
展示集群的 CPU 和内存使用率历史趋势图。数据通过 `GET /api/resources/allocation-trend` 或 `GET /api/clusters/{id}/details` (含历史) 获取。
-   **区域标题**: "资源用量趋势"。
-   **筛选与操作**:
    -   **集群选择**: 下拉框，选择目标 K8s 集群。
    -   **时间范围选择**: 下拉框，选项包括 "24小时", "7天", "30天"。
    -   **资源类型选择**: (若API支持) CPU/Memory。
    -   **刷新按钮**: 图标 `ReloadOutlined`。
-   **图表展示**:
    -   并排展示两个 ECharts 图表 (CPU, Memory)。
    -   X轴: 时间点。 Y轴: 使用率/分配率百分比。
    -   数据系列: 折线图 (smooth)，显示使用率、分配率。
    -   标记线: （可选）显示策略中相关的阈值。
    -   图表应能响应窗口大小变化，并有 Tooltip。

### 3.4. 监控策略管理区 (Strategy Management)
管理弹性伸缩的监控策略。通过策略管理API (`GET /api/strategies/list`, `POST /api/strategies/create`等)交互。
-   **区域标题**: "监控策略管理"。
-   **操作**:
    -   **新建策略按钮**: 文本 "新建策略"，图标 `PlusOutlined`。点击打开新建策略模态框。
-   **筛选与列表操作**:
    -   **策略名称搜索**: 输入框。
    -   **策略状态筛选**: 下拉框，选项 "启用" (`enabled`), "禁用" (`disabled`)。
    -   **触发动作筛选**: 下拉框，选项 "入池" (`pool_entry`), "退池" (`pool_exit`)。
    -   **集群筛选**: (若适用) 多选或单选关联的集群。
    -   **重置筛选按钮**: 图标 `ReloadOutlined`。
-   **策略列表 (Table)**:
    -   **列定义**:
        -   **策略名称**: `name`。可点击打开策略详情抽屉。
        -   **描述**: `description`。
        -   **关联集群**: 列出关联的集群名称 (来自 `strategy_cluster_association`)。
        -   **阈值条件**: `threshold_trigger_action`, `cpu_threshold_value` (% `cpu_threshold_type`), `memory_threshold_value` (% `memory_threshold_type`), `condition_logic`。
        -   **动作**: `threshold_trigger_action` (入池/退池)。
        -   **配置**: `device_count`, `duration_minutes`, `cooldown_minutes`。
        -   **状态**: `status` ("启用"/"禁用")。
        -   **操作**: 编辑, 启用/禁用 (`PUT /api/strategies/{id}/status`), 删除 (`DELETE /api/strategies/{id}`), 查看执行历史 (`GET /api/strategies/{id}/execution-history`)。
    -   **分页**: 支持。

### 3.5. 全部订单与统计区 (Historical Orders & Statistics)
展示历史订单记录及统计信息。通过订单列表API (`GET /api/orders/list`) 及分析类API (`GET /api/analytics/order-stats`) 获取数据。
-   **区域标题**: "全部订单与统计"。
-   **筛选**:
    -   **时间范围选择**: "最近7天", "最近30天", "最近90天"。
    -   **集群筛选**: 可选。
    -   **订单状态/类型筛选**: 可选。
-   **内容布局**: 分为左右两栏。
    -   **左栏: 订单状态饼图**
        -   ECharts 饼图，显示各状态 (`pending`, `processing`, `completed`, `failed`, `cancelled`) 订单占比。
    -   **右栏: 订单列表与摘要**
        -   **订单状态摘要**: 各状态订单数。
        -   **订单列表 (Tabs)**: 按状态 ("处理中", "已完成", "全部"等) 切换。
            -   内容: 网格布局显示订单卡片。
            -   **历史订单卡片**: 类似3.2节订单卡片，但不含资源利用率动态信息。显示关键信息如订单号、类型、状态、集群、设备数、创建/完成时间。
            -   底部: "查看详情"按钮。

## 4. 模态框与抽屉 (Modals & Drawers)

### 4.1. 新建/编辑策略模态框
通过 `POST /api/strategies/create` 或 `PUT /api/strategies/{id}` 交互。
-   **标题**: "新建监控策略" / "编辑监控策略"。
-   **表单 (`Form`)**:
    -   **策略名称**: `name` (Input)。必填。
    -   **策略描述**: `description` (TextArea)。
    -   **关联集群**: (`strategy_cluster_association`) 多选下拉框/穿梭框。选择策略应用的集群。必填。
    -   **阈值触发动作**: `threshold_trigger_action` (Radio.Group: "入池", "退池")。必填。
    -   **CPU 监控**:
        -   **CPU 阈值 (%)**: `cpu_threshold_value` (NumberInput 0-100)。
        -   **CPU 阈值类型**: `cpu_threshold_type` (Select: "使用率", "分配率")。CPU阈值填写时必选。
    -   **内存监控**:
        -   **内存阈值 (%)**: `memory_threshold_value` (NumberInput 0-100)。
        -   **内存阈值类型**: `memory_threshold_type` (Select: "使用率", "分配率")。内存阈值填写时必选。
    -   **条件组合逻辑**: `condition_logic` (Radio.Group: "同时满足 AND", "满足其一 OR")。CPU和内存阈值都设置时必填。
    -   **满足时长 (分钟)**: `duration_minutes` (NumberInput)。
    -   **冷却时间 (分钟)**: `cooldown_minutes` (NumberInput)。
    -   **设备数量**: `device_count` (NumberInput)。必填。
    -   **节点选择器 (JSON)**: `node_selector` (TextArea)。可选。
    -   **初始状态**: `status` (Radio.Group: "启用", "禁用")。默认为 "启用"。
-   **操作按钮**: 保存, 取消。

### 4.2. 订单详情抽屉
通过 `GET /api/orders/{id}` 和 `GET /api/orders/{id}/devices` 获取数据。
-   **标题**: "订单详情 #{OrderNumber}"。
-   **内容**:
    -   **订单基本信息 (`Descriptions` 组件)**:
        -   订单号 (`order_number`), 类型 (`action_type`), 状态 (`status`), 关联集群 (`cluster_id`->name), 关联策略 (`strategy_id`->name), 请求设备数 (`device_count`), 创建信息 (`created_by`, `created_at`), 审批/执行信息 (`approver`, `executor`, `execution_time`), 完成时间 (`completion_time`), 失败原因 (`failure_reason`)。
    -   **涉及设备列表区**:
        -   标题: "涉及设备列表"。
        -   列表显示 `order_device`关联的设备及其在订单中的状态。
        -   设备项: 设备名称, IP, CPU, Memory, 在此订单中的状态。

### 4.3. 策略详情抽屉
通过 `GET /api/strategies/{id}` 获取数据。
-   **标题**: "策略详情: {StrategyName}"。
-   **内容**:
    -   **策略基本信息 (`Descriptions` 组件)**:
        -   显示策略表 (`elastic_scaling_strategy`) 中的所有字段信息，如名称、描述、关联集群、各项阈值和配置、状态、创建/更新信息。
    -   **策略执行历史**: (通过 `GET /api/strategies/{id}/execution-history` 获取)
        -   列表显示此策略的执行记录，包括执行时间、触发值、结果、关联订单号等。
    -   **相关订单区**: (通过 `GET /api/orders/list` with `strategy_id` filter 获取)
        -   列表显示与此策略关联的历史订单卡片。

## 5. 数据实体
以下为系统核心数据实体，对应数据库表设计。

### 5.1. `k8s_cluster` (K8s集群表)
存储K8s集群的基本信息。
| Field             | Type          | Column Name      | Description                                  |
|-------------------|---------------|------------------|----------------------------------------------|
| ID                | BIGINT        | id               | 主键 (自动增长)                                |
| CreatedAt         | TIMESTAMP     | created_at       | 创建时间                                       |
| UpdatedAt         | TIMESTAMP     | updated_at       | 更新时间                                       |
| DeletedAt         | TIMESTAMP     | deleted_at       | 删除时间 (软删除)                               |
| ClusterID         | VARCHAR(36)   | cluster_id       | 集群UUID (唯一, 默认空)                        |
| ClusterName       | VARCHAR(128)  | clustername      | 集群名称 (默认空)                               |
| ClusterNameCn     | VARCHAR(256)  | clusternamecn    | 集群中文名称 (默认空)                             |
| Alias             | VARCHAR(128)  | alias            | 集群别名 (默认空)                               |
| ApiServer         | VARCHAR(256)  | apiserver        | API Server地址 (默认空)                        |
| ApiServerVip      | VARCHAR(256)  | api_server_vip   | API Server VIP地址 (默认空)                    |
| EtcdServer        | VARCHAR(256)  | etcdserver       | ETCD Server地址 (默认空)                       |
| EtcdServerVip     | VARCHAR(1024) | etcd_server_vip  | ETCD Server VIP地址 (默认空)                   |
| IngressServername | VARCHAR(256)  | ingressservername| Ingress Controller 服务名 (默认空)              |
| IngressServerVip  | VARCHAR(256)  | ingressservervip | Ingress VIP地址 (默认空)                       |
| KubePromVersion   | VARCHAR(256)  | kubepromversion  | Kube-Prometheus版本 (默认空)                   |
| PromServer        | VARCHAR(256)  | promserver       | Prometheus Server地址 (默认空)                 |
| ThanosServer      | VARCHAR(256)  | thanosserver     | Thanos Server地址 (默认空)                     |
| Idc               | VARCHAR(36)   | idc              | IDC标识 (gl, ft, wg, qf等, 默认空)              |
| Zone              | VARCHAR(36)   | zone             | 区域标识 (egt, core等, 默认空)                   |
| Status            | VARCHAR(36)   | status           | 集群状态 (pending, running, maintaining等, 默认空)|
| ClusterType       | VARCHAR(36)   | clustertype      | 集群类型 (tool, work, game, cloud等, 默认空)    |
| KubeConfig        | VARCHAR(1024) | kubeconfig       | KubeConfig内容 (默认空)                        |
| Desc              | TEXT          | desc             | 描述 (默认空)                                  |
| Creator           | VARCHAR(128)  | creator          | 创建者 (默认空)                                |
| Group             | VARCHAR(128)  | group            | 分组信息 (wayne使用, 默认空)                      |
| EsServer          | VARCHAR(128)  | esserver         | Elasticsearch Server地址 (默认空)              |
| NetType           | VARCHAR(20)   | nettype          | 网络类型 (默认空)                               |
| Architecture      | VARCHAR(20)   | architecture     | 架构类型 (默认空)                               |
| FlowType          | VARCHAR(255)  | flow_type        | 流程类型 (默认空)                               |
| NovaName          | VARCHAR(255)  | nova_name        | Nova名称 (默认空)                              |
| Priority          | INT           | level            | 优先级 (默认0)                                 |
| ClusterGroup      | VARCHAR(128)  | cluster_group    | 集群分组 (同IDC, 默认空)                         |
| PodCidr           | VARCHAR(1024) | pod_cidr         | Pod CIDR (默认空)                              |
| ServiceCidr       | VARCHAR(1024) | service_cidr     | Service CIDR (默认空)                          |
| RrCicode          | VARCHAR(1024) | rr_cicode        | RR Cicode (默认空)                             |
| RrGroup           | VARCHAR(1024) | rr_group         | RR Group (默认空)                              |

### 5.2. `k8s_cluster_resource_snapshot` (集群资源快照表)
存储特定时间点集群的资源使用情况。
| Field               | Type        | Column Name        | Description        |
|---------------------|-------------|--------------------|--------------------|
| ID                  | BIGINT      | id                 | 主键 (自动增长)        |
| CreatedAt           | TIMESTAMP   | created_at         | 创建时间             |
| UpdatedAt           | TIMESTAMP   | updated_at         | 更新时间             |
| DeletedAt           | TIMESTAMP   | deleted_at         | 删除时间 (软删除)      |
| CpuCapacity         | FLOAT64     | cpu_capacity       | CPU总容量           |
| MemoryCapacity      | FLOAT64     | mem_capacity       | 内存总容量 (GB)      |
| CpuRequest          | FLOAT64     | cpu_request        | CPU请求总量         |
| MemRequest          | FLOAT64     | mem_request        | 内存请求总量 (GB)    |
| NodeCount           | INT64       | node_count         | 节点数量             |
| BMCount             | INT64       | bm_count           | 物理机数量           |
| VMCount             | INT64       | vm_count           | 虚拟机数量           |
| MaxCpuUsageRatio    | FLOAT64     | max_cpu            | 最大CPU使用率        |
| MaxMemoryUsageRatio | FLOAT64     | max_memory         | 最大内存使用率        |
| ClusterID           | UINT        | cluster_id         | 关联的K8sCluster的ID |
| PerNodeCpuRequest   | FLOAT64     | per_node_cpu_req   | 每节点平均CPU请求量    |
| PerNodeMemRequest   | FLOAT64     | per_node_mem_req   | 每节点平均内存请求量    |
| ResourceType        | VARCHAR     | resource_type      | 资源类型 (见模型定义)  |
| PodCount            | INT64       | pod_count          | Pod数量             |


### 5.3. `device` (设备/节点表)
存储集群中的设备（节点）信息。
| Field          | Type         | Column Name       | Description                            |
|----------------|--------------|-------------------|----------------------------------------|
| ID             | BIGINT       | id                | 主键 (自动增长)                          |
| CreatedAt      | TIMESTAMP    | created_at        | 创建时间                                 |
| UpdatedAt      | TIMESTAMP    | updated_at        | 更新时间                                 |
| DeletedAt      | TIMESTAMP    | deleted_at        | 删除时间 (软删除)                         |
| CICode         | VARCHAR(255) | ci_code           | 设备编码 (有索引)                          |
| IP             | VARCHAR(50)  | ip                | IP地址 (有索引)                           |
| ArchType       | VARCHAR(50)  | arch_type         | CPU架构                                |
| IDC            | VARCHAR(100) | idc               | IDC                                    |
| Room           | VARCHAR(100) | room              | 机房                                   |
| Cabinet        | VARCHAR(100) | cabinet           | 所属机柜                                 |
| CabinetNO      | VARCHAR(100) | cabinet_no        | 机柜编号                                 |
| InfraType      | VARCHAR(100) | infra_type        | 网络类型                                 |
| IsLocalization | BOOLEAN      | is_localization   | 是否国产化                               |
| NetZone        | VARCHAR(100) | net_zone          | 网络区域                                 |
| Group          | VARCHAR(100) | group             | 机器类别                                 |
| AppID          | VARCHAR(100) | appid             | APPID                                  |
| OsCreateTime   | VARCHAR(100) | os_create_time    | 操作系统创建时间                           |
| CPU            | FLOAT        | cpu               | CPU数量                                |
| Memory         | FLOAT        | memory            | 内存大小 (GB)                            |
| Model          | VARCHAR(100) | model             | 型号                                   |
| KvmIP          | VARCHAR(50)  | kvm_ip            | KVM IP                                 |
| OS             | VARCHAR(100) | os                | 操作系统                                 |
| Company        | VARCHAR(100) | company           | 厂商                                   |
| OSName         | VARCHAR(100) | os_name           | 操作系统名称                               |
| OSIssue        | VARCHAR(100) | os_issue          | 操作系统版本                               |
| OSKernel       | VARCHAR(100) | os_kernel         | 操作系统内核                               |
| Status         | VARCHAR(50)  | status            | 设备状态                                 |
| Role           | VARCHAR(100) | role              | 角色                                   |
| Cluster        | VARCHAR(255) | cluster           | 所属集群名称                               |
| ClusterID      | INT          | cluster_id        | 所属集群ID (关联k8s_cluster表的ID)       |
| AcceptanceTime | VARCHAR(100) | acceptance_time   | 验收时间                                 |
| DiskCount      | INT          | disk_count        | 磁盘数量                                 |
| DiskDetail     | TEXT         | disk_detail       | 磁盘详情                                 |
| NetworkSpeed   | VARCHAR(255) | network_speed     | 网络速度                                 |
| IsSpecial      | BOOLEAN      | is_special        | 是否为特殊设备 (计算得出, 只读)                |
| FeatureCount   | INT          | feature_count     | 特性数量 (计算得出, 只读)                    |
| AppName        | VARCHAR(255) | app_name          | 应用名称 (计算得出, 只读)                    |

### 5.4. `elastic_scaling_strategy` (弹性伸缩策略表)
| Field                      | Type                                   | Description                                                                                    |
|----------------------------|----------------------------------------|------------------------------------------------------------------------------------------------|
| id                         | BIGINT                                 | 主键                                                                                           |
| name                       | VARCHAR(255)                           | 策略名称                                                                                       |
| description                | TEXT                                   | 策略描述                                                                                       |
| threshold_trigger_action   | ENUM('pool_entry', 'pool_exit')        | 定义策略阈值是用于触发资源池进入 (扩容)还是退出 (缩容)                                                |
| cpu_threshold_value        | DECIMAL(5,2)                           | CPU 阈值 (百分比)。如果此策略不监控 CPU，则为 NULL。                                              |
| cpu_threshold_type         | ENUM('usage', 'allocated')             | CPU 阈值类型 (例如 'usage' 代表 CPU 使用率)。如果不监控 CPU，则为 NULL。                         |
| memory_threshold_value     | DECIMAL(5,2)                           | 内存阈值 (百分比)。如果此策略不监控内存，则为 NULL。                                             |
| memory_threshold_type      | ENUM('usage', 'allocated')             | 内存阈值类型 (例如 'usage' 代表内存使用率)。如果不监控内存，则为 NULL。                        |
| condition_logic            | ENUM('AND', 'OR')                      | 当 `cpu_threshold_value` 和 `memory_threshold_value` 都设置时，如何组合 CPU 和内存条件。如果只设置一个，默认为 'OR' (或等效单一条件)。 |
| duration_minutes           | INT                                    | 组合阈值条件必须满足的持续时间 (分钟)。                                                          |
| cooldown_minutes           | INT                                    | 策略执行后的冷却时间 (分钟)，在此期间策略不会再次触发。                                            |
| device_count               | INT                                    | 策略触发时要添加或移除的设备 (节点) 数量。                                                       |
| node_selector              | JSON                                   | 用于识别/定位特定节点以进行添加/移除的节点选择器标签。                                              |
| status                     | ENUM('enabled', 'disabled')            | 策略状态。                                                                                     |
| created_by                 | VARCHAR(100)                           | 创建者的用户名或系统标识符。                                                                     |
| created_at                 | TIMESTAMP                              | 策略创建时的时间戳。                                                                           |
| updated_at                 | TIMESTAMP                              | 策略最后更新的时间戳。                                                                         |

### 5.5. `strategy_cluster_association` (策略集群关联表)
| Field       | Type   | Description                                     |
|-------------|--------|-------------------------------------------------|
| strategy_id | BIGINT | 外键，关联 `elastic_scaling_strategy.id`。       |
| cluster_id  | BIGINT | 外键，关联 `k8s_cluster.id`。                   |
| PRIMARY KEY (strategy_id, cluster_id) |        | 确保关联唯一性。                                  |

### 5.6. `elastic_scaling_order` (弹性伸缩订单表)
| Field           | Type                                                       | Description                                              |
|-----------------|------------------------------------------------------------|----------------------------------------------------------|
| id              | BIGINT                                                     | 主键                                                       |
| order_number    | VARCHAR(50)                                                | 唯一订单号                                                 |
| cluster_id      | BIGINT                                                     | 外键，关联 `k8s_cluster.id`，指明订单针对哪个集群。           |
| strategy_id     | BIGINT                                                     | 外键，关联 `elastic_scaling_strategy.id` (手动订单可为 NULL) |
| action_type     | ENUM('pool_entry', 'pool_exit')                            | 订单操作类型 (入池/退池)                                      |
| status          | ENUM('pending', 'processing', 'completed', 'failed', 'cancelled') | 订单状态                                                 |
| device_count    | INT                                                        | 请求的设备数量                                             |
| approver        | VARCHAR(100)                                               | 审批人用户名 (若需要审批流程)                                |
| executor        | VARCHAR(100)                                               | 执行人用户名                                               |
| execution_time  | TIMESTAMP                                                  | 订单开始执行时间                                           |
| created_by      | VARCHAR(100)                                               | 创建者用户名/系统                                          |
| created_at      | TIMESTAMP                                                  | 创建时间                                                   |
| updated_at      | TIMESTAMP                                                  | 最后更新时间                                               |
| completion_time | TIMESTAMP                                                  | 订单完成时间                                               |
| failure_reason  | TEXT                                                       | 失败原因 (如果订单失败)                                      |

### 5.7. `order_device` (订单设备关联表)
记录订单中具体涉及的设备及其处理状态。
| Field      | Type                                               | Description                     |
|------------|----------------------------------------------------|---------------------------------|
| id         | BIGINT                                             | 主键                              |
| order_id   | BIGINT                                             | 外键, 关联 `elastic_scaling_order` |
| device_id  | BIGINT                                             | 外键, 关联 `device`               |
| status     | ENUM('pending', 'processing', 'completed', 'failed') | 此设备在此订单中的处理状态        |
| created_at | TIMESTAMP                                          | 创建时间                          |
| updated_at | TIMESTAMP                                          | 最后更新时间                        |

### 5.8. `strategy_execution_history` (策略执行历史表)
记录策略每次评估和触发的历史。
| Field           | Type                                            | Description                            |
|-----------------|-------------------------------------------------|----------------------------------------|
| id              | BIGINT                                          | 主键                                     |
| strategy_id     | BIGINT                                          | 外键, 关联 `elastic_scaling_strategy`    |
| execution_time  | TIMESTAMP                                       | 执行时间                                 |
| triggered_value | VARCHAR(255)                                    | 触发策略时的具体指标值(例如: "CPU:85%,MEM:N/A") |
| threshold_value | VARCHAR(255)                                    | 触发策略时的阈值设定(例如: "CPU:80%,MEM:N/A") |
| result          | ENUM('order_created', 'skipped', 'failed_check') | 执行结果 (skipped可能因为冷却或条件不完全满足) |
| order_id        | BIGINT                                          | 外键, 关联 `elastic_scaling_order` (如果创建了订单) |
| reason          | TEXT                                            | 执行结果的原因 (例如: 冷却中, 条件不满足的具体描述) |

### 5.9. `duty_roster` (值班表)
管理值班人员信息。
| Field        | Type         | Description |
|--------------|--------------|-------------|
| id           | BIGINT       | 主键          |
| user_id      | VARCHAR(100) | 用户ID/名称   |
| duty_date    | DATE         | 值班日期      |
| start_time   | TIME         | 开始时间      |
| end_time     | TIME         | 结束时间      |
| contact_info | VARCHAR(255) | 联系方式      |

### 5.10. `notification_log` (通知日志表)
记录发送给用户的通知。
| Field             | Type                               | Description            |
|-------------------|------------------------------------|------------------------|
| id                | BIGINT                             | 主键                     |
| order_id          | BIGINT                             | 外键, 关联 `elastic_scaling_order` (可选,若通知与订单相关) |
| strategy_id       | BIGINT                             | 外键, 关联 `elastic_scaling_strategy` (可选, 若通知与策略相关) |
| notification_type | ENUM('email', 'sms', 'im', 'system') | 通知类型                 |
| recipient         | VARCHAR(255)                       |接收人信息 (用户ID,邮箱等) |
| content           | TEXT                               | 通知内容                 |
| status            | ENUM('sent', 'failed', 'read')     | 发送状态                 |
| send_time         | TIMESTAMP                          | 发送时间                 |
| error_message     | TEXT                               | 错误信息 (如果发送失败)    |


## 6. 非功能性需求

-   **易用性**: 界面应基于 Ant Design 组件库，提供直观、一致的用户体验。操作流程应简明。
-   **数据可视化**: 使用 ECharts 等图表库清晰展示资源趋势和统计数据。
-   **响应式**: 关键组件 (如图表、列表) 应能适应不同屏幕尺寸。
-   **可维护性**: 前端代码 (React) 和后端代码 (如Go) 应结构清晰，模块化，易于扩展和维护。
-   **性能**:
    -   列表和图表数据加载应有良好性能，对于大量数据应考虑分页、虚拟滚动、后端聚合等技术。
    -   API响应时间应在可接受范围内 (例如，大部分查询 < 1s)。
    -   策略评估引擎的性能应能支持所管理的集群和策略数量。
    -   数据库查询优化，合理使用索引。
    -   利用缓存机制 (如Redis) 缓存热点数据和配置。
-   **实时性/数据一致性**:
    -   关键数据 (如待处理订单数、资源使用率) 应提供手动刷新功能，并考虑定时轮询或WebSocket推送更新。
    -   确保操作的原子性和数据的一致性，特别是在订单处理和设备状态变更时。
-   **可配置性**:
    -   策略的各项参数应可灵活配置。
    -   系统级参数 (如监控间隔、通知方式、审批流程) 应可配置。
-   **可靠性与可用性**:
    -   后端服务应设计为高可用，支持水平扩展。
    -   关键任务 (如策略评估、订单执行) 应有重试和容错机制。
    -   数据库应有备份和恢复策略。
-   **安全性**:
    -   API接口需进行认证和授权 (如JWT, RBAC)。
    -   防止常见Web攻击 (XSS, CSRF, SQL注入等)。
    -   敏感配置信息加密存储。
    -   操作日志记录关键用户行为和系统事件。
-   **可扩展性**: 系统设计应易于未来增加新的监控指标、策略类型或集成其他系统。

## 7. 后端系统概述

### 7.1. 架构概览
系统后端可采用微服务架构，主要服务包括：
-   **API网关服务**: 统一入口，负责请求路由、认证、限流等。
-   **策略管理服务**: 负责策略的CRUD、存储，以及提供给评估引擎。
-   **订单管理服务**: 负责订单的CRUD、状态流转、与设备操作的协调。
-   **资源监控服务**: 定期从各K8s集群采集资源快照数据。
-   **策略评估引擎服务**: 核心服务，定期获取最新资源数据和策略配置，进行评估，满足条件则创建订单。
-   **设备操作服务**: (可选，或集成在订单服务中) 负责与K8s集群API交互，执行节点添加、移除等操作。
-   **通知服务**: 负责向相关人员发送通知。
-   **用户与权限服务**: 管理用户、角色和权限。

### 7.2. 核心工作流程
-   **策略评估与订单生成 (自动监控核心)**:
    1.  **周期性监控**: 资源监控服务按照预定间隔（例如，每5分钟，此间隔应可在系统配置中调整）从所有纳管的Kubernetes集群 (`k8s_cluster`表记录的集群) 拉取最新的资源利用率数据。这些数据包括CPU使用率、内存使用率、节点状态等，并存入 `k8s_cluster_resource_snapshot` 表。
    2.  **策略加载**: 策略评估引擎服务同样以预定间隔（可与资源监控间隔相同或不同，应可配置）从数据库加载所有状态为 "enabled" 的弹性伸缩策略 (`elastic_scaling_strategy` 表) 及其关联的集群信息 (`strategy_cluster_association` 表)。
    3.  **逐条策略评估**: 对于每一个加载的启用策略：
        a.  **集群数据获取**: 引擎获取该策略所关联的每一个集群的最新（或一段时间内的平均/峰值，根据策略具体配置）资源快照数据。
        b.  **条件检查**:
            i.  **阈值比较**: 将集群的当前资源指标（CPU、内存的 `usage` 或 `allocated`，根据策略的 `cpu_threshold_type` 和 `memory_threshold_type`）与策略中设定的 `cpu_threshold_value` 和 `memory_threshold_value` 进行比较。
            ii. **逻辑组合**: 如果策略同时定义了CPU和内存阈值，则根据 `condition_logic` ('AND' 或 'OR') 判断组合条件是否满足。
            iii. **持续时间**: 判断满足阈值条件的状况是否已持续达到策略中定义的 `duration_minutes`。这可能需要查询历史快照数据或维护一个内存状态来跟踪持续时间。
        c.  **冷却期检查**: 检查该策略上次成功触发执行后，是否已超过 `cooldown_minutes` 定义的冷却时间。如果策略在冷却期内，则本次跳过，不生成订单。
        d.  **订单生成**: 如果所有条件（阈值、持续时间、不在冷却期）均满足：
            i.  系统根据策略的 `threshold_trigger_action` ('pool_entry' 或 'pool_exit') 和 `device_count` 生成一个新的弹性伸缩订单，存入 `elastic_scaling_order` 表，初始状态为 `pending`。
            ii. 在 `strategy_execution_history` 表中记录本次策略执行的详细信息，包括触发时间、触发时的资源值、阈值、执行结果 ('order_created') 以及关联的订单ID。
        e.  **跳过记录**: 如果条件不满足或在冷却期内而未生成订单，也可选择性地在 `strategy_execution_history` 中记录一次 'skipped' 的执行，并注明原因，便于追踪策略评估行为。
-   **订单处理 (入池/退池)**:
    1.  新订单状态为 `pending`。通知服务可通知值班人员或根据配置自动流转。
    2.  (可选审批流程) 值班人员或系统根据配置审批订单，状态变为 `processing`。
    3.  订单管理服务根据订单类型 (`action_type`) 和设备数量 (`device_count`)：
        -   **入池**: 从可用设备中选择符合 `node_selector` 的设备，通过设备操作服务将其加入目标集群。
        -   **退池**: 从目标集群中选择合适的节点（考虑负载、污点等），通过设备操作服务将其移除。
    4.  在 `order_device` 中记录涉及的设备及处理状态。
    5.  操作完成后，更新订单状态为 `completed` 或 `failed` (及 `failure_reason`)。更新设备在 `device` 表中的状态。
    6.  通知相关人员订单结果。

## 8. 待明确项与进一步考虑

-   **`device` 表的维护**: 如何发现、注册和更新设备信息。设备与K8s Node的精确对应关系。
-   **节点选择逻辑 (退池)**:
    -   **目标**: 当触发退池操作时，系统需要智能地从目标集群中选择出N个最适合移除的节点。此逻辑将主要由 `server/portal/internal/service/device_query.go` 提供支持，并由订单管理服务调用和协调。
    -   **核心原则**:
        -   **优先选择非特殊设备**: `device_query.go` 在筛选候选节点时，应将 `device.IsSpecial = false` 的设备作为高优先级。特殊设备（`IsSpecial = true`）应尽量避免被自动退池，除非明确配置或手动选择。
    -   **辅助筛选与排序标准** (可配置优先级或作为评分因子):
        1.  **最低资源利用率**: 优先选择当前CPU和内存请求/使用率最低的节点。相关数据可从 `k8s_cluster_resource_snapshot` (可能是节点级别快照，或聚合计算) 或通过Kubernetes API实时获取。
        2.  **最少运行Pod数/最低影响Pod**: 优先选择运行Pod数量最少的节点。进一步地，可以评估节点上运行的Pod对业务的影响程度（例如，基于Pod优先级、标签、注解等元数据），选择影响最小的。这需要与Kubernetes API交互获取节点上的Pod列表及相关信息。
        3.  **无关键污点/标签**: 避免选择带有特定保护性污点（如 `NoSchedule` 且阻止驱逐）或关键业务标签的节点，除非策略明确允许。
        4.  **最长在池时间 (可选策略)**: 作为一种轮换机制，可以选择在资源池中停留时间最长的节点，实现类似FIFO的退池。这需要记录节点的入池时间。
        5.  **节点健康状态**: 必须确保只选择健康 (K8s Node `Ready`状态) 且当前未被手动Cordon的节点执行退池操作。
    -   **实现建议**:
        -   `server/portal/internal/service/device_query.go` 应提供一个函数，输入参数包括 `cluster_id`、需要退池的 `device_count` 以及可能的 `node_selector` (来自策略)。
        -   该函数首先过滤出目标集群中所有当前在池内的节点。
        -   应用上述筛选标准（特别是 `IsSpecial=false`），然后根据其他标准（如资源利用率、Pod数量等）对候选节点进行排序或评分。
        -   返回一个排序后的候选节点列表给订单管理服务。
        -   订单管理服务从列表顶部选取N个节点，执行标准的Kubernetes节点退役流程：先将节点标记为 `Unschedulable` (Cordon)，然后驱逐 (Drain) 节点上的所有Pod，最后从Kubernetes集群中移除该节点，并更新 `device` 表中对应设备的状态（如标记为"可用"或"待清理"等）。
-   **手动订单创建界面与设备维护流程**:
    -   **背景**: 除了策略自动触发的伸缩订单外，系统需要支持手动创建订单，特别是针对设备维护场景。此流程通常涉及与外部设备管理系统或团队的协作。
    -   **流程步骤**:
        1.  **维护请求接收 (API)**: 系统提供一个API接口，供下游系统（如设备组的维护管理系统）调用，以请求对特定设备/节点进行维护。请求参数应包括：
            -   设备标识 (如 `device.CICode` 或 `device.ID`)。
            -   期望的维护开始时间 (`maintenance_start_time`) 和结束时间 (`maintenance_end_time`)。
            -   外部系统工单号 (`external_ticket_id`)，用于追踪。
            -   维护原因/描述。
        2.  **创建维护请求订单**: 接收到请求后，系统内部创建一个类型为 `maintenance_request` 的 `elastic_scaling_order`。
            -   `device_id` 字段记录目标设备。
            -   初始状态可设为 `pending_confirmation`。
        3.  **调度确认与回调**:
            -   系统（或运维人员通过界面操作）确认维护窗口是否可行。
            -   通过API回调通知下游请求系统，告知已收到请求、内部订单号以及确认/建议的维护时间。
        4.  **下游系统确认**: 下游系统通过API调用，确认维护安排。订单状态更新为 `scheduled_for_maintenance`。
        5.  **执行维护前置操作 (K8s层面)**:
            -   在维护开始前，K8s平台运维人员（或自动化脚本）对目标节点执行 `cordon` 操作，使其不再接收新的Pod调度。
            -   根据需要，可能执行 `drain` 操作（优雅地驱逐节点上的现有Pod）。
            -   此操作完成后，可更新维护订单状态为 `maintenance_in_progress`。
        6.  **通知下游开始维护**: 一旦节点在K8s层面准备就绪（例如已Cordon），系统通过API回调或其他通知方式，通知下游系统可以开始物理维护。
        7.  **下游执行物理维护**: 下游团队对设备进行实际的维护操作。
        8.  **维护完成通知 (API)**: 下游系统维护完成后，通过API调用通知本系统维护已结束。
        9.  **创建"节点恢复服务"订单**: 收到维护完成通知后，系统自动或由运维人员手动创建一个新的关联订单，类型为 `maintenance_uncordon`。此订单的目标是使节点恢复服务。
            -   此订单关联到同一个 `device_id`。
            -   初始状态为 `pending` 或 `processing`。
        10. **执行Uncordon操作**: K8s平台运维人员（或自动化脚本）执行此订单，对节点进行 `uncordon` 操作，使其重新变为可调度状态。
            -   **注意**: 此操作与标准的"入池"不同，它不涉及重新加入集群或复杂的配置，主要是解除Cordon状态。
        11. **状态更新与关闭**: `maintenance_uncordon` 订单完成后，状态更新为 `completed`。原 `maintenance_request` 订单也可相应关闭或标记为 `completed`。
    -   **界面支持**:
        -   需要界面来查看和管理这些手动创建的维护类订单。
        -   应能清晰展示订单的各个阶段、关联的设备、以及与下游系统的交互状态。
    -   **数据模型扩展**: `elastic_scaling_order` 表的 `action_type` 和 `status` ENUM类型需要按上述流程进行扩展。相关字段如 `device_id`, `maintenance_start_time`, `maintenance_end_time`, `external_ticket_id` 已添加。
-   **前端状态管理和实时更新策略**: 对于列表和统计数据的实时性要求,保持与k8s_cluster_resource_snapshot的实时同步即可。
-   **API响应与错误码**: API响应应遵循 `pkg/middleware/render/json.go` 中定义的通用结构 (`{ "code": <http_status_code>, "msg": "<message>", "data": <optional_data> }`)。错误码直接使用HTTP状态码，如 `200` (成功), `400` (错误请求), `401` (未授权), `403` (禁止访问), `404` (未找到), `500` (服务器内部错误)。具体的业务逻辑错误信息通过 `msg` 字段传递。

## 8. 节点选择逻辑（退池）

### 8.1 概述
节点选择逻辑主要由 `server/portal/internal/service/device_query.go` 提供支持，并由订单管理服务调用和协调。该逻辑用于在触发退池操作时，从目标集群中智能选择最适合移除的节点。

### 8.2 核心原则
- **非特殊设备优先**：`device.IsSpecial = false` 的设备作为高优先级选择对象
- **特殊设备保护**：`IsSpecial = true` 的设备应避免被自动退池，除非明确配置或手动选择

### 8.3 筛选与排序标准
以下标准可按优先级配置或作为评分因子：

1. **资源利用率最低**
   - 优先选择 CPU 和内存请求/使用率最低的节点
   - 数据来源：`k8s_cluster_resource_snapshot` 或 Kubernetes API 实时数据

2. **运行 Pod 数最少/影响最小**
   - 优先选择运行 Pod 数量最少的节点
   - 评估指标：
     - Pod 优先级
     - 标签信息
     - 注解信息
   - 需要与 Kubernetes API 交互获取节点 Pod 信息

3. **无关键污点/标签**
   - 避免选择带有保护性污点的节点（如 `NoSchedule` 且阻止驱逐）
   - 避免选择带有关键业务标签的节点
   - 例外：策略明确允许的情况

4. **最长在池时间**（可选策略）
   - 实现类似 FIFO 的轮换机制
   - 需要记录节点的入池时间

5. **节点健康状态**
   - 必须处于 Kubernetes `Ready` 状态
   - 必须未被手动 Cordon

### 8.4 实现细节
- **函数签名**：`SelectNodesForRemoval(cluster_id, device_count, node_selector)`
- **处理流程**：
  1. 过滤目标集群中当前在池内的节点
  2. 应用筛选标准（特别是 `IsSpecial=false`）
  3. 根据其他标准（资源利用率、Pod 数量等）对候选节点排序/评分
  4. 返回排序后的候选节点列表给订单管理服务
  5. 订单管理服务从列表顶部选取 N 个节点
  6. 执行标准 Kubernetes 节点退役流程：
     - 标记为 `Unschedulable`（Cordon）
     - 驱逐（Drain）节点上的所有 Pod
     - 从 Kubernetes 集群中移除节点
     - 更新 `device` 表中对应设备状态

## 9. 手动订单创建与设备维护

### 9.1 背景
除了策略自动触发的伸缩订单外，系统需要支持手动创建订单，特别是针对设备维护场景。此流程通常涉及与外部设备管理系统或团队的协作。

### 9.2 流程步骤

#### 9.2.1 维护请求接收
- **API 接口**：供下游系统（如设备组维护管理系统）调用
- **请求参数**：
  - 设备标识（`device.CICode` 或 `device.ID`）
  - 维护时间窗口（`maintenance_start_time`、`maintenance_end_time`）
  - 外部系统工单号（`external_ticket_id`）
  - 维护原因/描述

#### 9.2.2 创建维护请求订单
- 创建类型为 `maintenance_request` 的 `elastic_scaling_order`
- 记录目标设备到 `device_id` 字段
- 初始状态设为 `pending_confirmation`

#### 9.2.3 调度确认与回调
- 系统/运维人员确认维护窗口可行性
- 通过 API 回调通知下游系统：
  - 请求接收确认
  - 内部订单号
  - 确认/建议的维护时间

#### 9.2.4 下游系统确认
- 外部系统确认维护安排
- 更新订单状态为 `scheduled_for_maintenance`

#### 9.2.5 执行维护前置操作
- 在维护开始前执行 Kubernetes 层面操作：
  - 执行 `cordon` 操作
  - 可选执行 `drain` 操作
- 更新订单状态为 `maintenance_in_progress`

#### 9.2.6 通知下游开始维护
- 节点在 Kubernetes 层面准备就绪后
- 通知下游系统开始物理维护

#### 9.2.7 下游执行物理维护
- 下游团队执行实际维护操作

#### 9.2.8 维护完成通知
- 下游系统通过 API 通知维护完成

#### 9.2.9 创建节点恢复服务订单
- 创建类型为 `maintenance_uncordon` 的关联订单
- 关联到同一 `device_id`
- 初始状态：`pending` 或 `processing`

#### 9.2.10 执行 Uncordon 操作
- 执行 `uncordon` 操作使节点恢复可调度状态
- 注意：此操作与标准"入池"不同，仅解除 Cordon 状态

#### 9.2.11 状态更新与关闭
- 更新 `maintenance_uncordon` 订单状态为 `completed`
- 关闭原 `maintenance_request` 订单

### 9.3 界面支持
- 提供维护类订单的查看和管理界面
- 清晰展示：
  - 订单各阶段状态
  - 关联设备信息
  - 与下游系统的交互状态

### 9.4 数据模型扩展
- 扩展 `elastic_scaling_order` 表：
  - 新增字段：
    - `device_id`
    - `maintenance_start_time`
    - `maintenance_end_time`
    - `external_ticket_id`
  - 更新 `action_type` 和 `status` 的 ENUM 类型

## 10. 前端状态管理和实时更新策略

### 10.1 实时数据同步
- 列表和统计数据的实时性要求
- 保持与 `k8s_cluster_resource_snapshot` 的实时同步

### 10.2 API 响应规范
- 遵循 `pkg/middleware/render/json.go` 中定义的通用结构：
  ```json
  {
    "code": <http_status_code>,
    "msg": "<message>",
    "data": <optional_data>
  }
  ```
- 错误码使用标准 HTTP 状态码：
  - `200`：成功
  - `400`：错误请求
  - `401`：未授权
  - `403`：禁止访问
  - `404`：未找到
  - `500`：服务器内部错误
- 具体业务逻辑错误信息通过 `msg` 字段传递