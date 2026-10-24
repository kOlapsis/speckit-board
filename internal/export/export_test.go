package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/spec"
)

const tasks = `## Phase 1
- [x] T001 Setup project
- [~] T002 Handle <b>markup</b> & "quotes" in internal/http/login.go
- [ ] T003 Wire routes
## Remediation (verify)
- [ ] T004 Reject empty passwords
`

func lane(feature, body string, light sessions.Light) Lane {
	p := spec.Project{Name: "demo", Feature: feature, HasTasks: body != ""}
	p.Tasks = spec.ParseTasks(strings.NewReader(body))
	p.Verification = spec.Verification{Verdict: "FAIL", OK: []string{"AC-1 Log in"}}
	return Lane{Project: p, Light: light}
}

func render(t *testing.T, lanes ...Lane) string {
	t.Helper()
	var buf bytes.Buffer
	if err := HTML(&buf, lanes); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestHTMLShowsTheBoard(t *testing.T) {
	out := render(t, lane("003-auth", tasks, sessions.Amber))
	for _, want := range []string{
		"<!doctype html>", `name="viewport"`, "demo", "003-auth", "1/4", "waiting", "verify FAIL",
		"Blocked", "Ready", "In progress", "Done", "Remediation", "Verified",
		"T001", "T002", "T003", "T004", "AC-1 Log in", "Setup project",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

func TestHTMLIsSelfContained(t *testing.T) {
	out := render(t, lane("003-auth", tasks, sessions.Gray))
	for _, banned := range []string{"<script", "http://", "https://", "<link", "@import", "src="} {
		if strings.Contains(out, banned) {
			t.Errorf("output contains %q", banned)
		}
	}
}

func TestHTMLEscapesTaskText(t *testing.T) {
	out := render(t, lane("003-auth", tasks, sessions.Gray))
	if strings.Contains(out, "<b>markup</b>") || !strings.Contains(out, "&lt;b&gt;markup&lt;/b&gt;") {
		t.Error("task text must be escaped")
	}
}

func TestHTMLWithoutTasks(t *testing.T) {
	out := render(t, lane("", "", sessions.Gray), lane("004-x", tasks, sessions.Green))
	if !strings.Contains(out, "no tasks.md") || !strings.Contains(out, "T003") {
		t.Error("an empty project must not hide the others")
	}
}

func TestHTMLNoLanes(t *testing.T) {
	if out := render(t); !strings.Contains(out, "<!doctype html>") {
		t.Error("empty export must still be a page")
	}
}
