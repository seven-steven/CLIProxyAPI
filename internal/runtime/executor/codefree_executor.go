package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codefree"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/thinking"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// CodefreeExecutor 实现 Codefree 特定的请求处理
// 它组合持有 OpenAICompatExecutor 来复用 OpenAI 兼容的请求逻辑
type CodefreeExecutor struct {
	compatExecutor *OpenAICompatExecutor
	cfg            *config.Config
}

// NewCodefreeExecutor 创建新的 Codefree 执行器
func NewCodefreeExecutor(cfg *config.Config) *CodefreeExecutor {
	return &CodefreeExecutor{
		compatExecutor: NewOpenAICompatExecutor("codefree", cfg),
		cfg:            cfg,
	}
}

// Identifier 返回执行器标识符
func (e *CodefreeExecutor) Identifier() string {
	return "codefree"
}

// PrepareRequest 准备请求，注入 Codefree 特定的 headers
func (e *CodefreeExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth, requestBody []byte) error {
	if req == nil {
		return nil
	}

	// 从 auth.Attributes 获取凭据
	var apiKey, userID string
	if auth != nil && auth.Attributes != nil {
		apiKey = auth.Attributes["api_key"]
		userID = auth.Attributes["user_id"]
	}

	// 注入 Codefree 特定的 headers
	if apiKey != "" {
		req.Header.Set("apiKey", apiKey)
	}
	if userID != "" {
		req.Header.Set("userId", userID)
	}

	// 从请求体提取 modelName（使用 gjson）
	if len(requestBody) > 0 {
		modelName := gjson.GetBytes(requestBody, "model").String()
		if modelName != "" {
			// 移除可能的 thinking suffix
			modelName = thinking.ParseSuffix(modelName).ModelName
			req.Header.Set("modelName", modelName)
		}
		// 如果提取失败，静默跳过（根据技术规范）
	}

	// 设置固定的 clientType header
	req.Header.Set("clientType", "codefree-cli")

	// 应用自定义 headers（如果有 header: 前缀的属性）
	if auth != nil {
		util.ApplyCustomHeadersFromAttrs(req, auth.Attributes)
	}

	return nil
}

// checkResponseError 检查 HTTP 响应错误，返回相应的错误类型
// 如果响应正常返回 nil，否则返回错误
func (e *CodefreeExecutor) checkResponseError(httpResp *http.Response, body []byte) error {
	// 检测 401 错误（Token 过期）
	if httpResp.StatusCode == http.StatusUnauthorized {
		return codefree.NewCodefreeError(codefree.ErrTokenExpired,
			fmt.Errorf("received 401 unauthorized from codefree API"))
	}

	// 检测其他错误状态码
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		log.Debugf("codefree executor: request error, status: %d, body: %s", httpResp.StatusCode, string(body))
		return statusErr{code: httpResp.StatusCode, msg: string(body)}
	}

	return nil
}

// Execute 执行非流式请求
func (e *CodefreeExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	// 调用内部执行方法
	httpResp, err := e.execute(ctx, auth, req, opts, false)
	if err != nil {
		return resp, err
	}
	defer func() {
		if errClose := httpResp.Body.Close(); errClose != nil {
			log.Errorf("codefree executor: close response body error: %v", errClose)
		}
	}()

	// 读取响应体
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查响应错误
	if err := e.checkResponseError(httpResp, body); err != nil {
		return resp, err
	}

	// 构造响应
	resp = cliproxyexecutor.Response{
		Payload: body,
		Headers: httpResp.Header.Clone(),
	}

	return resp, nil
}

// ExecuteStream 执行流式请求
func (e *CodefreeExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (_ *cliproxyexecutor.StreamResult, err error) {
	// 调用内部执行方法
	httpResp, err := e.execute(ctx, auth, req, opts, true)
	if err != nil {
		return nil, err
	}

	// 检查响应错误（流式请求需要先读取部分 body 来检查错误）
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if err := e.checkResponseError(httpResp, b); err != nil {
			return nil, err
		}
	}

	// 创建 chunk channel
	out := make(chan cliproxyexecutor.StreamChunk)

	// 启动 goroutine 读取流式响应
	go func() {
		defer close(out)
		defer func() {
			if errClose := httpResp.Body.Close(); errClose != nil {
				log.Errorf("codefree executor: close response body error: %v", errClose)
			}
		}()

		scanner := bufio.NewScanner(httpResp.Body)
		scanner.Buffer(nil, 52_428_800) // 50MB

		for scanner.Scan() {
			line := scanner.Bytes()
			appendAPIResponseChunk(ctx, e.cfg, line)

			// 发送 chunk
			out <- cliproxyexecutor.StreamChunk{Payload: line}
		}

		if errScan := scanner.Err(); errScan != nil {
			recordAPIResponseError(ctx, e.cfg, errScan)
			out <- cliproxyexecutor.StreamChunk{Err: errScan}
		}
	}()

	// 构造流式结果
	result := &cliproxyexecutor.StreamResult{
		Headers: httpResp.Header.Clone(),
		Chunks:  out,
	}

	return result, nil
}

// execute 内部执行方法，构造并发送 HTTP 请求
func (e *CodefreeExecutor) execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options, stream bool) (*http.Response, error) {
	baseModel := thinking.ParseSuffix(req.Model).ModelName

	// 获取凭据
	var baseURL, apiKey string
	if auth != nil && auth.Attributes != nil {
		baseURL = auth.Attributes["base_url"]
		apiKey = auth.Attributes["api_key"]
	}

	if baseURL == "" {
		return nil, statusErr{code: http.StatusUnauthorized, msg: "missing codefree baseURL"}
	}

	// 翻译请求格式
	from := opts.SourceFormat
	to := sdktranslator.FromString("openai")

	originalPayloadSource := req.Payload
	if len(opts.OriginalRequest) > 0 {
		originalPayloadSource = opts.OriginalRequest
	}
	originalPayload := originalPayloadSource
	originalTranslated := sdktranslator.TranslateRequest(from, to, baseModel, originalPayload, stream)
	translated := sdktranslator.TranslateRequest(from, to, baseModel, req.Payload, stream)
	requestedModel := payloadRequestedModel(opts, req.Model)
	translated = applyPayloadConfigWithRoot(e.cfg, baseModel, to.String(), "", translated, originalTranslated, requestedModel)

	// 应用 thinking（如果需要）
	var err error
	translated, err = thinking.ApplyThinking(translated, req.Model, from.String(), to.String(), e.Identifier())
	if err != nil {
		return nil, err
	}

	// 构造请求 URL
	endpoint := "/api/acbackend/codechat/v1/completions"
	url := strings.TrimSuffix(baseURL, "/") + endpoint

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(translated))
	if err != nil {
		return nil, err
	}

	// 设置基础 headers
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer codefree")
	}
	httpReq.Header.Set("User-Agent", "cli-proxy-codefree")

	// 准备 Codefree 特定的 headers
	if err := e.PrepareRequest(httpReq, auth, translated); err != nil {
		return nil, err
	}

	// 设置流式请求的额外 headers
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
	}

	// 记录请求日志
	var authID, authLabel, authType, authValue string
	if auth != nil {
		authID = auth.ID
		authLabel = auth.Label
		authType, authValue = auth.AccountInfo()
	}
	recordAPIRequest(ctx, e.cfg, upstreamRequestLog{
		URL:       url,
		Method:    http.MethodPost,
		Headers:   httpReq.Header.Clone(),
		Body:      translated,
		Provider:  e.Identifier(),
		AuthID:    authID,
		AuthLabel: authLabel,
		AuthType:  authType,
		AuthValue: authValue,
	})

	// 执行请求
	httpClient := newProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		recordAPIResponseError(ctx, e.cfg, err)
		return nil, err
	}

	// 记录响应元数据
	recordAPIResponseMetadata(ctx, e.cfg, httpResp.StatusCode, httpResp.Header.Clone())

	return httpResp, nil
}

// CountTokens 计算 token 数量（委托给 OpenAI 兼容的实现）
func (e *CodefreeExecutor) CountTokens(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	// 委托给 OpenAI 兼容执行器
	return e.compatExecutor.CountTokens(ctx, auth, req, opts)
}

// Refresh 刷新凭据（Codefree 不支持自动刷新）
func (e *CodefreeExecutor) Refresh(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	// Codefree 不提供 refresh token，返回错误提示用户重新登录
	return nil, codefree.NewCodefreeError(codefree.ErrTokenExpired,
		fmt.Errorf("codefree does not support token refresh, please run codefree-login again"))
}

// HttpRequest 注入凭据并执行 HTTP 请求
func (e *CodefreeExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("codefree executor: request is nil")
	}
	if ctx == nil {
		ctx = req.Context()
	}

	// 创建新的请求（复制原始请求）
	httpReq := req.WithContext(ctx)

	// 注入 Codefree 特定的 headers
	// 注意：HttpRequest 方法没有 requestBody 参数，所以我们无法提取 modelName
	// 这种情况下，modelName header 将被省略
	if err := e.PrepareRequest(httpReq, auth, nil); err != nil {
		return nil, err
	}

	// 执行请求
	httpClient := newProxyAwareHTTPClient(ctx, e.cfg, auth, 0)
	return httpClient.Do(httpReq)
}
