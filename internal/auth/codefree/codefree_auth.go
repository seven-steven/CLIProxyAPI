package codefree

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	log "github.com/sirupsen/logrus"
)

// truncateString 截断字符串用于日志输出
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// OAuth 配置常量
// 注意：ClientSecret 是 Codefree 服务端提供的固定值，用于 CLI 客户端认证
// 与其他 OAuth 客户端（如 Claude）的实现模式一致
// 可通过环境变量覆盖: CODEFREE_CLIENT_ID, CODEFREE_CLIENT_SECRET, CODEFREE_REDIRECT_URI
const (
	defaultClientID     = "250320a317ba7dcvofvm"
	defaultClientSecret = "250320a318034D2xWtybKuEiSsjxzhWvKrMtgIRA"
	defaultRedirectURI  = "https://www.srdcloud.cn/login/oauth-srdcloud-redirect"
	ProjectID           = "CLI"
	// APIKeyProjectID 是获取 API Key 时使用的 projectId
	// 注意：官方客户端使用 "0"，而不是 "CLI"
	APIKeyProjectID = "0"
)

var (
	// ClientID 可以通过环境变量 CODEFREE_CLIENT_ID 覆盖
	ClientID = getEnvOrDefault("CODEFREE_CLIENT_ID", defaultClientID)
	// ClientSecret 可以通过环境变量 CODEFREE_CLIENT_SECRET 覆盖
	ClientSecret = getEnvOrDefault("CODEFREE_CLIENT_SECRET", defaultClientSecret)
	// RedirectURI 可以通过环境变量 CODEFREE_REDIRECT_URI 覆盖
	RedirectURI = getEnvOrDefault("CODEFREE_REDIRECT_URI", defaultRedirectURI)
)

// getEnvOrDefault 获取环境变量值，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// API 端点常量
const (
	BaseURL         = "https://www.srdcloud.cn"
	AuthURL         = BaseURL + "/login/oauth/authorize"
	TokenURL        = BaseURL + "/login/oauth/access_token"
	APIKeyURL       = BaseURL + "/api/acbackend/usermanager/v1/users/apikey"
	ModelsURLFormat = BaseURL + "/api/acbackend/modelmgr/v1/clients/" + ProjectID + "/versions/%s"
)

// ModelInfo 表示模型信息
type ModelInfo struct {
	Name         string `json:"modelName"`
	Manufacturer string `json:"manufacturer"`
	MaxTokens    int    `json:"maxTokens"`
}

// tokenResponse 表示 OAuth token 响应
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	UID          string `json:"uid"`                    // 用户ID（当id_token为空时使用）
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	OriSessionID string `json:"ori_session_id,omitempty"` // 保留以映射完整 API 响应
}

// GetUserID 返回用户ID（优先使用id_token，其次使用uid，最后使用ori_session_id）
func (t *tokenResponse) GetUserID() string {
	if t.IDToken != "" {
		log.Debugf("GetUserID: using id_token")
		return t.IDToken
	}
	if t.UID != "" {
		log.Debugf("GetUserID: using uid=%s", t.UID)
		return t.UID
	}
	log.Debugf("GetUserID: using ori_session_id=%s", t.OriSessionID)
	return t.OriSessionID
}

// apikeyResponse 表示 API Key 响应
type apikeyResponse struct {
	EncryptedAPIKey string `json:"encryptedApiKey"`
}

// modelsResponse 表示模型列表响应
type modelsResponse struct {
	Data []ModelInfo `json:"data"`
}

// ErrModelFetchFailed 表示获取模型列表失败的错误
var ErrModelFetchFailed = &CodefreeError{
	Type:    "model_fetch_failed",
	Message: "Failed to fetch model list",
	Code:    http.StatusInternalServerError,
}

// CodefreeAuth 处理 Codefree OAuth 认证
type CodefreeAuth struct {
	httpClient *http.Client
	config     *config.Config
}

// NewCodefreeAuth 创建新的 Codefree 认证服务
func NewCodefreeAuth(cfg *config.Config) *CodefreeAuth {
	return &CodefreeAuth{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: cfg,
	}
}

// GenerateAuthURL 生成授权 URL
func (o *CodefreeAuth) GenerateAuthURL(state string) string {
	params := url.Values{
		"client_id":     {ClientID},
		"redirect_uri":  {RedirectURI},
		"response_type": {"code"},
		"state":         {state},
	}

	authURL := fmt.Sprintf("%s?%s", AuthURL, params.Encode())
	log.Debugf("Generated auth URL: %s", authURL)
	return authURL
}

// ExchangeCodeForTokens 交换授权码获取 tokens
func (o *CodefreeAuth) ExchangeCodeForTokens(code string) (*tokenResponse, error) {
	params := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {ClientID},
		"client_secret": {ClientSecret},
		"code":          {code},
		"redirect_uri":  {RedirectURI},
	}

	reqURL := fmt.Sprintf("%s?%s", TokenURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, NewCodefreeError(ErrCodeExchangeFailed, err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, NewCodefreeError(ErrCodeExchangeFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewCodefreeError(ErrCodeExchangeFailed, err)
	}

	log.Debugf("Token exchange response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Token exchange failed with status %d: %s", resp.StatusCode, string(body))
		return nil, NewCodefreeError(ErrCodeExchangeFailed,
			fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	// Codefree API 返回的是 application/x-www-form-urlencoded 格式
	// 例如: access_token=xxx&id_token=yyy&...
	values, err := url.ParseQuery(string(body))
	if err != nil {
		log.Errorf("Failed to parse token response as URL query: %v", err)
		return nil, NewCodefreeError(ErrCodeExchangeFailed,
			fmt.Errorf("failed to parse response: %w", err))
	}

	// 注意：官方客户端使用 ori_session_id (UUID格式) 作为 access_token 保存
	// 而不是 JWT 格式的 access_token
	// 参考: docs/codefree/codefree.har
	oriSessionID := values.Get("ori_session_id")
	jwtAccessToken := values.Get("access_token")

	// 如果 ori_session_id 存在，优先使用它作为 access_token
	// 否则回退到 JWT 格式的 access_token
	accessToken := jwtAccessToken
	if oriSessionID != "" {
		accessToken = oriSessionID
	}

	tokenResp := &tokenResponse{
		AccessToken:  accessToken,
		IDToken:      values.Get("id_token"),
		UID:          values.Get("uid"),
		OriSessionID: oriSessionID,
		TokenType:   values.Get("token_type"),
	}

	// 解析 expires_in
	if expStr := values.Get("expires_in"); expStr != "" {
		var expiresIn int64
		if _, err := fmt.Sscanf(expStr, "%d", &expiresIn); err == nil {
			tokenResp.ExpiresIn = expiresIn
		}
	}

	log.Debug("Successfully exchanged code for tokens")
	return tokenResp, nil
}

// FetchAPIKey 获取加密的 API Key
func (o *CodefreeAuth) FetchAPIKey(accessToken, userID string) (string, error) {
	req, err := http.NewRequest("GET", APIKeyURL, nil)
	if err != nil {
		return "", NewCodefreeError(ErrAPIKeyFetchFailed, err)
	}

	// 设置必需的 headers
	req.Header.Set("sessionId", accessToken)
	req.Header.Set("userId", userID)
	req.Header.Set("projectId", APIKeyProjectID) // 使用 "0" 而不是 "CLI"
	req.Header.Set("Authorization", "Bearer "+accessToken)

	log.Debugf("FetchAPIKey request URL: %s", APIKeyURL)
	log.Debugf("FetchAPIKey headers: sessionId=%s..., userId=%s, projectId=%s",
		truncateString(accessToken, 20), userID, APIKeyProjectID)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", NewCodefreeError(ErrAPIKeyFetchFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", NewCodefreeError(ErrAPIKeyFetchFailed, err)
	}

	log.Debugf("FetchAPIKey response status: %d, body: %s", resp.StatusCode, truncateString(string(body), 500))

	if resp.StatusCode != http.StatusOK {
		log.Errorf("API key fetch failed with status %d: %s", resp.StatusCode, string(body))
		return "", NewCodefreeError(ErrAPIKeyFetchFailed,
			fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	var apikeyResp apikeyResponse
	if err := json.Unmarshal(body, &apikeyResp); err != nil {
		return "", NewCodefreeError(ErrAPIKeyFetchFailed, err)
	}

	log.Debug("Successfully fetched API key")
	return apikeyResp.EncryptedAPIKey, nil
}

// FetchModels 获取模型列表
func (o *CodefreeAuth) FetchModels(userID, apiKey, cliVersion string) ([]ModelInfo, error) {
	modelsURL := fmt.Sprintf(ModelsURLFormat, cliVersion)

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		return nil, NewCodefreeError(ErrModelFetchFailed, err)
	}

	// 设置必需的 headers
	req.Header.Set("userId", userID)
	req.Header.Set("apiKey", apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, NewCodefreeError(ErrModelFetchFailed, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewCodefreeError(ErrModelFetchFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Model list fetch failed with status %d: %s", resp.StatusCode, string(body))
		return nil, NewCodefreeError(ErrModelFetchFailed,
			fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
	}

	var modelsResp modelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, NewCodefreeError(ErrModelFetchFailed, err)
	}

	log.Debugf("Successfully fetched %d models", len(modelsResp.Data))
	return modelsResp.Data, nil
}

// RefreshModels 刷新模型列表
func (o *CodefreeAuth) RefreshModels(authFilePath, cliVersion string) ([]ModelInfo, error) {
	// 加载凭据
	storage, err := LoadTokenFromFile(authFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// 检查是否过期
	if storage.IsExpired() {
		return nil, ErrTokenExpired
	}

	// 获取模型列表
	models, err := o.FetchModels(storage.IDToken, storage.APIKey, cliVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	return models, nil
}

// Authenticate 执行完整的 OAuth 认证流程
// 注意：此方法提供完整的 OAuth 流程实现，供需要直接集成的场景使用
// 对于 CLI 命令，推荐使用 codefree_login.go 中的 DoCodefreeLogin 函数
func (o *CodefreeAuth) Authenticate(ctx context.Context, authFilePath string, manualMode bool) error {
	// 创建 OAuth 服务器
	server := NewCodefreeOAuthServer()

	if !manualMode {
		// 启动回调服务器
		if err := server.Start(); err != nil {
			log.Warnf("Failed to start OAuth server: %v, falling back to manual mode", err)
			manualMode = true
		}
	}

	// 生成 randomCode
	randomCode, err := GenerateRandomCode()
	if err != nil {
		return fmt.Errorf("failed to generate random code: %w", err)
	}

	// 构造 state 参数
	var state string
	if !manualMode {
		server.SetExpectedRandomCode(randomCode)
		state = server.BuildStateParam(randomCode)
	} else {
		// 手动模式：使用固定的回调 URL 作为 state
		state = "manual_" + randomCode
	}

	// 生成授权 URL
	authURL := o.GenerateAuthURL(state)

	if !manualMode {
		// 等待回调（5 分钟超时）
		result, err := server.WaitForCallback(5 * time.Minute)
		if err != nil {
			_ = server.Stop(ctx)
			return err
		}

		_ = server.Stop(ctx)

		if result.Error != "" {
			return fmt.Errorf("OAuth error: %s", result.Error)
		}

		// 交换 code 获取 tokens
		tokenResp, err := o.ExchangeCodeForTokens(result.Code)
		if err != nil {
			return err
		}

		// 获取 API Key (可选步骤，失败时继续)
		apiKey, err := o.FetchAPIKey(tokenResp.AccessToken, tokenResp.GetUserID())
		if err != nil {
			log.Warnf("Failed to fetch API key (non-fatal): %v", err)
			// 不返回错误，继续保存凭据
		}

		// 保存凭据
		storage := &CodefreeTokenStorage{
			Type:        "codefree",
			AccessToken: tokenResp.AccessToken,
			IDToken:     tokenResp.GetUserID(),
			APIKey:      apiKey,
			ExpiresIn:   tokenResp.ExpiresIn,
			BaseURL:     BaseURL,
			TokenType:   tokenResp.TokenType,
			CreatedAt:   0, // 将在 SaveTokenToFile 中设置
		}

		if err := storage.SaveTokenToFile(authFilePath); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		log.Info("Codefree authentication successful")
		return nil
	}

	// 手动模式：返回特殊错误，提示需要手动处理
	return fmt.Errorf("manual mode: please visit %s and enter the code", authURL)
}
