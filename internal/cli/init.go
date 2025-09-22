package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/internal/infra"
	"github.com/douhashi/soba/internal/infra/git"
	"github.com/douhashi/soba/internal/infra/github"
	"github.com/douhashi/soba/pkg/errors"
	"github.com/douhashi/soba/pkg/logging"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize soba configuration",
		Long:  `Initialize soba configuration by creating a .soba/config.yml file in the current directory`,
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	err := runInitWithClient(context.Background(), args, nil)
	if err == nil {
		cmd.Printf("Successfully created config file\n")
	}
	return err
}

// runInitWithClient allows dependency injection for testing
func runInitWithClient(ctx context.Context, _ []string, gitHubClient GitHubLabelsClient) error {
	log := logging.NewMockLogger()

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error(ctx, "Failed to get current directory", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to get current directory")
	}

	// Check if current directory is a git repository
	gitClient, err := git.NewClient(currentDir)
	if err != nil {
		log.Error(ctx, "Current directory is not a git repository", logging.Field{Key: "error", Value: err.Error()})
		return errors.NewValidationError("current directory is not a git repository")
	}

	// Try to get repository information
	repository, err := gitClient.GetRepository()
	if err != nil {
		// Log warning but continue with default
		log.Warn(ctx, "Failed to detect repository from git remote", logging.Field{Key: "error", Value: err.Error()})
		repository = ""
	} else {
		log.Info(ctx, "Detected repository from git remote", logging.Field{Key: "repository", Value: repository})
	}

	// Define paths
	sobaDir := filepath.Join(currentDir, ".soba")
	configPath := filepath.Join(sobaDir, "config.yml")

	log.Debug(ctx, "Initializing soba configuration",
		logging.Field{Key: "directory", Value: sobaDir},
		logging.Field{Key: "config", Value: configPath},
	)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		log.Warn(ctx, "Config file already exists", logging.Field{Key: "path", Value: configPath})
		var conflictErr error = errors.NewConflictError("config file already exists")
		conflictErr = errors.WithContext(conflictErr, "path", configPath)
		return conflictErr
	}

	// Create .soba directory if it doesn't exist
	if err := os.MkdirAll(sobaDir, 0755); err != nil {
		if os.IsPermission(err) {
			log.Error(ctx, "Permission denied", logging.Field{Key: "directory", Value: sobaDir})
			return infra.NewConfigLoadError(sobaDir, "permission denied: cannot create directory")
		}
		log.Error(ctx, "Failed to create directory", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to create directory")
	}

	log.Debug(ctx, "Created directory", logging.Field{Key: "path", Value: sobaDir})

	// Generate config template
	var configContent string
	if repository != "" {
		opts := &config.TemplateOptions{
			Repository: repository,
		}
		configContent = config.GenerateTemplateWithOptions(opts)
	} else {
		configContent = config.GenerateTemplate()
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		if os.IsPermission(err) {
			log.Error(ctx, "Permission denied", logging.Field{Key: "file", Value: configPath})
			return infra.NewConfigLoadError(configPath, "permission denied: cannot write file")
		}
		log.Error(ctx, "Failed to write config file", logging.Field{Key: "error", Value: err.Error()})
		return errors.WrapInternal(err, "failed to write config file")
	}

	log.Info(ctx, "Successfully created config file", logging.Field{Key: "path", Value: configPath})

	// Try to create GitHub labels if repository is configured
	if err := createGitHubLabelsIfConfigured(ctx, configPath, gitHubClient, log); err != nil {
		// Log the error but don't fail the init command
		log.Warn(ctx, "Failed to create GitHub labels", logging.Field{Key: "error", Value: err.Error()})
	}

	return nil
}

// GitHubLabelsClient はGitHubラベル操作のインターフェース
type GitHubLabelsClient interface {
	CreateLabel(ctx context.Context, owner, repo string, request github.CreateLabelRequest) (*github.Label, error)
	ListLabels(ctx context.Context, owner, repo string) ([]github.Label, error)
}

// createGitHubLabelsIfConfigured はGitHubリポジトリが設定されている場合にラベルを作成する
func createGitHubLabelsIfConfigured(ctx context.Context, configPath string, client GitHubLabelsClient, log logging.Logger) error {
	// 設定ファイルを読み込む
	cfg, err := config.Load(configPath)
	if err != nil {
		return errors.WrapInternal(err, "failed to load config")
	}

	// リポジトリが設定されていない場合はスキップ
	if cfg.GitHub.Repository == "" {
		log.Debug(ctx, "No GitHub repository configured, skipping label creation")
		return nil
	}

	// リポジトリ文字列からowner/repoを分離
	parts := strings.Split(cfg.GitHub.Repository, "/")
	if len(parts) != 2 {
		log.Warn(ctx, "Invalid repository format", logging.Field{Key: "repository", Value: cfg.GitHub.Repository})
		return nil
	}
	owner, repo := parts[0], parts[1]

	// クライアントが提供されていない場合は作成
	if client == nil {
		tokenProvider := github.NewDefaultTokenProvider()
		githubClient, clientErr := github.NewClient(tokenProvider, &github.ClientOptions{
			Logger: log,
		})
		if clientErr != nil {
			return errors.WrapInternal(clientErr, "failed to create GitHub client")
		}
		client = githubClient
	}

	log.Info(ctx, "Creating GitHub labels", logging.Field{Key: "repository", Value: cfg.GitHub.Repository})

	// 既存のラベルを取得
	existingLabels, err := client.ListLabels(ctx, owner, repo)
	if err != nil {
		return errors.WrapInternal(err, "failed to list existing labels")
	}

	// 既存ラベル名のセットを作成
	existingLabelNames := make(map[string]bool)
	for _, label := range existingLabels {
		existingLabelNames[label.Name] = true
	}

	// sobaラベルを作成
	sobaLabels := github.GetSobaLabels()
	createdCount := 0
	skippedCount := 0

	for _, labelRequest := range sobaLabels {
		if existingLabelNames[labelRequest.Name] {
			log.Debug(ctx, "Label already exists, skipping", logging.Field{Key: "label", Value: labelRequest.Name})
			skippedCount++
			continue
		}

		_, err := client.CreateLabel(ctx, owner, repo, labelRequest)
		if err != nil {
			log.Warn(ctx, "Failed to create label",
				logging.Field{Key: "label", Value: labelRequest.Name},
				logging.Field{Key: "error", Value: err.Error()},
			)
			continue
		}

		log.Debug(ctx, "Created label", logging.Field{Key: "label", Value: labelRequest.Name})
		createdCount++
	}

	log.Info(ctx, "GitHub labels creation completed",
		logging.Field{Key: "created", Value: createdCount},
		logging.Field{Key: "skipped", Value: skippedCount},
		logging.Field{Key: "total", Value: len(sobaLabels)},
	)

	return nil
}
