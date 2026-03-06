# First-Run Config Setup

## Problem

`yt-browse` requires `YT_BROWSE_API_KEY` but has no guided setup. Users must know to set an env var or manually create a config file. The other two settings (`cache_dir`, `cache_ttl`) also lack discoverability.

## Design

### Flow

1. `config.Load()` checks for `~/.config/yt-browse/config.toml`.
2. If the file doesn't exist, returns `config.ErrConfigNotFound` (sentinel error).
3. `main.go` catches this sentinel and runs a setup Bubble Tea program.
4. Setup program: styled text input for API key. Enter to submit, Esc/Ctrl+C to abort.
5. On submit: creates `~/.config/yt-browse/`, writes `config.toml` with entered API key + defaults.
6. `main.go` calls `config.Load()` again, proceeds normally.

### Config file written on first run

```toml
api_key = "the-key-they-entered"
cache_dir = "~/.yt-browse/cache"
cache_ttl = "24h"
```

### Tilde expansion

`config.Load()` expands `~/` prefixes in `cache_dir` using `os.UserHomeDir()`. This allows the config file to use portable `~` paths.

### Precedence

Environment variables > config file values > built-in defaults. Unchanged from current behavior.

### Setup TUI

Minimal Bubble Tea program using `textinput` bubble. Styled with lipgloss consistent with the main app.

```
Welcome to yt-browse!

Enter your YouTube Data API v3 key:
> ____________________

(Get one at https://console.cloud.google.com)
```

### File layout

- `internal/config/setup.go` -- setup Bubble Tea model + `RunSetup() error`
- `internal/config/config.go` -- `ErrConfigNotFound` sentinel, tilde expansion, updated `loadConfigFile`
- `cmd/yt-browse/main.go` -- catches sentinel, calls `config.RunSetup()`

### Edge cases

- User presses Esc/Ctrl+C during setup: exit cleanly with no file written.
- Config file exists but is missing `api_key`: existing error path (no setup flow, just error message mentioning both config file and env var).
- Env var set but no config file: works without setup (env var takes precedence, no file needed).
