package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/internal/infra/tmux"
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
	log          *zap.SugaredLogger
}

// NewClosedIssueCleanupService は新しいClosedIssueCleanupServiceを作成する
func NewClosedIssueCleanupService(
	githubClient *github.ClientImpl,
	tmuxClient tmux.TmuxClient,
	owner, repo, sessionName string,
	enabled bool,
	interval time.Duration,
) *ClosedIssueCleanupService {
	return &ClosedIssueCleanupService{
		githubClient: githubClient,
		tmuxClient:   tmuxClient,
		owner:        owner,
		repo:         repo,
		sessionName:  sessionName,
		enabled:      enabled,
		interval:     interval,
		log:          zap.NewNop().Sugar(),
	}
}

// SetLogger はロガーを設定する
func (s *ClosedIssueCleanupService) SetLogger(log *zap.SugaredLogger) {
	s.log = log
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
			s.log.Info("Closed issue cleanup service is disabled")
		}
		return nil
	}

	if s.log != nil {
		s.log.Info("Starting closed issue cleanup service",
			"owner", s.owner,
			"repo", s.repo,
			"interval", s.interval)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 最初の実行
	if err := s.cleanupOnce(ctx); err != nil {
		if s.log != nil {
			s.log.Errorw("Failed to cleanup closed issues", "error", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			if s.log != nil {
				s.log.Info("Stopping closed issue cleanup service")
			}
			return ctx.Err()
		case <-ticker.C:
			if err := s.cleanupOnce(ctx); err != nil {
				if s.log != nil {
					s.log.Errorw("Failed to cleanup closed issues", "error", err)
				}
				// エラーがあっても継続
			}
		}
	}
}

// cleanupOnce は1回のクリーンアップ処理を実行する
func (s *ClosedIssueCleanupService) cleanupOnce(ctx context.Context) error {
	if s.log != nil {
		s.log.Debug("Starting cleanup of closed issues")
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
			s.log.Errorw("Failed to list closed issues", "error", err)
		}
		return fmt.Errorf("failed to list closed issues: %w", err)
	}

	if s.log != nil {
		s.log.Debugw("Found closed issues", "count", len(issues))
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
				s.log.Errorw("Failed to check window existence",
					"session", s.sessionName,
					"window", windowName,
					"error", err)
			}
			continue
		}

		if !exists {
			if s.log != nil {
				s.log.Debugw("Window does not exist",
					"session", s.sessionName,
					"window", windowName)
			}
			continue
		}

		// ウィンドウを削除
		if err := s.tmuxClient.DeleteWindow(s.sessionName, windowName); err != nil {
			if s.log != nil {
				s.log.Errorw("Failed to delete tmux window",
					"window", windowName,
					"issue", issue.Number,
					"error", err)
			}
			continue
		}

		if s.log != nil {
			s.log.Infow("Deleted tmux window for closed issue",
				"window", windowName,
				"issue", issue.Number)
		}
	}

	return nil
}
