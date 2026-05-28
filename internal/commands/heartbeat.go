package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
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
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			var resp api.StreamResponse
			_ = client.PostJSON(reqCtx, "/api/stream", body, &resp)
			cancel()
		}
	}
}
