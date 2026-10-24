package status

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kOlapsis/speckit-board/internal/spec"
)

func project(feature, tasks, verdict string) spec.Project {
	p := spec.Project{Name: "demo", Feature: feature, HasTasks: tasks != ""}
	p.Tasks = spec.ParseTasks(strings.NewReader(tasks))
	p.Verification = spec.Verification{Verdict: verdict}
	return p
}

const tasks = "## Phase 1\n- [x] T001 first\n- [x] T002 second\n- [~] T003 Add login handler\n- [ ] T004 last\n"

func TestLine(t *testing.T) {
	cases := []struct {
		name string
		p    spec.Project
		want string
	}{
		{"full", project("003-auth", tasks, "PASS"), "003-auth 2/4 · ▶ T003 Add login handler · verify PASS"},
		{"no current", project("003-auth", "- [x] T001 a\n- [ ] T002 b\n", ""), "003-auth 1/2"},
		{"no tasks file", project("003-auth", "", ""), "003-auth no tasks.md"},
		{"no feature", spec.Project{}, "speckit-board: no feature"},
		{"truncated", project("003-auth", "- [~] T001 "+strings.Repeat("x", 80)+"\n", "FAIL"),
			"003-auth 0/1 · ▶ T001 " + strings.Repeat("x", 39) + "… · verify FAIL"},
	}
	for _, c := range cases {
		if got := FromProject(c.p).Line(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestJSON(t *testing.T) {
	b, err := FromProject(project("003-auth", tasks, "PASS")).JSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	cur, _ := got["current"].(map[string]any)
	if got["feature"] != "003-auth" || got["done"] != 2.0 || got["total"] != 4.0 ||
		got["verdict"] != "PASS" || cur["id"] != "T003" {
		t.Errorf("%s", b)
	}
}

func TestJSONWithoutCurrent(t *testing.T) {
	b, _ := FromProject(spec.Project{}).JSON()
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["current"] != nil || got["feature"] != "" {
		t.Errorf("%s", b)
	}
}
