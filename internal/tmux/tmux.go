package tmux

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrNotInTmux = errors.New("not running inside tmux")
	ErrNoPane    = errors.New("no Claude Code pane found for this project")
)

type Client struct {
	Run    func(args ...string) (string, error)
	Getenv func(string) string
}

func New() *Client {
	return &Client{
		Run: func(args ...string) (string, error) {
			out, err := exec.Command("tmux", args...).Output()
			return string(out), err
		},
		Getenv: os.Getenv,
	}
}

func (c *Client) Inside() bool {
	return c.Getenv("TMUX") != ""
}

func (c *Client) FindPane(root, target string) (string, error) {
	if target != "" {
		return target, nil
	}
	if !c.Inside() {
		return "", ErrNotInTmux
	}
	cur, _ := c.Run("display-message", "-p", "-t", c.Getenv("TMUX_PANE"), "#{session_name}")
	cur = strings.TrimSpace(cur)
	out, err := c.Run("list-panes", "-a", "-F", "#{session_name}\t#{pane_id}\t#{pane_current_command}\t#{pane_current_path}")
	if err != nil {
		return "", err
	}
	var found string
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 4 || !strings.Contains(f[2], "claude") || !within(f[3], root) {
			continue
		}
		if f[0] == cur {
			return f[1], nil
		}
		if found == "" {
			found = f[1]
		}
	}
	if found == "" {
		return "", ErrNoPane
	}
	return found, nil
}

func within(path, root string) bool {
	return path == root || strings.HasPrefix(path, strings.TrimSuffix(root, "/")+"/")
}

func (c *Client) Send(pane, text string) error {
	if !c.Inside() {
		return ErrNotInTmux
	}
	if _, err := c.Run("send-keys", "-t", pane, "-l", text); err != nil {
		return err
	}
	_, err := c.Run("send-keys", "-t", pane, "Enter")
	return err
}

func (c *Client) Jump(pane string) error {
	if !c.Inside() {
		return ErrNotInTmux
	}
	for _, args := range [][]string{{"select-window", "-t", pane}, {"select-pane", "-t", pane}, {"switch-client", "-t", pane}} {
		if _, err := c.Run(args...); err != nil {
			return err
		}
	}
	return nil
}
