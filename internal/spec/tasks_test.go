package spec

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func open(t *testing.T, name string) Tasks {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return ParseTasks(bytes.NewReader(b))
}

func byID(ts Tasks, id string) Task {
	for _, x := range ts.Items {
		if x.ID == id {
			return x
		}
	}
	return Task{}
}

func TestParseTasksEnglish(t *testing.T) {
	ts := open(t, "tasks_en.md")
	if len(ts.Items) != 12 {
		t.Fatalf("got %d tasks, want 12", len(ts.Items))
	}
	cases := []struct {
		id       string
		state    State
		parallel bool
		phase    string
		idx      int
		remed    bool
		files    []string
	}{
		{"T001", Done, false, "Phase 1: Setup", 1, false, nil},
		{"T002", Done, true, "Phase 1: Setup", 1, false, []string{"go.mod"}},
		{"T003", Done, true, "Phase 1: Setup", 1, false, []string{".golangci.yml"}},
		{"T004", Done, false, "Phase 2: Foundational", 2, false, []string{"db/migrations/003_users.sql"}},
		{"T005", Doing, true, "Phase 2: Foundational", 2, false, []string{"internal/auth/hash.go"}},
		{"T006", Todo, true, "Phase 2: Foundational", 2, false, []string{"internal/auth/store.go"}},
		{"T009", Todo, false, "Phase 3: User Story 1 - Log in (Priority: P1)", 3, false, []string{"cmd/server/main.go"}},
		{"T010", Todo, false, "Phase 4: Polish", 4, false, []string{"docs/README.md"}},
		{"T011", Todo, false, "Remediation (verify)", 5, true, []string{"internal/auth/hash.go"}},
		{"T012", Done, false, "Remediation (verify)", 5, true, nil},
	}
	for _, c := range cases {
		got := byID(ts, c.id)
		if got.State != c.state || got.Parallel != c.parallel || got.Phase != c.phase ||
			got.PhaseIdx != c.idx || got.Remediation != c.remed || !reflect.DeepEqual(got.Files, c.files) {
			t.Errorf("%s: got %+v", c.id, got)
		}
	}
	if got := byID(ts, "T008").Text; got != "[US1] Add login form in web/src/Login.vue" {
		t.Errorf("T008 text = %q", got)
	}
	if got := byID(ts, "T008").Line; got != 21 {
		t.Errorf("T008 line = %d", got)
	}
}

func TestParseTasksFrench(t *testing.T) {
	ts := open(t, "tasks_fr.md")
	if len(ts.Items) != 5 {
		t.Fatalf("got %d tasks, want 5", len(ts.Items))
	}
	if byID(ts, "T002").State != Done {
		t.Error("uppercase X must be done")
	}
	if byID(ts, "T003").State != Doing {
		t.Error("star bullet with [~] must be doing")
	}
	if !byID(ts, "T005").Remediation {
		t.Error("Remédiation heading must start a remediation section")
	}
}

func TestParseTasksTolerance(t *testing.T) {
	ts := open(t, "tasks_messy.md")
	var ids []string
	for _, x := range ts.Items {
		ids = append(ids, x.ID)
	}
	want := []string{"T001", "T002", "L8", "T005", "T006", "T007"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	if x := byID(ts, "T001"); x.PhaseIdx != 0 || x.Phase != "" {
		t.Errorf("task before any heading: %+v", x)
	}
	if byID(ts, "T002").State != Done {
		t.Error("indented task must be parsed")
	}
	if x := byID(ts, "T005"); !x.Parallel || x.Text != "twice" {
		t.Errorf("T005 = %+v", x)
	}
	if !byID(ts, "T007").Remediation {
		t.Error("deep remediation heading must be recognised")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestParseTasksNeverFails(t *testing.T) {
	if n := len(ParseTasks(strings.NewReader("")).Items); n != 0 {
		t.Errorf("empty input gave %d tasks", n)
	}
	if n := len(ParseTasks(failingReader{}).Items); n != 0 {
		t.Errorf("failing reader gave %d tasks", n)
	}
	if byID(open(t, "tasks_binary.md"), "T001").ID == "" {
		t.Error("task after binary garbage must survive")
	}
}

func ids(c Column) []string {
	var out []string
	for _, x := range c.Cards {
		out = append(out, x.ID)
	}
	return out
}

func column(cols []Column, id ColumnID) Column {
	for _, c := range cols {
		if c.ID == id {
			return c
		}
	}
	return Column{}
}

func TestColumnsEnglish(t *testing.T) {
	cols := Columns(open(t, "tasks_en.md"), Verification{OK: []string{"AC-1 Log in"}})
	order := []ColumnID{Blocked, Ready, InProgress, Finished, Remediation, Verified}
	for i, c := range cols {
		if c.ID != order[i] {
			t.Fatalf("column %d = %q, want %q", i, c.ID, order[i])
		}
	}
	want := map[ColumnID][]string{
		Blocked:     {"T007", "T008", "T009", "T010"},
		Ready:       {"T006"},
		InProgress:  {"T005"},
		Finished:    {"T001", "T002", "T003", "T004", "T012"},
		Remediation: {"T011"},
		Verified:    {"AC-1 Log in"},
	}
	for id, w := range want {
		if got := ids(column(cols, id)); !reflect.DeepEqual(got, w) {
			t.Errorf("%s = %v, want %v", id, got, w)
		}
	}
}

func TestColumnsFirstPhaseIsReady(t *testing.T) {
	cols := Columns(open(t, "tasks_fr.md"), Verification{})
	if got := ids(column(cols, Ready)); !reflect.DeepEqual(got, []string{"T004"}) {
		t.Errorf("ready = %v", got)
	}
	if got := ids(column(cols, Remediation)); !reflect.DeepEqual(got, []string{"T005"}) {
		t.Errorf("remediation = %v", got)
	}
}

func TestColumnsRemediationNeverBlocks(t *testing.T) {
	ts := ParseTasks(strings.NewReader("## Phase 1\n- [x] T001 a\n## Remediation\n- [ ] T002 fix\n## Phase 2\n- [ ] T003 b\n"))
	if got := ids(column(Columns(ts, Verification{}), Ready)); !reflect.DeepEqual(got, []string{"T003"}) {
		t.Errorf("ready = %v", got)
	}
}

func TestColumnsEmpty(t *testing.T) {
	if got := len(Columns(Tasks{}, Verification{})); got != 6 {
		t.Errorf("empty board must still have 6 columns, got %d", got)
	}
}
