package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ClaudeSettingsPath() (string, error) {
	if v := os.Getenv("CLAUDE_SETTINGS_PATH"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

type hookSpec struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookEntry struct {
	Matcher string     `json:"matcher,omitempty"`
	Hooks   []hookSpec `json:"hooks"`
}

var managedEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Stop", "SessionEnd", "PreCompact", "Notification",
}

func managedCommand(event string) string {
	return "tokenmogged hook " + toSnake(event)
}

func toSnake(s string) string {
	out := []byte{}
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, byte(r-'A'+'a'))
		} else {
			out = append(out, byte(r))
		}
	}
	return string(out)
}

// InstallHooks merges tokenmogged's hooks into ~/.claude/settings.json,
// preserving any existing user hooks. Idempotent.
func InstallHooks() (string, error) {
	path, err := ClaudeSettingsPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if jerr := json.Unmarshal(data, &root); jerr != nil {
			return "", fmt.Errorf("parse existing %s: %w", path, jerr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	for _, evt := range managedEvents {
		entriesRaw, _ := hooks[evt].([]any)
		entries := []hookEntry{}
		for _, raw := range entriesRaw {
			var e hookEntry
			b, _ := json.Marshal(raw)
			if err := json.Unmarshal(b, &e); err == nil {
				filtered := e.Hooks[:0]
				for _, h := range e.Hooks {
					if h.Type == "command" && len(h.Command) >= len("tokenmogged hook") && h.Command[:len("tokenmogged hook")] == "tokenmogged hook" {
						continue
					}
					filtered = append(filtered, h)
				}
				e.Hooks = filtered
				if len(e.Hooks) > 0 {
					entries = append(entries, e)
				}
			}
		}
		entries = append(entries, hookEntry{
			Matcher: "*",
			Hooks:   []hookSpec{{Type: "command", Command: managedCommand(evt)}},
		})
		hooks[evt] = entries
	}

	root["hooks"] = hooks
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// UninstallHooks removes tokenmogged entries from ~/.claude/settings.json.
func UninstallHooks() error {
	path, err := ClaudeSettingsPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	root := map[string]any{}
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}
	for evt, raw := range hooks {
		entriesRaw, _ := raw.([]any)
		entries := []any{}
		for _, e := range entriesRaw {
			b, _ := json.Marshal(e)
			var entry hookEntry
			if err := json.Unmarshal(b, &entry); err == nil {
				filtered := entry.Hooks[:0]
				for _, h := range entry.Hooks {
					if h.Type == "command" && len(h.Command) >= len("tokenmogged hook") && h.Command[:len("tokenmogged hook")] == "tokenmogged hook" {
						continue
					}
					filtered = append(filtered, h)
				}
				entry.Hooks = filtered
				if len(entry.Hooks) > 0 {
					entries = append(entries, entry)
				}
			}
		}
		if len(entries) == 0 {
			delete(hooks, evt)
		} else {
			hooks[evt] = entries
		}
	}
	root["hooks"] = hooks
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}
