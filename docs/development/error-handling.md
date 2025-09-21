# エラーハンドリング

## 概要

sobaプロジェクトでは、一貫性のあるエラーハンドリングを実現するため、独自のエラー機構を実装しています。
この機構により、エラーの分類、コンテキスト情報の付与、適切なログ出力が可能になります。

## エラーコード

以下のエラーコードが定義されています：

- `VALIDATION`: 入力検証エラー
- `NOT_FOUND`: リソースが見つからない
- `INTERNAL`: 内部エラー
- `CONFLICT`: 競合エラー
- `TIMEOUT`: タイムアウト
- `EXTERNAL`: 外部システムエラー
- `UNKNOWN`: 不明なエラー

## 基本的な使い方

### 新しいエラーの作成

```go
import "github.com/douhashi/soba/pkg/errors"

// 検証エラー
err := errors.NewValidationError("invalid email format")

// リソースが見つからない
err := errors.NewNotFoundError("user not found")

// 内部エラー
err := errors.NewInternalError("database connection failed")
```

### エラーのラップ

既存のエラーにコンテキストを追加：

```go
// 基本的なラップ
if err != nil {
    return errors.Wrap(err, "failed to process request")
}

// 特定のエラータイプとしてラップ
if err != nil {
    return errors.WrapValidation(err, "input validation failed")
}
```

### コンテキスト情報の追加

エラーに追加情報を付与：

```go
err := errors.NewValidationError("invalid input")
err = errors.WithContext(err, "field", "email")
err = errors.WithContext(err, "value", userInput)
```

## レイヤー別エラーハンドリング

### ドメイン層

```go
import "github.com/douhashi/soba/internal/domain"

// Issue が見つからない
err := domain.NewIssueNotFoundError(issueNumber)

// フィールド検証エラー
err := domain.NewValidationError("title", "must not be empty")

// フェーズ遷移エラー
err := domain.NewPhaseTransitionError("doing", "todo", issueNum)
```

### インフラストラクチャ層

```go
import "github.com/douhashi/soba/internal/infra"

// GitHub API エラー
err := infra.NewGitHubAPIError(404, "/repos/owner/repo", "not found")

// Tmux実行エラー
err := infra.NewTmuxExecutionError(command, exitCode, stderr)

// 設定ファイルエラー
err := infra.NewConfigLoadError(filePath, "invalid format")
```

### サービス層

```go
import "github.com/douhashi/soba/internal/service"

// ワークフローエラー
err := service.NewWorkflowExecutionError(workflow, phase, reason)

// Issue処理エラー
err := service.NewIssueProcessingError(issueNum, operation, reason)

// デーモンエラー
err := service.NewDaemonError(component, reason)
```

## エラーの判定

```go
// エラーコードの確認
if errors.IsValidationError(err) {
    // 検証エラーの処理
}

if errors.IsNotFoundError(err) {
    // 404レスポンスを返す
}

// エラーチェーンの確認
if errors.Is(err, originalErr) {
    // 元のエラーと一致
}

// エラー型の取得
var baseErr *errors.BaseError
if errors.As(err, &baseErr) {
    code := baseErr.Code
    context := baseErr.Context
}
```

## ベストプラクティス

1. **エラーは即座にラップする**: エラーが発生した場所でコンテキストを追加
2. **適切なエラーコードを使用**: エラーの性質に応じて適切なコードを選択
3. **コンテキスト情報を付与**: デバッグに役立つ情報を追加
4. **エラーメッセージは簡潔に**: 詳細はコンテキストに含める

## 例: 完全なエラーハンドリング

```go
func ProcessIssue(issueNum int) error {
    log := logger.GetLogger()

    // Issue の取得
    issue, err := repository.GetIssue(issueNum)
    if err != nil {
        if errors.IsNotFoundError(err) {
            log.Warn("Issue not found", "number", issueNum)
            return domain.NewIssueNotFoundError(issueNum)
        }
        log.Error("Failed to get issue", "error", err, "number", issueNum)
        return service.WrapServiceError(err, "failed to get issue")
    }

    // 処理の実行
    if err := processWorkflow(issue); err != nil {
        var baseErr *errors.BaseError
        if errors.As(err, &baseErr) {
            log.Error("Workflow failed",
                "error", err,
                "code", baseErr.Code,
                "context", baseErr.Context)
        }
        return service.NewWorkflowExecutionError(
            "issue-processor",
            "execution",
            err.Error())
    }

    return nil
}
```