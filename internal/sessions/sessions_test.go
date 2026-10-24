package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func put(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/state")
	if got := Dir(); got != "/tmp/state/speckit-board/sessions" {
		t.Errorf("got %q", got)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/x")
	if got := Dir(); got != "/home/x/.local/state/speckit-board/sessions" {
		t.Errorf("got %q", got)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "a.json", `{"session_id":"a","cwd":"/work/app","state":"waiting","message":"needs permission","pane":"%7","updated_at":"2026-10-24T10:00:00Z"}`)
	put(t, dir, "b.json", `{"session_id":"b","cwd":"/work/app","state":"done","updated_at":"2026-10-24T09:00:00Z"}`)
	put(t, dir, "broken.json", `{{{`)
	put(t, dir, "notes.txt", `ignored`)
	all := List(dir)
	if len(all) != 2 || all[0].ID != "a" || all[0].Pane != "%7" || all[1].ID != "b" {
		t.Fatalf("%+v", all)
	}
	if got := List(filepath.Join(dir, "missing")); len(got) != 0 {
		t.Errorf("%+v", got)
	}
}

func at(h int) time.Time { return time.Date(2026, 10, 24, h, 0, 0, 0, time.UTC) }

func TestSummarize(t *testing.T) {
	all := []Session{
		{ID: "old-wait", Cwd: "/work/app", State: StateWaiting, UpdatedAt: at(8)},
		{ID: "new-wait", Cwd: "/work/app/web", State: StateWaiting, UpdatedAt: at(9)},
		{ID: "done", Cwd: "/work/app", State: StateDone, UpdatedAt: at(10)},
		{ID: "elsewhere", Cwd: "/work/other", State: StateWaiting, UpdatedAt: at(11)},
		{ID: "sibling", Cwd: "/work/app-two", State: StateWaiting, UpdatedAt: at(12)},
	}
	light, s := Summarize(all, "/work/app")
	if light != Amber || s == nil || s.ID != "new-wait" {
		t.Errorf("waiting: %v %+v", light, s)
	}
	light, s = Summarize(all[2:3], "/work/app")
	if light != Green || s.ID != "done" {
		t.Errorf("done: %v %+v", light, s)
	}
	light, s = Summarize([]Session{{ID: "w", Cwd: "/work/app", State: StateWorking}}, "/work/app")
	if light != Gray || s == nil || s.ID != "w" {
		t.Errorf("working: %v %+v", light, s)
	}
	if light, s = Summarize(all[3:], "/work/app"); light != Gray || s != nil {
		t.Errorf("none: %v %+v", light, s)
	}
}

func TestFirstWaiting(t *testing.T) {
	all := []Session{
		{ID: "a", State: StateDone, UpdatedAt: at(10)},
		{ID: "b", State: StateWaiting, UpdatedAt: at(8)},
		{ID: "c", State: StateWaiting, UpdatedAt: at(9)},
	}
	if s := FirstWaiting(all); s == nil || s.ID != "c" {
		t.Errorf("%+v", s)
	}
	if s := FirstWaiting(all[:1]); s != nil {
		t.Errorf("%+v", s)
	}
}
