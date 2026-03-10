package cmd

import (
	"fmt"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codefree"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
)

// DoCodefreeRefresh 刷新 Codefree 模型列表
// 注意：此命令用于显示当前可用的模型列表
// 模型的实际路由由 synthesizer 和 registry 包在运行时动态处理
func DoCodefreeRefresh(cfg *config.Config) {
	fmt.Println("Refreshing Codefree model list...")

	// 获取 auth 目录
	authDir, _ := util.ResolveAuthDir(cfg.AuthDir)
	authFilePath := authDir + "/" + codefree.CodefreeCredentialsFilename

	// 加载凭据
	storage, err := codefree.LoadTokenFromFile(authFilePath)
	if err != nil {
		fmt.Printf("Failed to load credentials: %v\n", err)
		fmt.Println("Please run 'codefree-login' first")
		return
	}

	// 检查是否过期
	if storage.IsExpired() {
		fmt.Println("Token has expired. Please run 'codefree-login' again")
		return
	}

	// 获取 CLI 版本
	cliVersion := cfg.CodefreeCliVersion
	if cliVersion == "" {
		cliVersion = "0.3.4" // 默认版本
	}

	// 创建认证服务
	auth := codefree.NewCodefreeAuth(cfg)

	// 获取模型列表
	models, err := auth.FetchModels(storage.IDToken, storage.APIKey, cliVersion)
	if err != nil {
		fmt.Printf("Failed to fetch model list: %v\n", err)
		return
	}

	fmt.Printf("Successfully fetched %d models\n", len(models))

	// 显示模型列表
	fmt.Println("\nAvailable models:")
	for i, model := range models {
		fmt.Printf("%d. %s (Manufacturer: %s, MaxTokens: %d)\n",
			i+1, model.Name, model.Manufacturer, model.MaxTokens)
	}

	fmt.Println("\nModel list refreshed successfully!")
}
