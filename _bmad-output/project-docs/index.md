# CLIProxyAPI 项目文档索引

**生成日期:** 2026-03-06
**项目版本:** v6

---

## 项目概览

- **类型:** 单体后端服务
- **主要语言:** Go 1.26.0
- **架构:** API 代理 + 格式转换器

CLIProxyAPI 是一个代理服务器，为 CLI 工具提供 OpenAI/Gemini/Claude/Codex 兼容的 API 接口。

---

## 快速参考

### 技术栈

| 类别      | 技术                    |
| --------- | ----------------------- |
| 语言      | Go 1.26.0               |
| Web 框架  | Gin 1.10.1              |
| WebSocket | gorilla/websocket 1.5.3 |
| TUI       | Bubble Tea 1.3.10       |
| 数据库    | PostgreSQL (pgx/v5)     |
| 对象存储  | MinIO                   |

### 入口点

| 入口     | 文件                 |
| -------- | -------------------- |
| 主服务器 | `cmd/server/main.go` |
| 默认端口 | 8317                 |

---

## 生成的文档

### 核心文档

| 文档                                    | 描述                    |
| --------------------------------------- | ----------------------- |
| [项目概览](./project-overview.md)       | 项目简介和功能概述      |
| [源码树分析](./source-tree-analysis.md) | 目录结构和文件组织      |
| [架构文档](./architecture.md)           | 系统架构和设计模式      |
| [API 契约](./api-contracts.md)          | API 端点和请求/响应格式 |
| [数据模型](./data-models.md)            | 核心数据结构和存储模式  |
| [开发指南](./development-guide.md)      | 本地开发和测试指南      |
| [部署指南](./deployment-guide.md)       | 部署和运维指南          |

---

## 现有 SDK 文档

原始 SDK 文档位于 `docs/` 目录：

| 文档                                                 | 描述                      |
| ---------------------------------------------------- | ------------------------- |
| [SDK 使用指南](../../docs/sdk-usage.md)              | SDK 基本使用方法          |
| [SDK 使用指南 (中文)](../../docs/sdk-usage_CN.md)    | SDK 基本使用方法 (中文版) |
| [SDK 高级用法](../../docs/sdk-advanced.md)           | 执行器和转换器            |
| [SDK 高级用法 (中文)](../../docs/sdk-advanced_CN.md) | 执行器和转换器 (中文版)   |
| [SDK 访问控制](../../docs/sdk-access.md)             | 访问控制配置              |
| [SDK 访问控制 (中文)](../../docs/sdk-access_CN.md)   | 访问控制配置 (中文版)     |
| [SDK 监控](../../docs/sdk-watcher.md)                | 文件监控功能              |
| [SDK 监控 (中文)](../../docs/sdk-watcher_CN.md)      | 文件监控功能 (中文版)     |

---

## 入门指南

### 1. 快速开始

```bash
# 克隆仓库
git clone https://github.com/router-for-me/CLIProxyAPI.git
cd CLIProxyAPI

# 创建配置
cp config.example.yaml config.yaml

# 运行
go run ./cmd/server
```

### 2. OAuth 登录

```bash
# Gemini 登录
./CLIProxyAPI -login

# Claude 登录
./CLIProxyAPI -claude-login

# Codex 登录
./CLIProxyAPI -codex-login
```

### 3. 测试 API

```bash
curl http://localhost:8317/v1/models \
  -H "Authorization: Bearer your-api-key"
```

---

## API 端点概览

### OpenAI 兼容

- `POST /v1/chat/completions` - 聊天补全
- `POST /v1/completions` - 文本补全
- `GET /v1/models` - 模型列表
- `POST /v1/messages` - Claude 消息
- `POST /v1/responses` - OpenAI Responses

### Gemini 兼容

- `GET/POST /v1beta/models/*` - Gemini API

### 管理 API

- `GET/PUT /v0/management/config.yaml` - 配置管理
- `GET /v0/management/auth-files` - 认证文件管理
- `GET /v0/management/usage` - 使用统计

---

## 关键目录

```
CLIProxyAPI/
├── cmd/server/          # 主入口
├── internal/
│   ├── api/            # HTTP API
│   ├── auth/           # 认证提供商
│   ├── store/          # 存储后端
│   └── translator/     # 格式转换
├── sdk/                 # 公共 SDK
├── docs/                # SDK 文档
└── _bmad-output/        # BMAD 生成的文档
    └── project-docs/    # 项目文档
```

---

## 支持的提供商

| 提供商      | 认证方式       | 模型示例                         |
| ----------- | -------------- | -------------------------------- |
| OpenAI      | API Key, OAuth | gpt-4o, o1-preview               |
| Claude      | API Key, OAuth | claude-sonnet-4, claude-opus-4   |
| Gemini      | API Key, OAuth | gemini-2.5-pro, gemini-2.0-flash |
| Codex       | OAuth          | codex-1                          |
| Qwen        | OAuth          | qwen-\*                          |
| iFlow       | OAuth, Cookie  | iflow-\*                         |
| Antigravity | OAuth          | antigravity-\*                   |
| Vertex AI   | 服务账号       | vertex-\*                        |

---

## 外部资源

- **在线文档**: [https://help.router-for.me/](https://help.router-for.me/)
- **GitHub**: [https://github.com/router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
- **Docker Hub**: `eceasy/cli-proxy-api`

---

## AI 辅助开发指南

当使用 AI 工具（如 Claude Code）进行开发时：

1. **参考文档**: 将本文档索引作为上下文入口
2. **架构理解**: 先阅读 [架构文档](./architecture.md) 了解系统设计
3. **API 开发**: 参考 [API 契约](./api-contracts.md) 了解端点格式
4. **数据操作**: 查看 [数据模型](./data-models.md) 了解数据结构

---

_此文档由 BMAD 项目文档化工作流自动生成_
