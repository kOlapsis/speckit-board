package hook

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kOlapsis/speckit-board/internal/events"
	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/spec"
)

type Env struct {
	SessionsDir string
	Now         func() time.Time
	Getenv      func(string) string
}

type payload struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Event     string `json:"hook_event_name"`
	Message   string `json:"message"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

func DefaultEnv() Env {
	return Env{SessionsDir: sessions.Dir(), Now: time.Now, Getenv: os.Getenv}
}

func Handle(in io.Reader, env Env) error {
	var p payload
	if err := json.NewDecoder(in).Decode(&p); err != nil {
		return err
	}
	if !sessions.ValidID(p.SessionID) {
		return errors.New("missing or invalid session_id")
	}
	switch p.Event {
	case "Notification":
		return touch(p, env, sessions.StateWaiting, p.Message)
	case "Stop":
		return touch(p, env, sessions.StateDone, "")
	case "PostToolUse":
		err := touch(p, env, sessions.StateWorking, "")
		if isTaskEdit(p) {
			err = errors.Join(err, record(p, env))
		}
		return err
	}
	return nil
}

func touch(p payload, env Env, state, message string) error {
	s, _ := sessions.Get(env.SessionsDir, p.SessionID)
	s.ID, s.State, s.Message, s.UpdatedAt = p.SessionID, state, message, env.Now()
	if p.Cwd != "" {
		s.Cwd = p.Cwd
	}
	if pane := env.Getenv("TMUX_PANE"); pane != "" {
		s.Pane = pane
	}
	return sessions.Write(env.SessionsDir, s)
}

func isTaskEdit(p payload) bool {
	switch p.ToolName {
	case "Edit", "Write", "MultiEdit":
		return filepath.Base(p.ToolInput.FilePath) == "tasks.md"
	}
	return false
}

func record(p payload, env Env) error {
	dir := filepath.Dir(p.ToolInput.FilePath)
	specs := filepath.Dir(dir)
	if filepath.Base(specs) != "specs" {
		return nil
	}
	root, feature := filepath.Dir(specs), filepath.Base(dir)
	f, err := os.Open(p.ToolInput.FilePath)
	if err != nil {
		return nil
	}
	tasks := spec.ParseTasks(f)
	_ = f.Close()
	prev := events.LastStates(events.Read(root), feature)
	var errs []error
	for _, t := range tasks.Items {
		from := prev[t.ID]
		if from == "" {
			from = spec.Todo.String()
		}
		if t.ID[0] == 'L' || from == t.State.String() {
			continue
		}
		errs = append(errs, events.Append(root, events.Event{
			TS: env.Now(), Feature: feature, Task: t.ID, From: from, To: t.State.String(), Session: p.SessionID,
		}))
	}
	return errors.Join(errs...)
}

func isOurs(command string) bool {
	f := strings.Fields(command)
	return len(f) >= 2 && filepath.Base(f[0]) == "speckit-board" && f[1] == "hook"
}
