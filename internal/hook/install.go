package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const command = "speckit-board hook"

var wanted = []struct{ event, matcher string }{
	{"Notification", ""},
	{"Stop", ""},
	{"PostToolUse", "Edit|Write|MultiEdit"},
}

type Result struct {
	Path    string
	Changed bool
	Added   []string
	Content []byte
}

func Install(path string, dryRun bool) (Result, error) {
	res := Result{Path: path}
	settings := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		if err := dec.Decode(&settings); err != nil {
			return res, fmt.Errorf("%s is not valid JSON, refusing to modify it: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return res, err
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		if _, present := settings["hooks"]; present {
			return res, fmt.Errorf("%s: \"hooks\" is not an object", path)
		}
		hooks = map[string]any{}
	}
	for _, w := range wanted {
		groups, ok := hooks[w.event].([]any)
		if _, present := hooks[w.event]; present && !ok {
			return res, fmt.Errorf("%s: hooks.%s is not a list", path, w.event)
		}
		if installed(groups) {
			continue
		}
		group := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command}}}
		if w.matcher != "" {
			group["matcher"] = w.matcher
		}
		hooks[w.event] = append(groups, group)
		res.Added = append(res.Added, w.event)
	}
	if len(res.Added) == 0 {
		return res, nil
	}
	settings["hooks"] = hooks
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		return res, err
	}
	res.Changed, res.Content = true, buf.Bytes()
	if dryRun {
		return res, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return res, err
	}
	return res, os.WriteFile(path, res.Content, 0o644)
}

func installed(groups []any) bool {
	for _, g := range groups {
		group, _ := g.(map[string]any)
		list, _ := group["hooks"].([]any)
		for _, h := range list {
			if c, _ := h.(map[string]any)["command"].(string); isOurs(c) {
				return true
			}
		}
	}
	return false
}
