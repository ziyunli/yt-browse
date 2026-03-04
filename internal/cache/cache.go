package cache

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/nroyalty/yt-browse/internal/youtube"
)

const maxCacheBytes int64 = 200 * 1024 * 1024 // 200 MB

type entry[T any] struct {
	FetchedAt time.Time `json:"fetched_at"`
	Data      T         `json:"data"`
}

type Store struct {
	dir string
	ttl time.Duration
}

func NewStore(dir string, ttl time.Duration) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache dir %s: %w", dir, err)
	}
	return &Store{dir: dir, ttl: ttl}, nil
}

func (s *Store) channelDir(channelID string) string {
	return filepath.Join(s.dir, channelID)
}

func (s *Store) GetChannel(channelID string) (*youtube.Channel, error) {
	var e entry[youtube.Channel]
	if err := s.load(filepath.Join(s.channelDir(channelID), "channel.json"), &e); err != nil {
		return nil, nil // cache miss
	}
	if s.expired(e.FetchedAt) {
		return nil, nil
	}
	return &e.Data, nil
}

func (s *Store) SetChannel(ch *youtube.Channel) error {
	e := entry[youtube.Channel]{FetchedAt: time.Now(), Data: *ch}
	if err := s.save(filepath.Join(s.channelDir(ch.ID), "channel.json"), e); err != nil {
		return err
	}
	// Drop a handle marker file so humans can identify channel directories
	if ch.Handle != "" {
		_ = os.WriteFile(filepath.Join(s.channelDir(ch.ID), ch.Handle), nil, 0644)
	}
	return nil
}

func (s *Store) GetPlaylists(channelID string) ([]youtube.Playlist, error) {
	var e entry[[]youtube.Playlist]
	if err := s.load(filepath.Join(s.channelDir(channelID), "playlists.json"), &e); err != nil {
		return nil, nil
	}
	if s.expired(e.FetchedAt) {
		return nil, nil
	}
	return e.Data, nil
}

func (s *Store) SetPlaylists(channelID string, playlists []youtube.Playlist) error {
	e := entry[[]youtube.Playlist]{FetchedAt: time.Now(), Data: playlists}
	return s.save(filepath.Join(s.channelDir(channelID), "playlists.json"), e)
}

func (s *Store) GetVideos(channelID string) ([]youtube.Video, error) {
	var e entry[[]youtube.Video]
	if err := s.load(filepath.Join(s.channelDir(channelID), "videos.json"), &e); err != nil {
		return nil, nil
	}
	if s.expired(e.FetchedAt) {
		return nil, nil
	}
	return e.Data, nil
}

func (s *Store) SetVideos(channelID string, videos []youtube.Video) error {
	e := entry[[]youtube.Video]{FetchedAt: time.Now(), Data: videos}
	return s.save(filepath.Join(s.channelDir(channelID), "videos.json"), e)
}

func (s *Store) GetPlaylistVideos(channelID, playlistID string) ([]youtube.Video, error) {
	var e entry[[]youtube.Video]
	filename := fmt.Sprintf("playlist_%s_videos.json", playlistID)
	if err := s.load(filepath.Join(s.channelDir(channelID), filename), &e); err != nil {
		return nil, nil
	}
	if s.expired(e.FetchedAt) {
		return nil, nil
	}
	return e.Data, nil
}

func (s *Store) SetPlaylistVideos(channelID, playlistID string, videos []youtube.Video) error {
	filename := fmt.Sprintf("playlist_%s_videos.json", playlistID)
	e := entry[[]youtube.Video]{FetchedAt: time.Now(), Data: videos}
	return s.save(filepath.Join(s.channelDir(channelID), filename), e)
}

// CleanExpired removes channel directories where all cache files are expired.
func (s *Store) CleanExpired() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		chDir := filepath.Join(s.dir, e.Name())
		if s.allExpired(chDir) {
			os.RemoveAll(chDir)
		}
	}
	return nil
}

// PurgeOverSize removes the least recently used channel caches until the
// total cache size is under maxCacheBytes. Channels are ranked by the most
// recent mod time of any file in their directory.
func (s *Store) PurgeOverSize() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type channelCache struct {
		name      string
		size      int64
		newestMod time.Time
	}

	var channels []channelCache
	var totalSize int64

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cc := channelCache{name: e.Name()}
		chDir := filepath.Join(s.dir, e.Name())
		files, err := os.ReadDir(chDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			info, err := f.Info()
			if err != nil {
				continue
			}
			cc.size += info.Size()
			if info.ModTime().After(cc.newestMod) {
				cc.newestMod = info.ModTime()
			}
		}
		totalSize += cc.size
		channels = append(channels, cc)
	}

	if totalSize <= maxCacheBytes {
		return nil
	}

	// Sort oldest first so we evict least recently used
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].newestMod.Before(channels[j].newestMod)
	})

	for _, cc := range channels {
		if totalSize <= maxCacheBytes {
			break
		}
		os.RemoveAll(filepath.Join(s.dir, cc.name))
		totalSize -= cc.size
	}
	return nil
}

func (s *Store) allExpired(dir string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < s.ttl {
			return false
		}
	}
	return true
}

func (s *Store) expired(fetchedAt time.Time) bool {
	return time.Since(fetchedAt) > s.ttl
}

func (s *Store) load(path string, v any) error {
	// Try compressed file first, fall back to uncompressed for backwards compat
	gzPath := path + ".gz"
	if data, err := os.ReadFile(gzPath); err == nil {
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer gr.Close()
		decoded, err := io.ReadAll(gr)
		if err != nil {
			return err
		}
		return json.Unmarshal(decoded, v)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (s *Store) save(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return err
	}
	if _, err := gw.Write(data); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	gzPath := path + ".gz"
	if err := os.WriteFile(gzPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	// Clean up old uncompressed file if it exists
	os.Remove(path)
	return nil
}
