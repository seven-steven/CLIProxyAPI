package codefree

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// CodefreeOAuthServer 处理 Codefree OAuth 本地回调
type CodefreeOAuthServer struct {
	server             *http.Server
	port               int
	listenAddr         string // 监听地址，默认 127.0.0.1
	resultChan         chan *OAuthResult
	errorChan          chan error
	mu                 sync.Mutex
	running            bool
	expectedRandomCode string
	manualMode         bool
}

// OAuthResult 包含 OAuth 回调结果
type OAuthResult struct {
	Code  string
	State string
	Error string
}

// NewCodefreeOAuthServer 创建新的 OAuth 回调服务器
func NewCodefreeOAuthServer() *CodefreeOAuthServer {
	return &CodefreeOAuthServer{
		resultChan: make(chan *OAuthResult, 1),
		errorChan:  make(chan error, 1),
		listenAddr: "127.0.0.1", // 默认监听本地地址
	}
}

// SetListenAddr 设置监听地址
// 例如: "127.0.0.1" (默认), "0.0.0.0" (所有接口)
func (s *CodefreeOAuthServer) SetListenAddr(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if addr != "" {
		s.listenAddr = addr
	}
}

// GetListenAddr 获取当前监听地址
func (s *CodefreeOAuthServer) GetListenAddr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listenAddr
}

// GenerateRandomCode 生成随机验证码
// 使用加密安全的随机数生成器，返回 hex 编码的字符串
func GenerateRandomCode() (string, error) {
	b := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// findAvailablePort 查找可用端口
func findAvailablePort(listenAddr string) (int, error) {
	addr := listenAddr
	if addr == "" {
		addr = "127.0.0.1"
	}
	listener, err := net.Listen("tcp", addr+":0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close()
	}()

	addr2 := listener.Addr().(*net.TCPAddr)
	return addr2.Port, nil
}

// Start 启动 OAuth 回调服务器
func (s *CodefreeOAuthServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// 使用配置的监听地址
	listenAddr := s.listenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}

	// 查找可用端口
	port, err := findAvailablePort(listenAddr)
	if err != nil {
		return NewCodefreeError(ErrServerStartFailed, err)
	}
	s.port = port

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2callback", s.handleCallback)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", listenAddr, s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.server = server
	s.running = true

	// 在 goroutine 中启动服务器
	// 注意：捕获 server 变量，避免竞态访问 s.server
	go func(srv *http.Server) {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.errorChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}(server)

	return nil
}

// Stop 停止 OAuth 回调服务器
func (s *CodefreeOAuthServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.server == nil {
		return nil
	}

	log.Debug("Stopping Codefree OAuth callback server")

	// Use context.Background() as fallback if ctx is nil
	if ctx == nil {
		ctx = context.Background()
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.server.Shutdown(shutdownCtx)
	s.running = false
	s.server = nil

	return err
}

// WaitForCallback 等待 OAuth 回调
func (s *CodefreeOAuthServer) WaitForCallback(timeout time.Duration) (*OAuthResult, error) {
	select {
	case result := <-s.resultChan:
		return result, nil
	case err := <-s.errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, ErrCallbackTimeout
	}
}

// handleCallback 处理 OAuth 回调
func (s *CodefreeOAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	log.WithField("remote_addr", r.RemoteAddr).Debug("Received Codefree OAuth callback")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	randomCode := query.Get("randomCode")
	errorParam := query.Get("error")

	// 检查错误参数
	if errorParam != "" {
		log.WithFields(log.Fields{
			"error": errorParam,
			"state": state,
		}).Error("OAuth error received")
		result := &OAuthResult{Error: errorParam}
		s.sendResult(result)
		s.writeErrorPage(w, errorParam)
		return
	}

	// 验证必填参数
	if code == "" {
		log.WithField("state", state).Error("No authorization code received")
		result := &OAuthResult{Error: "no_code"}
		s.sendResult(result)
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// 验证 randomCode
	s.mu.Lock()
	expectedCode := s.expectedRandomCode
	s.mu.Unlock()

	if expectedCode != "" && randomCode != expectedCode {
		log.WithFields(log.Fields{
			"expected": expectedCode,
			"actual":   randomCode,
		}).Error("Invalid randomCode")
		result := &OAuthResult{Error: "invalid_random_code"}
		s.sendResult(result)
		http.Error(w, "Invalid randomCode", http.StatusBadRequest)
		return
	}

	// 发送成功结果
	result := &OAuthResult{
		Code:  code,
		State: state,
	}
	s.sendResult(result)

	// 显示成功页面
	s.writeSuccessPage(w)
}

// sendResult 发送 OAuth 结果到通道
func (s *CodefreeOAuthServer) sendResult(result *OAuthResult) {
	select {
	case s.resultChan <- result:
		log.Debug("OAuth result sent to channel")
	default:
		log.Warn("OAuth result channel is full, result dropped")
	}
}

// writeSuccessPage 写入成功页面
func (s *CodefreeOAuthServer) writeSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>认证成功</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
        }
        .container {
            text-align: center;
            background: white;
            padding: 60px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.2);
        }
        .icon {
            font-size: 80px;
            margin-bottom: 20px;
        }
        h1 {
            margin: 0 0 20px 0;
            color: #667eea;
        }
        p {
            color: #666;
            font-size: 16px;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✅</div>
        <h1>认证成功！</h1>
        <p>Codefree 凭据已保存。</p>
        <p>您现在可以关闭此窗口。</p>
    </div>
</body>
</html>`

	_, _ = w.Write([]byte(html))
}

// writeErrorPage 写入错误页面
func (s *CodefreeOAuthServer) writeErrorPage(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>认证失败</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #ff6b6b 0%%, #ee5a6f 100%%);
            color: #333;
        }
        .container {
            text-align: center;
            background: white;
            padding: 60px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.2);
        }
        .icon {
            font-size: 80px;
            margin-bottom: 20px;
        }
        h1 {
            margin: 0 0 20px 0;
            color: #ee5a6f;
        }
        p {
            color: #666;
            font-size: 16px;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">❌</div>
        <h1>认证失败</h1>
        <p>错误: %s</p>
        <p>请重试或联系支持。</p>
    </div>
</body>
</html>`, errorMsg)

	_, _ = w.Write([]byte(html))
}

// BuildStateParam 构造 base64 编码的 state 参数
func (s *CodefreeOAuthServer) BuildStateParam(randomCode string) string {
	// state 格式: http://127.0.0.1:{port}/oauth2callback?from=cli&randomCode={randomCode}
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/oauth2callback?from=cli&randomCode=%s", s.port, randomCode)

	// Base64 编码
	return base64.StdEncoding.EncodeToString([]byte(callbackURL))
}

// GetPort 获取当前端口
func (s *CodefreeOAuthServer) GetPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// SetExpectedRandomCode 设置期望的 randomCode
func (s *CodefreeOAuthServer) SetExpectedRandomCode(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectedRandomCode = code
}

// IsRunning 检查服务器是否正在运行
func (s *CodefreeOAuthServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// SetManualMode 设置手动模式
func (s *CodefreeOAuthServer) SetManualMode(manual bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manualMode = manual
}

// IsManualMode 检查是否为手动模式
func (s *CodefreeOAuthServer) IsManualMode() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.manualMode
}

// ParseStateFromCallback 从回调 URL 解析 state 参数
func ParseStateFromCallback(callbackURL string) (string, error) {
	// 解析 base64 编码的 state
	decoded, err := base64.StdEncoding.DecodeString(callbackURL)
	if err != nil {
		return "", fmt.Errorf("failed to decode state: %w", err)
	}

	return string(decoded), nil
}

// ExtractCodeFromURL 从回调 URL 提取 code 参数
func ExtractCodeFromURL(urlStr string) (string, error) {
	// 使用标准库解析 URL，正确处理 URL 编码
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	code := parsedURL.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("code parameter not found in URL")
	}

	return code, nil
}

// ExtractRandomCodeFromState 从 state 参数中提取 randomCode
func ExtractRandomCodeFromState(state string) (string, error) {
	// 解码 base64
	decoded, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return "", fmt.Errorf("failed to decode state: %w", err)
	}

	// 使用标准库解析 URL，正确处理 URL 编码
	parsedURL, err := url.Parse(string(decoded))
	if err != nil {
		return "", fmt.Errorf("failed to parse state URL: %w", err)
	}

	randomCode := parsedURL.Query().Get("randomCode")
	if randomCode == "" {
		return "", fmt.Errorf("randomCode not found in state")
	}

	return randomCode, nil
}

// StateParam 表示 state 参数的结构
type StateParam struct {
	CallbackURL string `json:"callback_url"`
	RandomCode  string `json:"random_code"`
}

// BuildStateParamJSON 构造 JSON 格式的 state 参数
// 注意：这是备用方案，当前实现使用 URL 格式 (BuildStateParam)
// 保留此方法以备未来需要切换到 JSON 格式
func BuildStateParamJSON(port int, randomCode string) string {
	state := StateParam{
		CallbackURL: fmt.Sprintf("http://127.0.0.1:%d/oauth2callback", port),
		RandomCode:  randomCode,
	}

	data, _ := json.Marshal(state)
	return base64.StdEncoding.EncodeToString(data)
}

// ParseStateParamJSON 解析 JSON 格式的 state 参数
// 注意：这是备用方案，当前实现使用 URL 格式
func ParseStateParamJSON(state string) (*StateParam, error) {
	decoded, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	var param StateParam
	if err := json.Unmarshal(decoded, &param); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	return &param, nil
}

// GetPortString 获取已分配端口的字符串表示
func (s *CodefreeOAuthServer) GetPortString() string {
	return strconv.Itoa(s.GetPort())
}

// GetRandomPortString 是 GetPortString 的别名，保留以向后兼容
// Deprecated: 使用 GetPortString 代替
func (s *CodefreeOAuthServer) GetRandomPortString() string {
	return s.GetPortString()
}
