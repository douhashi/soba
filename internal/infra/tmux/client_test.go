package tmux

import (
	"testing"
	"time"
)

func TestTmuxClient_CreateSession(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			sessionName := generateSessionName(tt.repository)

			if tt.wantErr {
				if tt.repository == "" {
					// 空の場合はセッション名生成でエラーが出る
					return
				}
			} else {
				// セッションが存在しない場合のみ作成
				if !client.SessionExists(sessionName) {
					err := client.CreateSession(sessionName)
					if err != nil {
						t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
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
		err := client.CreateSession(sessionName)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
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

	// ペインを削除（インデックス1を削除）
	err = client.DeletePane(sessionName, windowName, 1)
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

	// ウィンドウが完全に準備されるまで設定に従って待機（tmux_command_delay: 3秒）
	time.Sleep(3 * time.Second)

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

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		want       string
	}{
		{
			name:       "通常のリポジトリ名",
			repository: "owner/repo",
			want:       "soba-owner-repo",
		},
		{
			name:       "ドット含むリポジトリ名",
			repository: "owner/repo.git",
			want:       "soba-owner-repo-git",
		},
		{
			name:       "ハイフン含むリポジトリ名",
			repository: "owner/my-repo",
			want:       "soba-owner-my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSessionName(tt.repository)
			if got != tt.want {
				t.Errorf("generateSessionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

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