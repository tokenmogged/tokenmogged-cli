package commands

import (
	"context"
	"errors"
	"fmt"
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
	return watchForMatch(ctx)
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
	return nil
}

func watchForMatch(ctx context.Context) error {
	client, err := api.New()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
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
			active := &state.ActiveMatch{
				MatchID:    m.MatchID,
				MatchToken: m.MatchToken,
				PlayerSide: m.PlayerSide,
				State:      m.State,
				Mode:       m.Mode,
				TopicID:    m.TopicID,
				StartedAt:  startedAt,
			}
			if err := state.SaveActiveMatch(active); err != nil {
				return err
			}
			fmt.Printf("Match bound: %s (%s)\n", m.MatchID, m.Mode)
			fmt.Println("Open the browser tab to see the topic. Hooks will start streaming when you run `claude`.")
			return nil
		}
	}
}

func SubmitNow(ctx context.Context) error {
	return errors.New("manual /submit is wired through the Claude Code slash command — run `claude` and use `/submit`")
}
