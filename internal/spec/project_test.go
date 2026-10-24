package spec

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyFixture(t *testing.T, name, dst string) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	write(t, dst, string(b))
}

func TestResolveFeatureFromBranch(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".git/HEAD"), "ref: refs/heads/003-auth\n")
	write(t, filepath.Join(root, "specs/003-auth/tasks.md"), "")
	t.Setenv("SPECIFY_FEATURE", "")
	if got := ResolveFeature(root); got != "003-auth" {
		t.Errorf("got %q", got)
	}
}

func TestResolveFeatureFromWorktreeFile(t *testing.T) {
	root := t.TempDir()
	gitdir := filepath.Join(t.TempDir(), "wt")
	write(t, filepath.Join(gitdir, "HEAD"), "ref: refs/heads/003-auth\n")
	write(t, filepath.Join(root, ".git"), "gitdir: "+gitdir+"\n")
	write(t, filepath.Join(root, "specs/003-auth/tasks.md"), "")
	t.Setenv("SPECIFY_FEATURE", "")
	if got := ResolveFeature(root); got != "003-auth" {
		t.Errorf("got %q", got)
	}
}

func TestResolveFeatureFromFeatureJSON(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".git/HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(root, ".specify/feature.json"), `{"feature_directory":"specs/004-export"}`)
	write(t, filepath.Join(root, "specs/004-export/tasks.md"), "")
	t.Setenv("SPECIFY_FEATURE", "")
	if got := ResolveFeature(root); got != "004-export" {
		t.Errorf("got %q", got)
	}
}

func TestResolveFeatureFromEnv(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "specs/005-env/tasks.md"), "")
	t.Setenv("SPECIFY_FEATURE", "005-env")
	if got := ResolveFeature(root); got != "005-env" {
		t.Errorf("got %q", got)
	}
}

func TestResolveFeatureNewestTasks(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SPECIFY_FEATURE", "")
	old := filepath.Join(root, "specs/001-old/tasks.md")
	write(t, old, "")
	write(t, filepath.Join(root, "specs/002-new/tasks.md"), "")
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	if got := ResolveFeature(root); got != "002-new" {
		t.Errorf("got %q", got)
	}
}

func TestResolveFeatureNone(t *testing.T) {
	t.Setenv("SPECIFY_FEATURE", "")
	if got := ResolveFeature(t.TempDir()); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestLoadProject(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SPECIFY_FEATURE", "003-auth")
	dir := filepath.Join(root, "specs/003-auth")
	copyFixture(t, "tasks_en.md", filepath.Join(dir, "tasks.md"))
	copyFixture(t, "acceptance_en.md", filepath.Join(dir, "acceptance-tests.md"))
	copyFixture(t, "verification_pass.md", filepath.Join(dir, "verification.md"))
	p := Load(root)
	if p.Name != filepath.Base(root) || p.Feature != "003-auth" || !p.HasTasks {
		t.Fatalf("%+v", p)
	}
	if done, total := p.Counts(); done != 5 || total != 12 {
		t.Errorf("counts = %d/%d", done, total)
	}
	if cur := p.Current(); cur == nil || cur.ID != "T005" {
		t.Errorf("current = %+v", cur)
	}
	if p.Acceptance.Reference != "1a2b3c4" || p.Verification.Verdict != "PASS" {
		t.Errorf("%+v %+v", p.Acceptance, p.Verification)
	}
}

func TestLoadProjectWithoutSpecs(t *testing.T) {
	t.Setenv("SPECIFY_FEATURE", "")
	p := Load(t.TempDir())
	if p.HasTasks || p.Feature != "" || p.Current() != nil {
		t.Errorf("%+v", p)
	}
	if done, total := p.Counts(); done != 0 || total != 0 {
		t.Errorf("counts = %d/%d", done, total)
	}
	if len(p.Columns()) != 6 {
		t.Error("board must still have columns")
	}
}

func TestLoadProjectMissingRoot(t *testing.T) {
	t.Setenv("SPECIFY_FEATURE", "")
	p := Load("/does/not/exist")
	if p.HasTasks {
		t.Errorf("%+v", p)
	}
}
