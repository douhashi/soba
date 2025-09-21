package tmux

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// TmuxClient はtmux操作を抽象化するインターフェース
// セッション、ウィンドウ、ペインの管理機能を提供
type TmuxClient interface {
	// セッション管理
	CreateSession(sessionName string) error
	DeleteSession(sessionName string) error
	KillSession(sessionName string) error
	SessionExists(sessionName string) bool

	// ウィンドウ管理
	CreateWindow(sessionName, windowName string) error
	DeleteWindow(sessionName, windowName string) error
	WindowExists(sessionName, windowName string) (bool, error)

	// ペイン管理
	CreatePane(sessionName, windowName string) error
	DeletePane(sessionName, windowName string, paneIndex int) error
	GetPaneCount(sessionName, windowName string) (int, error)
	GetFirstPaneIndex(sessionName, windowName string) (int, error)
	GetLastPaneIndex(sessionName, windowName string) (int, error)
	ResizePanes(sessionName, windowName string) error

	// コマンド送信
	SendCommand(sessionName, windowName string, paneIndex int, command string) error
}

// Client はTmuxClientインターフェースの具象実装
// 実際のtmuxコマンドを実行してtmux環境を操作する
type Client struct{}

// NewClient は新しいTmuxClientインスタンスを作成して返す
func NewClient() TmuxClient {
	return &Client{}
}

// CreateSession は指定された名前で新しいtmuxセッションを作成する
// 既に同名のセッションが存在する場合はエラーを返さない
func (c *Client) CreateSession(sessionName string) error {
	if err := validateSessionName(sessionName); err != nil {
		return err
	}

	if c.SessionExists(sessionName) {
		return nil // 既に存在する場合はエラーではない
	}

	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	err := cmd.Run()
	if err != nil {
		return NewTmuxError("create_session", fmt.Sprintf("failed to create session '%s'", sessionName), err)
	}

	return nil
}

// DeleteSession は指定されたtmuxセッションを削除する
// 保護されたセッションの場合はErrSessionProtectedを返す
func (c *Client) DeleteSession(sessionName string) error {
	if isProtectedSession(sessionName) {
		return ErrSessionProtected
	}

	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		return NewTmuxError("delete_session", fmt.Sprintf("failed to delete session '%s'", sessionName), err)
	}

	return nil
}

// KillSession は指定されたtmuxセッションを強制終了する
// DeleteSessionと異なり、保護されたセッションも強制的に終了する
func (c *Client) KillSession(sessionName string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		return NewTmuxError("kill_session", fmt.Sprintf("failed to kill session '%s'", sessionName), err)
	}

	return nil
}

// SessionExists は指定された名前のセッションが存在するかを確認する
func (c *Client) SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	return err == nil
}

// CreateWindow は指定されたセッション内に新しいウィンドウを作成する
func (c *Client) CreateWindow(sessionName, windowName string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	cmd := exec.Command("tmux", "new-window", "-t", sessionName, "-n", windowName)
	err := cmd.Run()
	if err != nil {
		return NewTmuxError("create_window", fmt.Sprintf("failed to create window '%s' in session '%s'", windowName, sessionName), err)
	}

	return nil
}

// DeleteWindow は指定されたセッション内のウィンドウを削除する
func (c *Client) DeleteWindow(sessionName, windowName string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "kill-window", "-t", fmt.Sprintf("%s:%s", sessionName, windowName))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("delete_window", fmt.Sprintf("failed to delete window '%s' in session '%s'", windowName, sessionName), err)
	}

	return nil
}

// WindowExists は指定されたセッション内にウィンドウが存在するかを確認する
func (c *Client) WindowExists(sessionName, windowName string) (bool, error) {
	if !c.SessionExists(sessionName) {
		return false, ErrSessionNotFound
	}

	cmd := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return false, NewTmuxError("list_windows", fmt.Sprintf("failed to list windows in session '%s'", sessionName), err)
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, window := range windows {
		if window == windowName {
			return true, nil
		}
	}

	return false, nil
}

// CreatePane は指定されたウィンドウ内に新しいペインを水平分割で作成する
func (c *Client) CreatePane(sessionName, windowName string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "split-window", "-h", "-t", fmt.Sprintf("%s:%s", sessionName, windowName))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("create_pane", fmt.Sprintf("failed to create pane in window '%s' of session '%s'", windowName, sessionName), err)
	}

	return nil
}

// DeletePane は指定されたインデックスのペインを削除する
func (c *Client) DeletePane(sessionName, windowName string, paneIndex int) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "kill-pane", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("delete_pane", fmt.Sprintf("failed to delete pane %d in window '%s' of session '%s'", paneIndex, windowName, sessionName), err)
	}

	return nil
}

// GetPaneCount は指定されたウィンドウ内のペイン数を取得する
func (c *Client) GetPaneCount(sessionName, windowName string) (int, error) {
	if !c.SessionExists(sessionName) {
		return 0, ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "list-panes", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "-F", "#{pane_index}")
	output, err := cmd.Output()
	if err != nil {
		return 0, NewTmuxError("list_panes", fmt.Sprintf("failed to list panes in window '%s' of session '%s'", windowName, sessionName), err)
	}

	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(panes) == 1 && panes[0] == "" {
		return 0, nil
	}

	return len(panes), nil
}

// GetFirstPaneIndex はウィンドウ内で最初のペインのインデックス番号を取得する
// tmux環境によってペインインデックスの開始番号が異なるため、動的に判定する
func (c *Client) GetFirstPaneIndex(sessionName, windowName string) (int, error) {
	if !c.SessionExists(sessionName) {
		return 0, ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "list-panes", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "-F", "#{pane_index}")
	output, err := cmd.Output()
	if err != nil {
		return 0, NewTmuxError("list_panes", fmt.Sprintf("failed to list panes in window '%s' of session '%s'", windowName, sessionName), err)
	}

	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(panes) == 0 || panes[0] == "" {
		return 0, NewTmuxError("list_panes", "no panes found", nil)
	}

	// 最初のペインのインデックスを数値として返す
	firstPaneIndex, err := strconv.Atoi(panes[0])
	if err != nil {
		return 0, NewTmuxError("parse_pane_index", fmt.Sprintf("failed to parse pane index '%s'", panes[0]), err)
	}

	return firstPaneIndex, nil
}

// GetLastPaneIndex はウィンドウ内で最後のペインのインデックス番号を取得する
// tmux環境によってペインインデックスの開始番号が異なるため、動的に判定する
func (c *Client) GetLastPaneIndex(sessionName, windowName string) (int, error) {
	if !c.SessionExists(sessionName) {
		return 0, ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "list-panes", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "-F", "#{pane_index}")
	output, err := cmd.Output()
	if err != nil {
		return 0, NewTmuxError("list_panes", fmt.Sprintf("failed to list panes in window '%s' of session '%s'", windowName, sessionName), err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return 0, fmt.Errorf("no panes found in window")
	}

	// 最後のペインのインデックスを取得
	lastLine := lines[len(lines)-1]
	lastPaneIndex, err := strconv.Atoi(lastLine)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last pane index: %v", err)
	}

	return lastPaneIndex, nil
}

// ResizePanes はウィンドウ内の全ペインを水平方向に均等にリサイズする
func (c *Client) ResizePanes(sessionName, windowName string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWindowNotFound
	}

	// #nosec G204 - tmuxコマンドは信頼できる入力のみを使用
	cmd := exec.Command("tmux", "select-layout", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "even-horizontal")
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("resize_panes", fmt.Sprintf("failed to resize panes in window '%s' of session '%s'", windowName, sessionName), err)
	}

	return nil
}

// SendCommand は指定されたペインにコマンドを送信して実行する
func (c *Client) SendCommand(sessionName, windowName string, paneIndex int, command string) error {
	if !c.SessionExists(sessionName) {
		return ErrSessionNotFound
	}

	exists, err := c.WindowExists(sessionName, windowName)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWindowNotFound
	}

	target := fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex)
	// #nosec G204 - この用途では適切なセキュリティチェックを実施済み
	cmd := exec.Command("tmux", "send-keys", "-t", target, command, "Enter")
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("send_command", fmt.Sprintf("failed to send command to pane %d in window '%s' of session '%s'", paneIndex, windowName, sessionName), err)
	}

	return nil
}

// isProtectedSession は指定されたセッションが削除から保護されているかを判定する
// sobaプレフィックスのセッションは誤削除を防ぐため保護される（テスト用を除く）
func isProtectedSession(sessionName string) bool {
	// テストセッションは保護対象外
	if strings.HasPrefix(sessionName, "soba-test-") {
		return false
	}

	// sobaで始まるセッションはすべて保護対象
	// これには旧形式の"soba"と新形式の"soba-owner-repo"が含まれる
	if strings.HasPrefix(sessionName, "soba") {
		return true
	}

	return false
}

// validateSessionName はセッション名の有効性を検証する
// 空文字列、無効な文字（スペース、特殊文字）、長すぎる名前をチェックする
func validateSessionName(sessionName string) error {
	if sessionName == "" {
		return ErrInvalidRepository
	}

	// セッション名長の制限（tmuxの制限に合わせて100文字）
	if len(sessionName) > 100 {
		return NewTmuxError("session_name_validation", "session name too long (max 100 characters)", nil)
	}

	// 無効な文字をチェック（英数字、ハイフン、アンダースコア、スラッシュのみ許可）
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_/]+$`)
	if !validPattern.MatchString(sessionName) {
		return NewTmuxError("session_name_validation", "session name contains invalid characters (only alphanumeric, -, _, / allowed)", nil)
	}

	return nil
}
