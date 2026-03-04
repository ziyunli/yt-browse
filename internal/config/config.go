package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	APIKey   string
	CacheDir string
	CacheTTL time.Duration
}

func Load() (*Config, error) {
	apiKey := os.Getenv("YT_BROWSE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("YT_BROWSE_API_KEY environment variable is required.\n\nTo get an API key:\n1. Go to https://console.cloud.google.com\n2. Create a project and enable YouTube Data API v3\n3. Create an API key under Credentials\n4. Export it: export YT_BROWSE_API_KEY=your_key_here\n\nThe API is free. The default quota (~10,000 units/day) is enough to browse\n~500k playlists or ~250k videos per day. Results are cached locally for\n24 hours, so repeat visits don't cost any quota.\nIf you hit the limit, requests fail until the quota resets at midnight\nPT. You can request a quota increase in the Google Cloud Console.")
	}

	cacheDir := os.Getenv("YT_BROWSE_CACHE_DIR")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".yt-browse", "cache")
	}

	cacheTTL := 24 * time.Hour
	if ttlStr := os.Getenv("YT_BROWSE_CACHE_TTL"); ttlStr != "" {
		parsed, err := time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("parsing YT_BROWSE_CACHE_TTL %q: %w", ttlStr, err)
		}
		cacheTTL = parsed
	}

	return &Config{
		APIKey:   apiKey,
		CacheDir: cacheDir,
		CacheTTL: cacheTTL,
	}, nil
}
