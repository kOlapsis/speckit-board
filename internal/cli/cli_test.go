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
