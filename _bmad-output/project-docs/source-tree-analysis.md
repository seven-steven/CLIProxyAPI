# 源码树分析

**生成日期:** 2026-03-06

---

## 完整目录结构

```
CLIProxyAPI/                           # 项目根目录
├── cmd/                               # 应用入口点
│   └── server/                        # 主服务器入口
│       └── main.go                    # 主程序入口 [ENTRY POINT]
│
├── internal/                          # 私有应用代码
│   ├── access/                        # 访问控制和配置提供者
│   │   ├── config_access/             # 配置访问提供者
│   │   └── reconcile.go               # 访问协调器
│   │
│   ├── api/                           # HTTP API 层
│   │   ├── handlers/                  # API 处理器
│   │   │   └── management/            # 管理 API 处理器
│   │   ├── middleware/                # HTTP 中间件
│   │   │   ├── request_logging.go     # 请求日志中间件
│   │   │   └── response_writer.go     # 响应写入器
│   │   ├── modules/                   # API 模块
│   │   │   └── amp/                   # Amp CLI 集成模块
│   │   └── server.go                  # Gin 服务器 [CRITICAL]
│   │
│   ├── auth/                          # 认证提供者
│   │   ├── antigravity/               # Antigravity OAuth
│   │   ├── claude/                    # Claude OAuth
│   │   ├── codex/                     # OpenAI Codex OAuth
│   │   ├── gemini/                    # Gemini 认证
│   │   ├── iflow/                     # iFlow OAuth
│   │   ├── kimi/                      # Kimi OAuth
│   │   ├── qwen/                      # Qwen OAuth
│   │   └── vertex/                    # Vertex AI 凭证
│   │
│   ├── browser/                       # 浏览器自动化
│   ├── buildinfo/                     # 构建信息
│   ├── cache/                         # 签名缓存
│   ├── cmd/                           # CLI 命令处理
│   ├── config/                        # 配置加载和管理
│   ├── constant/                      # 常量定义
│   ├── interfaces/                    # 接口定义
│   ├── logging/                       # 日志配置
│   ├── managementasset/               # 管理面板资源
│   ├── misc/                          # 杂项工具
│   ├── registry/                      # 模型注册表
│   ├── runtime/                       # 运行时配置
│   ├── store/                         # 存储后端
│   │   ├── postgresstore.go           # PostgreSQL 存储
│   │   ├── gitstore.go                # Git 存储
│   │   └── objectstore.go             # 对象存储
│   ├── thinking/                      # 思考模式处理
│   ├── translator/                    # API 格式转换器
│   ├── tui/                           # 终端用户界面
│   ├── usage/                         # 使用统计
│   ├── util/                          # 工具函数
│   ├── watcher/                       # 文件监控
│   └── wsrelay/                       # WebSocket 中继
│
├── sdk/                               # 公共 SDK（可嵌入）
│   ├── access/                        # 访问控制 SDK
│   ├── api/                           # API 处理器 SDK
│   │   └── handlers/                  # 处理器实现
│   │       ├── claude/                # Claude 处理器
│   │       ├── gemini/                # Gemini 处理器
│   │       └── openai/                # OpenAI 处理器
│   ├── auth/                          # 认证 SDK
│   ├── cliproxy/                      # 核心 SDK
│   │   ├── auth/                      # 认证管理
│   │   ├── executor/                  # 执行器
│   │   └── usage/                     # 使用追踪
│   ├── config/                        # SDK 配置
│   ├── logging/                       # SDK 日志
│   └── translator/                    # SDK 格式转换
│
├── docs/                              # 文档目录
│   ├── sdk-usage.md                   # SDK 使用指南
│   ├── sdk-advanced.md                # SDK 高级用法
│   ├── sdk-access.md                  # SDK 访问控制
│   └── sdk-watcher.md                 # SDK 监控
│
├── examples/                          # 示例代码
│   ├── custom-provider/               # 自定义提供者示例
│   ├── http-request/                  # HTTP 请求示例
│   └── translator/                    # 转换器示例
│
├── test/                              # 集成测试
│   ├── amp_management_test.go         # Amp 管理测试
│   ├── builtin_tools_translation_test.go  # 工具转换测试
│   └── thinking_conversion_test.go    # 思考模式转换测试
│
├── .github/                           # GitHub 配置
│   ├── workflows/                     # CI/CD 工作流
│   │   ├── docker-image.yml           # Docker 镜像构建
│   │   └── pr-test-build.yml          # PR 测试构建
│   └── ISSUE_TEMPLATE/                # Issue 模板
│
├── assets/                            # 静态资源
├── auths/                             # 认证文件目录
├── go.mod                             # Go 模块定义
├── go.sum                             # 依赖校验和
├── Dockerfile                         # Docker 构建文件
├── docker-compose.yml                 # Docker Compose 配置
├── .goreleaser.yml                    # GoReleaser 配置
├── config.example.yaml                # 配置示例
├── README.md                          # 项目说明（英文）
├── README_CN.md                       # 项目说明（中文）
└── LICENSE                            # MIT 许可证
```

---

## 关键目录说明

### cmd/server/ [入口点]

主程序入口，负责：

- 命令行参数解析
- 配置加载
- 存储后端初始化
- 服务启动（服务器/TUI 模式）

### internal/api/ [核心 API 层]

HTTP API 服务器实现：

- `server.go`: Gin 服务器配置、路由设置、中间件
- `handlers/management/`: 管理 API 端点
- `modules/amp/`: Amp CLI 集成模块

### internal/auth/ [认证层]

各提供商的 OAuth 认证实现：

- Claude (Anthropic)
- Codex (OpenAI)
- Gemini (Google)
- Qwen、iFlow、Kimi、Antigravity

### internal/store/ [存储层]

多后端存储支持：

- PostgreSQL: 企业级持久化
- Git: 版本控制的配置存储
- Object Store: MinIO/S3 兼容存储

### sdk/ [公共 SDK]

可嵌入的 SDK，允许将代理功能集成到其他应用：

- `api/handlers/`: OpenAI/Claude/Gemini 处理器
- `cliproxy/`: 核心代理功能
- `translator/`: API 格式转换

### internal/translator/ [格式转换器]

在 OpenAI/Claude/Gemini API 格式之间转换请求和响应。

---

## 文件统计

| 类别                  | 数量  |
| --------------------- | ----- |
| Go 源文件 (internal/) | ~120+ |
| Go 源文件 (sdk/)      | ~80+  |
| Go 源文件 (cmd/)      | 1     |
| 测试文件              | ~50+  |
| 配置文件              | 5     |
| 文档文件              | 8+    |

---

## 入口点

| 入口点      | 文件                               | 用途               |
| ----------- | ---------------------------------- | ------------------ |
| 主服务器    | `cmd/server/main.go`               | 启动代理服务器     |
| OAuth 登录  | `cmd/server/main.go -login`        | Google/Gemini 登录 |
| Codex 登录  | `cmd/server/main.go -codex-login`  | OpenAI Codex OAuth |
| Claude 登录 | `cmd/server/main.go -claude-login` | Claude OAuth       |
| TUI 模式    | `cmd/server/main.go -tui`          | 终端管理界面       |
