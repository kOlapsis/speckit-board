package tui

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/watch"
)

const pollInterval = time.Second

func Run(opts Options) error {
	if opts.SessionsDir == "" {
		opts.SessionsDir = sessions.Dir()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var dirs []string
	for _, r := range opts.Roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			abs = r
		}
		dirs = append(dirs, watch.Dirs(abs, opts.SessionsDir)...)
	}
	opts.Reload = watch.Start(ctx, dirs, pollInterval)
	_, err := tea.NewProgram(New(opts), tea.WithAltScreen()).Run()
	return err
}
