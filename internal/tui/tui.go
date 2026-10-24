package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kOlapsis/speckit-board/internal/events"
	"github.com/kOlapsis/speckit-board/internal/sessions"
	"github.com/kOlapsis/speckit-board/internal/spec"
	"github.com/kOlapsis/speckit-board/internal/tmux"
)

type Tmux interface {
	Inside() bool
	FindPane(root, target string) (string, error)
	Send(pane, text string) error
	Jump(pane string) error
}

type Options struct {
	Roots       []string
	Target      string
	SessionsDir string
	Tmux        Tmux
	Reload      <-chan struct{}
}

type ReloadMsg struct{}

type Selection struct {
	Lane   int
	Column spec.ColumnID
	Card   spec.Card
	OK     bool
}

type lane struct {
	project spec.Project
	cols    []spec.Column
	events  []events.Event
	light   sessions.Light
	session *sessions.Session
}

type Model struct {
	opts      Options
	lanes     []lane
	w, h      int
	laneIdx   int
	colIdx    int
	rowIdx    int
	message   string
	showHelp  bool
	shortHelp help.Model
}

type keys struct{ jump, verify, enter, helpKey, quit key.Binding }

var shortKeys = keys{
	jump:    key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "jump")),
	verify:  key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "verify")),
	enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "implement")),
	helpKey: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
}

func (k keys) ShortHelp() []key.Binding {
	return []key.Binding{k.jump, k.verify, k.enter, k.helpKey, k.quit}
}

func (k keys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

func New(opts Options) *Model {
	if opts.Tmux == nil {
		opts.Tmux = tmux.New()
	}
	m := &Model{opts: opts, shortHelp: help.New()}
	m.reload()
	for i, c := range m.lanes[0].cols {
		if len(c.Cards) > 0 {
			m.colIdx = i
			break
		}
	}
	return m
}

func (m *Model) reload() {
	all := sessions.List(m.opts.SessionsDir)
	m.lanes = m.lanes[:0]
	for _, root := range m.opts.Roots {
		p := spec.Load(root)
		l := lane{project: p, cols: p.Columns(), events: events.Read(p.Root)}
		l.light, l.session = sessions.Summarize(all, p.Root)
		m.lanes = append(m.lanes, l)
	}
	m.clamp()
}

func (m *Model) clamp() {
	cards := m.lanes[m.laneIdx].cols[m.colIdx].Cards
	if m.rowIdx >= len(cards) {
		m.rowIdx = max(len(cards)-1, 0)
	}
}

func (m *Model) Selection() Selection {
	l := m.lanes[m.laneIdx]
	col := l.cols[m.colIdx]
	s := Selection{Lane: m.laneIdx, Column: col.ID}
	if m.rowIdx < len(col.Cards) {
		s.Card, s.OK = col.Cards[m.rowIdx], true
	}
	return s
}

func (m *Model) Init() tea.Cmd {
	return m.waitReload()
}

func (m *Model) waitReload() tea.Cmd {
	ch := m.opts.Reload
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return nil
		}
		return ReloadMsg{}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case ReloadMsg:
		m.reload()
		return m, m.waitReload()
	case tea.KeyMsg:
		return m, m.onKey(msg)
	}
	return m, nil
}

func (m *Model) onKey(msg tea.KeyMsg) tea.Cmd {
	m.message = ""
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit
	case "?":
		m.showHelp = !m.showHelp
	case "left", "h":
		m.colIdx = max(m.colIdx-1, 0)
		m.rowIdx = 0
	case "right", "l":
		m.colIdx = min(m.colIdx+1, len(m.lanes[m.laneIdx].cols)-1)
		m.rowIdx = 0
	case "up":
		m.rowIdx = max(m.rowIdx-1, 0)
	case "down":
		m.rowIdx++
	case "tab", "shift+tab":
		step := 1
		if msg.String() == "shift+tab" {
			step = len(m.lanes) - 1
		}
		m.laneIdx = (m.laneIdx + step) % len(m.lanes)
		m.rowIdx = 0
	case "j":
		m.jump()
	case "v":
		m.send("/speckit-verify")
	case "enter":
		s := m.Selection()
		if s.Column != spec.Remediation || !s.OK || s.Card.Task == nil {
			m.message = "Enter only works on a remediation task"
			break
		}
		m.send("/speckit-implement " + s.Card.ID)
	}
	m.clamp()
	return nil
}

func (m *Model) send(text string) {
	if !m.opts.Tmux.Inside() {
		m.message = tmux.ErrNotInTmux.Error()
		return
	}
	pane, err := m.opts.Tmux.FindPane(m.lanes[m.laneIdx].project.Root, m.opts.Target)
	if err == nil {
		err = m.opts.Tmux.Send(pane, text)
	}
	if err != nil {
		m.message = err.Error()
		return
	}
	m.message = fmt.Sprintf("sent %q to %s", text, pane)
}

func (m *Model) jump() {
	if !m.opts.Tmux.Inside() {
		m.message = tmux.ErrNotInTmux.Error()
		return
	}
	var target *sessions.Session
	for i := range m.lanes {
		l := m.lanes[(m.laneIdx+i)%len(m.lanes)]
		if l.light == sessions.Amber {
			target = l.session
			break
		}
	}
	switch {
	case target == nil:
		m.message = "no session is waiting"
	case target.Pane == "":
		m.message = "the waiting session has no tmux pane recorded"
	default:
		if err := m.opts.Tmux.Jump(target.Pane); err != nil {
			m.message = err.Error()
			return
		}
		m.message = "jumped to " + target.Pane
	}
}
