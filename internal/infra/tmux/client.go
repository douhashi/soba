package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// TmuxClient はtmux操作のインターフェース
type TmuxClient interface {
	// セッション管理
	CreateSession(sessionName string) error
	DeleteSession(sessionName string) error
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
	ResizePanes(sessionName, windowName string) error

	// コマンド送信
	SendCommand(sessionName, windowName string, paneIndex int, command string) error
}

// Client はTmuxClientの実装
type Client struct{}

// NewClient は新しいTmuxClientを作成する
func NewClient() TmuxClient {
	return &Client{}
}

// CreateSession は新しいtmuxセッションを作成する
func (c *Client) CreateSession(sessionName string) error {
	if sessionName == "" {
		return ErrInvalidRepository
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

// DeleteSession はtmuxセッションを削除する
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

// SessionExists はセッションが存在するかを確認する
func (c *Client) SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()
	return err == nil
}

// CreateWindow は新しいウィンドウを作成する
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

// DeleteWindow はウィンドウを削除する
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

	cmd := exec.Command("tmux", "kill-window", "-t", fmt.Sprintf("%s:%s", sessionName, windowName))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("delete_window", fmt.Sprintf("failed to delete window '%s' in session '%s'", windowName, sessionName), err)
	}

	return nil
}

// WindowExists はウィンドウが存在するかを確認する
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

// CreatePane は新しいペインを作成する（水平分割）
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

	cmd := exec.Command("tmux", "split-window", "-h", "-t", fmt.Sprintf("%s:%s", sessionName, windowName))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("create_pane", fmt.Sprintf("failed to create pane in window '%s' of session '%s'", windowName, sessionName), err)
	}

	return nil
}

// DeletePane はペインを削除する
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

	cmd := exec.Command("tmux", "kill-pane", "-t", fmt.Sprintf("%s:%s.%d", sessionName, windowName, paneIndex))
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("delete_pane", fmt.Sprintf("failed to delete pane %d in window '%s' of session '%s'", paneIndex, windowName, sessionName), err)
	}

	return nil
}

// GetPaneCount はウィンドウ内のペイン数を取得する
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

// GetFirstPaneIndex はウィンドウの最初のペインのインデックスを取得する
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

// ResizePanes はペインを均等にリサイズする
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

	cmd := exec.Command("tmux", "select-layout", "-t", fmt.Sprintf("%s:%s", sessionName, windowName), "even-horizontal")
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("resize_panes", fmt.Sprintf("failed to resize panes in window '%s' of session '%s'", windowName, sessionName), err)
	}

	return nil
}

// SendCommand はペインにコマンドを送信する
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
	cmd := exec.Command("tmux", "send-keys", "-t", target, command, "Enter")
	err = cmd.Run()
	if err != nil {
		return NewTmuxError("send_command", fmt.Sprintf("failed to send command to pane %d in window '%s' of session '%s'", paneIndex, windowName, sessionName), err)
	}

	return nil
}

// generateSessionName はリポジトリ名からセッション名を生成する
func generateSessionName(repository string) string {
	if repository == "" {
		return ""
	}

	// スラッシュとドットをハイフンに置換
	sessionName := strings.ReplaceAll(repository, "/", "-")
	sessionName = strings.ReplaceAll(sessionName, ".", "-")

	return "soba-" + sessionName
}

// isProtectedSession は保護されたセッションかどうかを判定する
func isProtectedSession(sessionName string) bool {
	// 現在の開発セッション（soba-douhashi-soba）を保護
	protectedSessions := []string{
		"soba-douhashi-soba",
	}

	for _, protected := range protectedSessions {
		if sessionName == protected {
			return true
		}
	}

	return false
}