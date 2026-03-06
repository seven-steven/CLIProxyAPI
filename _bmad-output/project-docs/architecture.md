# 架构文档

**生成日期:** 2026-03-06
**项目:** CLIProxyAPI v6

---

## 执行摘要

CLIProxyAPI 是一个 Go 语言实现的 API 代理服务器，采用模块化的单体架构。它作为中间层代理各种 AI 提供商的 API，提供格式转换、认证管理、负载均衡等功能。系统支持多种存储后端和部署模式。

---

## 技术栈

### 核心技术

| 类别          | 技术              | 版本   | 用途               |
| ------------- | ----------------- | ------ | ------------------ |
| **语言**      | Go                | 1.26.0 | 主要编程语言       |
| **Web 框架**  | Gin               | 1.10.1 | HTTP 服务器和路由  |
| **WebSocket** | gorilla/websocket | 1.5.3  | WebSocket 连接处理 |
| **TUI 框架**  | Bubble Tea        | 1.3.10 | 终端用户界面       |

### 数据存储

| 类别           | 技术                | 版本   | 用途               |
| -------------- | ------------------- | ------ | ------------------ |
| **关系数据库** | PostgreSQL (pgx/v5) | 5.7.6  | 可选的持久化存储   |
| **对象存储**   | MinIO               | 7.0.66 | 可选的对象存储后端 |

### 工具库

| 类别          | 技术             | 版本         | 用途               |
| ------------- | ---------------- | ------------ | ------------------ |
| **JSON 处理** | gjson/sjson      | 1.18.0/1.2.5 | 高性能 JSON 解析   |
| **日志**      | logrus           | 1.9.3        | 结构化日志         |
| **配置**      | yaml.v3          | 3.0.1        | YAML 配置解析      |
| **加密**      | bcrypt, utls     | -            | 密码哈希, TLS 指纹 |
| **OAuth**     | x/oauth2         | 0.30.0       | OAuth2 认证流程    |
| **压缩**      | brotli, compress | 1.0.6/1.17.4 | HTTP 响应压缩      |

---

## 架构模式

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIProxyAPI                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  HTTP API   │  │  WebSocket  │  │  TUI Mode   │             │
│  │  (Gin)      │  │  (gorilla)  │  │  (BubbleTea)│             │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘             │
│         │                │                                      │
│  ┌──────▼────────────────▼──────┐                              │
│  │      Authentication Layer     │                              │
│  │  (OAuth, API Key, Token Store)│                              │
│  └──────────────┬────────────────┘                              │
│                 │                                                │
│  ┌──────────────▼────────────────┐                              │
│  │      Format Translator        │                              │
│  │  (OpenAI ↔ Claude ↔ Gemini)   │                              │
│  └──────────────┬────────────────┘                              │
│                 │                                                │
│  ┌──────────────▼────────────────┐                              │
│  │      Provider Execution        │                              │
│  │  (Load Balancing, Retry)       │                              │
│  └──────────────┬────────────────┘                              │
│                 │                                                │
└─────────────────┼───────────────────────────────────────────────┘
                  │
    ┌─────────────┼─────────────┬─────────────┬─────────────┐
    │             │             │             │             │
    ▼             ▼             ▼             ▼             ▼
┌───────┐   ┌───────┐   ┌───────┐   ┌───────┐   ┌───────┐
│ Gemini│   │Claude │   │ Codex │   │ Qwen  │   │ iFlow │
│  API  │   │  API  │   │  API  │   │  API  │   │  API  │
└───────┘   └───────┘   └───────┘   └───────┘   └───────┘
```

### 核心设计模式

1. **代理模式**: 作为中间层代理 AI API 请求
2. **适配器模式**: 格式转换器适配不同 API 格式
3. **策略模式**: 多种存储后端和认证策略
4. **中间件链**: HTTP 请求处理管道

---

## 数据架构

### 认证数据模型

```go
type Auth struct {
    ID               string            // 唯一标识符
    Provider         string            // 提供商 (gemini, claude, codex, etc.)
    FileName         string            // 存储文件名
    Label            string            // 显示标签
    Status           AuthStatus        // 状态 (active, cooldown, disabled)
    Attributes       map[string]string // 附加属性
    Metadata         map[string]any    // 提供商特定元数据
    Storage          TokenStorage      // 令牌存储接口
    CreatedAt        time.Time
    UpdatedAt        time.Time
    LastRefreshedAt  time.Time
    NextRefreshAfter time.Time
}
```

### 配置数据模型

```yaml
# 主配置结构
port: 8317 # 服务端口
host: "" # 绑定地址
auth-dir: "auths" # 认证目录
debug: false # 调试模式

# API 密钥配置
gemini-key: [] # Gemini API 密钥列表
claude-key: [] # Claude API 密钥列表
codex-key: [] # Codex API 密钥列表

# OpenAI 兼容配置
openai-compatibility: [] # 兼容提供商配置

# 远程管理
remote-management:
  secret-key: "" # 管理密码
  disable-control-panel: false

# Amp 集成
ampcode:
  upstream-url: "" # Amp 上游 URL
  model-mappings: [] # 模型映射规则
```

---

## API 设计

### OpenAI 兼容端点

| 方法     | 路径                        | 描述                 |
| -------- | --------------------------- | -------------------- |
| GET      | `/v1/models`                | 列出可用模型         |
| POST     | `/v1/chat/completions`      | 聊天补全             |
| POST     | `/v1/completions`           | 文本补全             |
| POST     | `/v1/messages`              | Claude 消息          |
| POST     | `/v1/messages/count_tokens` | Token 计数           |
| GET/POST | `/v1/responses`             | OpenAI Responses API |

### Gemini 兼容端点

| 方法 | 路径                     | 描述             |
| ---- | ------------------------ | ---------------- |
| GET  | `/v1beta/models`         | 列出 Gemini 模型 |
| POST | `/v1beta/models/*action` | Gemini 操作      |

### 管理 API 端点

| 方法 | 路径                         | 描述         |
| ---- | ---------------------------- | ------------ |
| GET  | `/v0/management/config`      | 获取配置     |
| PUT  | `/v0/management/config.yaml` | 更新配置     |
| GET  | `/v0/management/auth-files`  | 列出认证文件 |
| GET  | `/v0/management/usage`       | 使用统计     |
| GET  | `/v0/management/logs`        | 日志查看     |

### OAuth 回调端点

| 路径                    | 提供商       |
| ----------------------- | ------------ |
| `/anthropic/callback`   | Claude       |
| `/codex/callback`       | OpenAI Codex |
| `/google/callback`      | Gemini       |
| `/iflow/callback`       | iFlow        |
| `/antigravity/callback` | Antigravity  |

---

## 组件结构

### internal/api/ - API 层

**server.go**: Gin 服务器主实现

- 路由设置
- 中间件配置
- CORS 处理
- TLS 支持

**handlers/**: API 处理器

- 管理 API 处理器
- OAuth 回调处理
- 配置管理

**modules/amp/**: Amp CLI 集成模块

- 模型映射
- 请求代理
- 响应重写

### internal/auth/ - 认证层

每个提供商的认证实现：

- OAuth 流程
- Token 刷新
- PKCE 支持
- 会话管理

### internal/store/ - 存储层

**PostgresStore**: PostgreSQL 后端

- 配置和认证持久化
- 本地镜像同步

**GitTokenStore**: Git 后端

- 版本控制配置存储
- 自动提交

**ObjectTokenStore**: 对象存储后端

- MinIO/S3 兼容
- 分布式存储

### sdk/ - 公共 SDK

可嵌入的 SDK 组件：

- API 处理器工厂
- 认证管理器
- 格式转换器
- 使用追踪

---

## 测试策略

### 测试文件分布

- **单元测试**: 与源文件同目录 (`*_test.go`)
- **集成测试**: `test/` 目录
- **测试覆盖**: 约 50+ 测试文件

### 测试类型

1. **API 处理器测试**: 请求/响应验证
2. **认证测试**: OAuth 流程模拟
3. **转换器测试**: 格式转换验证
4. **存储测试**: 后端操作验证

---

## 部署架构

### Docker 部署

```dockerfile
# 多阶段构建
FROM golang:1.26-alpine AS builder
# 构建二进制文件

FROM alpine:3.22.0
# 运行时镜像
EXPOSE 8317
```

### 支持的平台

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

### CI/CD 流程

1. **PR 测试**: 构建验证
2. **Docker 构建**: 多架构镜像
3. **GoReleaser**: 发布二进制文件

---

## 安全考虑

### 认证

- Bearer Token 认证
- API Key 验证
- OAuth 2.0 流程
- 本地管理密码

### 网络安全

- TLS 支持
- CORS 配置
- Localhost 限制（管理端点）

### 数据安全

- 敏感数据加密存储
- 环境变量配置
- 文件权限控制
