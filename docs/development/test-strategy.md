# テスト戦略

## テストピラミッド

### レベル1: ユニットテスト (70%)
**対象**: 個別関数・メソッド
**ツール**: testing標準パッケージ
**実行時間**: < 10ms/test

```go
func TestParseIssueNumber(t *testing.T) {
    got := ParseIssueNumber("#123")
    want := 123
    if got != want {
        t.Errorf("ParseIssueNumber() = %v, want %v", got, want)
    }
}
```

### レベル2: 統合テスト (20%)
**対象**: モジュール間連携
**ツール**: testify, gomock
**実行時間**: < 100ms/test

```go
func TestIssueProcessor_ProcessIssue(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockGitHub := mocks.NewMockGitHubClient(ctrl)
    mockGitHub.EXPECT().GetIssue(123).Return(testIssue, nil)

    processor := NewIssueProcessor(mockGitHub)
    err := processor.ProcessIssue(context.Background(), 123)
    assert.NoError(t, err)
}
```

### レベル3: E2Eテスト (10%)
**対象**: 完全なワークフロー
**ツール**: 実環境またはtestcontainers
**実行時間**: < 30s/test

## テスト戦略

### 1. テーブル駆動テスト
```go
tests := []struct {
    name    string
    input   string
    want    int
    wantErr bool
}{
    {"valid issue", "#123", 123, false},
    {"invalid format", "abc", 0, true},
    {"negative number", "#-1", 0, true},
}
```

### 2. モック戦略
**外部依存のモック化**:
- GitHub API → gomock
- tmuxコマンド → コマンドモック
- ファイルシステム → afero
- 時刻 → clock interface

### 3. テストデータ管理
```go
// testdata/fixtures.go
var (
    ValidIssue = &Issue{
        Number: 123,
        Title:  "Test Issue",
        Labels: []string{"soba:todo"},
    }
)
```

## カバレッジ目標

### パッケージ別目標
| パッケージ | カバレッジ目標 | 理由 |
|-----------|--------------|------|
| domain | 95% | ビジネスロジックのコア |
| service | 85% | 主要な処理 |
| infra | 70% | 外部連携 |
| cli | 60% | UI層 |
| config | 90% | 設定の重要性 |

### 全体目標: 80%以上

## テスト自動化

### CI/CDパイプライン
```yaml
name: Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

### ローカル実行
```makefile
test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-short:
	go test -short ./...
```

## テストのベストプラクティス

### 1. 明確なテスト名
```go
// Good
func TestIssueProcessor_ProcessIssue_WithValidInput_ReturnsSuccess(t *testing.T)

// Bad
func TestProcess(t *testing.T)
```

### 2. AAA パターン
```go
func TestExample(t *testing.T) {
    // Arrange
    processor := NewProcessor()
    input := "test"

    // Act
    result, err := processor.Process(input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

### 3. テストヘルパー
```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    // セットアップロジック
    return db
}
```

## パフォーマンステスト

### ベンチマーク
```go
func BenchmarkProcessIssue(b *testing.B) {
    processor := NewIssueProcessor()
    issue := &Issue{Number: 123}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        processor.ProcessIssue(context.Background(), issue)
    }
}
```

### 負荷テスト
```go
func TestConcurrentProcessing(t *testing.T) {
    processor := NewIssueProcessor()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            err := processor.ProcessIssue(context.Background(), n)
            assert.NoError(t, err)
        }(i)
    }
    wg.Wait()
}
```

## テストの実行モード

### 短縮モード
```go
if testing.Short() {
    t.Skip("skipping integration test in short mode")
}
```

### 並列実行
```go
func TestParallel(t *testing.T) {
    t.Parallel()
    // テストロジック
}
```

## 失敗時の診断

### デバッグ情報
```go
t.Logf("Processing issue #%d with state %s", issue.Number, issue.State)
```

### エラーメッセージ
```go
assert.Equal(t, expected, actual,
    "Issue #%d should transition from %s to %s",
    issueNumber, fromState, toState)
```