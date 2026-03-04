package tui

import (
	"context"
	"os/exec"
	"runtime"

	tea "charm.land/bubbletea/v2"
	"github.com/nroyalty/yt-browse/internal/cache"
	"github.com/nroyalty/yt-browse/internal/recent"
	"github.com/nroyalty/yt-browse/internal/youtube"
)

func resolveChannelCmd(client *youtube.Client, store *cache.Store, input string, knownChannelID string) tea.Cmd {
	return func() tea.Msg {
		// If we know the channel ID (e.g. from recent list), try cache first
		if knownChannelID != "" {
			if cached, _ := store.GetChannel(knownChannelID); cached != nil {
				return channelResolvedMsg{channel: cached}
			}
		}

		ctx := context.Background()
		ch, err := client.ResolveChannel(ctx, input)
		if err != nil {
			return channelErrorMsg{err: err}
		}
		// Cache the channel
		_ = store.SetChannel(ch)
		return channelResolvedMsg{channel: ch}
	}
}

func fetchPlaylistsCmd(client *youtube.Client, store *cache.Store, channelID string, skipCache bool) tea.Cmd {
	return func() tea.Msg {
		if !skipCache {
			if cached, _ := store.GetPlaylists(channelID); cached != nil {
				return playlistsFetchedMsg{playlists: cached}
			}
		}

		ctx := context.Background()
		playlists, err := client.FetchPlaylists(ctx, channelID)
		if err != nil {
			return playlistsErrorMsg{err: err}
		}

		_ = store.SetPlaylists(channelID, playlists)
		return playlistsFetchedMsg{playlists: playlists}
	}
}

func fetchVideosCmd(client *youtube.Client, store *cache.Store, uploadsPlaylistID, channelID string, gen int, ctx context.Context, progressCh chan<- videoLoadingMsg, skipCache bool) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)

		if !skipCache {
			if cached, _ := store.GetVideos(channelID); cached != nil {
				return videosFetchedMsg{videos: cached, gen: gen}
			}
		}

		videos, err := client.FetchVideos(ctx, uploadsPlaylistID, channelID, func(total, loaded int) {
			progressCh <- videoLoadingMsg{total: total, loaded: loaded, gen: gen}
		})
		if err != nil {
			return videosErrorMsg{err: err, gen: gen}
		}
		_ = store.SetVideos(channelID, videos)
		return videosFetchedMsg{videos: videos, gen: gen}
	}
}

func listenForVideoProgress(ch <-chan videoLoadingMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func fetchPlaylistVideosCmd(client *youtube.Client, store *cache.Store, playlistID, channelID string, skipCache bool) tea.Cmd {
	return func() tea.Msg {
		if !skipCache {
			if cached, _ := store.GetPlaylistVideos(channelID, playlistID); cached != nil {
				return playlistVideosFetchedMsg{videos: cached}
			}
		}

		ctx := context.Background()
		videos, err := client.FetchVideos(ctx, playlistID, channelID, nil)
		if err != nil {
			return playlistVideosErrorMsg{err: err}
		}
		_ = store.SetPlaylistVideos(channelID, playlistID, videos)
		return playlistVideosFetchedMsg{videos: videos}
	}
}

func loadRecentCmd(store *recent.Store) tea.Cmd {
	return func() tea.Msg {
		return recentChannelsLoadedMsg{entries: store.Load()}
	}
}

func saveRecentCmd(store *recent.Store, ch *youtube.Channel) tea.Cmd {
	return func() tea.Msg {
		store.Add(ch.ID, ch.Title, ch.Handle)
		return nil
	}
}

func removeRecentCmd(store *recent.Store, id string) tea.Cmd {
	return func() tea.Msg {
		store.Remove(id)
		return recentChannelRemovedMsg{}
	}
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Start()
		return nil
	}
}
