package events

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	TS      time.Time `json:"ts"`
	Feature string    `json:"feature"`
	Task    string    `json:"task"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Session string    `json:"session,omitempty"`
}

type Stamps struct {
	Started  time.Time
	Finished time.Time
}

func Path(root string) string {
	return filepath.Join(root, ".specify", "events.jsonl")
}

func Read(root string) []Event {
	f, err := os.Open(Path(root))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.Task != "" && !e.TS.IsZero() {
			out = append(out, e)
		}
	}
	return out
}

func StampsFor(evs []Event, feature, task string) Stamps {
	var s Stamps
	for _, e := range evs {
		if e.Feature != feature || e.Task != task {
			continue
		}
		switch e.To {
		case "doing":
			s.Started = e.TS
		case "done":
			s.Finished = e.TS
		}
	}
	return s
}

func Append(root string, e Event) error {
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func LastStates(evs []Event, feature string) map[string]string {
	out := map[string]string{}
	for _, e := range evs {
		if e.Feature == feature {
			out[e.Task] = e.To
		}
	}
	return out
}
