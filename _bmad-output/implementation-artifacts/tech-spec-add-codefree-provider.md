---
title: "添加 Codefree Provider"
slug: "add-codefree-provider"
created: "2026-03-09"
status: "code-review-fixed"
stepsCompleted: [1, 2, 3, 4, 5]
codeReviewIssues:
  - issue: "Config field naming inconsistency"
    severity: "HIGH"
    fix: "Renamed CodefreeCLIVersion to CodefreeCliVersion (Go idiomatic)"
  - issue: "Missing executor tests"
    severity: "HIGH"
    fix: "Added codefree_executor_test.go with 6 test functions"
  - issue: "Missing OAuth server tests"
    severity: "HIGH"
    fix: "Added oauth_server_test.go with 17 test functions"
  - issue: "Hardcoded CLI in URL"
    severity: "MEDIUM"
    fix: "Changed ModelsURLFormat to use ProjectID constant"
  - issue: "Redundant generateRandomCode wrapper"
    severity: "LOW"
    fix: "Removed wrapper, use codefree.GenerateRandomCode directly"
tech_stack:
  - Go 1.21+
  - OAuth2
  - HTTP Client
  - SSE Streaming
files_to_modify:
  - "internal/auth/codefree/codefree_auth.go (新建)"
  - "internal/auth/codefree/token.go (新建)"
  - "internal/auth/codefree/oauth_server.go (新建)"
  - "internal/auth/codefree/filename.go (新建)"
  - "internal/auth/codefree/errors.go (新建)"
  - "internal/watcher/synthesizer/file.go (修改)"
  - "internal/runtime/executor/codefree_executor.go (新建)"
  - "internal/runtime/executor/registry.go (修改)"
  - "internal/config/config.go (修改)"
  - "internal/cmd/codefree_login.go (新建)"
  - "internal/cmd/codefree_refresh.go (新建)"
code_patterns:
  - "Provider 识别: auth 文件 type 字段 → provider 名称"
  - "Credentials 映射: auth.Attributes[api_key/base_url/header:X]"
  - "Executor 复用: OpenAI 兼容 API 复用 openai_compat_executor"
  - "Token 存储: TokenStorage 结构 + SaveTokenToFile 方法"
test_patterns:
  - "单元测试: internal/auth/codefree/*_test.go"
  - "集成测试: synthesizer/file_test.go 扩展"
  - "E2E 测试: executor 测试"
---

# Tech-Spec: 添加 Codefree Provider

**Created:** 2026-03-09

## Overview

### Problem Statement

需要添加一个新的 Provider 来支持 CodeFree AI 服务。该服务提供 GLM-4.7、DeepSeek-V3.1-Terminus、GLM-4.6V 等模型。用户通过 OAuth 登录获取凭据文件，或直接提供预先获取的凭据文件，应用从中解析认证信息并转发请求。

### Solution

1. 实现 `internal/auth/codefree/` 认证模块，支持 OAuth 登录（本地回调服务器）和凭据文件解析
2. 创建 `codefree_executor` 处理 CodeFree 特定的 headers（包装 `openai_compat_executor`）
3. 实现模型列表刷新功能（用户主动触发），调用 CodeFree 模型管理 API

### Scope

**In Scope:**

- OAuth 认证流程（本地回调服务器）
- 凭据文件解析（从 `codefree.json` 格式提取 `apikey`, `access_token`, `id_token`）
- 请求转发到 `https://www.srdcloud.cn/api/acbackend/codechat/v1/completions`
- 模型列表刷新（用户主动触发，调用 `/api/acbackend/modelmgr/v1/clients/CLI/versions/{version}`）
- CLI 版本配置项（用于模型列表 API）
- CodeFree 特定 headers 注入 (`apiKey`, `userId`, `modelName`, `clientType`)

**Out of Scope:**

- Token 自动刷新（服务端不提供 refresh token）
- 非 OpenAI 兼容的 API 格式转换

## Context for Development

### Codebase Patterns

**Provider 识别机制:**

- Auth 文件通过 `type` 字段识别 provider（如 `"type": "codefree"`）
- `synthesizer/file.go` 的 `synthesizeFileAuths` 函数处理文件解析
- Provider 名称自动转为小写
  **Credentials 映射模式:**

```go
// auth.Attributes 中存储凭据
auth.Attributes["api_key"] = "xxx"           // API 密钥
auth.Attributes["base_url"] = "https://..."  // 基础 URL
auth.Attributes["user_id"] = "267306"        // userId（用于 header）
auth.Attributes["header:clientType"] = "codefree-cli"  // 固定 header
```

**Executor 复用策略:**

- `openai_compat_executor.go` 的 `resolveCredentials` 从 `auth.Attributes` 获取凭据
- `PrepareRequest` 方法注入 `Authorization: Bearer` header
- `ApplyCustomHeadersFromAttrs` 应用 `header:` 前缀的自定义 headers
  **CodeFree 特殊需求:**
- 需要额外的 headers: `apiKey`, `userId`, `modelName`, `clientType`
- `modelName` header 需要从请求体动态提取（失败时跳过）

### Files to Reference

| File                                                  | Purpose                           | 修改类型 |
| ----------------------------------------------------- | --------------------------------- | -------- |
| `internal/watcher/synthesizer/file.go`                | Auth 文件解析，添加 codefree 处理 | 修改     |
| `internal/runtime/executor/openai_compat_executor.go` | OpenAI 兼容执行器参考             | 参考     |
| `internal/runtime/executor/codefree_executor.go`      | CodeFree 执行器，处理特殊 headers | **新建** |
| `internal/auth/claude/oauth_server.go`                | OAuth 回调服务器参考              | 参考     |
| `internal/auth/claude/token.go`                       | Token 存储结构参考                | 参考     |
| `internal/auth/claude/anthropic_auth.go`              | OAuth 认证流程参考                | 参考     |
| `internal/config/config.go`                           | 添加 CodefreeCLIVersion 配置      | 修改     |
| `internal/cmd/anthropic_login.go`                     | Login 命令参考                    | 参考     |
| `docs/codefree/codefree.har`                          | API 请求链路分析                  | 参考     |
| `docs/codefree/oauth_creds.json`                      | OAuth 凭据格式示例                | 参考     |

### Technical Decisions

| 决策项         | 结论                                                           | 理由                                                                                                 |
| -------------- | -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Executor       | 创建 `codefree_executor` **组合**持有 `openai_compat_executor` | 需要注入额外的 headers (`apiKey`, `userId`, `modelName`, `clientType`)，组合优于嵌入避免方法集复杂性 |
| 认证方式       | OAuth 登录 → 生成含 `apikey` 的凭据文件                        | 符合现有模式                                                                                         |
| baseUrl        | 固定为 `https://www.srdcloud.cn`                               | 简化配置                                                                                             |
| 模型列表       | 用户主动刷新时获取                                             | 避免不必要的 API 调用                                                                                |
| CLI 版本       | 配置项 `codefree.cli_version`，默认 `"0.3.4"`                  | 灵活支持版本更新                                                                                     |
| 多语言         | 跟随现有 `internal/tui/i18n.go` 设计                           | 保持一致性                                                                                           |
| 错误处理       | Token 过期时提示用户重新登录                                   | 无 refresh token                                                                                     |
| OAuth 参数     | `client_id`, `client_secret` 硬编码                            | 与现有实现一致                                                                                       |
| 模型名提取失败 | 跳过 header 设置                                               | 简化处理逻辑                                                                                         |
| 端口冲突处理   | 引导用户手动输入 OAuth code                                    | 避免复杂的端口重试逻辑，用户体验更可控                                                               |
| 凭据文件写入   | 原子写入（temp file + rename）                                 | 防止部分写入导致凭据损坏                                                                             |
| 凭据文件校验   | 必填字段校验 (`type`, `apikey`, `id_token`, `baseUrl`)         | 提前发现格式问题，提供清晰错误提示                                                                   |
| access_token   | 使用 `ori_session_id` (UUID) 而非 JWT 格式的 `access_token`    | 官方客户端使用 `ori_session_id`，API Key 接口需要 UUID 格式的 sessionId                              |
| projectId      | API Key 接口用 `"0"`，模型列表 URL 用 `"CLI"`                  | 两个接口使用不同的 projectId 值，参考 HAR 文件分析                                                   |
| 过期检测       | 基于 `expires_in` + `created_at` 计算，不主动阻止请求          | 过期后服务端返回 401，由 executor 检测并提示                                                         |
| 模型刷新命令   | 独立的 `codefree-refresh` CLI 命令                             | 用户主动触发，避免不必要的 API 调用                                                                  |
| 401 检测位置   | 在 `CodefreeExecutor.execute` 方法中检测                       | 返回特定错误类型 `ErrTokenExpired`，上层处理提示                                                     |
| 凭据文件名     | `{auth_dir}/codefree.json`                                     | 与其他 provider 保持一致，固定文件名                                                                 |
| 网络重试策略   | 不自动重试，直接返回错误                                       | 网络问题由用户决定是否重试；5xx 错误记录日志但不重试                                                 |
| 多账户支持     | 单文件模式，不支持多账户                                       | 简化实现；如需多账户可后续扩展                                                                       |
| SSE 流超时     | 复用 `openai_compat_executor` 的超时设置                       | 与现有实现保持一致                                                                                   |

## Implementation Plan

### Tasks

**Phase 1: Auth 模块 (internal/auth/codefree/)**

- [x] **Task 1**: 创建 `internal/auth/codefree/filename.go`
  - File: `internal/auth/codefree/filename.go`
  - Action: 创建 `CodefreeCredentialsFilename` 常量，值为 `"codefree.json"`
  - Notes: 参考 `claude/filename.go`，与其他 provider 保持命名一致
- [x] **Task 1.1**: 创建 `internal/auth/codefree/errors.go`
  - File: `internal/auth/codefree/errors.go`
  - Action: 定义错误类型:
    - `ErrTokenExpired`: Token 过期错误
    - `ErrInvalidCredentials`: 凭据格式错误
    - `ErrAuthTimeout`: OAuth 超时错误
  - Notes: 使用 `errors.Is()` 兼容的模式
- [x] **Task 2**: 创建 `internal/auth/codefree/token.go`
  - File: `internal/auth/codefree/token.go`
  - Action: 创建 `CodefreeTokenStorage` 结构体
  - Action: 实现 `saveTokenToFile` 方法（**原子写入**: 写入临时文件后 rename，避免部分写入导致凭据损坏）
  - Action: **文件权限**: 设置凭据文件权限为 `0600`（仅所有者可读写）
  - Notes: 字段: `type`, `access_token`, `id_token` (userId), `apikey`, `expires_in`, `baseUrl`, `created_at` (写入时自动设置)
  - Action: 实现 `Validate` 方法（校验必填字段: `type`, `apikey`, `id_token`, `baseUrl`）
  - Action: 实现 `IsExpired` 方法（基于 `expires_in` 和创建时间计算是否过期）
  - Action: 实现 `GetExpiresAt` 方法（返回过期时间的 Unix 时间戳）
  - Action: 定义常量: `DefaultCredentialsFilename`, `CodefreeCredentialsFile`
  - Notes: 字段: `type`, `access_token`, `id_token` (userId), `apikey`, `expires_in`, `baseUrl`, `created_at` (写入时自动设置)
- [x] **Task 3**: 创建 `internal/auth/codefree/oauth_server.go`
  - File: `internal/auth/codefree/oauth_server.go`
  - Action: 创建 `CodefreeOAuthServer` 结构体
  - Action: 实现 `Start` 方法（启动回调服务器）
    - 使用随机可用端口（非固定端口）
    - 监听路径: `/oauth2callback`
    - 生成 `randomCode` 用于安全验证
  - Action: 实现 `waitForCallback` 方法
    - **randomCode 验证**:
      - 使用 `sync.Mutex` 保护 `expectedRandomCode` 字段
      - 验证回调中的 `randomCode` 参数与 `expectedRandomCode` 匹配
    - 提取 `code` 参数
  - Action: 实现 `stop` 方法（关闭服务器、停止 goroutine）
  - Action: 实现 `stop` 方法
  - Action: 实现 `BuildStateParam` 方法（构造 base64 编码的 state 参数）
  - Action: **手动模式 fallback**: 若用户选择 `--manual`，跳过回调服务器，直接提示输入 code
  - Notes: 参考 `claude/oauth_server.go`，回调地址格式 `http://127.0.0.1:{random_port}/oauth2callback`
- [x] **Task 4**: 创建 `internal/auth/codefree/codefree_auth.go`
  - File: `internal/auth/codefree/codefree_auth.go`
  - Action: 创建 `CodefreeAuth` 结构体
  - Action: 定义 `ModelInfo` 结构体（字段: `Name`, `Manufacturer`, `MaxTokens`）
  - Action: 定义 OAuth 常量: `clientID`, `clientSecret`, `redirectURI`
  - Action: 定义 `APIKeyProjectID = "0"` 常量（用于 API Key 接口）
  - Action: 实现 `GenerateAuthURL(state string)` 方法
    - 返回完整授权 URL（包含 client_id, redirect_uri, state 等参数）
  - Action: 实现 `ExchangeCodeForTokens` 方法（OAuth code → access_token + id_token）
    - **重要**: 使用 `ori_session_id` (UUID) 作为 `access_token`，而非 JWT 格式的 `access_token`
  - Action: 实现 `FetchAPIKey` 方法（**关键步骤**: 调用 `/api/acbackend/usermanager/v1/users/apikey` 获取 encryptedApiKey）
    - Headers: `sessionId` (ori_session_id), `userId` (id_token/uid), `projectId` (固定值 `"0"`)
  - Action: 实现 `FetchModels` 方法（获取模型列表）
    - 返回值: `[]ModelInfo`
    - 使用配置的 `codefree.cli_version` 构造 API 路径
  - Notes: OAuth 常量硬编码，API Key 接口 `projectId` 为 `"0"`，模型列表 URL 使用 `"CLI"`
    **Phase 2: Synthesizer 集成**
- [x] **Task 5**: 修改 `internal/watcher/synthesizer/file.go`
  - File: `internal/watcher/synthesizer/file.go`
  - Action: 在 `synthesizeFileAuths` 函数中添加 codefree provider 处理
  - Action: 映射 `apikey` → `auth.Attributes["api_key"]`
  - Action: 映射 `id_token` → `auth.Attributes["user_id"]`
  - Action: 映射 `baseUrl` → `auth.Attributes["base_url"]`
  - Action: 设置 `header:clientType` = `codefree-cli`
  - Notes: 添加 `case "codefree":` 处理逻辑
    **Phase 3: Executor 实现**
- [x] **Task 6**: 创建 `internal/runtime/executor/codefree_executor.go`
  - File: `internal/runtime/executor/codefree_executor.go`
  - Action: 创建 `CodefreeExecutor` 结构体，**组合持有** `*Openai_compat_executor`（而非嵌入）
  - Action: 实现 `Identifier()` 方法，返回 `"codefree"`
  - Action: 实现 `PrepareRequest` 方法
    - 从 `auth.Attributes` 获取 `api_key`, `user_id`
    - 从请求体提取 `modelName`（使用 gjson，失败时静默跳过）
    - 注入 headers: `apiKey`, `userId`, `modelName`, `clientType`
  - Action: 实现 `execute` 方法（委托给底层 executor）
  - Action: **401 检测**: 在 execute 方法中检测 401 响应，返回特定错误类型 `ErrTokenExpired`
    - `ErrTokenExpired` 定义在 `internal/auth/codefree/errors.go` 中
  - Notes: 组合模式避免 Go 嵌入的方法集复杂性，更清晰的委托关系
- [x] **Task 6.1**: 注册 CodefreeExecutor
  - File: `internal/runtime/executor/registry.go`（或对应的注册文件）
  - Action: 在 executor 注册表中添加 codefree provider → CodefreeExecutor 的映射
  - Notes: 参考 `openai_compat_executor` 的注册方式
    **Phase 4: 配置与命令**
- [x] **Task 7**: 修改 `internal/config/config.go`
  - File: `internal/config/config.go`
  - Action: 添加 `CodefreeCLIVersion` 字段
  - Action: 添加默认值 `"0.3.4"`
  - Notes: 添加到 `Config` 结构体
- [x] **Task 8**: 创建 `internal/cmd/codefree_login.go`
  - File: `internal/cmd/codefree_login.go`
  - Action: 创建 `codefree-login` 命令
  - Action: 实现 `run` 方法（触发 OAuth 流程）
  - Action: 支持命令行参数:
    - `--manual`: 跳过自动回调，手动输入 OAuth code
  - Action: **浏览器自动打开**:
    - 使用 Go 的 `os/exec` 包
    - Linux: `xdg-open`
    - macOS: `open`
    - Windows: `cmd /c start`
    - 打开失败时输出 URL 让用户手动访问
  - Action: **OAuth 流程**:
    1. 启动本地回调服务器（随机端口）
    2. 生成 randomCode，构造 state 参数
    3. 生成授权 URL，打开浏览器
    4. 等待回调或超时（默认 5 分钟）
    5. 回调成功后获取 code
    6. 调用 ExchangeCodeForTokens
    7. 调用 FetchAPIKey
    8. 保存凭据到 `{auth_dir}/codefree.json`
  - Action: **手动输入 code 流程**（`--manual` 模式）:
    - 输出授权 URL 让用户手动访问
    - 提示用户授权后从服务端页面复制 code
    - 读取用户输入的 code，继续 OAuth 流程
  - Action: 集成到现有的命令系统
  - Notes: 参考 `anthropic_login.go`
- [x] **Task 8.1**: 创建 `internal/cmd/codefree_refresh.go`
  - File: `internal/cmd/codefree_refresh.go`
  - Action: 创建 `codefree-refresh` 命令（触发模型列表刷新）
  - Action: **认证方式**: 从 `{auth_dir}/codefree.json` 读取凭据
  - Action: 凭据不存在时提示用户先执行 `codefree-login`
  - Action: 调用 `FetchModels` 方法，将返回的模型列表更新到注册表
    - 调用 `internal/registry` 包中的模型注册函数（参考现有实现）
  - Action: 输出刷新结果（新增/更新的模型数量）
  - Action: **命令帮助文本**:
    - Short: `CodeFree refresh model list`
    - Usage: `codefree-refresh`
  - Notes: 用户主动触发的模型列表刷新功能

### Acceptance Criteria

**OAuth 登录**

- [ ] AC1: Given 用户启动 `codefree-login` 命令，when OAuth 流程完成，then access token 已保存到 `{auth_dir}/codefree.json`
- [ ] AC2: Given 有效的凭据文件，when 解析凭据文件时，then `auth.Attributes` 应包含正确的 `api_key`, `user_id`, `base_url`
- [ ] AC3: Given 凭据文件存在，when 发送 API 请求时，then `codefree_executor` 正确注入以下 headers:
  - `apiKey` (从 `auth.Attributes["api_key"]`)
  - `userId` (从 `auth.Attributes["user_id"]`)
  - `modelName` (从请求体提取)
  - `clientType` (固定值 `codefree-cli`)
- [ ] AC4: Given 用户触发模型列表刷新，when 调用 `FetchModels` 方法时，then 返回有效的模型列表并更新到注册表
- [ ] AC5: Given Token 过期（检测到 401 响应），then 返回友好的错误提示，引导用户重新登录
- [ ] AC6: Given 请求体无 `model` 字段时，then 跳过 `modelName` header 设置，请求仍能正常发送
- [ ] AC7: Given OAuth 回调端口被占用，when 启动登录命令时，then 提示用户手动输入 OAuth code
- [ ] AC8: Given 凭据文件缺失必填字段 (`type`, `apikey`, `id_token`, `baseUrl`)，when 解析文件时，then 返回清晰的错误提示说明缺失字段

## Additional Context

### Dependencies

- 现有 auth 模块基础设施（`internal/auth/claude/` 参考）
- `openai_compat_executor` 执行器
- 本地回调服务器基础设施
- i18n 系统 (`internal/tui/i18n.go`)
- `github.com/tidwall/gjson` - JSON 解析

- `internal/util` 包 (ResolveAuthDir 函数)

**auth_dir 路径获取:**

- 通过 `internal/util.ResolveAuthDir(cfg.AuthDir)` 函数获取
- 参考 `internal/watcher/clients.go:80` 中的用法
- 默认值: `{config_dir}/auth`（可通过 `--config` 参数或 `~/.codefree` 茳覆盖）

- 默认值: `{auth_dir}/.codefree`（文件名固定为 `codefree.json`）

**配置示例 (`~/.codefree/config.yaml`):**

```yaml
# 添加到现有配置文件中
codefree:
  cli_version: "0.3.4"
```

**API 错误处理:**

- 4xx 错误: 记录日志，返回原始错误信息
- 连接错误: 包装为友好错误消息， 使用 i18n
- 超时: 取消 context，返回超时错误
- 其他错误: 直接返回，让上层处理

**凭据文件权限:**

- 保存时设置文件权限为 `0600` (仅所有者可读写)
- 防止凭据泄露- 现有 auth 模块基础设施（`internal/auth/claude/` 参考）
- `openai_compat_executor` 执行器
- 本地回调服务器基础设施
- i18n 系统 (`internal/tui/i18n.go`)
- `github.com/tidwall/gjson` - JSON 解析
- 现有 auth 模块基础设施
- `openai_compat_executor` 执行器
- 本地回调服务器基础设施
- i18n 系统
- `github.com/tidwall/gjson`

- **配置示例**（添加到 `config.yaml`）:
  ```yaml
  codefree:
    cli_version: "0.3.4"
  ```

**auth_dir 路径获取:**

- 通过 `internal/util.ResolveAuthDir(cfg.AuthDir)` 函数获取
- 参考 `internal/watcher/clients.go:80` 中的用法

### Testing Strategy

**单元测试:**

- `internal/auth/codefree/token_test.go` - Token 存储测试
- `internal/auth/codefree/codefree_auth_test.go` - OAuth 流程测试
- `internal/runtime/executor/codefree_executor_test.go` - Executor 测试
  **集成测试:**
- `internal/watcher/synthesizer/file_test.go` - 添加 codefree 文件解析测试
  **E2E 测试:**
- 完整 OAuth 流程（mock server）
- API 请求转发测试（需要 mock server）

**Mock Server 配置:**

- 使用 `httptest` 包创建本地 mock server
- Mock 端点:
  - `GET /login/oauth/access_token` - 返回模拟 token 响应
  - `GET /api/acbackend/usermanager/v1/users/apikey` - 返回模拟 apikey
  - `GET /api/acbackend/modelmgr/v1/clients/CLI/versions/{version}` - 返回模拟模型列表
  - `POST /api/acbackend/codechat/v1/completions` - 返回模拟 SSE 流式响应
- 测试时通过设置 `baseUrl` 指向 mock server

### Notes

**CodeFree API 关键信息:**

1. **OAuth Token 端点**: `GET https://www.srdcloud.cn/login/oauth/access_token`
   - 参数: `grant_type=authorization_code`, `client_id`, `client_secret`, `code`, `redirect_uri`
   - 返回字段:
     - `access_token`: JWT 格式（**不使用**）
     - `ori_session_id`: UUID 格式（**实际使用的 access_token**）
     - `id_token`: 用户 ID（可能为空，需 fallback 到 `uid` 字段）
     - `uid`: 用户 ID（当 `id_token` 为空时使用）
     - `ori_token`: 另一个 UUID 格式 token
     - `expires_in`: 过期时间（秒）
   - **重要**: 保存凭据时，`access_token` 字段应使用 `ori_session_id` 的值（UUID 格式），而非 JWT 格式的 `access_token`
2. **API Key 获取**: `GET https://www.srdcloud.cn/api/acbackend/usermanager/v1/users/apikey`
   - Headers: `sessionId` (ori_session_id), `userId` (id_token/uid), `projectId` (固定值 `"0"`)
   - 返回: `encryptedApiKey`
   - **此步骤必须在 OAuth token 交换后立即执行**
   - **注意**: `projectId` 必须是 `"0"`，不是 `"CLI"`
3. **模型列表**: `GET https://www.srdcloud.cn/api/acbackend/modelmgr/v1/clients/CLI/versions/{version}`
   - Headers: `userId`, `apiKey`
   - 返回: `{"data":[{"modelName":"...","manufacturer":"...","maxTokens":...}],...}`
   - 版本号从配置项 `codefree.cli_version` 获取
4. **Completions**: `POST https://www.srdcloud.cn/api/acbackend/codechat/v1/completions`
   - Headers: `apiKey`, `userId`, `modelName`, `clientType: codefree-cli`, `authorization: Bearer codefree`
   - Body: OpenAI 兼容格式 `{"model":"...","messages":[...]}`
   - Response: OpenAI 兼容 SSE 流式格式

**OAuth 详细配置:**

- **client_id**: `250320a317ba7dcvofvm`（硬编码）
- **client_secret**: `250320a318034D2xWtybKuEiSsjxzhWvKrMtgIRA`（硬编码）
- **redirect_uri**: `https://www.srdcloud.cn/login/oauth-srdcloud-redirect`
- **本地回调机制**: 通过 `state` 参数传递本地回调地址（base64 编码）
  - state 格式: `base64(http://127.0.0.1:{port}/oauth2callback?from=cli&randomCode={randomCode})`
  - 示例: `aHR0cDovLzEyNy4wLjAuMTo0NTQ0MS9vYXV0aDJjYWxsYmFjaz9mcm9tPWNsaSZyYW5kb21Db2RlPTc3MjU=`
  - 解码后: `http://127.0.0.1:45441/oauth2callback?from=cli&randomCode=7725`
- **授权 URL**: `https://www.srdcloud.cn/login/oauth/authorize`
  - 完整示例: `https://www.srdcloud.cn/login/oauth/authorize?client_id=250320a317ba7dcvofvm&redirect_uri=https%3A%2F%2Fwww.srdcloud.cn%2Flogin%2Foauth-srdcloud-redirect&response_type=code&state={base64_state}`
  - 参数说明:
    - `client_id`: `250320a317ba7dcvofvm`
    - `redirect_uri`: `https://www.srdcloud.cn/login/oauth-srdcloud-redirect` (URL 编码)
    - `response_type`: `code`
    - `state`: base64 编码的本地回调地址
- **回调端口**: 随机可用端口（如 45441）
- **回调路径**: `/oauth2callback`
- **安全验证**: `randomCode` 参数用于验证回调合法性

**OAuth 完整流程:**

```
1. 用户执行 codefree-login 命令
2. 启动本地回调服务器（随机端口，如 45441）
   - 监听路径: /oauth2callback
   - 生成 randomCode 用于验证
3. 构造 state 参数 = base64(http://127.0.0.1:{port}/oauth2callback?from=cli&randomCode={code})
4. 打开浏览器访问授权 URL（包含 state 参数）
5. 用户授权后，CodeFree 重定向到 redirect_uri
6. CodeFree 服务端解析 state，向本地回调服务器发送请求
7. 本地服务器验证 randomCode，提取 code 参数
8. 调用 OAuth Token 端点 → 获取 ori_session_id (作为 access_token), id_token/uid
9. 调用 API Key 端点 (projectId="0") → 获取 apikey
10. 保存完整凭据到 {auth_dir}/codefree.json
```

     **凭据文件格式 (`codefree.json`):**

```json
{
  "access_token": "ori_session_id (UUID格式，非JWT)",
  "token_type": "bearer",
  "expires_in": 86400,
  "id_token": "userId",
  "baseUrl": "https://www.srdcloud.cn",
  "apikey": "encrypted-api-key",
  "type": "codefree",
  "created_at": 1709875200
}
```

**关键代码路径:**

```
auth 文件 (type: codefree)
    ↓
synthesizer/file.go (synthesizeFileAuths)
    ↓
auth.Attributes 映射
    ↓
codefree_executor.go (prepareRequest)
    ↓
注入 headers → openai_compat_executor
    ↓
CodeFree API
```

**i18n 消息键 (参考 `internal/tui/i18n.go`):**

| 键名                              | 中文                                                | 英文                                                             |
| --------------------------------- | --------------------------------------------------- | ---------------------------------------------------------------- |
| `codefree.login.start`            | 正在启动 CodeFree 登录...                           | Starting CodeFree login...                                       |
| `codefree.login.browser`          | 请在浏览器中完成授权                                | Please complete authorization in browser                         |
| `codefree.login.callback_wait`    | 等待授权回调...                                     | Waiting for authorization callback...                            |
| `codefree.login.callback_timeout` | 等待授权超时，请重试                                | Authorization timeout, please retry                              |
| `codefree.login.manual_prompt`    | 请在浏览器完成授权后，从地址栏复制 code 参数并粘贴: | After authorization, copy the code parameter from URL and paste: |
| `codefree.login.success`          | 登录成功！凭据已保存                                | Login successful! Credentials saved                              |
| `codefree.login.failed`           | 登录失败: {error}                                   | Login failed: {error}                                            |
| `codefree.refresh.start`          | 正在刷新模型列表...                                 | Refreshing model list...                                         |
| `codefree.refresh.success`        | 模型列表已更新，新增 {added} 个，更新 {updated} 个  | Model list updated, {added} added, {updated} updated             |
| `codefree.refresh.no_creds`       | 未找到凭据，请先执行 codefree-login                 | Credentials not found, please run codefree-login first           |
| `codefree.token.expired`          | Token 已过期，请重新执行 codefree-login             | Token expired, please run codefree-login again                   |
| `codefree.creds.invalid`          | 凭据文件格式错误: 缺少 {fields}                     | Invalid credentials file: missing {fields}                       |
| `codefree.port.in_use`            | 端口 {port} 已被占用，使用手动模式                  | Port {port} in use, using manual mode                            |
