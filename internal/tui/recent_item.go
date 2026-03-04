package tui

import (
	"github.com/nroyalty/yt-browse/internal/recent"
)

// RecentItem wraps recent.Entry and implements list.DefaultItem.
type RecentItem struct {
	entry recent.Entry
}

func (r RecentItem) Title() string       { return r.entry.Handle }
func (r RecentItem) Description() string { return r.entry.Title }
func (r RecentItem) FilterValue() string { return r.entry.Handle + " " + r.entry.Title }
