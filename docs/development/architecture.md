# システムアーキテクチャ設計

## アーキテクチャ概要

### レイヤードアーキテクチャ
```
┌─────────────────────────────────────┐
│          CLI Layer (cmd/)           │
├─────────────────────────────────────┤
│       Application Layer              │
│         (internal/cli)              │
├─────────────────────────────────────┤
│        Service Layer                │
│       (internal/service)            │
├─────────────────────────────────────┤
│         Domain Layer                │
│        (internal/domain)            │
├─────────────────────────────────────┤
│     Infrastructure Layer            │
│        (internal/infra)             │
└─────────────────────────────────────┘
```

## コンポーネント設計

### 1. CLI Layer
**責務**: ユーザーインターフェース
```go
cmd/
└── soba/
    └── main.go  // エントリーポイント
```

### 2. Application Layer
**責務**: コマンド処理、DI
```go
internal/cli/
├── root.go      // ルートコマンド
├── init.go      // 初期化コマンド
├── start.go     // 起動コマンド
├── status.go    // 状態確認
└── stop.go      // 停止コマンド
```

### 3. Service Layer
**責務**: ビジネスロジック
```go
internal/service/
├── issue_processor.go     // Issue処理
├── workflow_executor.go   // ワークフロー実行
├── daemon.go             // Daemon管理
├── tmux_manager.go       // tmuxセッション
└── git_workspace.go      // Git操作
```

### 4. Domain Layer
**責務**: ビジネスエンティティ
```go
internal/domain/
├── issue.go      // Issueモデル
├── phase.go      // フェーズ定義
├── label.go      // ラベル管理
└── errors.go     // ドメインエラー
```

### 5. Infrastructure Layer
**責務**: 外部システム連携
```go
internal/infra/
├── github/
│   ├── client.go     // GitHub API
│   └── token.go      // 認証
├── tmux/
│   └── client.go     // tmux操作
└── slack/
    └── notifier.go   // 通知
```

## データフロー

### Issue処理フロー
```
GitHub Issue
    ↓
IssueWatcher (ポーリング)
    ↓
IssueProcessor (オーケストレーション)
    ↓
PhaseStrategy (フェーズ判定)
    ↓
WorkflowExecutor (実行)
    ├→ GitWorkspace (worktree)
    └→ TmuxManager (AI実行)
    ↓
GitHub API (結果更新)
```

## シーケンシャル処理アーキテクチャ

### 単一処理ループ
```go
// 1Issue 1プロセス - 並列実行なし
func (w *IssueWatcher) WatchAndProcess(ctx context.Context) {
    for {
        issues := w.fetchTodoIssues()
        for _, issue := range issues {
            // 1つずつ順番に処理
            if err := w.processIssue(ctx, issue); err != nil {
                slog.Error("failed to process issue",
                    "issue_number", issue.Number,
                    "error", err)
                continue
            }
            // 完了まで待機
            w.waitUntilCompleted(ctx, issue)
        }
        time.Sleep(w.interval)
    }
}
```

## 状態管理

### Single Source of Truth
- **GitHub Issueのラベル**: 唯一の信頼できる状態源
- **永続化なし**: ローカルファイルへの状態保存は行わない
- **起動時同期**: 起動時にGitHub APIから現在の状態を取得
- **メモリ内管理**: 実行中の状態はメモリ内でのみ管理

## エラーハンドリング

### エラー階層
```go
// ドメインエラー
type IssueNotFoundError struct {
    Number int
}

// インフラエラー
type GitHubAPIError struct {
    StatusCode int
    Message    string
}

// サービスエラー
type WorkflowExecutionError struct {
    Phase   string
    Cause   error
}
```

### リトライ戦略
```go
type RetryPolicy struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func WithRetry(policy RetryPolicy, fn func() error) error {
    delay := policy.InitialDelay
    for i := 0; i < policy.MaxAttempts; i++ {
        if err := fn(); err == nil {
            return nil
        }
        time.Sleep(delay)
        delay = time.Duration(float64(delay) * policy.Multiplier)
        if delay > policy.MaxDelay {
            delay = policy.MaxDelay
        }
    }
    return ErrMaxRetriesExceeded
}
```

## セキュリティアーキテクチャ

### 認証・認可
```go
// トークン管理
type TokenProvider interface {
    GetToken() (string, error)
}

// 実装
type EnvTokenProvider struct{}     // 環境変数
type GhCliTokenProvider struct{}    // gh CLI
type FileTokenProvider struct{}     // ファイル
```

### シークレット保護
- 環境変数での管理
- 設定ファイルの権限チェック (0600)
- ログでのマスキング

## 監視・可観測性

### メトリクス
```go
type Metrics struct {
    IssuesProcessed   counter
    ProcessingTime    histogram
    ActiveSessions    gauge
    ErrorRate         counter
}
```

### ログ戦略
```go
// 構造化ログ
slog.Info("issue processed",
    "issue_number", issue.Number,
    "phase", phase,
    "duration_ms", duration.Milliseconds(),
    "success", success,
)
```

## デプロイメントアーキテクチャ

### バイナリ配布
```
soba-linux-amd64
soba-linux-arm64
soba-darwin-amd64
soba-darwin-arm64
```

### 設定管理
```yaml
# .soba/config.yml
github:
  token: ${GITHUB_TOKEN}
  repository: owner/repo

workflow:
  interval: 20
  use_tmux: true
```