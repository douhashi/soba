package tmux

// Window represents a tmux window
type Window struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
}
