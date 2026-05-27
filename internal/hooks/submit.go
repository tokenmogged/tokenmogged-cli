package hooks

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
	"github.com/tokenmogged/tokenmogged-cli/internal/submit"
)

func triggerSubmission(active *state.ActiveMatch, payload map[string]any) error {
	client, err := api.New()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	reason := "manual_submit"
	if payload != nil {
		if v, ok := payload["reason"].(string); ok && v != "" {
			reason = v
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := submit.Run(ctx, client, active, cwd, reason); err != nil {
		return err
	}
	if err := state.ClearActiveMatch(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
