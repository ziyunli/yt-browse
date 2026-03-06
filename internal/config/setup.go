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
	ti.SetWidth(50)
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
	m.err = ""
	return m, cmd
}

func (m setupModel) View() tea.View {
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
	return tea.NewView(b.String())
}

// RunSetup launches an interactive setup wizard to configure yt-browse.
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
