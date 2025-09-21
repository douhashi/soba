# Go言語コーディング規約

## 基本原則

### 1. Effective Go準拠
- [Effective Go](https://golang.org/doc/effective_go.html)のガイドラインに従う
- Go標準ライブラリのコーディングスタイルを参考にする

### 2. シンプルさ優先
- 明確で読みやすいコード
- 過度な抽象化を避ける
- YAGNIの原則

## 命名規則

### パッケージ名
```go
// Good
package github
package config

// Bad
package githubClient
package ConfigManager
```

### 変数・関数名
```go
// Public
type IssueProcessor struct {}
func NewIssueProcessor() *IssueProcessor {}

// Private
type issueCache struct {}
func parseConfig() error {}
```

### 定数
```go
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)
```

## 構造体とインターフェース

### インターフェース定義
```go
// 小さく保つ（1-3メソッド）
type Reader interface {
    Read([]byte) (int, error)
}

// 使用側で定義
type IssueService interface {
    GetIssue(number int) (*Issue, error)
}
```

### エラーハンドリング
```go
// カスタムエラー型
type ValidationError struct {
    Field string
    Value interface{}
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %v", e.Field, e.Value)
}

// エラーラップ
if err != nil {
    return fmt.Errorf("failed to process issue #%d: %w", issueNumber, err)
}
```

## 並行処理

### Context使用
```go
func ProcessIssue(ctx context.Context, issue *Issue) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // 処理継続
    }
}
```

### Goroutineパターン
```go
// WaitGroup使用
var wg sync.WaitGroup
for _, issue := range issues {
    wg.Add(1)
    go func(i *Issue) {
        defer wg.Done()
        processIssue(i)
    }(issue)
}
wg.Wait()
```

## ファイル構造

### インポート順序
```go
import (
    // 標準ライブラリ
    "context"
    "fmt"

    // 外部ライブラリ
    "github.com/google/go-github/v64/github"
    "github.com/spf13/cobra"

    // 内部パッケージ
    "github.com/douhashi/soba/internal/config"
    "github.com/douhashi/soba/internal/service"
)
```

## テスト

### テーブル駆動テスト
```go
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {"valid config", validConfig, false},
        {"missing token", invalidConfig, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateConfig(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
```

## ログ

### 構造化ログ
```go
slog.Info("processing issue",
    "issue_number", issue.Number,
    "phase", phase,
    "duration", time.Since(start),
)
```

## コメント

### パッケージコメント
```go
// Package service provides the core business logic for issue processing,
// workflow execution, and daemon management.
package service
```

### 関数コメント
```go
// ProcessIssue handles the complete lifecycle of a GitHub issue,
// from initial detection through implementation to merge.
func ProcessIssue(ctx context.Context, issue *Issue) error {
    // 実装
}
```

## 開発コマンド

### ビルド・テスト
```bash
# ビルド
make build

# テスト実行（推奨）
make test

# テストカバレッジ確認
make test-coverage

# リント実行
make lint

# フォーマット
make fmt

# クリーン
make clean
```

## リンター設定

### golangci-lint
```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - unused
```

## 禁止事項

1. **panic使用禁止**（ライブラリ初期化除く）
2. **グローバル変数禁止**（設定除く）
3. **init関数の複雑な処理禁止**
4. **空のインターフェース濫用禁止**