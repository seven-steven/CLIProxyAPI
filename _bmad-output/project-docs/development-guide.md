# 开发指南

**生成日期:** 2026-03-06

---

## 先决条件

### 必需软件

| 软件     | 版本要求 | 用途       |
| -------- | -------- | ---------- |
| **Go**   | 1.26.0+  | 编程语言   |
| **Git**  | 最新版   | 版本控制   |
| **Make** | 可选     | 构建自动化 |

### 可选软件

| 软件           | 用途           |
| -------------- | -------------- |
| **Docker**     | 容器化部署     |
| **PostgreSQL** | 持久化存储后端 |
| **MinIO**      | 对象存储后端   |

---

## 环境设置

### 1. 克隆仓库

```bash
git clone https://github.com/router-for-me/CLIProxyAPI.git
cd CLIProxyAPI
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 创建配置文件

```bash
cp config.example.yaml config.yaml
```

### 4. 配置环境变量 (可选)

```bash
# .env 文件
DEPLOY=local                    # 部署模式
MANAGEMENT_PASSWORD=yourpwd     # 管理密码
```

---

## 本地开发

### 构建项目

```bash
# 直接构建
go build -o CLIProxyAPI ./cmd/server

# 使用构建脚本
./docker-build.sh
```

### 运行服务器

```bash
# 基本运行
./CLIProxyAPI

# 带配置文件
./CLIProxyAPI -config /path/to/config.yaml

# TUI 模式
./CLIProxyAPI -tui

# 独立 TUI 模式（内嵌服务器）
./CLIProxyAPI -tui -standalone

# 调试模式
./CLIProxyAPI -config config.yaml
# 在 config.yaml 中设置 debug: true
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/api/...

# 运行特定测试
go test -run TestFunctionName ./...

# 带覆盖率
go test -cover ./...
```

---

## OAuth 登录

### Gemini 登录

```bash
./CLIProxyAPI -login
# 或指定项目 ID
./CLIProxyAPI -login -project_id your-project-id
```

### Codex 登录

```bash
# OAuth 登录
./CLIProxyAPI -codex-login

# 设备码登录
./CLIProxyAPI -codex-device-login
```

### Claude 登录

```bash
./CLIProxyAPI -claude-login
```

### 其他提供商

```bash
# Qwen
./CLIProxyAPI -qwen-login

# iFlow
./CLIProxyAPI -iflow-login

# iFlow Cookie
./CLIProxyAPI -iflow-cookie

# Antigravity
./CLIProxyAPI -antigravity-login

# Kimi
./CLIProxyAPI -kimi-login
```

### 登录选项

```bash
# 不自动打开浏览器
./CLIProxyAPI -login -no-browser

# 指定 OAuth 回调端口
./CLIProxyAPI -login -oauth-callback-port 9090
```

---

## 项目结构

```
CLIProxyAPI/
├── cmd/server/          # 应用入口
├── internal/            # 私有代码
│   ├── api/            # API 层
│   ├── auth/           # 认证
│   ├── config/         # 配置
│   ├── store/          # 存储
│   ├── translator/     # 格式转换
│   └── tui/            # 终端 UI
├── sdk/                 # 公共 SDK
├── docs/                # 文档
├── examples/            # 示例
└── test/                # 集成测试
```

---

## 配置说明

### 基本配置

```yaml
# 服务器配置
port: 8317
host: "" # 空字符串表示绑定所有接口
debug: false

# 认证目录
auth-dir: auths

# TLS 配置 (可选)
tls:
  enable: false
  cert: /path/to/cert.pem
  key: /path/to/key.pem

# 远程管理
remote-management:
  secret-key: "your-password"
  disable-control-panel: false
```

### API 密钥配置

```yaml
# 直接配置 API 密钥
gemini-key:
  - "your-gemini-api-key"

claude-key:
  - "your-claude-api-key"

codex-key:
  - "your-codex-api-key"
```

### OpenAI 兼容配置

```yaml
openai-compatibility:
  - name: "openrouter"
    base-url: "https://openrouter.ai/api/v1"
    api-key-entries:
      - api-key: "sk-or-xxx"
        models:
          - "openai/gpt-4o"
          - "anthropic/claude-3.5-sonnet"
```

---

## 存储后端配置

### PostgreSQL 后端

```bash
# 环境变量
export PGSTORE_DSN="postgres://user:pass@localhost:5432/dbname"
export PGSTORE_SCHEMA="cliproxy"          # 可选
export PGSTORE_LOCAL_PATH="/var/lib/cliproxy"  # 可选
```

### Git 后端

```bash
export GITSTORE_GIT_URL="https://github.com/user/repo.git"
export GITSTORE_GIT_USERNAME="username"
export GITSTORE_GIT_TOKEN="token"
export GITSTORE_LOCAL_PATH="/var/lib/gitstore"
```

### 对象存储后端

```bash
export OBJECTSTORE_ENDPOINT="https://s3.example.com"
export OBJECTSTORE_ACCESS_KEY="access-key"
export OBJECTSTORE_SECRET_KEY="secret-key"
export OBJECTSTORE_BUCKET="cliproxy"
export OBJECTSTORE_LOCAL_PATH="/var/lib/objectstore"
```

---

## 常见开发任务

### 添加新的认证提供者

1. 在 `internal/auth/` 创建新目录
2. 实现 OAuth 流程
3. 实现 `TokenStorage` 接口
4. 在 `internal/cmd/` 添加登录命令
5. 在 `sdk/auth/` 添加 SDK 支持

### 添加新的 API 端点

1. 在 `internal/api/handlers/` 添加处理器
2. 在 `internal/api/server.go` 注册路由
3. 添加相应的测试

### 添加新的格式转换器

1. 在 `sdk/translator/` 添加转换逻辑
2. 实现 `Translator` 接口
3. 注册到转换器注册表

---

## 调试技巧

### 启用调试日志

```yaml
debug: true
request-log: true
logging-to-file: true
```

### 查看 API 请求日志

```bash
# 通过管理 API
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/request-log

# 查看错误日志
curl -H "Authorization: Bearer $PASSWORD" \
  http://localhost:8317/v0/management/request-error-logs
```

### pprof 性能分析

```yaml
pprof:
  enable: true
  addr: "127.0.0.1:8316"
```

访问 `http://127.0.0.1:8316/debug/pprof/`

---

## 代码风格

### Go 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go)
- 使用 `gofmt` 格式化代码
- 使用 `go vet` 检查代码

### 注释规范

```go
// FunctionName 做某事。
//
// 参数:
//   - arg1: 参数描述
//
// 返回:
//   - 返回值描述
func FunctionName(arg1 string) error {
    // ...
}
```

---

## 贡献指南

### 提交 PR

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 提交信息规范

```
type(scope): subject

body

footer
```

类型:

- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具

---

## SDK 使用

### 嵌入代理功能

```go
package main

import (
    "context"
    "log"

    "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy"
    "github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 创建服务
    service, err := cliproxy.NewService(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 执行请求
    resp, err := service.Execute(context.Background(), "gpt-4o", payload)
    if err != nil {
        log.Fatal(err)
    }

    // 处理响应
    log.Println(string(resp))
}
```

详细 SDK 文档请参考:

- [SDK 使用指南](./sdk-usage.md)
- [SDK 高级用法](./sdk-advanced.md)
- [SDK 访问控制](./sdk-access.md)
- [SDK 监控](./sdk-watcher.md)
