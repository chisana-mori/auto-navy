# 项目编码规则

本文档旨在统一项目编码风格，提高代码可读性和可维护性。

## 1. 通用规则

- **版本控制**: 使用 Git 进行版本控制，遵循清晰的分支策略（例如 Gitflow 或 GitHub Flow）。
- **提交信息**: 遵循 Conventional Commits 规范编写提交信息。
- **代码格式化**: 使用自动化工具（如 Go Fmt, Prettier）统一代码格式。
- **命名**: 变量、函数、类、文件等命名应清晰、简洁、有意义，并遵循对应语言的惯例。
- **注释**: 对复杂的逻辑、重要的决策或公共 API 添加必要的注释。优先使用英文注释，如需使用中文，请保持一致。
- **文档**: 重要模块和公共 API 应编写必要的文档（例如 Go Doc, Swagger, JSDoc）。
- **依赖管理**: 明确管理项目依赖，定期更新并移除不再使用的依赖。

## 2. 后端 (Go)

基于 `server/portal/` 目录下的代码。

- **目录结构**: 生成的业务代码需要满足以下代码树结构：
```
navy-ng/
├── models/                # 数据模型层 - 使用GORM进行数据库操作
│   └── portal/            # 门户模块数据模型
│       └── object.go      # 核心数据模型定义文件
│
├── pkg/                   # 公共库代码 - 可被外部项目引用
│   └── middleware/        # 中间件
│       └── render         # controller层渲染相关方法
│            └── json.go   # 渲染方法
│
├── server/                # 后端服务层
│   └── portal/            # 门户模块服务
│       └── internal/      # 内部实现(不对外暴露)
│           ├── main.go    # 服务启动入口
│           ├── conf/      # 配置管理(环境变量/配置文件)
│           ├── docs/      # swagger文档
│           ├── routers/   # Gin路由定义与控制器
│           └── service/   # 业务逻辑实现
│
├── job/                   # 巡检任务job
│   └── email/             # 任务job集合，每个job对应发送一种类型的邮件
├── web/                   # 前端应用层
│   └── navy-fe/           # React前端项目
│
├── scripts/               # 开发运维脚本
│                           # 构建/部署/测试等自动化脚本
│
# 项目配置文件
├── .gitignore            # Git忽略规则
├── .golangci.yaml        # Go代码静态分析配置
├── Makefile              # 项目构建/测试/运行命令
├── README.md             # 项目文档说明
├── go.mod                # Go模块依赖管理
└── go.sum                # 依赖版本校验
```
- **命名规范**:
    - 包名：小写，简洁，避免下划线或 `-`。
    - 变量/函数：`camelCase`。
    - 导出类型/常量/函数：`PascalCase`。
    - 常量：根据上下文使用 `PascalCase` 或 `ALL_CAPS_SNAKE_CASE` (如 `SQLConstantName`, `MaxConnections`)。
    - 文件名：`snake_case.go` (例如 `device_query.go`)。
- **代码风格**:
    - 使用 `gofmt` 或 `goimports` 自动格式化代码。
    - 遵循 `.golangci.yaml` 中定义的 Lint 规则，特别是 `depguard` 规则以维护层级依赖关系。
    - 错误处理：使用标准 `error` 类型，必要时使用 `fmt.Errorf` 添加上下文。优先处理错误，避免嵌套过深。
- **依赖库**:
    - Web 框架: `github.com/gin-gonic/gin`
    - ORM: `gorm.io/gorm`
    - 时间处理: `github.com/jinzhu/now`
    - API 文档: `github.com/swaggo/gin-swagger`, `github.com/swaggo/swag`
- **API 设计**: 遵循 RESTful 原则设计 API 接口。
- **Redis 缓存**: 以下规则旨在提供一致且高效的缓存策略。
    - **键命名**: 应使用统一的方式（例如助手函数或结构体，如 `pkg/redis/keys.go` 中的 `KeyBuilder`）生成缓存键，以确保一致性。推荐格式为 `项目前缀:模块:版本:类型:唯一标识` (例如: `navy:portal:v1:device:123` 或 `navy:portal:v1:device_list:query_hash`)。版本号 (`v1`) 可用于区分不同的缓存结构或版本。
    - **序列化**: 推荐使用标准库（如 `encoding/json`）对复杂数据结构进行序列化和反序列化后再存入 Redis。
    - **过期时间**: 
        - 为不同类型的缓存数据定义明确的 Go `time.Duration` 常量（如 `service/device_cache.go` 中所示），以便于管理和调整。
        - 操作 Redis 时务必设置合理的过期时间（例如使用 `SETEX` 或 `SetWithExpireTime`），避免缓存数据永久驻留。
        - 考虑为相近的缓存类型设置带有轻微随机抖动（Jitter）的过期时间，以防止缓存雪崩（当前代码未显式实现，但建议考虑）。
    - **缓存封装**: 建议创建专门的缓存服务抽象层（类似 `service/device_cache.go` 中的 `DeviceCache`），封装 Redis 客户端操作、键构建、序列化/反序列化以及可能的缓存统计逻辑，使缓存操作对核心业务逻辑透明。
    - **错误处理**: 
        - 必须显式处理缓存未命中 (Cache Miss) 的情况，通常此时需要从数据源（如数据库）加载数据，并可能回填缓存。
        - 妥善处理 Redis 操作（连接、读写）错误以及数据序列化/反序列化过程中可能出现的错误。
    - **缓存失效**: 
        - 当底层数据发生变更（如数据库记录更新或删除）时，必须主动失效或更新相关的缓存条目（如 `InvalidateDevice` 示例）。
        - 对于需要批量失效的缓存（例如列表缓存），应使用 `SCAN` 命令（如 `pkg/redis/handler.go` 中的 `ScanKeys` 封装）基于模式匹配来查找并删除键，**严禁**在生产环境使用 `KEYS` 命令，因为它可能阻塞 Redis 服务器。
    - **缓存统计**: （可选）实现简单的缓存统计（命中率、未命中率等，如 `CacheStats` 结构体所示



## 3. 前端 (React/TypeScript)

- **目录结构**: 遵循组件化和功能模块化的原则组织代码：
    - `src/`: 源代码根目录。
    - `src/components/`: 可重用的 UI 组件，可按功能或页面分组。
    - `src/pages/` (建议): 页面级组件。
    - `src/services/` 或 `src/api/`: API 请求封装。
    - `src/store/` 或 `src/hooks/` (建议): 状态管理和自定义 Hooks。
    - `src/styles/`: 全局样式和主题文件。
    - `src/types/`: TypeScript 类型定义。
    - `src/utils/`: 通用工具函数。
- **命名规范**:
    - 文件名/目录名：`kebab-case` (例如 `device-query.css`) 或 `PascalCase` (组件，例如 `DeviceTable.tsx`)。保持一致性。
    - 组件：`PascalCase` (例如 `DeviceList`)。
    - 变量/函数：`camelCase`。
    - 常量：`UPPER_SNAKE_CASE` 或 `PascalCase` (根据上下文)。
    - 类型/接口：`PascalCase` (例如 `DeviceQueryOptions`)。
- **代码风格**:
    - 使用 Prettier 自动格式化代码。
    - 使用 ESLint 进行代码检查，遵循推荐的 React 和 TypeScript 规则集。
    - 组件：优先使用函数组件和 Hooks。
    - 类型：为 Props, State, API 响应等添加明确的 TypeScript 类型。
    - 样式：推荐使用 CSS Modules, Styled Components 或 Tailwind CSS 以避免全局样式污染。当前观察到使用普通 CSS 文件，注意命名和作用域管理。
- **依赖库**:
    - 框架: React
    - 语言: TypeScript
- **API 请求**: 在 `services/` 或 `api/` 目录下封装 API 请求，统一处理请求逻辑、错误和加载状态。
