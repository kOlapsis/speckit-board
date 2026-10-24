package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func expect(t *testing.T, ch <-chan struct{}, within time.Duration) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(within):
		t.Fatalf("no signal within %v", within)
	}
}

func drain(ch <-chan struct{}) {
	for {
		select {
		case <-ch:
		case <-time.After(300 * time.Millisecond):
			return
		}
	}
}

func TestSignalsOnFileChange(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := Start(ctx, []string{dir}, time.Hour)
	drain(ch)
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] T001 a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect(t, ch, time.Second)
}

func TestWatchesNewSubdirectories(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := Start(ctx, []string{dir}, time.Hour)
	drain(ch)
	sub := filepath.Join(dir, "003-auth")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	expect(t, ch, time.Second)
	drain(ch)
	if err := os.WriteFile(filepath.Join(sub, "tasks.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect(t, ch, time.Second)
}

func TestFallsBackToPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := Start(ctx, []string{filepath.Join(t.TempDir(), "missing")}, 50*time.Millisecond)
	expect(t, ch, time.Second)
	expect(t, ch, time.Second)
}

func TestStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := Start(ctx, []string{t.TempDir()}, time.Hour)
	cancel()
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("channel not closed after cancel")
		}
	}
}

func TestDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "specs", "003-auth"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := Dirs(root, "/state/sessions")
	want := map[string]bool{
		filepath.Join(root, "specs"): true, filepath.Join(root, "specs", "003-auth"): true,
		filepath.Join(root, ".specify"): true, "/state/sessions": true,
	}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected %s", d)
		}
	}
}
