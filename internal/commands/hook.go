package commands

import (
	"errors"

	"github.com/tokenmogged/tokenmogged-cli/internal/hooks"
)

func Hook(args []string) error {
	if len(args) < 1 {
		return errors.New("hook: missing event type")
	}
	return hooks.Run(args[0])
}
