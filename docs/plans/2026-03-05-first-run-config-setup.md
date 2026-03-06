# First-Run Config Setup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When no config file exists, run a Bubble Tea setup wizard that prompts for the API key and writes `~/.config/yt-browse/config.toml` with defaults.

**Architecture:** `config.Load()` returns a sentinel `ErrConfigNotFound` when the TOML file is missing. `main.go` catches it, runs a mini Bubble Tea setup program, then retries `Load()`. Config file supports `~` in paths via tilde expansion.

**Tech Stack:** Go, BurntSushi/toml, Bubble Tea v2, lipgloss v2

---

### Task 1: Add `ErrConfigNotFound` sentinel and tilde expansion to config.go

**Files:**
- Modify: `internal/config/config.go:26-41` (loadConfigFile) and `internal/config/config.go:58-68` (cacheDir handling)

**Step 1: Add sentinel error and config path helper**

Add at the top of `config.go`, after the imports:

```go
import "errors"

var ErrConfigNotFound = errors.New("config file not found")

// ConfigDir returns the path to the config directory (~/.config/yt-browse).
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "yt-browse"), nil
}

// ConfigPath returns the full path to config.toml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}
```

**Step 2: Update `loadConfigFile` to return sentinel**

Change the `os.IsNotExist` branch in `loadConfigFile` from returning `&fileConfig{}, nil` to returning `nil, ErrConfigNotFound`.

**Step 3: Add `expandTilde` helper**

```go
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[2:]), nil
}
```

**Step 4: Update `Load()` to handle sentinel and expand tilde**

In `Load()`:
- When `loadConfigFile` returns `ErrConfigNotFound`, check if `YT_BROWSE_API_KEY` env var is set. If yes, proceed with an empty `fileConfig` (current behavior for env-var-only users). If no, return `ErrConfigNotFound` to the caller.
- After reading `cacheDir` from file config, call `expandTilde()` on it.

```go
func Load() (*Config, error) {
	fc, err := loadConfigFile()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			// No config file. If API key is in env, proceed without file.
			if os.Getenv("YT_BROWSE_API_KEY") != "" {
				fc = &fileConfig{}
			} else {
				return nil, ErrConfigNotFound
			}
		} else {
			return nil, err
		}
	}

	// ... rest of Load unchanged except cacheDir tilde expansion ...

	// After resolving cacheDir from env/file/default:
	if cacheDir != "" {
		cacheDir, err = expandTilde(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("expanding cache_dir path: %w", err)
		}
	}
```

**Step 5: Build and verify**

Run: `go build ./...`
Expected: compiles cleanly.

**Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "Add ErrConfigNotFound sentinel and tilde expansion to config"
```

---

### Task 2: Create the setup Bubble Tea program

**Files:**
- Create: `internal/config/setup.go`

**Step 1: Write the setup model**

Create `internal/config/setup.go` with a Bubble Tea model containing:
- A `textinput.Model` for the API key
- A `done` bool (user pressed Enter)
- A `cancelled` bool (user pressed Esc/Ctrl+C)

Key behaviors:
- `Init`: focus the text input, blink cursor
- `Update`: Enter sets `done=true` and quits. Esc/Ctrl+C sets `cancelled=true` and quits. Otherwise delegate to text input.
- `View`: styled welcome message, prompt, text input, hint about where to get a key.

Use lipgloss styles consistent with the main app (orange `#FF6600` for accents, `#888888` for muted text, white for primary text).

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	setupTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6600")).
		MarginBottom(1)

	setupPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	setupHintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginTop(1)

	setupErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		MarginTop(1)
)

type setupModel struct {
	textInput textinput.Model
	done      bool
	cancelled bool
	err       string
}

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Placeholder = "AIza..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return setupModel{textInput: ti}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.textInput.Value())
			if val == "" {
				m.err = "API key cannot be empty."
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.err = "" // clear error on new input
	return m, cmd
}

func (m setupModel) View() string {
	var b strings.Builder

	b.WriteString(setupTitleStyle.Render("Welcome to yt-browse!"))
	b.WriteString("\n")
	b.WriteString(setupPromptStyle.Render("Enter your YouTube Data API v3 key:"))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(setupErrorStyle.Render(m.err))
		b.WriteString("\n")
	}
	b.WriteString(setupHintStyle.Render("Get one at https://console.cloud.google.com"))
	b.WriteString("\n")
	b.WriteString(setupHintStyle.Render("Press Esc to cancel."))

	return b.String()
}
```

**Step 2: Add `RunSetup()` function**

In the same file, add the public function that `main.go` will call:

```go
// RunSetup runs the interactive setup wizard and writes the config file.
// Returns nil on success, or an error if cancelled/failed.
func RunSetup() error {
	m := newSetupModel()
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	final := result.(setupModel)
	if final.cancelled {
		return fmt.Errorf("setup cancelled")
	}

	apiKey := strings.TrimSpace(final.textInput.Value())
	return writeConfigFile(apiKey)
}
```

**Step 3: Add `writeConfigFile` helper**

```go
const defaultCacheDir = "~/.yt-browse/cache"
const defaultCacheTTL = "24h"

func writeConfigFile(apiKey string) error {
	configDir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	content := fmt.Sprintf("api_key = %q\ncache_dir = %q\ncache_ttl = %q\n",
		apiKey, defaultCacheDir, defaultCacheTTL)

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
```

Note: file permissions `0o600` since it contains an API key.

**Step 4: Build and verify**

Run: `go build ./...`
Expected: compiles cleanly.

**Step 5: Commit**

```bash
git add internal/config/setup.go
git commit -m "Add Bubble Tea setup wizard for first-run config"
```

---

### Task 3: Wire up setup in main.go

**Files:**
- Modify: `cmd/yt-browse/main.go:45-49`

**Step 1: Update main.go to catch ErrConfigNotFound**

Replace the current `config.Load()` error handling:

```go
// before:
cfg, err := config.Load()
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err)
    os.Exit(1)
}

// after:
cfg, err := config.Load()
if errors.Is(err, config.ErrConfigNotFound) {
    if setupErr := config.RunSetup(); setupErr != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", setupErr)
        os.Exit(1)
    }
    cfg, err = config.Load()
}
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %s\n", err)
    os.Exit(1)
}
```

Add `"errors"` to the imports.

**Step 2: Update help text to mention config file**

In the help text section, update the "Environment variables" section to mention the config file:

```go
fmt.Println("Configuration:")
fmt.Println("  Config file: ~/.config/yt-browse/config.toml (created on first run)")
fmt.Println()
fmt.Println("Environment variables (override config file):")
fmt.Println("  YT_BROWSE_API_KEY    YouTube Data API v3 key")
fmt.Println("  YT_BROWSE_CACHE_DIR  Cache directory (default: ~/.yt-browse/cache)")
fmt.Println("  YT_BROWSE_CACHE_TTL  Cache TTL (default: 24h)")
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: compiles cleanly.

**Step 4: Manual smoke test**

Temporarily move existing config (if any) and run the binary to verify the setup flow appears:

```bash
mv ~/.config/yt-browse/config.toml ~/.config/yt-browse/config.toml.bak 2>/dev/null
go run ./cmd/yt-browse
# Should show setup wizard. Press Esc to cancel.
mv ~/.config/yt-browse/config.toml.bak ~/.config/yt-browse/config.toml 2>/dev/null
```

**Step 5: Commit**

```bash
git add cmd/yt-browse/main.go
git commit -m "Wire up first-run config setup in main"
```
