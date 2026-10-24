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
