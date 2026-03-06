# CLIProxyAPI 项目概览

**生成日期:** 2026-03-06
**项目版本:** v6
**文档版本:** 1.0.0

---

## 项目简介

CLIProxyAPI 是一个代理服务器，为 CLI 工具提供 OpenAI/Gemini/Claude/Codex 兼容的 API 接口。它允许用户使用本地或多账户 CLI 访问与 OpenAI（包括 Responses API）/Gemini/Claude 兼容的客户端和 SDK。

### 核心功能

- **多提供商支持**: OpenAI、Gemini、Claude、Codex、Qwen、iFlow、Antigravity
- **OAuth 认证**: 支持 Claude Code、Codex、Gemini CLI 等的 OAuth 登录
- **多账户负载均衡**: 支持多个账户的轮询负载均衡
- **流式响应**: 支持流式和非流式响应
- **函数调用**: 支持工具/函数调用
- **多模态输入**: 支持文本和图像输入
- **API 格式转换**: 在 OpenAI/Claude/Gemini 格式之间自动转换
- **Amp CLI 集成**: 支持 Amp CLI 和 IDE 扩展

---

## 技术栈摘要

| 类别          | 技术                | 版本          |
| ------------- | ------------------- | ------------- |
| **语言**      | Go                  | 1.26.0        |
| **Web 框架**  | Gin                 | 1.10.1        |
| **WebSocket** | gorilla/websocket   | 1.5.3         |
| **TUI 框架**  | Bubble Tea          | 1.3.10        |
| **数据库**    | PostgreSQL (pgx/v5) | 5.7.6 (可选)  |
| **对象存储**  | MinIO               | 7.0.66 (可选) |
| **日志**      | logrus              | 1.9.3         |
| **JSON 处理** | gjson/sjson         | 1.18.0/1.2.5  |

---

## 架构类型

**类型:** API 代理服务 (Backend)
**模式:** 单体应用 (Monolith)
**部署:** Docker / 原生二进制

### 架构特点

1. **API 代理模式** - 作为中间层代理各种 AI API
2. **格式转换器** - 在 OpenAI/Claude/Gemini 格式之间转换
3. **多存储后端** - 支持文件系统、PostgreSQL、Git、对象存储
4. **模块化设计** - internal/ 私有代码 + sdk/ 公共 SDK

---

## 仓库结构

```
CLIProxyAPI/
├── cmd/                    # 应用入口点
│   └── server/            # 主服务器入口
├── internal/               # 私有应用代码
│   ├── api/               # API 处理器和服务器
│   ├── auth/              # 认证逻辑（多提供商）
│   ├── config/            # 配置管理
│   ├── store/             # 存储后端
│   ├── translator/        # API 格式转换
│   ├── tui/               # 终端 UI
│   └── ...
├── sdk/                    # 公共 SDK（可嵌入）
│   ├── api/               # SDK API 处理器
│   ├── auth/              # SDK 认证
│   ├── cliproxy/          # 核心 SDK
│   └── translator/        # SDK 格式转换
├── docs/                   # 文档
├── examples/               # 示例代码
├── test/                   # 集成测试
└── .github/                # GitHub 工作流
```

---

## 快速参考

| 项目         | 值                                                  |
| ------------ | --------------------------------------------------- |
| **入口点**   | `cmd/server/main.go`                                |
| **默认端口** | 8317                                                |
| **配置文件** | `config.yaml` (可选 `config.example.yaml` 作为模板) |
| **认证目录** | `auths/` (可配置)                                   |
| **许可证**   | MIT                                                 |

---

## 相关文档

- [源码树分析](./source-tree-analysis.md)
- [架构文档](./architecture.md)
- [API 契约](./api-contracts.md)
- [数据模型](./data-models.md)
- [开发指南](./development-guide.md)
- [部署指南](./deployment-guide.md)

---

## 外部资源

- **官方文档**: [https://help.router-for.me/](https://help.router-for.me/)
- **GitHub**: [https://github.com/router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
- **Docker Hub**: `eceasy/cli-proxy-api`
