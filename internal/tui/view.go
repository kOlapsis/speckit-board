package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/kOlapsis/speckit-board/internal/events"
	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/spec"
)

const (
	detailHeight = 7
	minLaneRows  = 4
	timeLayout   = "2006-01-02 15:04"
)

var (
	titles = map[spec.ColumnID]string{
		spec.Blocked: "Blocked", spec.Ready: "Ready", spec.InProgress: "In progress",
		spec.Finished: "Done", spec.Remediation: "Remediation", spec.Verified: "Verified",
	}
	colors = map[spec.ColumnID]lipgloss.Color{
		spec.Blocked: "245", spec.Ready: "75", spec.InProgress: "214",
		spec.Finished: "42", spec.Remediation: "203", spec.Verified: "135",
	}
	lights = map[sessions.Light]struct {
		label string
		color lipgloss.Color
	}{
		sessions.Amber: {"waiting", "214"},
		sessions.Green: {"done", "42"},
		sessions.Gray:  {"unknown", "245"},
	}
	dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selected = lipgloss.NewStyle().Reverse(true)
)

const helpText = `Arrows or h/l   move between columns, up/down between cards
tab             next project
j               Jump to the session that is waiting (tmux)
v               send /speckit-verify to the project's Claude Code pane
enter           on a remediation task, send /speckit-implement <id>
?               toggle this help
q               quit`

func cell(text string, w int, st lipgloss.Style) string {
	return st.Width(w).MaxWidth(w).Render(ansi.Truncate(text, w, "…"))
}

func (m *Model) View() string {
	if m.w == 0 {
		m.w, m.h = 100, 30
	}
	if m.w < 20 || m.h < 6 {
		return clip("terminal too small", m.w, m.h)
	}
	var body []string
	reserved := detailHeight + 1
	if m.showHelp && m.message == "" {
		reserved += strings.Count(helpText, "\n") + 1
	}
	laneRows := max((m.h-reserved)/len(m.lanes), minLaneRows)
	for i := range m.lanes {
		body = append(body, m.laneView(i, laneRows)...)
	}
	body = append(body, m.detailView()...)
	foot := m.shortHelp.ShortHelpView(shortKeys.ShortHelp())
	switch {
	case m.message != "":
		foot = m.message
	case m.showHelp:
		body = append(body, strings.Split(helpText, "\n")...)
	}
	if len(body) > m.h-1 {
		body = body[:m.h-1]
	}
	for len(body) < m.h-1 {
		body = append(body, "")
	}
	return clip(strings.Join(append(body, foot), "\n"), m.w, m.h)
}

func clip(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, w, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) laneView(idx, rows int) []string {
	l := m.lanes[idx]
	done, total := l.project.Counts()
	li := lights[l.light]
	dot := lipgloss.NewStyle().Foreground(li.color).Render("● " + li.label)
	feature := l.project.Feature
	if feature == "" {
		feature = "no feature"
	}
	head := fmt.Sprintf("%s  %s · %s · %d/%d", dot, lipgloss.NewStyle().Bold(true).Render(l.project.Name), feature, done, total)
	out := []string{head}
	if !l.project.HasTasks {
		return append(out, dim.Render("  no tasks.md found"))
	}
	colW := max((m.w-(len(l.cols)-1))/len(l.cols), 4)
	cardRows := rows - 2
	headers := make([]string, len(l.cols))
	cols := make([][]string, len(l.cols))
	for ci, c := range l.cols {
		st := lipgloss.NewStyle().Bold(true).Foreground(colors[c.ID])
		headers[ci] = cell(fmt.Sprintf("%s (%d)", titles[c.ID], len(c.Cards)), colW, st)
		sel := -1
		if idx == m.laneIdx && ci == m.colIdx {
			sel = m.rowIdx
		}
		cols[ci] = columnCards(c, colW, cardRows, sel)
	}
	out = append(out, strings.Join(headers, " "))
	for r := 0; r < cardRows; r++ {
		parts := make([]string, len(cols))
		for ci := range cols {
			parts[ci] = cols[ci][r]
		}
		out = append(out, strings.Join(parts, " "))
	}
	return out
}

func columnCards(c spec.Column, w, rows, sel int) []string {
	out := make([]string, rows)
	n := len(c.Cards)
	visible, start := n, 0
	if n > rows {
		visible = rows - 1
		start = min(max(sel-visible+1, 0), n-visible)
	}
	for r := range out {
		out[r] = cell("", w, lipgloss.NewStyle())
		if r >= visible {
			break
		}
		card := c.Cards[start+r]
		st := lipgloss.NewStyle()
		if start+r == sel {
			st = selected
		}
		out[r] = cell(card.ID+" "+cardTitle(card), w, st)
	}
	if n > rows {
		out[rows-1] = cell(fmt.Sprintf("+%d more", n-visible), w, dim)
	}
	return out
}

func cardTitle(c spec.Card) string {
	if c.Task == nil {
		return ""
	}
	return c.Task.Text
}

func (m *Model) detailView() []string {
	rule := dim.Render(strings.Repeat("─", m.w))
	out := []string{rule}
	s := m.Selection()
	l := m.lanes[m.laneIdx]
	var lines []string
	switch {
	case !s.OK:
		lines = []string{dim.Render("No card selected")}
	case s.Card.Task == nil:
		lines = []string{lipgloss.NewStyle().Bold(true).Render(s.Card.Title), "Criterion rated OK in the last verification"}
	default:
		lines = taskDetail(*s.Card.Task, l, m.w)
	}
	if l.session != nil && l.session.Message != "" {
		lines = append(lines, dim.Render("Claude: "+l.session.Message))
	}
	return append(out, pad(lines, detailHeight-1)...)
}

func taskDetail(t spec.Task, l lane, w int) []string {
	flags := t.State.String()
	if t.Parallel {
		flags += ", parallel"
	}
	if t.Remediation {
		flags += ", remediation"
	}
	phase := t.Phase
	if phase == "" {
		phase = "(no phase)"
	}
	lines := []string{lipgloss.NewStyle().Bold(true).Render(t.ID) + "  " + flags + "  " + phase}
	lines = append(lines, strings.Split(lipgloss.NewStyle().Width(w).Render(t.Text), "\n")...)
	if len(t.Files) > 0 {
		lines = append(lines, "Files: "+strings.Join(t.Files, ", "))
	}
	st := events.StampsFor(l.events, l.project.Feature, t.ID)
	if ts := stamps(st); ts != "" {
		lines = append(lines, ts)
	}
	return lines
}

func stamps(st events.Stamps) string {
	var parts []string
	if !st.Started.IsZero() {
		parts = append(parts, "Started "+st.Started.Local().Format(timeLayout))
	}
	if !st.Finished.IsZero() {
		parts = append(parts, "Finished "+st.Finished.Local().Format(timeLayout))
	}
	return strings.Join(parts, "  ")
}

func pad(lines []string, n int) []string {
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}
