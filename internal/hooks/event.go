package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/config"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
	"github.com/tokenmogged/tokenmogged-cli/internal/submit"
)

const StateFile = "active_match.json"

func stateFilePath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, StateFile), nil
}

// Run handles a single hook invocation. Reads raw event JSON from stdin,
// inspects it, and POSTs to /api/stream if an active match is bound.
func Run(eventType string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	var payload map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			payload = map[string]any{"raw": string(raw)}
		}
	} else {
		payload = map[string]any{}
	}

	active, err := state.LoadActiveMatch()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if active == nil {
		return nil
	}

	if !cwdInScratch(payload, active.ScratchDir) {
		return nil
	}

	creds, err := config.Load()
	if err != nil {
		return nil
	}

	client := &api.Client{BaseURL: creds.APIBase, Token: creds.Token, HTTP: nil}
	client = mustClient(client)

	sessionUUID := ""
	if v, ok := payload["session_id"].(string); ok {
		sessionUUID = v
	} else if v, ok := payload["session"].(map[string]any); ok {
		if s, ok := v["id"].(string); ok {
			sessionUUID = s
		}
	}

	modelID := ""
	tokens := extractTokens(payload)
	if v, ok := payload["model"].(string); ok {
		modelID = v
	} else if v, ok := payload["model_id"].(string); ok {
		modelID = v
	}

	body := api.StreamEvent{
		MatchID:     active.MatchID,
		MatchToken:  active.MatchToken,
		EventID:     newEventID(),
		SessionUUID: sessionUUID,
		EventType:   eventType,
		ClientTs:    time.Now().UTC().Format(time.RFC3339Nano),
		ModelID:     modelID,
		Tokens:      tokens,
		Payload:     payload,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var resp api.StreamResponse
	if err := client.PostGzip(ctx, "/api/stream", body, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "[tokenmogged] hook post failed: %v\n", err)
		return nil
	}

	if !resp.Accepted {
		if resp.Reason == "token_cap_exceeded" || resp.Reason == "match_ended" {
			emitBlockedHint("Token cap reached. Match is being judged.")
		}
		return nil
	}

	if (eventType == "session_end" || isCompaction(payload, eventType)) && (active.State == "active" || active.State == "lobby") {
		// When claude is spawned by `tokenmogged play`, the parent process
		// triggers submission after claude exits — it's a stable, long-lived
		// process and won't be killed mid-upload. The hook subprocess is
		// lifecycle-bound to claude and can get SIGKILL'd before a multi-step
		// upload completes. Skip here; let the parent handle it.
		if os.Getenv("TOKENMOGGED_PLAY") == "1" {
			return nil
		}
		fmt.Fprintln(os.Stderr, "[tokenmogged] uploading submission...")
		cwd, _ := os.Getwd()
		reason := "manual_submit"
		if v, ok := payload["reason"].(string); ok && v != "" {
			reason = v
		}
		if err := submit.Trigger(active, cwd, reason); err != nil {
			fmt.Fprintf(os.Stderr, "[tokenmogged] submission failed: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "[tokenmogged] submission uploaded.")
		}
	}
	return nil
}

func extractTokens(p map[string]any) *api.TokenCounts {
	usageRaw, ok := p["usage"].(map[string]any)
	if !ok {
		if r, ok := p["message"].(map[string]any); ok {
			usageRaw, _ = r["usage"].(map[string]any)
		}
	}
	if usageRaw == nil {
		return nil
	}
	get := func(k string) int {
		switch v := usageRaw[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		default:
			return 0
		}
	}
	return &api.TokenCounts{
		Input:         get("input_tokens"),
		Output:        get("output_tokens"),
		CacheRead:     get("cache_read_input_tokens"),
		CacheCreation: get("cache_creation_input_tokens"),
	}
}

func isCompaction(payload map[string]any, eventType string) bool {
	if eventType == "pre_compact" {
		return true
	}
	if v, ok := payload["compact"].(bool); ok && v {
		return true
	}
	if v, ok := payload["reason"].(string); ok && strings.Contains(strings.ToLower(v), "compact") {
		return true
	}
	return false
}

func emitBlockedHint(msg string) {
	out := map[string]any{
		"continue":     false,
		"stopReason":   msg,
		"systemMessage": "[tokenmogged] " + msg,
	}
	b, _ := json.Marshal(out)
	fmt.Fprintln(os.Stdout, string(b))
}

func newEventID() string {
	return fmt.Sprintf("ev_%d_%d", time.Now().UnixNano(), os.Getpid())
}

// cwdInScratch returns true if the hook's cwd belongs to the active match's scratch dir.
// When the active match has no scratch dir (e.g., legacy state file pre-v0.1.1), it
// falls back to allowing the event — preserves backward compatibility.
func cwdInScratch(payload map[string]any, scratch string) bool {
	if scratch == "" {
		return true
	}
	cwd, _ := payload["cwd"].(string)
	if cwd == "" {
		return false
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return false
	}
	scratchAbs, err := filepath.Abs(scratch)
	if err != nil {
		return false
	}
	cwdAbs = filepath.Clean(cwdAbs) + string(filepath.Separator)
	scratchAbs = filepath.Clean(scratchAbs) + string(filepath.Separator)
	return strings.HasPrefix(cwdAbs, scratchAbs)
}

func mustClient(c *api.Client) *api.Client {
	if c.HTTP == nil {
		c.HTTP = newHTTPClient()
	}
	return c
}
