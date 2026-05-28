// Package transcript parses Claude Code session transcript .jsonl files.
//
// Claude Code writes session events to ~/.claude/projects/<encoded-cwd>/<session-uuid>.jsonl.
// Each line is one event; assistant turns carry a `message.usage` object with
// the authoritative token counts. The file is append-only.
//
// This package is shared by:
//   - submit.Run (final submission tarball)
//   - the SessionEnd hook (forwards summary on match end)
//   - the heartbeat goroutine (live token telemetry every ~60s)
//   - the hook event forwarder (live token telemetry per Claude tool use)
package transcript

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Summary is the aggregated view of one transcript file.
type Summary struct {
	SessionUUID       string
	TotalInput        int
	TotalOutput       int
	TotalCacheRead    int
	TotalCacheCreate  int
	ModelsUsed        map[string]int
	LatestModel       string
	EventCount        int
	HasCompactionLine bool
}

// FindLatest returns the most-recently-modified .jsonl under the
// Claude Code project directory derived from cwd. Returns "" if none.
func FindLatest(cwd string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claude", "projects", EncodeProjectPath(cwd))

	var latest string
	var latestMod time.Time
	walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".jsonl") {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if info.ModTime().After(latestMod) {
			latest = p
			latestMod = info.ModTime()
		}
		return nil
	})
	if walkErr != nil {
		// dir-doesn't-exist is fine; means claude hasn't started yet
		if errors.Is(walkErr, fs.ErrNotExist) {
			return "", nil
		}
		return "", walkErr
	}
	return latest, nil
}

// ReadFile parses a single transcript .jsonl and returns the aggregated
// Summary. Empty file or non-existent path returns an empty Summary with
// no error (mirrors how Claude Code's append-only writes can race CLI reads).
func ReadFile(path string) (Summary, error) {
	summary := Summary{ModelsUsed: map[string]int{}}
	if path == "" {
		return summary, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return summary, nil
		}
		return summary, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		summary.EventCount++
		if summary.SessionUUID == "" {
			if v, ok := rec["sessionId"].(string); ok {
				summary.SessionUUID = v
			}
		}
		msg, ok := rec["message"].(map[string]any)
		if !ok {
			continue
		}
		usage, ok := msg["usage"].(map[string]any)
		if ok {
			summary.TotalInput += toInt(usage["input_tokens"])
			summary.TotalOutput += toInt(usage["output_tokens"])
			summary.TotalCacheRead += toInt(usage["cache_read_input_tokens"])
			summary.TotalCacheCreate += toInt(usage["cache_creation_input_tokens"])
		}
		if model, ok := msg["model"].(string); ok && model != "" {
			summary.LatestModel = model
			if usage != nil {
				summary.ModelsUsed[model] += toInt(usage["input_tokens"]) + toInt(usage["output_tokens"])
			}
		}
		if v, ok := rec["isCompactSummary"].(bool); ok && v {
			summary.HasCompactionLine = true
		}
	}
	return summary, nil
}

// EncodeProjectPath converts a filesystem cwd to the directory name Claude
// Code uses under ~/.claude/projects. The encoding replaces path separators
// and spaces with hyphens and prepends a leading hyphen.
func EncodeProjectPath(cwd string) string {
	r := strings.NewReplacer(string(os.PathSeparator), "-", " ", "-")
	out := r.Replace(cwd)
	if strings.HasPrefix(out, "-") {
		return out
	}
	return "-" + out
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}
