package tmux

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fake struct {
	calls [][]string
	panes string
	cur   string
	fail  string
}

func (f *fake) run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	if args[0] == f.fail {
		return "", errors.New("tmux failed")
	}
	switch args[0] {
	case "list-panes":
		return f.panes, nil
	case "display-message":
		return f.cur + "\n", nil
	}
	return "", nil
}

func client(f *fake, inside bool) *Client {
	env := map[string]string{}
	if inside {
		env["TMUX"] = "/tmp/tmux-1000/default,1,0"
		env["TMUX_PANE"] = "%1"
	}
	return &Client{Run: f.run, Getenv: func(k string) string { return env[k] }}
}

const panes = "work\t%1\tzsh\t/work/app\n" +
	"work\t%2\tclaude\t/work/app-two\n" +
	"other\t%3\tclaude\t/work/app\n" +
	"work\t%4\tclaude\t/work/app/web\n" +
	"work\t%5\tvim\t/work/app\n"

func TestFindPanePrefersCurrentSession(t *testing.T) {
	c := client(&fake{panes: panes, cur: "work"}, true)
	got, err := c.FindPane("/work/app", "")
	if err != nil || got != "%4" {
		t.Errorf("got %q %v", got, err)
	}
	c = client(&fake{panes: panes, cur: "other"}, true)
	if got, _ := c.FindPane("/work/app", ""); got != "%3" {
		t.Errorf("got %q", got)
	}
}

func TestFindPaneTargetOverrides(t *testing.T) {
	f := &fake{panes: panes}
	got, err := client(f, true).FindPane("/work/app", "%9")
	if err != nil || got != "%9" || len(f.calls) != 0 {
		t.Errorf("got %q %v %v", got, err, f.calls)
	}
}

func TestFindPaneNone(t *testing.T) {
	_, err := client(&fake{panes: "work\t%1\tzsh\t/work/app\n", cur: "work"}, true).FindPane("/work/app", "")
	if !errors.Is(err, ErrNoPane) {
		t.Errorf("got %v", err)
	}
}

func TestOutsideTmux(t *testing.T) {
	c := client(&fake{}, false)
	if c.Inside() {
		t.Error("must not be inside")
	}
	if _, err := c.FindPane("/work/app", ""); !errors.Is(err, ErrNotInTmux) {
		t.Errorf("find: %v", err)
	}
	if err := c.Send("%1", "x"); !errors.Is(err, ErrNotInTmux) {
		t.Errorf("send: %v", err)
	}
	if err := c.Jump("%1"); !errors.Is(err, ErrNotInTmux) {
		t.Errorf("jump: %v", err)
	}
}

func TestSend(t *testing.T) {
	f := &fake{}
	if err := client(f, true).Send("%3", "/speckit-verify"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"send-keys", "-t", "%3", "-l", "/speckit-verify"}, {"send-keys", "-t", "%3", "Enter"}}
	if !reflect.DeepEqual(f.calls, want) {
		t.Errorf("%v", f.calls)
	}
}

func TestSendError(t *testing.T) {
	if err := client(&fake{fail: "send-keys"}, true).Send("%3", "x"); err == nil {
		t.Error("error must propagate")
	}
}

func TestJump(t *testing.T) {
	f := &fake{}
	if err := client(f, true).Jump("%3"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range f.calls {
		got = append(got, strings.Join(c, " "))
	}
	want := []string{"select-window -t %3", "select-pane -t %3", "switch-client -t %3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%v", got)
	}
}

func TestJumpError(t *testing.T) {
	if err := client(&fake{fail: "switch-client"}, true).Jump("%3"); err == nil {
		t.Error("error must propagate")
	}
}
