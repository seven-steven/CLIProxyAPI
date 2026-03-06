# 数据模型文档

**生成日期:** 2026-03-06

---

## 概述

CLIProxyAPI 使用多种数据模型来管理认证、配置和 API 请求/响应。本文档描述核心数据结构和存储模式。

---

## 核心数据模型

### Auth (认证记录)

认证记录表示一个提供商的认证凭据。

```go
type Auth struct {
    // 唯一标识符
    ID               string            `json:"id"`

    // 提供商标识 (gemini, claude, codex, qwen, iflow, antigravity, kimi)
    Provider         string            `json:"provider"`

    // 存储文件名
    FileName         string            `json:"file_name"`

    // 显示标签 (通常是邮箱或项目ID)
    Label            string            `json:"label"`

    // 认证状态
    Status           AuthStatus        `json:"status"`

    // 附加属性
    Attributes       map[string]string `json:"attributes,omitempty"`

    // 提供商特定元数据
    Metadata         map[string]any    `json:"metadata,omitempty"`

    // 令牌存储接口
    Storage          TokenStorage      `json:"-"`

    // 时间戳
    CreatedAt        time.Time         `json:"created_at"`
    UpdatedAt        time.Time         `json:"updated_at"`
    LastRefreshedAt  time.Time         `json:"last_refreshed_at"`
    NextRefreshAfter time.Time         `json:"next_refresh_after"`

    // 是否禁用
    Disabled         bool              `json:"disabled"`
}
```

### AuthStatus (认证状态)

```go
type AuthStatus int

const (
    StatusActive   AuthStatus = iota  // 活跃
    StatusCooldown                     // 冷却中
    StatusDisabled                     // 已禁用
)
```

### TokenStorage (令牌存储接口)

```go
type TokenStorage interface {
    // 保存令牌到文件
    SaveTokenToFile(path string) error

    // 从文件加载令牌
    LoadTokenFromFile(path string) error

    // 检查令牌是否需要刷新
    NeedsRefresh() bool

    // 刷新令牌
    Refresh(ctx context.Context) error
}
```

---

## 配置模型

### Config (主配置)

```go
type Config struct {
    // 嵌入 SDK 配置
    SDKConfig `yaml:",inline"`

    // 服务器配置
    Host string `yaml:"host"`  // 绑定地址
    Port int    `yaml:"port"`  // 服务端口

    // TLS 配置
    TLS TLSConfig `yaml:"tls"`

    // 远程管理配置
    RemoteManagement RemoteManagement `yaml:"remote-management"`

    // 认证目录
    AuthDir string `yaml:"auth-dir"`

    // 调试模式
    Debug bool `yaml:"debug"`

    // pprof 配置
    Pprof PprofConfig `yaml:"pprof"`

    // 日志配置
    LoggingToFile      bool `yaml:"logging-to-file"`
    LogsMaxTotalSizeMB int  `yaml:"logs-max-total-size-mb"`
    ErrorLogsMaxFiles  int  `yaml:"error-logs-max-files"`

    // 请求日志
    RequestLog bool `yaml:"request-log"`

    // 使用统计
    UsageStatisticsEnabled bool `yaml:"usage-statistics-enabled"`

    // 代理 URL
    ProxyURL string `yaml:"proxy-url"`

    // 商业模式
    CommercialMode bool `yaml:"commercial-mode"`

    // WebSocket 认证
    WebsocketAuth bool `yaml:"websocket-auth"`

    // 重试配置
    RequestRetry        string `yaml:"request-retry"`
    MaxRetryInterval    int    `yaml:"max-retry-interval"`
    MaxRetryCredentials int    `yaml:"max-retry-credentials"`

    // 配额冷却
    DisableCooling bool `yaml:"disable-cooling"`

    // 配额超限配置
    QuotaExceededSwitchProject    bool   `yaml:"quota-exceeded-switch-project"`
    QuotaExceededSwitchPreviewModel bool  `yaml:"quota-exceeded-switch-preview-model"`

    // API 密钥配置
    GeminiKey  []string `yaml:"gemini-key"`
    ClaudeKey  []string `yaml:"claude-key"`
    CodexKey   []string `yaml:"codex-key"`

    // OpenAI 兼容配置
    OpenAICompatibility []OpenAICompatEntry `yaml:"openai-compatibility"`

    // Vertex 兼容配置
    VertexCompatAPIKey []VertexCompatEntry `yaml:"vertex-api-key"`

    // OAuth 排除模型
    OAuthExcludedModels []string `yaml:"oauth-excluded-models"`

    // OAuth 模型别名
    OAuthModelAlias map[string]string `yaml:"oauth-model-alias"`

    // Amp 配置
    AmpCode AmpCodeConfig `yaml:"ampcode"`

    // 路由策略
    RoutingStrategy string `yaml:"routing-strategy"`

    // 强制模型前缀
    ForceModelPrefix string `yaml:"force-model-prefix"`
}
```

### SDKConfig (SDK 配置)

```go
type SDKConfig struct {
    // Gemini API 密钥
    GeminiKey []string `yaml:"gemini-key"`

    // Claude API 密钥
    ClaudeKey []string `yaml:"claude-key"`

    // Codex API 密钥
    CodexKey []string `yaml:"codex-key"`

    // 流式配置
    Streaming StreamingConfig `yaml:"streaming"`

    // 非流式保活间隔
    NonStreamKeepAliveInterval int `yaml:"non-stream-keep-alive-interval"`

    // 透传 Headers
    PassthroughHeaders bool `yaml:"passthrough-headers"`
}
```

### TLSConfig (TLS 配置)

```go
type TLSConfig struct {
    Enable bool   `yaml:"enable"`  // 启用 TLS
    Cert   string `yaml:"cert"`    // 证书文件路径
    Key    string `yaml:"key"`     // 私钥文件路径
}
```

### RemoteManagement (远程管理配置)

```go
type RemoteManagement struct {
    SecretKey             string `yaml:"secret-key"`              // 管理密码
    DisableControlPanel   bool   `yaml:"disable-control-panel"`   // 禁用控制面板
    PanelGitHubRepository string `yaml:"panel-github-repository"` // 面板仓库
}
```

### AmpCodeConfig (Amp 配置)

```go
type AmpCodeConfig struct {
    UpstreamURL              string            `yaml:"upstream-url"`
    UpstreamAPIKey           string            `yaml:"upstream-api-key"`
    RestrictManagementToLocalhost bool          `yaml:"restrict-management-to-localhost"`
    ModelMappings            []ModelMapping    `yaml:"model-mappings"`
    ForceModelMappings       bool              `yaml:"force-model-mappings"`
    UpstreamAPIKeys          []UpstreamAPIKey  `yaml:"upstream-api-keys"`
}

type ModelMapping struct {
    From string `yaml:"from"`  // 原始模型名
    To   string `yaml:"to"`    // 映射目标模型名
}
```

---

## API 请求/响应模型

### OpenAI 格式

#### ChatCompletionRequest

```go
type ChatCompletionRequest struct {
    Model            string                 `json:"model"`
    Messages         []ChatMessage          `json:"messages"`
    Temperature      *float64               `json:"temperature,omitempty"`
    TopP             *float64               `json:"top_p,omitempty"`
    MaxTokens        *int                   `json:"max_tokens,omitempty"`
    Stream           bool                   `json:"stream"`
    Tools            []Tool                 `json:"tools,omitempty"`
    ToolChoice       interface{}            `json:"tool_choice,omitempty"`
    ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
    FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
    PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
    User             string                 `json:"user,omitempty"`
}

type ChatMessage struct {
    Role    string      `json:"role"`
    Content interface{} `json:"content"` // string 或 []ContentPart
}

type ContentPart struct {
    Type     string    `json:"type"` // "text" 或 "image_url"
    Text     string    `json:"text,omitempty"`
    ImageURL *ImageURL `json:"image_url,omitempty"`
}
```

#### ChatCompletionResponse

```go
type ChatCompletionResponse struct {
    ID                string   `json:"id"`
    Object            string   `json:"object"`
    Created           int64    `json:"created"`
    Model             string   `json:"model"`
    Choices           []Choice `json:"choices"`
    Usage             *Usage   `json:"usage,omitempty"`
    SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

type Choice struct {
    Index        int          `json:"index"`
    Message      *ChatMessage `json:"message,omitempty"`
    Delta        *ChatMessage `json:"delta,omitempty"`
    FinishReason string       `json:"finish_reason"`
}

type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

### Claude 格式

#### MessagesRequest

```go
type MessagesRequest struct {
    Model       string          `json:"model"`
    MaxTokens   int             `json:"max_tokens"`
    Messages    []ClaudeMessage `json:"messages"`
    System      string          `json:"system,omitempty"`
    Temperature *float64        `json:"temperature,omitempty"`
    Tools       []ClaudeTool    `json:"tools,omitempty"`
    Stream      bool            `json:"stream"`
}

type ClaudeMessage struct {
    Role    string      `json:"role"`
    Content interface{} `json:"content"`
}
```

#### MessagesResponse

```go
type MessagesResponse struct {
    ID           string          `json:"id"`
    Type         string          `json:"type"`
    Role         string          `json:"role"`
    Content      []ContentBlock  `json:"content"`
    Model        string          `json:"model"`
    StopReason   string          `json:"stop_reason,omitempty"`
    StopSequence string          `json:"stop_sequence,omitempty"`
    Usage        ClaudeUsage     `json:"usage"`
}

type ContentBlock struct {
    Type string `json:"type"` // "text", "tool_use", "image"
    Text string `json:"text,omitempty"`
    // ... 其他字段
}
```

### Gemini 格式

#### GenerateContentRequest

```go
type GenerateContentRequest struct {
    Contents         []GeminiContent `json:"contents"`
    GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
    SafetySettings   []SafetySetting  `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
    Role  string       `json:"role"`
    Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
    Text       string `json:"text,omitempty"`
    InlineData *Blob  `json:"inlineData,omitempty"`
}
```

---

## 存储模型

### PostgreSQL Schema

```sql
-- 配置存储表
CREATE TABLE config_store (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 认证存储表
CREATE TABLE auth_store (
    id TEXT PRIMARY KEY,
    content JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 文件系统存储

```
auths/
├── gemini/
│   ├── account1.json      # Gemini OAuth 令牌
│   └── account2.json
├── claude/
│   ├── account1.json      # Claude OAuth 令牌
│   └── account2.json
├── codex/
│   ├── account1.json      # Codex OAuth 令牌
│   └── account2.json
└── vertex/
    └── service-account.json # Vertex 服务账号
```

### 认证文件格式示例

```json
{
  "type": "gemini",
  "email": "user@example.com",
  "project_id": "my-project",
  "access_token": "ya29.xxx",
  "refresh_token": "1//xxx",
  "token_uri": "https://oauth2.googleapis.com/token",
  "client_id": "xxx.apps.googleusercontent.com",
  "client_secret": "xxx",
  "expiry": "2026-03-06T12:00:00Z"
}
```

---

## 使用统计模型

```go
type UsageStatistics struct {
    TotalRequests    int            `json:"total_requests"`
    TotalTokens      int64          `json:"total_tokens"`
    InputTokens      int64          `json:"input_tokens"`
    OutputTokens     int64          `json:"output_tokens"`
    ByModel          map[string]ModelUsage `json:"by_model"`
    ByProvider       map[string]int64      `json:"by_provider"`
    StartTime        time.Time      `json:"start_time"`
    LastUpdated      time.Time      `json:"last_updated"`
}

type ModelUsage struct {
    Requests     int   `json:"requests"`
    InputTokens  int64 `json:"input_tokens"`
    OutputTokens int64 `json:"output_tokens"`
}
```
