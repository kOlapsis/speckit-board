package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StateWaiting = "waiting"
	StateDone    = "done"
	StateWorking = "working"
)

type Light int

const (
	Gray Light = iota
	Amber
	Green
)

type Session struct {
	ID        string    `json:"session_id"`
	Cwd       string    `json:"cwd"`
	State     string    `json:"state"`
	Message   string    `json:"message,omitempty"`
	Pane      string    `json:"pane,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func Dir() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	return filepath.Join(base, "speckit-board", "sessions")
}

func List(dir string) []Session {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Session
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(b, &s) == nil && s.ID != "" {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}

func inside(cwd, root string) bool {
	return cwd == root || strings.HasPrefix(cwd, strings.TrimSuffix(root, "/")+"/")
}

func Summarize(all []Session, root string) (Light, *Session) {
	var latest *Session
	for i := range all {
		s := &all[i]
		if !inside(s.Cwd, root) {
			continue
		}
		better := latest == nil
		if !better {
			w, lw := s.State == StateWaiting, latest.State == StateWaiting
			better = (w && !lw) || (w == lw && s.UpdatedAt.After(latest.UpdatedAt))
		}
		if better {
			latest = s
		}
	}
	switch {
	case latest == nil:
		return Gray, nil
	case latest.State == StateWaiting:
		return Amber, latest
	case latest.State == StateDone:
		return Green, latest
	}
	return Gray, latest
}

func FirstWaiting(all []Session) *Session {
	var best *Session
	for i := range all {
		if s := &all[i]; s.State == StateWaiting && (best == nil || s.UpdatedAt.After(best.UpdatedAt)) {
			best = s
		}
	}
	return best
}
