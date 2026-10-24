package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kOlapsis/speckit-board/internal/events"
	"github.com/kOlapsis/speckit-board/internal/sessions"
)

var now = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

type fixture struct {
	root, state string
	env         Env
}

func setup(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{root: t.TempDir(), state: t.TempDir()}
	f.env = Env{
		SessionsDir: filepath.Join(f.state, "sessions"),
		Now:         func() time.Time { return now },
		Getenv:      func(k string) string { return map[string]string{"TMUX_PANE": "%7"}[k] },
	}
	return f
}

func (f *fixture) payload(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "ROOT", f.root)
}

func (f *fixture) handle(t *testing.T, name string) error {
	t.Helper()
	return Handle(strings.NewReader(f.payload(t, name)), f.env)
}

func (f *fixture) tasks(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(f.root, "specs", "003-auth", "tasks.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func session(t *testing.T, f *fixture) sessions.Session {
	t.Helper()
	all := sessions.List(f.env.SessionsDir)
	if len(all) != 1 {
		t.Fatalf("sessions = %+v", all)
	}
	return all[0]
}

func TestNotificationMarksSessionWaiting(t *testing.T) {
	f := setup(t)
	if err := f.handle(t, "notification.json"); err != nil {
		t.Fatal(err)
	}
	s := session(t, f)
	if s.ID != "abc123" || s.State != sessions.StateWaiting || s.Cwd != "/work/app" ||
		s.Pane != "%7" || s.Message != "Claude needs your permission to use Bash" || !s.UpdatedAt.Equal(now) {
		t.Errorf("%+v", s)
	}
}

func TestStopMarksSessionDone(t *testing.T) {
	f := setup(t)
	_ = f.handle(t, "notification.json")
	if err := f.handle(t, "stop.json"); err != nil {
		t.Fatal(err)
	}
	if s := session(t, f); s.State != sessions.StateDone || s.Message != "" || s.Pane != "%7" {
		t.Errorf("%+v", s)
	}
}

func TestSubagentStopIsIgnored(t *testing.T) {
	f := setup(t)
	_ = f.handle(t, "notification.json")
	if err := f.handle(t, "subagent_stop.json"); err != nil {
		t.Fatal(err)
	}
	if s := session(t, f); s.State != sessions.StateWaiting {
		t.Errorf("%+v", s)
	}
}

func TestToolUseMarksSessionWorking(t *testing.T) {
	f := setup(t)
	_ = f.handle(t, "notification.json")
	if err := f.handle(t, "post_bash.json"); err != nil {
		t.Fatal(err)
	}
	if s := session(t, f); s.State != sessions.StateWorking {
		t.Errorf("%+v", s)
	}
}

func TestEditOfTasksRecordsTransitions(t *testing.T) {
	f := setup(t)
	f.tasks(t, "## Phase 1\n- [x] T001 a\n- [~] T002 Add login handler\n- [ ] T003 c\n")
	for _, name := range []string{"post_edit_tasks.json", "post_write_tasks.json", "post_multiedit_tasks.json"} {
		if err := f.handle(t, name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	evs := events.Read(f.root)
	if len(evs) != 2 {
		t.Fatalf("a repeated edit must not duplicate events: %+v", evs)
	}
	want := map[string][2]string{"T001": {"todo", "done"}, "T002": {"todo", "doing"}}
	for _, e := range evs {
		w := want[e.Task]
		if e.From != w[0] || e.To != w[1] || e.Feature != "003-auth" || e.Session != "abc123" || !e.TS.Equal(now) {
			t.Errorf("%+v", e)
		}
	}
}

func TestLaterTransitionsBuildOnTheJournal(t *testing.T) {
	f := setup(t)
	f.tasks(t, "- [~] T002 Add login handler\n")
	_ = f.handle(t, "post_edit_tasks.json")
	f.tasks(t, "- [x] T002 Add login handler\n")
	_ = f.handle(t, "post_edit_tasks.json")
	evs := events.Read(f.root)
	if len(evs) != 2 || evs[1].From != "doing" || evs[1].To != "done" {
		t.Errorf("%+v", evs)
	}
}

func TestOtherFilesDoNotTouchTheJournal(t *testing.T) {
	f := setup(t)
	f.tasks(t, "- [x] T001 a\n")
	if err := f.handle(t, "post_edit_other.json"); err != nil {
		t.Fatal(err)
	}
	if evs := events.Read(f.root); len(evs) != 0 {
		t.Errorf("%+v", evs)
	}
}

func TestMissingTasksFileIsNotAnError(t *testing.T) {
	f := setup(t)
	if err := f.handle(t, "post_edit_tasks.json"); err != nil {
		t.Errorf("%v", err)
	}
}

func TestBadInput(t *testing.T) {
	f := setup(t)
	for _, in := range []string{"", "not json", `{"hook_event_name":"Notification"}`, `{"session_id":"../evil","hook_event_name":"Stop"}`} {
		if err := Handle(strings.NewReader(in), f.env); err == nil {
			t.Errorf("%q must fail", in)
		}
	}
	if _, err := os.Stat(f.env.SessionsDir); err == nil {
		entries, _ := os.ReadDir(f.env.SessionsDir)
		if len(entries) != 0 {
			t.Errorf("bad input wrote files: %v", entries)
		}
	}
}

func TestUnknownEventIsIgnored(t *testing.T) {
	f := setup(t)
	if err := Handle(strings.NewReader(`{"session_id":"abc123","hook_event_name":"SessionStart","cwd":"/x"}`), f.env); err != nil {
		t.Errorf("%v", err)
	}
	if all := sessions.List(f.env.SessionsDir); len(all) != 0 {
		t.Errorf("%+v", all)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func commands(t *testing.T, m map[string]any, event string) []string {
	t.Helper()
	var out []string
	hooks, _ := m["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	for _, g := range groups {
		for _, h := range g.(map[string]any)["hooks"].([]any) {
			out = append(out, h.(map[string]any)["command"].(string))
		}
	}
	return out
}

func TestInstallCreatesSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "settings.json")
	res, err := Install(path, false)
	if err != nil || !res.Changed {
		t.Fatalf("%+v %v", res, err)
	}
	m := readJSON(t, path)
	for _, ev := range []string{"Notification", "Stop", "PostToolUse"} {
		if got := commands(t, m, ev); len(got) != 1 || got[0] != "speckit-board hook" {
			t.Errorf("%s: %v", ev, got)
		}
	}
	groups := m["hooks"].(map[string]any)["PostToolUse"].([]any)
	if groups[0].(map[string]any)["matcher"] != "Edit|Write|MultiEdit" {
		t.Errorf("%v", groups)
	}
}

func TestInstallKeepsExistingSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	existing := `{
  "model": "opus",
  "permissions": {"allow": ["Bash(git status)"]},
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "notify-send done"}]}],
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard.sh"}]}]
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path, false); err != nil {
		t.Fatal(err)
	}
	m := readJSON(t, path)
	if m["model"] != "opus" || len(commands(t, m, "PreToolUse")) != 1 {
		t.Errorf("%v", m)
	}
	if got := commands(t, m, "Stop"); len(got) != 2 || got[0] != "notify-send done" || got[1] != "speckit-board hook" {
		t.Errorf("Stop = %v", got)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if _, err := Install(path, false); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	res, err := Install(path, false)
	if err != nil || res.Changed {
		t.Fatalf("%+v %v", res, err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("second install rewrote the file")
	}
}

func TestInstallRecognisesAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	body := `{"hooks":{"Notification":[{"hooks":[{"type":"command","command":"/usr/local/bin/speckit-board hook"}]}]}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path, false); err != nil {
		t.Fatal(err)
	}
	if got := commands(t, readJSON(t, path), "Notification"); len(got) != 1 {
		t.Errorf("%v", got)
	}
}

func TestInstallDryRunWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	res, err := Install(path, true)
	if err != nil || !res.Changed || !strings.Contains(string(res.Content), "speckit-board hook") {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("dry run wrote the file")
	}
}

func TestInstallRefusesBrokenSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path, false); err == nil {
		t.Error("must refuse to overwrite a file it cannot parse")
	}
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Error("file was modified")
	}
}
