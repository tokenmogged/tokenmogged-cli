package commands

import (
	"fmt"

	"github.com/tokenmogged/tokenmogged-cli/internal/hooks"
)

func Setup() error {
	path, err := hooks.InstallHooks()
	if err != nil {
		return err
	}
	fmt.Printf("Installed Claude Code hooks into:\n  %s\n\n", path)
	fmt.Println("Next steps:")
	fmt.Println("  tokenmogged play   # open the lobby and queue up")
	return nil
}

func UninstallHooks() error {
	if err := hooks.UninstallHooks(); err != nil {
		return err
	}
	fmt.Println("Removed tokenmogged entries from your Claude Code settings.")
	return nil
}
