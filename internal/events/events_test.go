package events

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadSkipsBadLines(t *testing.T) {
	root := t.TempDir()
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"ts":"2026-10-24T10:00:00Z","feature":"003-auth","task":"T005","from":"todo","to":"doing","session":"s1"}
not json
{"ts":"yesterday","feature":"003-auth","task":"T006","from":"todo","to":"doing"}

{"ts":"2026-10-24T11:30:00Z","feature":"003-auth","task":"T005","from":"doing","to":"done"}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	evs := Read(root)
	if len(evs) != 2 || evs[0].Task != "T005" || evs[0].Session != "s1" {
		t.Fatalf("%+v", evs)
	}
}

func TestReadMissingFile(t *testing.T) {
	if evs := Read(t.TempDir()); len(evs) != 0 {
		t.Errorf("%+v", evs)
	}
}

func TestStampsFor(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 10, 24, h, 0, 0, 0, time.UTC) }
	evs := []Event{
		{TS: at(9), Feature: "003-auth", Task: "T005", From: "todo", To: "doing"},
		{TS: at(10), Feature: "003-auth", Task: "T005", From: "doing", To: "todo"},
		{TS: at(11), Feature: "003-auth", Task: "T005", From: "todo", To: "doing"},
		{TS: at(12), Feature: "003-auth", Task: "T005", From: "doing", To: "done"},
		{TS: at(13), Feature: "004-other", Task: "T005", From: "todo", To: "doing"},
	}
	got := StampsFor(evs, "003-auth", "T005")
	if !got.Started.Equal(at(11)) || !got.Finished.Equal(at(12)) {
		t.Errorf("%+v", got)
	}
	if got := StampsFor(evs, "003-auth", "T999"); !got.Started.IsZero() || !got.Finished.IsZero() {
		t.Errorf("%+v", got)
	}
}
