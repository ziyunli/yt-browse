package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/nroyalty/yt-browse/internal/cache"
	"github.com/nroyalty/yt-browse/internal/config"
	"github.com/nroyalty/yt-browse/internal/recent"
	"github.com/nroyalty/yt-browse/internal/tui"
	"github.com/nroyalty/yt-browse/internal/youtube"
)

func main() {
	var channelInput string
	if len(os.Args) >= 2 {
		channelInput = os.Args[1]
	}

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

	// Clean expired cache entries on startup and enforce size limit
	_ = cacheStore.CleanExpired()
	_ = cacheStore.PurgeOverSize()

	// Recent channels store lives next to the cache dir
	recentPath := filepath.Join(filepath.Dir(cfg.CacheDir), "recent_channels.json")
	recentStore := recent.NewStore(recentPath)

	model := tui.New(cfg, ytClient, cacheStore, recentStore, channelInput)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
