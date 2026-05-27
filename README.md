# tokenmogged-cli

Open-source Go shim that streams Claude Code hook events to [tokenmogged.com](https://tokenmogged.com)
during a 1v1 match. Outside of an active match, it is a no-op.

## Install

```sh
curl tokenmogged.com/install | sh
```

## Usage

```sh
tokenmogged login    # bind this account
tokenmogged setup    # install Claude Code hooks
tokenmogged whoami   # verify
tokenmogged play     # opens the lobby in your browser
```

## What it does

- `tokenmogged setup` writes hook entries to `~/.claude/settings.json` so that every
  Claude Code event (`SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`,
  `Stop`, `SessionEnd`) runs `tokenmogged hook <event_type>` which reads the event
  JSON on stdin and POSTs it to `https://tokenmogged.com/api/stream` with your CLI
  bearer token.
- When you're not in a match, hooks are a fast no-op (~5ms).
- During a match, the CLI checks for the bound match every few seconds, includes the
  `match_id` and `match_token` in each hook POST, and triggers the submission flow
  at `SessionEnd` or `/submit`.

## License

MIT
