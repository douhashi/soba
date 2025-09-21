package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/douhashi/soba/internal/infra"
)

// TokenProvider はGitHubのアクセストークンを提供するインターフェース
type TokenProvider interface {
	GetToken(ctx context.Context) (string, error)
}

// GhCliTokenProvider は`gh auth token`コマンドからトークンを取得する
type GhCliTokenProvider struct {
	// テスト用にコマンド実行を差し替え可能にする
	commandExecutor func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewGhCliTokenProvider は新しいGhCliTokenProviderを作成する
func NewGhCliTokenProvider() *GhCliTokenProvider {
	return &GhCliTokenProvider{
		commandExecutor: defaultCommandExecutor,
	}
}

// GetToken は`gh auth token`コマンドを実行してトークンを取得する
func (p *GhCliTokenProvider) GetToken(ctx context.Context) (string, error) {
	executor := p.commandExecutor
	if executor == nil {
		executor = defaultCommandExecutor
	}

	output, err := executor(ctx, "gh", "auth", "token")
	if err != nil {
		return "", infra.NewGitHubAPIError(
			0,
			"gh auth token",
			fmt.Sprintf("failed to get token from gh cli: %v", err),
		)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", infra.NewGitHubAPIError(
			0,
			"gh auth token",
			"gh auth token returned empty",
		)
	}

	return token, nil
}

// EnvTokenProvider は環境変数からトークンを取得する
type EnvTokenProvider struct {
	envKey string
}

// NewEnvTokenProvider は新しいEnvTokenProviderを作成する
func NewEnvTokenProvider(envKey string) *EnvTokenProvider {
	return &EnvTokenProvider{
		envKey: envKey,
	}
}

// GetToken は環境変数からトークンを取得する
func (p *EnvTokenProvider) GetToken(ctx context.Context) (string, error) {
	key := p.envKey
	if key == "" {
		key = "GITHUB_TOKEN"
	}

	token := os.Getenv(key)
	if token == "" {
		return "", infra.NewGitHubAPIError(
			0,
			"environment",
			fmt.Sprintf("environment variable %s is not set", key),
		)
	}

	return token, nil
}

// ChainTokenProvider は複数のTokenProviderを順番に試す
type ChainTokenProvider struct {
	providers []TokenProvider
}

// NewChainTokenProvider は新しいChainTokenProviderを作成する
func NewChainTokenProvider(providers ...TokenProvider) *ChainTokenProvider {
	return &ChainTokenProvider{
		providers: providers,
	}
}

// GetToken は各プロバイダーを順番に試してトークンを取得する
func (p *ChainTokenProvider) GetToken(ctx context.Context) (string, error) {
	if len(p.providers) == 0 {
		return "", infra.NewGitHubAPIError(
			0,
			"token-chain",
			"no token providers configured",
		)
	}

	var lastErr error
	for _, provider := range p.providers {
		token, err := provider.GetToken(ctx)
		if err == nil && token != "" {
			return token, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", infra.NewGitHubAPIError(
			0,
			"token-chain",
			fmt.Sprintf("all token providers failed: %v", lastErr),
		)
	}

	return "", infra.NewGitHubAPIError(
		0,
		"token-chain",
		"all token providers failed",
	)
}

// NewDefaultTokenProvider はデフォルトのTokenProviderを作成する
// 1. gh auth token
// 2. GITHUB_TOKEN環境変数
// の順で試行する
func NewDefaultTokenProvider() TokenProvider {
	return NewChainTokenProvider(
		NewGhCliTokenProvider(),
		NewEnvTokenProvider("GITHUB_TOKEN"),
	)
}

// defaultCommandExecutor はデフォルトのコマンド実行関数
func defaultCommandExecutor(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}