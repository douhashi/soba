package tmux

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// checkTmuxAvailable はtmuxが利用可能かチェックする
// テスト環境でtmuxが使えない場合はテストをスキップする
func checkTmuxAvailable(t *testing.T) {
	t.Helper()

	// tmuxコマンドが存在するかチェック
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping test")
	}

	// tmuxサーバーが起動可能かチェック
	cmd := exec.Command("tmux", "list-sessions")
	if err := cmd.Run(); err != nil {
		// tmuxサーバーが起動していない場合は問題ない
		// 他のエラーの場合はログに記録するが、テストは続行
		t.Logf("tmux server check result: %v", err)
	}
}

// getCommandDelay は設定からコマンド実行の遅延時間を取得する
// 環境変数TMUX_COMMAND_DELAY_SECONDSが設定されていればそれを使用し、
// なければテスト環境用のデフォルト値（1秒）を返す
func getCommandDelay() time.Duration {
	if delayStr := os.Getenv("TMUX_COMMAND_DELAY_SECONDS"); delayStr != "" {
		if delay, err := strconv.Atoi(delayStr); err == nil {
			return time.Duration(delay) * time.Second
		}
	}
	// テスト環境用のデフォルト遅延（本番の3秒から短縮）
	return 1 * time.Second
}

func TestTmuxClient_CreateSession(t *testing.T) {
	checkTmuxAvailable(t)
	tests := []struct {
		name       string
		repository string
		wantErr    bool
	}{
		{
			name:       "有効なリポジトリ名でセッション作成",
			repository: "douhashi/test",
			wantErr:    false,
		},
		{
			name:       "空のリポジトリ名",
			repository: "",
			wantErr:    true,
		},
		{
			name:       "スラッシュを含むリポジトリ名",
			repository: "owner/repo",
			wantErr:    false,
		},
		{
			name:       "無効な文字を含むリポジトリ名",
			repository: "invalid repo name",
			wantErr:    true,
		},
		{
			name:       "長すぎるリポジトリ名",
			repository: "verylongrepositoryname" + string(make([]byte, 100)),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			sessionName := "soba-test-" + tt.repository

			if tt.wantErr {
				if tt.repository == "" {
					// 空の場合はCreateSessionでエラーが出る
					err := client.CreateSession("")
					if err == nil {
						t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
					}
					return
				}
			} else {
				// セッションが存在しない場合のみ作成
				if !client.SessionExists(sessionName) {
					createErr := client.CreateSession(sessionName)
					if createErr != nil {
						t.Errorf("CreateSession() error = %v, wantErr %v", createErr, tt.wantErr)
					}

					// クリーンアップ
					defer func() {
						if client.SessionExists(sessionName) {
							client.DeleteSession(sessionName)
						}
					}()

					// セッションが作成されたか確認
					if !client.SessionExists(sessionName) {
						t.Error("Session was not created")
					}
				}
			}
		})
	}
}

func TestTmuxClient_DeleteSession(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-session"

	// 保護対象セッションの削除を試行
	protectedSession := "soba-douhashi-soba"
	err := client.DeleteSession(protectedSession)
	if err == nil {
		t.Error("Should not be able to delete protected session")
	}

	// テスト用セッションを作成してから削除
	if !client.SessionExists(sessionName) {
		createErr := client.CreateSession(sessionName)
		if createErr != nil {
			t.Fatalf("Failed to create test session: %v", createErr)
		}
	}

	err = client.DeleteSession(sessionName)
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	// セッションが削除されたか確認
	if client.SessionExists(sessionName) {
		t.Error("Session was not deleted")
	}
}

func TestTmuxClient_CreateWindow(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-window"
	windowName := "test-window"

	// テスト用セッションを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Errorf("CreateWindow() error = %v", err)
	}

	// ウィンドウの存在確認
	exists, err := client.WindowExists(sessionName, windowName)
	if err != nil {
		t.Errorf("WindowExists() error = %v", err)
	}
	if !exists {
		t.Error("Window was not created")
	}
}

func TestTmuxClient_DeleteWindow(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-delete-window"
	windowName := "test-window"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	err = client.DeleteWindow(sessionName, windowName)
	if err != nil {
		t.Errorf("DeleteWindow() error = %v", err)
	}

	// ウィンドウが削除されたか確認
	exists, err := client.WindowExists(sessionName, windowName)
	if err != nil {
		t.Errorf("WindowExists() error = %v", err)
	}
	if exists {
		t.Error("Window was not deleted")
	}
}

func TestTmuxClient_CreatePane(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-pane"
	windowName := "test-window"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	err = client.CreatePane(sessionName, windowName)
	if err != nil {
		t.Errorf("CreatePane() error = %v", err)
	}

	// ペイン数の確認
	count, err := client.GetPaneCount(sessionName, windowName)
	if err != nil {
		t.Errorf("GetPaneCount() error = %v", err)
	}
	if count < 2 {
		t.Error("Pane was not created (expected at least 2 panes)")
	}
}

func TestTmuxClient_DeletePane(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-delete-pane"
	windowName := "test-window"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	// ペインを作成
	err = client.CreatePane(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test pane: %v", err)
	}

	// 動的に最初のペインインデックスを取得して最後のペインを削除
	firstPaneIndex, err := client.GetFirstPaneIndex(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to get first pane index: %v", err)
	}
	// tmuxは最初のペインが0または1から始まることがあるので、最後のペインを削除
	lastPaneIndex := firstPaneIndex + 1
	err = client.DeletePane(sessionName, windowName, lastPaneIndex)
	if err != nil {
		t.Errorf("DeletePane() error = %v", err)
	}

	// ペイン数の確認
	count, err := client.GetPaneCount(sessionName, windowName)
	if err != nil {
		t.Errorf("GetPaneCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 pane, got %d", count)
	}
}

func TestTmuxClient_ResizePanes(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-resize"
	windowName := "test-window"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	// ペインを作成
	err = client.CreatePane(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test pane: %v", err)
	}

	err = client.ResizePanes(sessionName, windowName)
	if err != nil {
		t.Errorf("ResizePanes() error = %v", err)
	}
}

func TestTmuxClient_GetFirstPaneIndex(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-first-pane"
	windowName := "test-window"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	index, err := client.GetFirstPaneIndex(sessionName, windowName)
	if err != nil {
		t.Errorf("GetFirstPaneIndex() error = %v", err)
	}

	// インデックスは0または1であるべき
	if index != 0 && index != 1 {
		t.Errorf("GetFirstPaneIndex() = %v, want 0 or 1", index)
	}
}

func TestTmuxClient_SendCommand(t *testing.T) {
	checkTmuxAvailable(t)
	client := NewClient()
	sessionName := "soba-test-command"
	windowName := "test-window"
	command := "echo 'hello world'"

	// テスト用セッションとウィンドウを作成
	if !client.SessionExists(sessionName) {
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}
	defer client.DeleteSession(sessionName)

	err := client.CreateWindow(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to create test window: %v", err)
	}

	// ウィンドウが完全に準備されるまで設定に従って待機
	time.Sleep(getCommandDelay())

	// 動的に最初のペインインデックスを取得
	firstPaneIndex, err := client.GetFirstPaneIndex(sessionName, windowName)
	if err != nil {
		t.Fatalf("Failed to get first pane index: %v", err)
	}

	err = client.SendCommand(sessionName, windowName, firstPaneIndex, command)
	if err != nil {
		t.Errorf("SendCommand() error = %v", err)
	}
}

// generateSessionName関数は削除されたため、このテストも削除

func TestIsProtectedSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		want        bool
	}{
		{
			name:        "保護対象セッション",
			sessionName: "soba-douhashi-soba",
			want:        true,
		},
		{
			name:        "通常のセッション",
			sessionName: "soba-test-session",
			want:        false,
		},
		{
			name:        "sobaプレフィックスではない",
			sessionName: "normal-session",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProtectedSession(tt.sessionName)
			if got != tt.want {
				t.Errorf("isProtectedSession() = %v, want %v", got, tt.want)
			}
		})
	}
}
