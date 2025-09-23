package service

import (
	"context"
	"fmt"
	"time"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
	"github.com/douhashi/soba/pkg/app"
	"github.com/douhashi/soba/pkg/logging"
)

// ClosedIssueCleanupService は閉じたIssueに対応するtmuxウィンドウを削除するサービス
type ClosedIssueCleanupService struct {
	githubClient *github.ClientImpl
	tmuxClient   tmux.TmuxClient
	owner        string
	repo         string
	sessionName  string
	enabled      bool
	interval     time.Duration
	log          logging.Logger
}

// NewClosedIssueCleanupService は新しいClosedIssueCleanupServiceを作成する
func NewClosedIssueCleanupService(
	githubClient *github.ClientImpl,
	tmuxClient tmux.TmuxClient,
	owner, repo, sessionName string,
	enabled bool,
	interval time.Duration,
) *ClosedIssueCleanupService {
	var logger logging.Logger
	// appが初期化されている場合のみロガーを取得
	if app.IsInitialized() {
		logger = app.LogFactory().CreateComponentLogger("cleanup")
	}

	return &ClosedIssueCleanupService{
		githubClient: githubClient,
		tmuxClient:   tmuxClient,
		owner:        owner,
		repo:         repo,
		sessionName:  sessionName,
		enabled:      enabled,
		interval:     interval,
		log:          logger,
	}
}

// SetLogger はロガーを設定する
func (s *ClosedIssueCleanupService) SetLogger(logger logging.Logger) {
	if logger != nil {
		s.log = logger
	}
}

// Configure は設定を更新する
func (s *ClosedIssueCleanupService) Configure(owner, repo, sessionName string, enabled bool, interval time.Duration) {
	s.owner = owner
	s.repo = repo
	s.sessionName = sessionName
	s.enabled = enabled
	s.interval = interval
}

// Start はサービスを開始する
func (s *ClosedIssueCleanupService) Start(ctx context.Context) error {
	if !s.enabled {
		if s.log != nil {
			s.log.Info(ctx, "Closed issue cleanup service is disabled")
		}
		return nil
	}

	if s.log != nil {
		s.log.Info(ctx, "Starting closed issue cleanup service",
			logging.Field{Key: "owner", Value: s.owner},
			logging.Field{Key: "repo", Value: s.repo},
			logging.Field{Key: "interval", Value: s.interval})
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 最初の実行
	if err := s.cleanupOnce(ctx); err != nil {
		if s.log != nil {
			s.log.Error(ctx, "Failed to cleanup closed issues", logging.Field{Key: "error", Value: err})
		}
	}

	for {
		select {
		case <-ctx.Done():
			if s.log != nil {
				s.log.Info(ctx, "Stopping closed issue cleanup service")
			}
			return ctx.Err()
		case <-ticker.C:
			if err := s.cleanupOnce(ctx); err != nil {
				if s.log != nil {
					s.log.Error(ctx, "Failed to cleanup closed issues", logging.Field{Key: "error", Value: err})
				}
				// エラーがあっても継続
			}
		}
	}
}

// cleanupOnce は1回のクリーンアップ処理を実行する
func (s *ClosedIssueCleanupService) cleanupOnce(ctx context.Context) error {
	if s.log != nil {
		s.log.Debug(ctx, "Starting cleanup of closed issues")
	}

	// githubClientがnilの場合は何もしない
	if s.githubClient == nil {
		return nil
	}

	// CloseされたIssueの一覧を取得
	opts := github.ListIssuesOptions{
		State: "closed",
	}

	issues, err := s.githubClient.ListIssues(ctx, s.owner, s.repo, opts)
	if err != nil {
		if s.log != nil {
			s.log.Error(ctx, "Failed to list closed issues", logging.Field{Key: "error", Value: err})
		}
		return fmt.Errorf("failed to list closed issues: %w", err)
	}

	if s.log != nil {
		s.log.Debug(ctx, "Found closed issues", logging.Field{Key: "count", Value: len(issues)})
	}

	// tmuxClientがnilの場合も何もしない
	if s.tmuxClient == nil {
		return nil
	}

	// 各Issueに対応するtmuxウィンドウを削除
	for _, issue := range issues {
		windowName := fmt.Sprintf("issue-%d", issue.Number)

		// ウィンドウの存在確認
		exists, err := s.tmuxClient.WindowExists(s.sessionName, windowName)
		if err != nil {
			if s.log != nil {
				s.log.Error(ctx, "Failed to check window existence",
					logging.Field{Key: "session", Value: s.sessionName},
					logging.Field{Key: "window", Value: windowName},
					logging.Field{Key: "error", Value: err})
			}
			continue
		}

		if !exists {
			if s.log != nil {
				s.log.Debug(ctx, "Window does not exist",
					logging.Field{Key: "session", Value: s.sessionName},
					logging.Field{Key: "window", Value: windowName})
			}
			continue
		}

		// ウィンドウを削除
		if err := s.tmuxClient.DeleteWindow(s.sessionName, windowName); err != nil {
			if s.log != nil {
				s.log.Error(ctx, "Failed to delete tmux window",
					logging.Field{Key: "window", Value: windowName},
					logging.Field{Key: "issue", Value: issue.Number},
					logging.Field{Key: "error", Value: err})
			}
			continue
		}

		if s.log != nil {
			s.log.Info(ctx, "Deleted tmux window for closed issue",
				logging.Field{Key: "window", Value: windowName},
				logging.Field{Key: "issue", Value: issue.Number})
		}
	}

	return nil
}
