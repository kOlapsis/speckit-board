package status

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kOlapsis/speckit-board/internal/spec"
)

const maxCurrent = 40

type Current struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Info struct {
	Project     string   `json:"project"`
	Feature     string   `json:"feature"`
	HasTasks    bool     `json:"has_tasks"`
	Done        int      `json:"done"`
	Total       int      `json:"total"`
	Remediation int      `json:"remediation"`
	Current     *Current `json:"current"`
	Verdict     string   `json:"verdict"`
}

func FromProject(p spec.Project) Info {
	i := Info{Project: p.Name, Feature: p.Feature, HasTasks: p.HasTasks, Verdict: p.Verification.Verdict}
	i.Done, i.Total = p.Counts()
	for _, t := range p.Tasks.Items {
		if t.Remediation && t.State != spec.Done {
			i.Remediation++
		}
	}
	if t := p.Current(); t != nil {
		i.Current = &Current{ID: t.ID, Text: t.Text}
	}
	return i
}

func (i Info) Line() string {
	if i.Feature == "" {
		return "speckit-board: no feature"
	}
	if !i.HasTasks {
		return i.Feature + " no tasks.md"
	}
	parts := []string{fmt.Sprintf("%s %d/%d", i.Feature, i.Done, i.Total)}
	if i.Current != nil {
		parts = append(parts, "▶ "+i.Current.ID+" "+truncate(i.Current.Text, maxCurrent))
	}
	if i.Verdict != "" {
		parts = append(parts, "verify "+i.Verdict)
	}
	return strings.Join(parts, " · ")
}

func (i Info) JSON() ([]byte, error) {
	return json.Marshal(i)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
