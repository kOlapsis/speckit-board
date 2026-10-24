package watch

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounce = 40 * time.Millisecond

func Dirs(root, sessionsDir string) []string {
	specs := filepath.Join(root, "specs")
	dirs := []string{specs, filepath.Join(root, ".specify"), sessionsDir}
	if entries, err := os.ReadDir(specs); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(specs, e.Name()))
			}
		}
	}
	return dirs
}

func Start(ctx context.Context, dirs []string, poll time.Duration) <-chan struct{} {
	out := make(chan struct{}, 1)
	w, err := fsnotify.NewWatcher()
	failed := err != nil
	if err == nil {
		for _, d := range dirs {
			if w.Add(d) != nil {
				failed = true
			}
		}
	}
	go func() {
		defer close(out)
		notify := func() {
			select {
			case out <- struct{}{}:
			default:
			}
		}
		var events chan fsnotify.Event
		var errs chan error
		if w != nil {
			defer func() { _ = w.Close() }()
			events, errs = w.Events, w.Errors
		}
		var tick <-chan time.Time
		if failed {
			t := time.NewTicker(poll)
			defer t.Stop()
			tick = t.C
		}
		timer := time.NewTimer(time.Hour)
		timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Create) {
					if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
						_ = w.Add(ev.Name)
					}
				}
				timer.Reset(debounce)
			case _, ok := <-errs:
				if !ok {
					return
				}
				timer.Reset(debounce)
			case <-timer.C:
				notify()
			case <-tick:
				notify()
			}
		}
	}()
	return out
}
