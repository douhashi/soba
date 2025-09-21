# Logger使用ガイド

## 概要

Sobaプロジェクトでは、Go標準ライブラリの`log/slog`を基盤とした統一的なロギング機構を提供しています。
構造化ログの出力により、デバッグやプロダクション環境での問題分析が容易になります。

## 基本的な使い方

### 初期化

アプリケーション起動時にロガーを初期化します：

```go
import "github.com/douhashi/soba/pkg/logger"

// 開発環境用
logger.Init(logger.Config{
    Environment: "development",  // "development" or "production"
    Level:       slog.LevelDebug,
})

// プロダクション環境用
logger.Init(logger.Config{
    Environment: "production",
    Level:       slog.LevelInfo,
})
```

### ロガーの取得

グローバルロガーを取得：

```go
log := logger.GetLogger()
log.Info("Application started")
```

### ログレベル

以下のログレベルが使用可能です：

- `Debug`: デバッグ情報
- `Info`: 一般的な情報
- `Warn`: 警告
- `Error`: エラー

```go
log := logger.GetLogger()

log.Debug("Debugging information", "variable", value)
log.Info("Operation completed", "duration", time.Since(start))
log.Warn("Resource usage high", "usage", 85)
log.Error("Operation failed", "error", err)
```

## 高度な使い方

### フィールドの追加

複数のフィールドを一括で追加：

```go
log := logger.GetLogger()
enhancedLog := logger.WithFields(log, logger.Fields{
    "user_id":    "123",
    "session_id": "abc-def",
    "request_id": "xyz-789",
})

enhancedLog.Info("User action", "action", "login")
// 出力: msg="User action" user_id=123 session_id=abc-def request_id=xyz-789 action=login
```

### エラー情報の構造化

エラーを構造化して記録：

```go
err := someOperation()
if err != nil {
    errorLog := logger.WithError(log, err)
    errorLog.Error("Operation failed")
    // 出力: msg="Operation failed" error="specific error message"
}
```

### コンテキストを使用したロガー管理

リクエスト処理などでコンテキスト経由でロガーを伝播：

```go
// ロガーをコンテキストに格納
ctx := context.Background()
requestLogger := log.With("request_id", generateRequestID())
ctx = logger.WithContext(ctx, requestLogger)

// 別の場所でコンテキストから取得
func handleRequest(ctx context.Context) {
    log := logger.FromContext(ctx)
    log.Info("Processing request")
}
```

### 環境変数による設定

環境変数からロガー設定を読み込み：

```go
// LOG_LEVEL=DEBUG APP_ENV=production として実行
logger.InitFromEnv(os.Stdout)
```

### 実行時のログレベル変更

アプリケーション実行中にログレベルを変更：

```go
// 詳細ログを有効化
logger.SetLevel(slog.LevelDebug)

// 本番用に制限
logger.SetLevel(slog.LevelWarn)
```

## 各レイヤーでの使用例

### CLIレイヤー

```go
package cli

import (
    "github.com/douhashi/soba/pkg/logger"
)

func initConfig() {
    logLevel := slog.LevelInfo
    if verbose {
        logLevel = slog.LevelDebug
    }

    logger.Init(logger.Config{
        Environment: "development",
        Level:       logLevel,
    })

    log := logger.GetLogger()
    log.Debug("Configuration initialized")
}
```

### サービスレイヤー

```go
package service

import (
    "context"
    "github.com/douhashi/soba/pkg/logger"
)

type IssueProcessor struct {
    log *slog.Logger
}

func NewIssueProcessor() *IssueProcessor {
    return &IssueProcessor{
        log: logger.GetLogger().With("component", "issue_processor"),
    }
}

func (p *IssueProcessor) Process(ctx context.Context, issueID int) error {
    log := logger.FromContext(ctx)
    log.Info("Processing issue", "issue_id", issueID)

    // 処理...

    if err != nil {
        errorLog := logger.WithError(log, err)
        errorLog.Error("Failed to process issue")
        return err
    }

    log.Info("Issue processed successfully")
    return nil
}
```

### インフラレイヤー

```go
package github

import (
    "github.com/douhashi/soba/pkg/logger"
)

type Client struct {
    log *slog.Logger
}

func NewClient(token string) *Client {
    log := logger.GetLogger().With("component", "github_client")

    log.Debug("Creating GitHub client")

    return &Client{
        log: log,
    }
}

func (c *Client) GetIssue(id int) (*Issue, error) {
    c.log.Debug("Fetching issue", "id", id)

    // API呼び出し...

    if err != nil {
        c.log.Error("Failed to fetch issue",
            "id", id,
            "error", err)
        return nil, err
    }

    c.log.Debug("Issue fetched successfully", "id", id)
    return issue, nil
}
```

## パフォーマンス考慮事項

### 遅延評価

コストの高い操作は必要なログレベルの時のみ実行：

```go
if log.Enabled(nil, slog.LevelDebug) {
    expensiveData := computeExpensiveDebugData()
    log.Debug("Debug info", "data", expensiveData)
}
```

### 構造化ログのメリット

- **パース可能**: JSONフォーマットにより機械的な処理が可能
- **検索可能**: 特定のフィールドでフィルタリングが容易
- **分析可能**: ログ集計ツールとの連携が簡単

## 出力フォーマット

### 開発環境（テキスト形式）

```
time=2024-01-01T10:00:00.000+09:00 level=INFO source=/path/to/file.go:42 msg="User logged in" user_id=123 session_id=abc
```

### プロダクション環境（JSON形式）

```json
{
  "time": "2024-01-01T10:00:00.000+09:00",
  "level": "INFO",
  "source": {
    "function": "github.com/douhashi/soba/internal/service.Process",
    "file": "/path/to/file.go",
    "line": 42
  },
  "msg": "User logged in",
  "user_id": "123",
  "session_id": "abc"
}
```

## ベストプラクティス

1. **適切なログレベルの使用**
   - Debug: 開発時のデバッグ情報
   - Info: 正常な処理の重要なイベント
   - Warn: 問題になる可能性があるが処理は継続
   - Error: エラーが発生したが処理は継続

2. **構造化フィールドの活用**
   - キー・バリュー形式でコンテキスト情報を追加
   - 一貫性のあるフィールド名を使用

3. **機密情報の除外**
   - パスワード、トークン、個人情報をログに含めない

4. **エラーログの充実**
   - エラーメッセージだけでなく、関連するコンテキストも記録

5. **パフォーマンスへの配慮**
   - 高頻度のループ内での過度なログ出力を避ける
   - デバッグログは本番環境では無効化

## トラブルシューティング

### ログが出力されない

```go
// ログレベルを確認
currentLevel := logger.GetLogger().Handler().Enabled(context.Background(), slog.LevelDebug)
```

### ログローテーション

ログローテーションは外部ツール（logrotate等）で対応することを推奨：

```bash
# /etc/logrotate.d/soba
/var/log/soba/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```