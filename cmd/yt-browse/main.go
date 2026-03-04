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
		switch os.Args[1] {
		case "-h", "-help", "--help":
			fmt.Println("Usage: yt-browse [channel]")
			fmt.Println()
			fmt.Println("Browse a YouTube channel's playlists and videos in the terminal.")
			fmt.Println()
			fmt.Println("  channel   @handle, channel URL, channel ID, or username")
			fmt.Println("            If omitted, opens a picker with recent channels.")
			fmt.Println()
			fmt.Println("Environment variables:")
			fmt.Println("  YT_BROWSE_API_KEY    YouTube Data API v3 key (required)")
			fmt.Println("  YT_BROWSE_CACHE_DIR  Cache directory (default: ~/.yt-browse/cache)")
			fmt.Println("  YT_BROWSE_CACHE_TTL  Cache TTL (default: 24h)")
			fmt.Println()
			fmt.Println("The YouTube Data API is free. The default quota (~10,000 units/day) is")
			fmt.Println("enough to browse ~500k playlists or ~250k videos per day. Results are")
			fmt.Println("cached locally for 24 hours, so repeat visits don't cost any quota.")
			fmt.Println("If you hit the limit, requests fail until the quota resets at midnight")
			fmt.Println("PT. You can request a quota increase in the Google Cloud Console.")
			os.Exit(0)
		default:
			channelInput = os.Args[1]
		}
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
