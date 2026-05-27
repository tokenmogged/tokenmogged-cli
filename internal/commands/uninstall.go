package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokenmogged/tokenmogged-cli/internal/config"
	"github.com/tokenmogged/tokenmogged-cli/internal/hooks"
)

func Uninstall(args []string) error {
	yes := false
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			yes = true
		}
	}

	binary, _ := os.Executable()
	configDir, _ := config.ConfigDir()
	scratchDir := matchesDir()

	fmt.Println("This will remove:")
	fmt.Println("  - tokenmogged hooks from ~/.claude/settings.json")
	if configDir != "" {
		fmt.Printf("  - %s (credentials + active match state)\n", configDir)
	}
	if scratchDir != "" {
		fmt.Printf("  - %s (match scratch directories)\n", scratchDir)
	}
	if binary != "" {
		fmt.Printf("  - %s (the tokenmogged binary itself)\n", binary)
	}
	fmt.Println()
	fmt.Println("Your PATH entry in ~/.zshrc / ~/.bashrc / etc. is NOT touched — remove manually if desired.")
	fmt.Println()

	if !yes {
		fmt.Print("Continue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	var failures []string

	if err := hooks.UninstallHooks(); err != nil {
		failures = append(failures, fmt.Sprintf("hooks: %v", err))
	} else {
		fmt.Println("✓ Removed tokenmogged entries from Claude Code settings.")
	}

	if configDir != "" {
		if err := os.RemoveAll(configDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Sprintf("config dir: %v", err))
		} else {
			fmt.Printf("✓ Removed %s\n", configDir)
		}
	}

	if scratchDir != "" {
		if err := os.RemoveAll(scratchDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Sprintf("scratch dir: %v", err))
		} else {
			fmt.Printf("✓ Removed %s\n", scratchDir)
		}
	}

	if binary != "" {
		if err := os.Remove(binary); err != nil {
			failures = append(failures, fmt.Sprintf("binary: %v (remove manually)", err))
		} else {
			fmt.Printf("✓ Removed %s\n", binary)
		}
	}

	if len(failures) > 0 {
		fmt.Println()
		fmt.Println("Some items could not be removed:")
		for _, f := range failures {
			fmt.Printf("  - %s\n", f)
		}
		return errors.New("partial uninstall")
	}

	fmt.Println()
	fmt.Println("Uninstalled. Sorry to see you go.")
	return nil
}

func matchesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".tokenmogged")
}
