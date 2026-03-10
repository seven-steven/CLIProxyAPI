package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codefree"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	log "github.com/sirupsen/logrus"
)

// DoCodefreeLogin 执行 Codefree OAuth 登录流程
func DoCodefreeLogin(cfg *config.Config, manualMode bool) {
	fmt.Println("Starting Codefree login...")

	// 获取 auth 目录
	authDir, err := util.ResolveAuthDir(cfg.AuthDir)
	if err != nil || authDir == "" {
		// 如果没有配置 auth 目录，使用默认目录
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			fmt.Printf("Failed to get working directory: %v\n", wdErr)
			return
		}
		authDir = wd + "/auth"
		// 创建目录
		if mkErr := os.MkdirAll(authDir, 0755); mkErr != nil && !os.IsExist(mkErr) {
			fmt.Printf("Failed to create auth directory: %v\n", mkErr)
			return
		}
		fmt.Printf("Using default auth directory: %s\n", authDir)
	}
	authFilePath := authDir + "/" + codefree.CodefreeCredentialsFilename

	// 创建 Codefree 认证服务
	auth := codefree.NewCodefreeAuth(cfg)

	// 创建 OAuth 服务器
	server := codefree.NewCodefreeOAuthServer()

	if !manualMode {
		// 尝试启动回调服务器
		if err := server.Start(); err != nil {
			log.Warnf("Failed to start OAuth server: %v, falling back to manual mode", err)
			fmt.Printf("Port conflict detected, using manual mode\n")
			manualMode = true
		}
	}

	// 生成 randomCode
	randomCode, err := codefree.GenerateRandomCode()
	if err != nil {
		fmt.Printf("Failed to generate random code: %v\n", err)
		return
	}

	// 构造 state 参数
	var state string
	if !manualMode {
		server.SetExpectedRandomCode(randomCode)
		state = server.BuildStateParam(randomCode)
	} else {
		// 手动模式：使用固定的 state
		state = "manual_" + randomCode
	}

	// 生成授权 URL
	authURL := auth.GenerateAuthURL(state)

	if manualMode {
		// 手动模式：显示 URL 让用户手动访问
		fmt.Printf("Please visit the following URL to authorize:\n%s\n\n", authURL)
		fmt.Print("After authorization, please enter the code parameter from the URL: ")

		var code string
		if _, err := fmt.Scanln(&code); err != nil {
			fmt.Printf("Failed to read code: %v\n", err)
			return
		}

		// 交换 code 获取 tokens
		tokenResp, err := auth.ExchangeCodeForTokens(code)
		if err != nil {
			fmt.Printf("Failed to exchange code: %v\n", err)
			return
		}

		// 获取 API Key (可选步骤，失败时继续)
		apiKey, err := auth.FetchAPIKey(tokenResp.AccessToken, tokenResp.GetUserID())
		if err != nil {
			log.Warnf("Failed to fetch API key (non-fatal): %v", err)
			fmt.Printf("Warning: Failed to fetch API key, continuing without it: %v\n", err)
			// 不返回，继续保存凭据
		}

		// 保存凭据
		storage := &codefree.CodefreeTokenStorage{
			Type:        "codefree",
			AccessToken: tokenResp.AccessToken,
			IDToken:     tokenResp.GetUserID(),
			APIKey:      apiKey,
			ExpiresIn:   tokenResp.ExpiresIn,
			BaseURL:     codefree.BaseURL,
			TokenType:   tokenResp.TokenType,
		}

		if err := storage.SaveTokenToFile(authFilePath); err != nil {
			fmt.Printf("Failed to save credentials: %v\n", err)
			return
		}

		fmt.Printf("Credentials saved to %s\n", authFilePath)
		fmt.Println("Codefree authentication successful!")
		return
	}

	// 自动模式：打开浏览器
	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If browser doesn't open, visit: %s\n", authURL)

	if err := openBrowser(authURL); err != nil {
		log.Warnf("Failed to open browser: %v", err)
		fmt.Printf("Please manually visit: %s\n", authURL)
	}

	// 等待回调（5 分钟超时）
	fmt.Println("Waiting for authorization callback...")
	result, err := server.WaitForCallback(5 * time.Minute)
	if err != nil {
		_ = server.Stop(context.Background())
		fmt.Printf("Authorization timeout: %v\n", err)
		return
	}

	_ = server.Stop(context.Background())

	if result.Error != "" {
		fmt.Printf("OAuth error: %s\n", result.Error)
		return
	}

	// 交换 code 获取 tokens
	tokenResp, err := auth.ExchangeCodeForTokens(result.Code)
	if err != nil {
		fmt.Printf("Failed to exchange code: %v\n", err)
		return
	}

	// 获取 API Key (可选步骤，失败时继续)
	apiKey, err := auth.FetchAPIKey(tokenResp.AccessToken, tokenResp.GetUserID())
	if err != nil {
		log.Warnf("Failed to fetch API key (non-fatal): %v", err)
		fmt.Printf("Warning: Failed to fetch API key, continuing without it: %v\n", err)
		// 不返回，继续保存凭据
	}

	// 保存凭据
	storage := &codefree.CodefreeTokenStorage{
		Type:        "codefree",
		AccessToken: tokenResp.AccessToken,
		IDToken:     tokenResp.GetUserID(),
		APIKey:      apiKey,
		ExpiresIn:   tokenResp.ExpiresIn,
		BaseURL:     codefree.BaseURL,
		TokenType:   tokenResp.TokenType,
	}

	if err := storage.SaveTokenToFile(authFilePath); err != nil {
		fmt.Printf("Failed to save credentials: %v\n", err)
		return
	}

	fmt.Printf("Credentials saved to %s\n", authFilePath)
	fmt.Println("Codefree authentication successful!")
}

// openBrowser 打开浏览器访问指定 URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
