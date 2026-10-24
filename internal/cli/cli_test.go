package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := New(Info{Version: "1.2.3", Commit: "abc1234", Date: "2026-10-24"})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("SPECIFY_FEATURE", "003-auth")
	dir := filepath.Join(root, "specs", "003-auth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tasks := "## Phase 1\n- [x] T001 first\n- [~] T002 Add login handler\n- [ ] T003 last\n"
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(tasks), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestStatusLine(t *testing.T) {
	out, err := run(t, "status", project(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := "003-auth 1/3 · ▶ T002 Add login handler\n"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestStatusJSON(t *testing.T) {
	out, err := run(t, "status", "--json", project(t))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["done"] != 1.0 {
		t.Errorf("%q %v", out, err)
	}
}

func TestStatusOutsideAnyProject(t *testing.T) {
	t.Setenv("SPECIFY_FEATURE", "")
	out, err := run(t, "status", t.TempDir())
	if err != nil || !strings.Contains(out, "no feature") {
		t.Errorf("%q %v", out, err)
	}
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil || out != "speckit-board 1.2.3 (commit abc1234, built 2026-10-24)\n" {
		t.Errorf("%q %v", out, err)
	}
}

func TestBoardNeedsATerminal(t *testing.T) {
	if _, err := run(t, project(t)); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Errorf("got %v", err)
	}
}

func runIn(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := New(Info{})
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestHookWritesSessionAndNeverFails(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	out, err := runIn(t, `{"session_id":"abc","cwd":"/w","hook_event_name":"Stop"}`, "hook")
	if err != nil || out != "" {
		t.Fatalf("%q %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(state, "speckit-board", "sessions", "abc.json")); err != nil {
		t.Error(err)
	}
	for _, in := range []string{"", "garbage", `{"hook_event_name":"Stop"}`} {
		if out, err := runIn(t, in, "hook"); err != nil || out != "" {
			t.Errorf("%q must exit 0 silently, got %q %v", in, out, err)
		}
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestInstallHooksProjectIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	out, err := run(t, "install-hooks")
	if err != nil || !strings.Contains(out, "Notification") {
		t.Fatalf("%q %v", out, err)
	}
	path := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "install-hooks", "--project")
	if err != nil || !strings.Contains(out, "already installed") {
		t.Errorf("%q %v", out, err)
	}
}

func TestInstallHooksUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := run(t, "install-hooks", "--user"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
		t.Error(err)
	}
}

func TestInstallHooksDryRun(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	out, err := run(t, "install-hooks", "--dry-run")
	if err != nil || !strings.Contains(out, "speckit-board hook") || !strings.Contains(out, "dry run") {
		t.Fatalf("%q %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); err == nil {
		t.Error("dry run wrote files")
	}
}

func TestInstallHooksScopesExclude(t *testing.T) {
	if _, err := run(t, "install-hooks", "--user", "--project"); err == nil {
		t.Error("--user and --project must be mutually exclusive")
	}
}

func TestInstallHooksReportsBrokenSettings(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.MkdirAll(".claude", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".claude/settings.json", []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "install-hooks"); err == nil {
		t.Error("broken settings must be reported")
	}
}
