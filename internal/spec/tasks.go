package spec

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type State int

const (
	Todo State = iota
	Doing
	Done
)

func (s State) String() string {
	return [...]string{"todo", "doing", "done"}[s]
}

type Task struct {
	ID          string
	Text        string
	State       State
	Parallel    bool
	Phase       string
	PhaseIdx    int
	Remediation bool
	Line        int
	Files       []string
}

type Tasks struct {
	Items []Task
}

var (
	taskRe    = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX~])\]\s+(.*\S)\s*$`)
	headingRe = regexp.MustCompile(`^(#{2,6})\s+(.*\S)\s*$`)
	remedRe   = regexp.MustCompile(`(?i)rem[eé]diation`)
	idRe      = regexp.MustCompile(`^(T\d+)\b\s*`)
	fileExtRe = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]{0,4}$`)
	spacesRe  = regexp.MustCompile(`\s+`)
)

func ParseTasks(r io.Reader) Tasks {
	var ts Tasks
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var phase string
	var idx int
	var remed bool
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if m := headingRe.FindStringSubmatch(text); m != nil {
			isRemed := remedRe.MatchString(m[2])
			switch {
			case len(m[1]) == 2:
				phase, remed = m[2], isRemed
				idx++
			case isRemed:
				phase, remed = m[2], true
				idx++
			}
			continue
		}
		m := taskRe.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		t := Task{State: stateOf(m[1]), Phase: phase, PhaseIdx: idx, Remediation: remed, Line: line}
		body := m[2]
		if id := idRe.FindStringSubmatch(body); id != nil {
			t.ID = id[1]
			body = body[len(id[0]):]
		} else {
			t.ID = "L" + strconv.Itoa(line)
		}
		t.Parallel = strings.Contains(body, "[P]")
		body = strings.ReplaceAll(body, "[P]", "")
		t.Text = strings.TrimSpace(spacesRe.ReplaceAllString(body, " "))
		t.Files = filesIn(t.Text)
		ts.Items = append(ts.Items, t)
	}
	return ts
}

func stateOf(marker string) State {
	switch marker {
	case "~":
		return Doing
	case " ":
		return Todo
	}
	return Done
}

func filesIn(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, tok := range strings.Fields(text) {
		tok = strings.Trim(tok, "`'\",;:()[]")
		tok = strings.TrimRight(tok, ".")
		if tok == "" || strings.Contains(tok, "://") || seen[tok] {
			continue
		}
		ext := fileExtRe.FindString(tok)
		if ext == "" || (len(ext) < 3 && !strings.Contains(tok, "/")) {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

type ColumnID string

const (
	Blocked     ColumnID = "blocked"
	Ready       ColumnID = "ready"
	InProgress  ColumnID = "doing"
	Finished    ColumnID = "done"
	Remediation ColumnID = "remediation"
	Verified    ColumnID = "verified"
)

func (id ColumnID) Title() string {
	switch id {
	case InProgress:
		return "In progress"
	case "":
		return ""
	}
	return strings.ToUpper(string(id[:1])) + string(id[1:])
}

type Card struct {
	ID    string
	Title string
	Task  *Task
}

type Column struct {
	ID    ColumnID
	Cards []Card
}

func Columns(ts Tasks, v Verification) []Column {
	cols := map[ColumnID]*Column{}
	order := []ColumnID{Blocked, Ready, InProgress, Finished, Remediation, Verified}
	for _, id := range order {
		cols[id] = &Column{ID: id}
	}
	open := map[int]bool{}
	for _, t := range ts.Items {
		if !t.Remediation && t.State != Done {
			open[t.PhaseIdx] = true
		}
	}
	for i := range ts.Items {
		t := &ts.Items[i]
		id := Ready
		switch {
		case t.State == Done:
			id = Finished
		case t.State == Doing:
			id = InProgress
		case t.Remediation:
			id = Remediation
		default:
			for p := range open {
				if p < t.PhaseIdx {
					id = Blocked
					break
				}
			}
		}
		cols[id].Cards = append(cols[id].Cards, Card{ID: t.ID, Title: t.Text, Task: t})
	}
	for _, c := range v.OK {
		cols[Verified].Cards = append(cols[Verified].Cards, Card{ID: c, Title: c})
	}
	out := make([]Column, len(order))
	for i, id := range order {
		out[i] = *cols[id]
	}
	return out
}
