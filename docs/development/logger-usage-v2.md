# Logger使用ガイド v2

## 概要

新しいContext-aware loggingシステムは、依存性注入とコンテキスト追跡を重視した設計です。

## 基本設計

### Logger Factory

```go
import "github.com/douhashi/soba/pkg/logging"

// Factoryの初期化
logFactory, err := logging.NewFactory(logging.Config{
    Level:     "info",    // debug, info, warn, error
    Format:    "json",    // json, text
    Output:    "stderr",  // stdout, stderr, file path
    AddSource: false,     // ソース位置を含めるか
})

// Loggerの作成
logger := logFactory.CreateLogger()

// Component Loggerの作成
daemonLogger := logFactory.CreateComponentLogger("daemon")
```

### Context利用

```go
ctx := context.Background()
ctx = logging.WithRequestID(ctx, "req-123")
ctx = logging.WithComponent(ctx, "service")

logger.Info(ctx, "Processing request",
    logging.Field{Key: "user", Value: "john"},
)
```

## DI実装

### Service層

```go
type DaemonService struct {
    logger logging.Logger
}

func NewDaemonService(logger logging.Logger) *DaemonService {
    return &DaemonService{
        logger: logger.WithFields(
            logging.Field{Key: "service", Value: "daemon"},
        ),
    }
}

func (s *DaemonService) Start(ctx context.Context) error {
    ctx = logging.WithRequestID(ctx, generateID())
    s.logger.Info(ctx, "Service starting")
    // ...
}
```

### テスト

```go
func TestService(t *testing.T) {
    // Mock loggerを使用
    mockFactory, _ := logging.NewMockFactory()
    service := NewService(mockFactory.CreateLogger())

    service.DoWork(context.Background())

    // ログの検証
    mockLogger := mockFactory.Handler.(*logging.MockLogger)
    assert.Equal(t, 1, mockLogger.CountLevel("INFO"))
}
```

## 主な変更点

1. **グローバル状態の排除** - 全ログはDIで注入
2. **Context追跡** - Request ID/Trace IDの自動追加
3. **Component Logger** - サービス毎の独立したLogger
4. **Mock対応** - テスト容易性の向上