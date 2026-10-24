package spec

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	Root         string
	Name         string
	Feature      string
	HasTasks     bool
	Tasks        Tasks
	Acceptance   Acceptance
	Verification Verification
}

func Load(root string) Project {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	p := Project{Root: root, Name: filepath.Base(root), Feature: ResolveFeature(root)}
	if p.Feature == "" {
		return p
	}
	dir := filepath.Join(root, "specs", p.Feature)
	if b, err := os.ReadFile(filepath.Join(dir, "tasks.md")); err == nil {
		p.Tasks, p.HasTasks = ParseTasks(bytes.NewReader(b)), true
	}
	if b, err := os.ReadFile(filepath.Join(dir, "acceptance-tests.md")); err == nil {
		p.Acceptance = ParseAcceptance(bytes.NewReader(b))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "verification.md")); err == nil {
		p.Verification = ParseVerification(bytes.NewReader(b))
	}
	return p
}

func (p Project) Columns() []Column {
	return Columns(p.Tasks, p.Verification)
}

func (p Project) Counts() (done, total int) {
	for _, t := range p.Tasks.Items {
		if t.State == Done {
			done++
		}
	}
	return done, len(p.Tasks.Items)
}

func (p Project) Current() *Task {
	for i := range p.Tasks.Items {
		if p.Tasks.Items[i].State == Doing {
			return &p.Tasks.Items[i]
		}
	}
	return nil
}

func ResolveFeature(root string) string {
	has := func(name string) bool {
		if name == "" || strings.ContainsAny(name, `/\`) {
			return false
		}
		st, err := os.Stat(filepath.Join(root, "specs", name))
		return err == nil && st.IsDir()
	}
	if f := os.Getenv("SPECIFY_FEATURE"); has(f) {
		return f
	}
	if b := branch(root); has(b) {
		return b
	}
	if f := featureJSON(root); has(f) {
		return f
	}
	return newestFeature(root)
}

func branch(root string) string {
	gitdir := filepath.Join(root, ".git")
	if st, err := os.Stat(gitdir); err == nil && !st.IsDir() {
		b, err := os.ReadFile(gitdir)
		if err != nil {
			return ""
		}
		gitdir = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(b)), "gitdir:"))
	}
	b, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "ref: refs/heads/")
}

func featureJSON(root string) string {
	b, err := os.ReadFile(filepath.Join(root, ".specify", "feature.json"))
	if err != nil {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	for _, v := range m {
		if s, ok := v.(string); ok && s != "" {
			if base := filepath.Base(filepath.Clean(s)); base != "." {
				if st, err := os.Stat(filepath.Join(root, "specs", base)); err == nil && st.IsDir() {
					return base
				}
			}
		}
	}
	return ""
}

func newestFeature(root string) string {
	matches, _ := filepath.Glob(filepath.Join(root, "specs", "*", "tasks.md"))
	var best string
	var bestTime int64
	for _, m := range matches {
		st, err := os.Stat(m)
		if err != nil {
			continue
		}
		if t := st.ModTime().UnixNano(); t > bestTime {
			best, bestTime = filepath.Base(filepath.Dir(m)), t
		}
	}
	return best
}
