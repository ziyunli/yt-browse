package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

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

type Config struct {
	APIKey   string
	CacheDir string
	CacheTTL time.Duration
}

// fileConfig mirrors the TOML file structure.
type fileConfig struct {
	APIKey   string `toml:"api_key"`
	CacheDir string `toml:"cache_dir"`
	CacheTTL string `toml:"cache_ttl"`
}

// loadConfigFile reads ~/.config/yt-browse/config.toml if it exists.
func loadConfigFile() (*fileConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}
	return &fc, nil
}

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

func Load() (*Config, error) {
	// Read environment variables first -- they take precedence and may
	// provide a complete configuration without needing the config file.
	envAPIKey := os.Getenv("YT_BROWSE_API_KEY")
	envCacheDir := os.Getenv("YT_BROWSE_CACHE_DIR")
	envCacheTTL := os.Getenv("YT_BROWSE_CACHE_TTL")

	// Only load the config file when env vars leave gaps.
	var fc fileConfig
	if envAPIKey == "" || envCacheDir == "" || envCacheTTL == "" {
		loaded, err := loadConfigFile()
		if err != nil {
			if envAPIKey != "" {
				// Env provides the required API key -- file errors are non-fatal.
			} else if errors.Is(err, ErrConfigNotFound) {
				return nil, ErrConfigNotFound
			} else {
				return nil, err
			}
		}
		if loaded != nil {
			fc = *loaded
		}
	}

	apiKey := envAPIKey
	if apiKey == "" {
		apiKey = fc.APIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("YouTube API key is required.\n\nSet it in ~/.config/yt-browse/config.toml:\n  api_key = \"your_key_here\"\n\nOr via environment variable:\n  export YT_BROWSE_API_KEY=your_key_here\n\nTo get an API key:\n1. Go to https://console.cloud.google.com\n2. Create a project and enable YouTube Data API v3\n3. Create an API key under Credentials\n\nThe API is free. The default quota (~10,000 units/day) is enough to browse\n~500k playlists or ~250k videos per day. Results are cached locally for\n24 hours, so repeat visits don't cost any quota.\nIf you hit the limit, requests fail until the quota resets at midnight\nPT. You can request a quota increase in the Google Cloud Console.")
	}

	cacheDir := envCacheDir
	if cacheDir == "" {
		cacheDir = fc.CacheDir
	}
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".yt-browse", "cache")
	}

	cacheDir, err := expandTilde(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("expanding cache dir path: %w", err)
	}

	cacheTTL := 24 * time.Hour
	if envCacheTTL != "" {
		parsed, err := time.ParseDuration(envCacheTTL)
		if err != nil {
			return nil, fmt.Errorf("parsing YT_BROWSE_CACHE_TTL %q: %w", envCacheTTL, err)
		}
		cacheTTL = parsed
	} else if fc.CacheTTL != "" {
		parsed, err := time.ParseDuration(fc.CacheTTL)
		if err != nil {
			return nil, fmt.Errorf("parsing cache_ttl %q from config file: %w", fc.CacheTTL, err)
		}
		cacheTTL = parsed
	}

	return &Config{
		APIKey:   apiKey,
		CacheDir: cacheDir,
		CacheTTL: cacheTTL,
	}, nil
}
