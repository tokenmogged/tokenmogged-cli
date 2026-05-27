package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/config"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
)

func Play(ctx context.Context) error {
	base := config.APIBase()
	url := strings.TrimRight(base, "/") + "/play"
	fmt.Printf("Opening: %s\n", url)
	openBrowser(url)
	fmt.Println("(I'll wait here. When a match starts, this CLI binds to it.)")
	fmt.Println()
	active, err := watchForMatch(ctx)
	if err != nil || active == nil {
		return err
	}
	return launchClaude(ctx, active)
}

func Status(ctx context.Context) error {
	active, err := state.LoadActiveMatch()
	if err != nil {
		return err
	}
	if active == nil {
		fmt.Println("No active match.")
		return nil
	}
	fmt.Printf("Active match: %s (%s, %s, side=%s)\n", active.MatchID, active.State, active.Mode, active.PlayerSide)
	if active.ScratchDir != "" {
		fmt.Printf("Scratch dir:  %s\n", active.ScratchDir)
	}
	return nil
}

func watchForMatch(ctx context.Context) (*state.ActiveMatch, error) {
	client, err := api.New()
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, nil
		case <-ticker.C:
			var resp api.ActiveMatchResponse
			cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
			err := client.GetJSON(cctx, "/api/match/active", &resp)
			cancel()
			if err != nil {
				continue
			}
			if resp.Match == nil {
				continue
			}
			m := resp.Match
			startedAt, _ := time.Parse(time.RFC3339, m.StartedAt)
			scratch, err := scratchDirFor(m.MatchID)
			if err != nil {
				return nil, err
			}
			active := &state.ActiveMatch{
				MatchID:    m.MatchID,
				MatchToken: m.MatchToken,
				PlayerSide: m.PlayerSide,
				State:      m.State,
				Mode:       m.Mode,
				TopicID:    m.TopicID,
				StartedAt:  startedAt,
				ScratchDir: scratch,
			}
			if err := state.SaveActiveMatch(active); err != nil {
				return nil, err
			}
			return active, nil
		}
	}
}

func scratchDirFor(matchID string) (string, error) {
	if override := os.Getenv("TOKENMOGGED_SCRATCH_DIR"); override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return abs, os.MkdirAll(abs, 0o755)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".tokenmogged", "matches", matchID)
	return dir, os.MkdirAll(dir, 0o755)
}

func launchClaude(ctx context.Context, active *state.ActiveMatch) error {
	bin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Printf("Match bound: %s (%s)\n", active.MatchID, active.Mode)
		fmt.Println()
		fmt.Println("`claude` not on PATH. Run it manually in this directory:")
		fmt.Printf("  cd %s\n", active.ScratchDir)
		fmt.Println("  claude")
		return nil
	}

	fmt.Printf("Match bound: %s (%s, topic=%s)\n", active.MatchID, active.Mode, active.TopicID)
	fmt.Printf("Scratch dir: %s\n", active.ScratchDir)
	fmt.Println("Launching Claude Code...")
	fmt.Println()

	cmd := exec.CommandContext(ctx, bin)
	cmd.Dir = active.ScratchDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && !errors.Is(err, context.Canceled) {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return fmt.Errorf("claude exited: %w", err)
	}
	return nil
}

func SubmitNow(ctx context.Context) error {
	return errors.New("manual /submit is wired through the Claude Code slash command — run `claude` and use `/submit`")
}
