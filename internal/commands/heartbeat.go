package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
	"github.com/tokenmogged/tokenmogged-cli/internal/transcript"
)

const heartbeatInterval = 60 * time.Second

// runHeartbeat fires a best-effort POST /api/stream every `heartbeatInterval`
// while ctx is alive. It exists to keep the match's matchLastEvent Redis key
// fresh during long Claude generations and reading time, when the standard
// Claude Code hooks (PreToolUse, PostToolUse, etc.) don't fire. The server
// side accepts `event_type: "heartbeat"` as a no-op fast-path: it refreshes
// the timer and exits without writing to hook_events, counting tokens, or
// broadcasting via Ably.
//
// Errors are intentionally swallowed. A heartbeat is best-effort liveness;
// network blips don't get surfaced. If the server is unreachable past
// STALE_MS, the match correctly transitions to disconnect_grace — that's
// the desired behavior, not a bug to paper over.
func runHeartbeat(ctx context.Context, active *state.ActiveMatch) {
	client, err := api.New()
	if err != nil {
		return
	}
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			body := api.StreamEvent{
				MatchID:     active.MatchID,
				MatchToken:  active.MatchToken,
				EventID:     fmt.Sprintf("hb_%d", time.Now().UnixNano()),
				SessionUUID: "heartbeat",
				EventType:   "heartbeat",
				ClientTs:    time.Now().UTC().Format(time.RFC3339Nano),
				Payload:     map[string]any{},
			}
			// Attach absolute cumulative tokens read from the latest Claude
			// .jsonl transcript in the scratch dir. Hooks fire on user
			// actions; this catches the silent moments (long generations,
			// reading time) so the live UI counter still ticks up.
			if path, err := transcript.FindLatest(active.ScratchDir); err == nil && path != "" {
				if s, err := transcript.ReadFile(path); err == nil && (s.TotalInput+s.TotalOutput) > 0 {
					body.Tokens = &api.TokenCounts{
						Input:         s.TotalInput,
						Output:        s.TotalOutput,
						CacheRead:     s.TotalCacheRead,
						CacheCreation: s.TotalCacheCreate,
					}
					if s.LatestModel != "" {
						body.ModelID = s.LatestModel
					}
				}
			}
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			var resp api.StreamResponse
			_ = client.PostJSON(reqCtx, "/api/stream", body, &resp)
			cancel()
		}
	}
}
