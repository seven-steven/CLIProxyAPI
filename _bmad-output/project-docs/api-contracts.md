# API 契约文档

**生成日期:** 2026-03-06
**基础 URL:** `http://localhost:8317`

---

## 概述

CLIProxyAPI 提供多个兼容端点，允许客户端使用 OpenAI、Claude 或 Gemini 格式的 API。所有端点都支持 Bearer Token 认证。

---

## 认证

所有 API 请求需要在 Header 中包含认证信息：

```
Authorization: Bearer YOUR_API_KEY
```

或者在管理 API 中：

```
Authorization: Bearer MANAGEMENT_PASSWORD
```

---

## OpenAI 兼容 API

### 聊天补全

**POST** `/v1/chat/completions`

请求体：

```json
{
  "model": "gpt-4o",
  "messages": [{ "role": "user", "content": "Hello!" }],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 4096
}
```

响应（非流式）：

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

### 文本补全

**POST** `/v1/completions`

请求体：

```json
{
  "model": "gpt-4o",
  "prompt": "Once upon a time",
  "max_tokens": 100,
  "stream": false
}
```

### 模型列表

**GET** `/v1/models`

响应：

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1234567890,
      "owned_by": "openai"
    }
  ]
}
```

### Responses API (OpenAI)

**POST** `/v1/responses`

请求体：

```json
{
  "model": "gpt-4o",
  "input": "What is the weather?",
  "stream": true
}
```

**GET** `/v1/responses` (WebSocket 升级)

用于流式响应的 WebSocket 连接。

---

## Claude 兼容 API

### 消息

**POST** `/v1/messages`

请求体：

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 4096,
  "messages": [{ "role": "user", "content": "Hello!" }],
  "stream": true
}
```

响应（非流式）：

```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you?"
    }
  ],
  "model": "claude-sonnet-4-20250514",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

### Token 计数

**POST** `/v1/messages/count_tokens`

请求体：

```json
{
  "model": "claude-sonnet-4-20250514",
  "messages": [{ "role": "user", "content": "Hello!" }]
}
```

响应：

```json
{
  "input_tokens": 10
}
```

---

## Gemini 兼容 API

### 模型操作

**POST** `/v1beta/models/{model}:{action}`

支持的 action：

- `generateContent`
- `streamGenerateContent`
- `countTokens`
- `embedContent`

请求体 (generateContent)：

```json
{
  "contents": [
    {
      "parts": [
        {
          "text": "Hello!"
        }
      ]
    }
  ],
  "generationConfig": {
    "temperature": 0.7,
    "maxOutputTokens": 4096
  }
}
```

响应：

```json
{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "text": "Hello! How can I help you?"
          }
        ],
        "role": "model"
      },
      "finishReason": "STOP"
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 10,
    "candidatesTokenCount": 20,
    "totalTokenCount": 30
  }
}
```

---

## 管理 API

### 配置管理

**GET** `/v0/management/config`

获取当前配置（JSON 格式）。

**GET** `/v0/management/config.yaml`

获取当前配置（YAML 格式）。

**PUT** `/v0/management/config.yaml`

更新配置。

### 认证文件管理

**GET** `/v0/management/auth-files`

列出所有认证文件。

响应：

```json
{
  "files": [
    {
      "id": "gemini-account-1",
      "provider": "gemini",
      "label": "user@example.com",
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-01T00:00:00Z"
    }
  ]
}
```

**POST** `/v0/management/auth-files`

上传认证文件。

**DELETE** `/v0/management/auth-files`

删除认证文件。

### OAuth 认证

**GET** `/v0/management/anthropic-auth-url`

获取 Claude OAuth 授权 URL。

**GET** `/v0/management/codex-auth-url`

获取 Codex OAuth 授权 URL。

**GET** `/v0/management/gemini-cli-auth-url`

获取 Gemini CLI OAuth 授权 URL。

**GET** `/v0/management/qwen-auth-url`

获取 Qwen OAuth 授权 URL。

### 使用统计

**GET** `/v0/management/usage`

获取使用统计。

响应：

```json
{
  "total_requests": 1000,
  "total_tokens": 50000,
  "by_model": {
    "gpt-4o": { "requests": 500, "tokens": 25000 },
    "claude-sonnet-4": { "requests": 500, "tokens": 25000 }
  }
}
```

### 日志管理

**GET** `/v0/management/logs`

获取服务器日志。

**GET** `/v0/management/request-error-logs`

获取请求错误日志。

---

## Amp CLI 集成 API

### Amp 配置

**GET** `/v0/management/ampcode/upstream-url`

获取 Amp 上游 URL。

**PUT** `/v0/management/ampcode/upstream-url`

设置 Amp 上游 URL。

**GET** `/v0/management/ampcode/model-mappings`

获取模型映射配置。

**PUT** `/v0/management/ampcode/model-mappings`

更新模型映射配置。

---

## 流式响应

### SSE 格式

流式响应使用 Server-Sent Events (SSE) 格式：

```
data: {"id":"xxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"},"index":0}]}

data: {"id":"xxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"!"},"index":0}]}

data: [DONE]
```

---

## 错误响应

所有错误响应遵循 OpenAI 格式：

```json
{
  "error": {
    "message": "Invalid API key",
    "type": "authentication_error",
    "code": "invalid_api_key"
  }
}
```

### 错误类型

| HTTP 状态码 | 错误类型              | 代码                  |
| ----------- | --------------------- | --------------------- |
| 400         | invalid_request_error | -                     |
| 401         | authentication_error  | invalid_api_key       |
| 403         | permission_error      | insufficient_quota    |
| 404         | invalid_request_error | model_not_found       |
| 429         | rate_limit_error      | rate_limit_exceeded   |
| 500+        | server_error          | internal_server_error |

---

## 支持的模型

### OpenAI 系列

- gpt-4o, gpt-4o-mini
- gpt-4-turbo, gpt-4
- o1-preview, o1-mini
- chatgpt-4o-latest

### Claude 系列

- claude-opus-4, claude-opus-4-5
- claude-sonnet-4, claude-sonnet-4-5
- claude-3-5-sonnet, claude-3-5-haiku

### Gemini 系列

- gemini-2.0-flash, gemini-2.5-pro
- gemini-1.5-pro, gemini-1.5-flash

### 其他

- qwen-\* (Qwen 系列)
- codex-\* (Codex 系列)
