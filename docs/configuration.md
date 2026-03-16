# Configuration

## Config file

`~/.config/yt-browse/config.toml` — created automatically on first run.

```toml
api_key = "your-youtube-data-api-v3-key"
cache_dir = "~/.yt-browse/cache"
cache_ttl = "24h"
```

| Key | Description | Default |
|-----|-------------|---------|
| `api_key` | YouTube Data API v3 key | *(required)* |
| `cache_dir` | Cache directory (supports `~/`) | `~/.yt-browse/cache` |
| `cache_ttl` | Cache time-to-live ([Go duration](https://pkg.go.dev/time#ParseDuration)) | `24h` |

## Environment variables

Environment variables override config file values:

| Variable | Overrides |
|----------|-----------|
| `YT_BROWSE_API_KEY` | `api_key` |
| `YT_BROWSE_CACHE_DIR` | `cache_dir` |
| `YT_BROWSE_CACHE_TTL` | `cache_ttl` |

## First-run setup

If no config file exists and `YT_BROWSE_API_KEY` is not set, yt-browse launches an interactive setup wizard that prompts for your API key and writes the config file.

To get an API key:
1. Go to https://console.cloud.google.com
2. Create a project and enable YouTube Data API v3
3. Create an API key under Credentials

The API is free. The default quota (~10,000 units/day) is enough for heavy use. Results are cached locally so repeat visits cost no quota.
