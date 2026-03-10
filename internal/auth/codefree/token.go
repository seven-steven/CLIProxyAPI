package codefree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/misc"
)

// CodefreeTokenStorage 存储 Codefree OAuth 凭据信息
type CodefreeTokenStorage struct {
	// Type 表示认证提供者类型，始终为 "codefree"
	Type string `json:"type"`

	// AccessToken 是 OAuth2 访问令牌
	AccessToken string `json:"access_token"`

	// IDToken 包含用户标识（userId）
	IDToken string `json:"id_token"`

	// APIKey 是加密的 API 密钥
	APIKey string `json:"apikey"`

	// ExpiresIn 是令牌的有效期（秒）
	ExpiresIn int64 `json:"expires_in"`

	// BaseURL 是 Codefree API 的基础 URL
	BaseURL string `json:"baseUrl"`

	// CreatedAt 是凭据创建时间的 Unix 时间戳
	CreatedAt int64 `json:"created_at"`

	// TokenType 是令牌类型
	TokenType string `json:"token_type,omitempty"`

	// Metadata 保存任意键值对
	Metadata map[string]any `json:"-"`

	// DecryptedAPIKey 是解密后的 UUID 格式 API 寁钥（运行时计算）
	DecryptedAPIKey string `json:"-"`

	// Models 是可用模型列表
	Models []ModelInfo `json:"models,omitempty"`
}

// GetDecryptedAPIKey 获取解密后的 UUID 格式 API 密钥
// 如果已解密则返回缓存值，否则进行解密
func (ts *CodefreeTokenStorage) GetDecryptedAPIKey() string {
	if ts.DecryptedAPIKey != "" {
		return ts.DecryptedAPIKey
	}

	// 解密 encryptedApiKey
	decrypted, err := DecryptAPIKey(ts.APIKey)
	if err != nil {
		return ""
	}

	ts.DecryptedAPIKey = decrypted
	return decrypted
}


func (ts *CodefreeTokenStorage) Validate() error {
	var missingFields []string

	if ts.Type == "" {
		missingFields = append(missingFields, "type")
	}
	if ts.APIKey == "" {
		missingFields = append(missingFields, "apikey")
	}
	if ts.IDToken == "" {
		missingFields = append(missingFields, "id_token")
	}
	if ts.BaseURL == "" {
		missingFields = append(missingFields, "baseUrl")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %v", missingFields)
	}

	return nil
}

// IsExpired 检查令牌是否已过期
func (ts *CodefreeTokenStorage) IsExpired() bool {
	if ts.CreatedAt == 0 || ts.ExpiresIn == 0 {
		return true
	}

	expiresAt := ts.GetExpiresAt()
	return time.Now().Unix() > expiresAt
}

// GetExpiresAt 返回过期时间的 Unix 时间戳
func (ts *CodefreeTokenStorage) GetExpiresAt() int64 {
	return ts.CreatedAt + ts.ExpiresIn
}

// SetMetadata 设置元数据
func (ts *CodefreeTokenStorage) SetMetadata(meta map[string]any) {
	ts.Metadata = meta
}

// SaveTokenToFile 将令牌存储序列化为 JSON 文件（原子写入）
func (ts *CodefreeTokenStorage) SaveTokenToFile(authFilePath string) error {
	misc.LogSavingCredentials(authFilePath)

	// 设置类型
	ts.Type = "codefree"

	// 如果未设置创建时间，设置为当前时间
	if ts.CreatedAt == 0 {
		ts.CreatedAt = time.Now().Unix()
	}

	// 创建目录结构（如果不存在）
	if err := os.MkdirAll(filepath.Dir(authFilePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 合并元数据
	data, errMerge := misc.MergeMetadata(ts, ts.Metadata)
	if errMerge != nil {
		return fmt.Errorf("failed to merge metadata: %w", errMerge)
	}

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// 原子写入：先写入临时文件，然后重命名
	tempFile := authFilePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write temporary token file: %w", err)
	}

	// 重命名临时文件为目标文件（原子操作）
	if err := os.Rename(tempFile, authFilePath); err != nil {
		// 清理临时文件
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to rename token file: %w", err)
	}

	return nil
}

// LoadTokenFromFile 从文件加载令牌存储
func LoadTokenFromFile(authFilePath string) (*CodefreeTokenStorage, error) {
	data, err := os.ReadFile(authFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("credentials file not found: %s", authFilePath)
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var storage CodefreeTokenStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// 验证必填字段
	if err := storage.Validate(); err != nil {
		return nil, NewCodefreeError(ErrInvalidCredentials, err)
	}

	return &storage, nil
}
