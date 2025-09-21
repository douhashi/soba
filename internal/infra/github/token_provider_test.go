package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("GhCliTokenProvider", func(t *testing.T) {
		t.Run("returns token from gh auth token command", func(t *testing.T) {
			// モックコマンドを作成
			provider := &GhCliTokenProvider{
				commandExecutor: func(ctx context.Context, name string, args ...string) ([]byte, error) {
					assert.Equal(t, "gh", name)
					assert.Equal(t, []string{"auth", "token"}, args)
					return []byte("test-gh-token"), nil
				},
			}

			token, err := provider.GetToken(ctx)
			require.NoError(t, err)
			assert.Equal(t, "test-gh-token", token)
		})

		t.Run("returns error when gh command fails", func(t *testing.T) {
			provider := &GhCliTokenProvider{
				commandExecutor: func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return nil, fmt.Errorf("command failed")
				},
			}

			token, err := provider.GetToken(ctx)
			assert.Error(t, err)
			assert.Empty(t, token)
			assert.Contains(t, err.Error(), "failed to get token from gh cli")
		})

		t.Run("returns error when token is empty", func(t *testing.T) {
			provider := &GhCliTokenProvider{
				commandExecutor: func(ctx context.Context, name string, args ...string) ([]byte, error) {
					return []byte(""), nil
				},
			}

			token, err := provider.GetToken(ctx)
			assert.Error(t, err)
			assert.Empty(t, token)
			assert.Contains(t, err.Error(), "gh auth token returned empty")
		})
	})

	t.Run("EnvTokenProvider", func(t *testing.T) {
		t.Run("returns token from environment variable", func(t *testing.T) {
			// 環境変数を設定
			originalValue := os.Getenv("GITHUB_TOKEN")
			defer func() {
				if originalValue != "" {
					os.Setenv("GITHUB_TOKEN", originalValue)
				} else {
					os.Unsetenv("GITHUB_TOKEN")
				}
			}()

			os.Setenv("GITHUB_TOKEN", "test-env-token")

			provider := &EnvTokenProvider{
				envKey: "GITHUB_TOKEN",
			}

			token, err := provider.GetToken(ctx)
			require.NoError(t, err)
			assert.Equal(t, "test-env-token", token)
		})

		t.Run("returns error when environment variable is not set", func(t *testing.T) {
			// 環境変数をクリア
			originalValue := os.Getenv("TEST_TOKEN")
			defer func() {
				if originalValue != "" {
					os.Setenv("TEST_TOKEN", originalValue)
				}
			}()
			os.Unsetenv("TEST_TOKEN")

			provider := &EnvTokenProvider{
				envKey: "TEST_TOKEN",
			}

			token, err := provider.GetToken(ctx)
			assert.Error(t, err)
			assert.Empty(t, token)
			assert.Contains(t, err.Error(), "environment variable TEST_TOKEN is not set")
		})

		t.Run("uses default GITHUB_TOKEN when envKey is empty", func(t *testing.T) {
			originalValue := os.Getenv("GITHUB_TOKEN")
			defer func() {
				if originalValue != "" {
					os.Setenv("GITHUB_TOKEN", originalValue)
				} else {
					os.Unsetenv("GITHUB_TOKEN")
				}
			}()

			os.Setenv("GITHUB_TOKEN", "default-token")

			provider := &EnvTokenProvider{}

			token, err := provider.GetToken(ctx)
			require.NoError(t, err)
			assert.Equal(t, "default-token", token)
		})
	})

	t.Run("ChainTokenProvider", func(t *testing.T) {
		t.Run("returns token from first successful provider", func(t *testing.T) {
			provider1 := &mockTokenProvider{
				token: "",
				err:   fmt.Errorf("provider1 failed"),
			}
			provider2 := &mockTokenProvider{
				token: "token-from-provider2",
				err:   nil,
			}
			provider3 := &mockTokenProvider{
				token: "token-from-provider3",
				err:   nil,
			}

			chain := &ChainTokenProvider{
				providers: []TokenProvider{provider1, provider2, provider3},
			}

			token, err := chain.GetToken(ctx)
			require.NoError(t, err)
			assert.Equal(t, "token-from-provider2", token)
			assert.True(t, provider1.called)
			assert.True(t, provider2.called)
			assert.False(t, provider3.called) // 3番目は呼ばれない
		})

		t.Run("returns error when all providers fail", func(t *testing.T) {
			provider1 := &mockTokenProvider{
				token: "",
				err:   fmt.Errorf("provider1 failed"),
			}
			provider2 := &mockTokenProvider{
				token: "",
				err:   fmt.Errorf("provider2 failed"),
			}

			chain := &ChainTokenProvider{
				providers: []TokenProvider{provider1, provider2},
			}

			token, err := chain.GetToken(ctx)
			assert.Error(t, err)
			assert.Empty(t, token)
			assert.Contains(t, err.Error(), "all token providers failed")
		})

		t.Run("returns error when no providers are configured", func(t *testing.T) {
			chain := &ChainTokenProvider{
				providers: []TokenProvider{},
			}

			token, err := chain.GetToken(ctx)
			assert.Error(t, err)
			assert.Empty(t, token)
			assert.Contains(t, err.Error(), "no token providers configured")
		})
	})

	t.Run("NewDefaultTokenProvider", func(t *testing.T) {
		t.Run("creates chain with both providers", func(t *testing.T) {
			provider := NewDefaultTokenProvider()

			chain, ok := provider.(*ChainTokenProvider)
			require.True(t, ok)
			assert.Len(t, chain.providers, 2)

			// 最初はGhCliTokenProvider
			_, isGhCli := chain.providers[0].(*GhCliTokenProvider)
			assert.True(t, isGhCli)

			// 次はEnvTokenProvider
			_, isEnv := chain.providers[1].(*EnvTokenProvider)
			assert.True(t, isEnv)
		})
	})
}

// mockTokenProvider is a test double for TokenProvider
type mockTokenProvider struct {
	token  string
	err    error
	called bool
}

func (m *mockTokenProvider) GetToken(ctx context.Context) (string, error) {
	m.called = true
	return m.token, m.err
}

// Integration test (実際のコマンドを使う場合)
func TestGhCliTokenProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 実際にgh auth tokenコマンドが使える環境でのみ実行
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh command not found")
	}

	provider := NewGhCliTokenProvider()
	ctx := context.Background()

	token, err := provider.GetToken(ctx)
	// エラーが発生するかトークンが取得できるか
	// （実行環境によってどちらかになる）
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get token from gh cli")
	} else {
		assert.NotEmpty(t, token)
	}
}