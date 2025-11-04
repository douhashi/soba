package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

// GhCommand はghコマンドのラッパー
type GhCommand struct {
	logger logging.Logger
}

// NewGhCommand は新しいGhCommandインスタンスを作成する
func NewGhCommand() *GhCommand {
	return &GhCommand{
		logger: logging.NewMockLogger(),
	}
}

// NewGhCommandWithLogger はロガーを指定してGhCommandインスタンスを作成する
func NewGhCommandWithLogger(logger logging.Logger) *GhCommand {
	return &GhCommand{
		logger: logger,
	}
}

// IsAvailable はghコマンドが利用可能かチェックする
func (g *GhCommand) IsAvailable() bool {
	cmd := exec.Command("gh", "--version")
	err := cmd.Run()
	return err == nil
}

// IsAuthenticated はghコマンドが認証済みかチェックする
func (g *GhCommand) IsAuthenticated(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "status")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 認証されていない場合、ghは非ゼロのexit codeを返す
		if strings.Contains(string(output), "not logged in") {
			return false, nil
		}
		return false, errors.WrapInternal(err, "failed to check auth status")
	}

	return true, nil
}

// ListLabels はリポジトリのラベル一覧を取得する
func (g *GhCommand) ListLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	repoPath := fmt.Sprintf("%s/%s", owner, repo)
	cmd := exec.CommandContext(ctx, "gh", "label", "list", "--repo", repoPath, "--json", "name,color,description")

	output, err := cmd.Output()
	if err != nil {
		g.logger.Error(ctx, "Failed to list labels",
			logging.Field{Key: "repo", Value: repoPath},
			logging.Field{Key: "error", Value: err.Error()},
		)
		return nil, errors.WrapInternal(err, "failed to list labels")
	}

	var labels []Label
	if err := json.Unmarshal(output, &labels); err != nil {
		return nil, errors.WrapInternal(err, "failed to parse label list")
	}

	return labels, nil
}

// CreateLabel は新しいラベルを作成する
func (g *GhCommand) CreateLabel(ctx context.Context, owner, repo string, request CreateLabelRequest) error {
	repoPath := fmt.Sprintf("%s/%s", owner, repo)

	// 既存のラベルをチェック
	existingLabels, err := g.ListLabels(ctx, owner, repo)
	if err != nil {
		// リストの取得に失敗しても作成を試みる
		g.logger.Warn(ctx, "Failed to list existing labels, attempting to create anyway",
			logging.Field{Key: "error", Value: err.Error()},
		)
	} else {
		// 既存のラベルがある場合はスキップ
		for _, label := range existingLabels {
			if label.Name == request.Name {
				g.logger.Debug(ctx, "Label already exists, skipping",
					logging.Field{Key: "label", Value: request.Name},
				)
				return nil
			}
		}
	}

	// ghコマンドでラベルを作成
	args := []string{
		"label", "create", request.Name,
		"--repo", repoPath,
		"--color", request.Color,
	}

	if request.Description != "" {
		args = append(args, "--description", request.Description)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()

		// ラベルが既に存在する場合はエラーとしない
		if strings.Contains(stderrStr, "already exists") {
			g.logger.Debug(ctx, "Label already exists",
				logging.Field{Key: "label", Value: request.Name},
			)
			return nil
		}

		g.logger.Error(ctx, "Failed to create label",
			logging.Field{Key: "label", Value: request.Name},
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "stderr", Value: stderrStr},
		)
		return errors.WrapInternal(err, fmt.Sprintf("failed to create label %s: %s", request.Name, stderrStr))
	}

	g.logger.Info(ctx, "Created label",
		logging.Field{Key: "label", Value: request.Name},
		logging.Field{Key: "color", Value: request.Color},
	)

	return nil
}

// CreateSobaLabels はsobaワークフローで使用するすべてのラベルを作成する
func (g *GhCommand) CreateSobaLabels(ctx context.Context, owner, repo string) (created int, skipped int, err error) {
	sobaLabels := GetSobaLabels()

	for _, label := range sobaLabels {
		if err := g.CreateLabel(ctx, owner, repo, label); err != nil {
			// エラーログは CreateLabel 内で出力済み
			continue
		}
		created++
	}

	skipped = len(sobaLabels) - created
	return created, skipped, nil
}

// GetSobaLabels はsobaワークフローで使用するラベル定義を返す
func GetSobaLabels() []CreateLabelRequest {
	return []CreateLabelRequest{
		{
			Name:        "soba:todo",
			Color:       "e1e4e8",
			Description: "New issue awaiting processing",
		},
		{
			Name:        "soba:todo:high",
			Color:       "d73a4a",
			Description: "High priority task",
		},
		{
			Name:        "soba:todo:low",
			Color:       "cfd3d7",
			Description: "Low priority task",
		},
		{
			Name:        "soba:queued",
			Color:       "fbca04",
			Description: "Selected for processing",
		},
		{
			Name:        "soba:planning",
			Color:       "d4c5f9",
			Description: "Claude creating implementation plan",
		},
		{
			Name:        "soba:ready",
			Color:       "0e8a16",
			Description: "Plan complete, awaiting implementation",
		},
		{
			Name:        "soba:doing",
			Color:       "1d76db",
			Description: "Claude working on implementation",
		},
		{
			Name:        "soba:review-requested",
			Color:       "f9d71c",
			Description: "PR created, awaiting review",
		},
		{
			Name:        "soba:reviewing",
			Color:       "a2eeef",
			Description: "Claude reviewing PR",
		},
		{
			Name:        "soba:done",
			Color:       "0e8a16",
			Description: "Review approved, ready to merge",
		},
		{
			Name:        "soba:requires-changes",
			Color:       "d93f0b",
			Description: "Review requested modifications",
		},
		{
			Name:        "soba:revising",
			Color:       "ff6347",
			Description: "Claude applying requested changes",
		},
		{
			Name:        "soba:lgtm",
			Color:       "0e8a16",
			Description: "Review approved, target for auto merge",
		},
	}
}