package codefree

import (
	"errors"
	"fmt"
	"net/http"
)

// CodefreeError 表示 Codefree 认证相关的错误
type CodefreeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Cause   error  `json:"-"`
}

// Error 实现 error 接口
func (e *CodefreeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *CodefreeError) Unwrap() error {
	return e.Cause
}

// 常见错误类型
var (
	// ErrTokenExpired 表示 Token 已过期
	ErrTokenExpired = &CodefreeError{
		Type:    "token_expired",
		Message: "Token has expired",
		Code:    http.StatusUnauthorized,
	}

	// ErrInvalidCredentials 表示凭据格式错误
	ErrInvalidCredentials = &CodefreeError{
		Type:    "invalid_credentials",
		Message: "Invalid credentials format",
		Code:    http.StatusBadRequest,
	}

	// ErrAuthTimeout 表示 OAuth 超时错误
	ErrAuthTimeout = &CodefreeError{
		Type:    "auth_timeout",
		Message: "Authentication timeout",
		Code:    http.StatusRequestTimeout,
	}

	// ErrCodeExchangeFailed 表示交换授权码失败
	ErrCodeExchangeFailed = &CodefreeError{
		Type:    "code_exchange_failed",
		Message: "Failed to exchange authorization code for tokens",
		Code:    http.StatusBadRequest,
	}

	// ErrAPIKeyFetchFailed 表示获取 API Key 失败
	ErrAPIKeyFetchFailed = &CodefreeError{
		Type:    "apikey_fetch_failed",
		Message: "Failed to fetch API key",
		Code:    http.StatusInternalServerError,
	}

	// ErrServerStartFailed 表示启动回调服务器失败
	ErrServerStartFailed = &CodefreeError{
		Type:    "server_start_failed",
		Message: "Failed to start OAuth callback server",
		Code:    http.StatusInternalServerError,
	}

	// ErrPortInUse 表示端口已被占用
	ErrPortInUse = &CodefreeError{
		Type:    "port_in_use",
		Message: "OAuth callback port is already in use",
		Code:    13, // Special exit code for port-in-use
	}

	// ErrCallbackTimeout 表示等待回调超时
	ErrCallbackTimeout = &CodefreeError{
		Type:    "callback_timeout",
		Message: "Timeout waiting for OAuth callback",
		Code:    http.StatusRequestTimeout,
	}

	// ErrInvalidRandomCode 表示 randomCode 验证失败
	ErrInvalidRandomCode = &CodefreeError{
		Type:    "invalid_random_code",
		Message: "Invalid randomCode in callback",
		Code:    http.StatusBadRequest,
	}
)

// NewCodefreeError 创建一个新的 Codefree 错误，包含原因
func NewCodefreeError(baseErr *CodefreeError, cause error) *CodefreeError {
	return &CodefreeError{
		Type:    baseErr.Type,
		Message: baseErr.Message,
		Code:    baseErr.Code,
		Cause:   cause,
	}
}

// IsCodefreeError 检查错误是否为 CodefreeError
func IsCodefreeError(err error) bool {
	var codefreeErr *CodefreeError
	return errors.As(err, &codefreeErr)
}

// IsTokenExpired 检查错误是否为 Token 过期错误
func IsTokenExpired(err error) bool {
	var codefreeErr *CodefreeError
	if errors.As(err, &codefreeErr) {
		return codefreeErr.Type == "token_expired"
	}
	return false
}

// GetUserFriendlyMessage 返回用户友好的错误消息
func GetUserFriendlyMessage(err error) string {
	if !IsCodefreeError(err) {
		return "An unexpected error occurred. Please try again."
	}

	var codefreeErr *CodefreeError
	errors.As(err, &codefreeErr)

	switch codefreeErr.Type {
	case "token_expired":
		return "Your authentication has expired. Please run codefree-login again."
	case "invalid_credentials":
		return "Invalid credentials format. Please run codefree-login again."
	case "auth_timeout", "callback_timeout":
		return "Authentication timed out. Please try again."
	case "port_in_use":
		return "The required port is already in use. Please try again or use --manual flag."
	case "code_exchange_failed":
		return "Failed to exchange authorization code. Please try again."
	case "apikey_fetch_failed":
		return "Failed to fetch API key. Please try again."
	default:
		return "Authentication failed. Please try again."
	}
}
