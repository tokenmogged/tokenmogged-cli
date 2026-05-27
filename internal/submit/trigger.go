package submit

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/api"
	"github.com/tokenmogged/tokenmogged-cli/internal/state"
)

// Trigger is the canonical submission entry point used by both `tokenmogged
// play` (after claude exits) and the SessionEnd hook (fallback for users
// who launch claude manually). Caller passes the scratch dir as cwd.
func Trigger(active *state.ActiveMatch, cwd string, reason string) error {
	client, err := api.New()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := Run(ctx, client, active, cwd, reason); err != nil {
		return err
	}
	if err := state.ClearActiveMatch(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
