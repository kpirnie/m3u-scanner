// Package models defines the core MediaEntry type used throughout the pipeline.
package models

import (
	"fmt"
	"strings"
)

// MediaEntry holds all metadata for a single media file, derived primarily
// from the folder structure and enriched by tag/ffprobe libraries.
type MediaEntry struct {
	// ── Required ──────────────────────────────────────────────────────────────
	Path       string // absolute filesystem path
	MediaType  string // "music" | "images" | "movies" | "shows"
	GroupTitle string // written as group-title= in M3U (same as MediaType)
	TVGName    string // written as tvg-name= in M3U
	Display    string // label after the comma on #EXTINF

	// ── Common ────────────────────────────────────────────────────────────────
	Duration int    // seconds; -1 = unknown
	Year     string // optional

	// ── Music ─────────────────────────────────────────────────────────────────
	Artist string
	Genre  string
	Album  string
	Disc   int // 0 = not set
	Track  int // 0 = not set

	// ── Shows ─────────────────────────────────────────────────────────────────
	Series       string
	Season       int // 0 = not set
	Episode      int // 0 = not set
	EpisodeTitle string

	// ── Sort ──────────────────────────────────────────────────────────────────
	sortKey string // computed once, used for stable ordering
}

// SortKey returns a pre-computed string that produces correct ordering when
// entries are sorted lexicographically:
//
//	music  → group / artist / album / disc (zero-padded) / track (zero-padded)
//	shows  → group / series / season (zero-padded) / episode (zero-padded)
//	movies → group / display title
//	images → group / display title
func (e *MediaEntry) SortKey() string {
	if e.sortKey != "" {
		return e.sortKey
	}
	switch e.MediaType {
	case "music":
		e.sortKey = strings.Join([]string{
			e.GroupTitle,
			strings.ToLower(e.Artist),
			strings.ToLower(e.Album),
			fmt.Sprintf("%04d", e.Disc),
			fmt.Sprintf("%04d", e.Track),
			strings.ToLower(e.Display),
		}, "\x00")
	case "shows":
		e.sortKey = strings.Join([]string{
			e.GroupTitle,
			strings.ToLower(e.Series),
			fmt.Sprintf("%04d", e.Season),
			fmt.Sprintf("%04d", e.Episode),
		}, "\x00")
	default:
		e.sortKey = strings.Join([]string{
			e.GroupTitle,
			strings.ToLower(e.Display),
		}, "\x00")
	}
	return e.sortKey
}
