package commands

import "fmt"

func Help(version string) {
	fmt.Printf(`tokenmogged v%s — 1v1 competitive coding with Claude Code

Usage:
  tokenmogged <command> [args]

Commands:
  login           Bind this machine to your tokenmogged.com account.
  logout          Forget credentials on this machine.
  whoami          Print the currently bound account.
  setup           Install Claude Code hooks into ~/.claude/settings.json.
  uninstall-hooks Remove tokenmogged entries from your Claude Code settings.
  play            Open the tokenmogged lobby in your browser.
  status          Print active match info (if any).
  submit          Force-submit the current working directory now.
  hook <event>    Internal: invoked by Claude Code hooks (reads JSON on stdin).
  version         Print the CLI version.
  help            This message.

Source: https://github.com/tokenmogged/tokenmogged-cli
`, version)
}
