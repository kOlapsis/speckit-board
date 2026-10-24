package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kOlapsis/speckit-board/internal/spec"
)

type fakeTmux struct {
	inside  bool
	pane    string
	findErr error
	sent    []string
	jumped  []string
}

func (f *fakeTmux) Inside() bool { return f.inside }
func (f *fakeTmux) FindPane(_, _ string) (string, error) {
	return f.pane, f.findErr
}
func (f *fakeTmux) Send(pane, text string) error {
	f.sent = append(f.sent, pane+" "+text)
	return nil
}
func (f *fakeTmux) Jump(pane string) error {
	f.jumped = append(f.jumped, pane)
	return nil
}

const tasksMD = `## Phase 1
- [x] T001 Setup project
- [~] T002 Add login handler in internal/http/login.go
- [ ] T003 Wire routes
## Remediation (verify)
- [ ] T004 Reject empty passwords
`

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

type env struct {
	root, sessions string
	tmux           *fakeTmux
	m              *Model
}

func setup(t *testing.T) *env {
	t.Helper()
	time.Local = time.UTC
	t.Setenv("SPECIFY_FEATURE", "003-auth")
	e := &env{root: t.TempDir(), sessions: t.TempDir(), tmux: &fakeTmux{inside: true, pane: "%9"}}
	write(t, filepath.Join(e.root, "specs/003-auth/tasks.md"), tasksMD)
	e.m = New(Options{Roots: []string{e.root}, SessionsDir: e.sessions, Tmux: e.tmux})
	e.size(120, 40)
	return e
}

func (e *env) size(w, h int) {
	e.m.Update(tea.WindowSizeMsg{Width: w, Height: h})
}

func (e *env) key(k string) tea.Cmd {
	var msg tea.KeyMsg
	switch k {
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	_, cmd := e.m.Update(msg)
	return cmd
}

func (e *env) goTo(t *testing.T, col spec.ColumnID) {
	t.Helper()
	for i := 0; i < 8; i++ {
		if e.m.Selection().Column == col {
			return
		}
		e.key("right")
	}
	t.Fatalf("column %s unreachable, at %s", col, e.m.Selection().Column)
}

func TestViewShowsBoard(t *testing.T) {
	e := setup(t)
	v := e.m.View()
	for _, want := range []string{"Blocked", "Ready", "In progress", "Done", "Remediation", "Verified", "T001", "T002", "T003", "T004", "003-auth", "1/4"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q", want)
		}
	}
}

func TestSelectionStartsOnFirstCardAndMoves(t *testing.T) {
	e := setup(t)
	if s := e.m.Selection(); !s.OK || s.Card.ID != "T003" || s.Column != spec.Ready {
		t.Fatalf("%+v", s)
	}
	e.key("l")
	if s := e.m.Selection(); s.Card.ID != "T002" {
		t.Errorf("after l: %+v", s)
	}
	e.key("h")
	e.key("left")
	if s := e.m.Selection(); s.Column != spec.Blocked || s.OK {
		t.Errorf("empty column must have no card: %+v", s)
	}
	e.key("down")
	e.goTo(t, spec.Finished)
	e.key("down")
	if s := e.m.Selection(); s.Card.ID != "T001" {
		t.Errorf("down must clamp: %+v", s)
	}
}

func TestDetailPanel(t *testing.T) {
	e := setup(t)
	write(t, filepath.Join(e.root, ".specify/events.jsonl"),
		`{"ts":"2026-10-24T10:05:00Z","feature":"003-auth","task":"T002","from":"todo","to":"doing"}`+"\n")
	e.m.Update(ReloadMsg{})
	e.key("l")
	v := e.m.View()
	for _, want := range []string{"Add login handler in internal/http/login.go", "Phase 1", "internal/http/login.go", "2026-10-24 10:05"} {
		if !strings.Contains(v, want) {
			t.Errorf("detail lacks %q\n%s", want, v)
		}
	}
}

func TestEnterOnRemediationSendsImplement(t *testing.T) {
	e := setup(t)
	e.goTo(t, spec.Remediation)
	e.key("enter")
	if len(e.tmux.sent) != 1 || e.tmux.sent[0] != "%9 /speckit-implement T004" {
		t.Errorf("%v", e.tmux.sent)
	}
}

func TestEnterElsewhereDoesNothing(t *testing.T) {
	e := setup(t)
	e.key("enter")
	if len(e.tmux.sent) != 0 || !strings.Contains(e.m.View(), "remediation") {
		t.Errorf("%v", e.tmux.sent)
	}
}

func TestVerifyKey(t *testing.T) {
	e := setup(t)
	e.key("v")
	if len(e.tmux.sent) != 1 || e.tmux.sent[0] != "%9 /speckit-verify" {
		t.Errorf("%v", e.tmux.sent)
	}
}

func TestActionsOutsideTmux(t *testing.T) {
	e := setup(t)
	e.tmux.inside = false
	e.key("v")
	e.key("j")
	e.goTo(t, spec.Remediation)
	e.key("enter")
	if len(e.tmux.sent)+len(e.tmux.jumped) != 0 || !strings.Contains(e.m.View(), "tmux") {
		t.Errorf("%v %v", e.tmux.sent, e.tmux.jumped)
	}
}

func TestVerifyWithoutPane(t *testing.T) {
	e := setup(t)
	e.tmux.findErr = errors.New("no Claude Code pane found for this project")
	e.key("v")
	if len(e.tmux.sent) != 0 || !strings.Contains(e.m.View(), "no Claude Code pane") {
		t.Errorf("%v", e.tmux.sent)
	}
}

func TestJumpToWaitingSession(t *testing.T) {
	e := setup(t)
	e.key("j")
	if len(e.tmux.jumped) != 0 || !strings.Contains(e.m.View(), "no session") {
		t.Fatalf("%v", e.tmux.jumped)
	}
	write(t, filepath.Join(e.sessions, "s1.json"),
		`{"session_id":"s1","cwd":"`+e.root+`","state":"waiting","message":"Claude needs your permission to use Bash","pane":"%5","updated_at":"2026-10-24T10:00:00Z"}`)
	e.m.Update(ReloadMsg{})
	if v := e.m.View(); !strings.Contains(v, "waiting") {
		t.Errorf("lane must show waiting:\n%s", v)
	}
	e.key("j")
	if len(e.tmux.jumped) != 1 || e.tmux.jumped[0] != "%5" {
		t.Errorf("%v", e.tmux.jumped)
	}
}

func TestSessionLights(t *testing.T) {
	e := setup(t)
	if !strings.Contains(e.m.View(), "unknown") {
		t.Error("no session must read unknown")
	}
	write(t, filepath.Join(e.sessions, "s1.json"),
		`{"session_id":"s1","cwd":"`+e.root+`","state":"done","updated_at":"2026-10-24T10:00:00Z"}`)
	e.m.Update(ReloadMsg{})
	if !strings.Contains(e.m.View(), "done ●") && !strings.Contains(e.m.View(), "● done") {
		t.Error("done session must be shown")
	}
}

func TestHelpAndQuit(t *testing.T) {
	e := setup(t)
	e.key("?")
	if !strings.Contains(e.m.View(), "tmux send-keys") && !strings.Contains(e.m.View(), "Jump") {
		t.Error("help not shown")
	}
	e.key("?")
	if strings.Contains(e.m.View(), "Jump to") {
		t.Error("help not hidden")
	}
	cmd := e.key("q")
	if cmd == nil {
		t.Fatal("q must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("got %T", cmd())
	}
}

func TestReloadPicksUpChanges(t *testing.T) {
	e := setup(t)
	write(t, filepath.Join(e.root, "specs/003-auth/tasks.md"), strings.Replace(tasksMD, "- [ ] T003", "- [x] T003", 1))
	e.m.Update(ReloadMsg{})
	if !strings.Contains(e.m.View(), "2/4") {
		t.Errorf("reload not applied:\n%s", e.m.View())
	}
}

func TestSeveralProjectsAndTab(t *testing.T) {
	e := setup(t)
	other := t.TempDir()
	e.m = New(Options{Roots: []string{e.root, other}, SessionsDir: e.sessions, Tmux: e.tmux})
	e.size(120, 40)
	if s := e.m.Selection(); s.Lane != 0 {
		t.Fatalf("%+v", s)
	}
	e.key("tab")
	if s := e.m.Selection(); s.Lane != 1 || s.OK {
		t.Errorf("%+v", s)
	}
	v := e.m.View()
	if !strings.Contains(v, filepath.Base(other)) || !strings.Contains(v, "no tasks.md") {
		t.Errorf("empty project must still render:\n%s", v)
	}
	e.key("tab")
	if e.m.Selection().Lane != 0 {
		t.Error("tab must wrap")
	}
}

func TestSmallTerminalDoesNotBreak(t *testing.T) {
	e := setup(t)
	for _, size := range [][2]int{{40, 10}, {20, 5}, {200, 60}, {1, 1}} {
		e.size(size[0], size[1])
		v := e.m.View()
		if v == "" {
			t.Errorf("%v: empty view", size)
		}
		for _, line := range strings.Split(v, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Errorf("%v: line too wide (%d): %q", size, lipgloss.Width(line), line)
				break
			}
		}
	}
}

func TestBeforeFirstResize(t *testing.T) {
	m := New(Options{Roots: []string{t.TempDir()}, SessionsDir: t.TempDir(), Tmux: &fakeTmux{}})
	if m.View() == "" {
		t.Error("view before size must not be empty")
	}
}

func TestColumnsStayAligned(t *testing.T) {
	e := setup(t)
	write(t, filepath.Join(e.root, "specs/003-auth/tasks.md"),
		"## Phase 1\n- [x] T001 first\n- [x] T002 second\n- [x] T003 third\n- [ ] T004 fourth\n")
	e.m.Update(ReloadMsg{})
	lines := strings.Split(e.m.View(), "\n")
	for i, line := range lines {
		if !strings.Contains(line, "Blocked (") {
			continue
		}
		want := lipgloss.Width(line[:strings.Index(line, "Done (")])
		for off, id := range []string{"T001", "T002", "T003"} {
			row := lines[i+1+off]
			if got := lipgloss.Width(row[:strings.Index(row, id)]); got != want {
				t.Errorf("%s starts at %d, Done column at %d", id, got, want)
			}
		}
		return
	}
	t.Fatal("no header line")
}
