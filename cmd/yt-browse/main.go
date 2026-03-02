package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/nroyalty/yt-browse/internal/cache"
	"github.com/nroyalty/yt-browse/internal/config"
	"github.com/nroyalty/yt-browse/internal/tui"
	"github.com/nroyalty/yt-browse/internal/youtube"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: yt-browse <channel>\n\n")
		fmt.Fprintf(os.Stderr, "  channel: YouTube handle (@ddrjake), URL, or channel ID\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  yt-browse @ddrjake\n")
		fmt.Fprintf(os.Stderr, "  yt-browse https://youtube.com/@ddrjake\n")
		os.Exit(1)
	}

	channelInput := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	ytClient, err := youtube.NewClient(ctx, cfg.APIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating YouTube client: %s\n", err)
		os.Exit(1)
	}

	cacheStore, err := cache.NewStore(cfg.CacheDir, cfg.CacheTTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cache: %s\n", err)
		os.Exit(1)
	}

	// Clean expired cache entries on startup
	_ = cacheStore.CleanExpired()

	model := tui.New(cfg, ytClient, cacheStore, channelInput)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
