// Package models defines the core MediaEntry type used throughout the pipeline.
package models

import (
	"fmt"
	"strings"
)

// Person represents a credited individual attached to a media entry.
type Person struct {
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

// MediaEntry holds all metadata for a single media file, derived primarily
// from the folder structure and enriched by tag/ffprobe libraries.
type MediaEntry struct {
	// ── Required ──────────────────────────────────────────────────────────────
	Path       string `json:"path"`
	MediaType  string `json:"media_type"`
	GroupTitle string `json:"group_title"`
	TVGName    string `json:"tvg_name"`
	Display    string `json:"display"`

	// ── Common ────────────────────────────────────────────────────────────────
	Duration int    `json:"duration"`
	Year     string `json:"year"`

	// ── Music ─────────────────────────────────────────────────────────────────
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Disc   int    `json:"disc"`
	Track  int    `json:"track"`

	// ── Shows ─────────────────────────────────────────────────────────────────
	Series       string `json:"series"`
	Season       int    `json:"season"`
	Episode      int    `json:"episode"`
	EpisodeTitle string `json:"episode_title"`

	// ── Extended metadata ─────────────────────────────────────────────────────
	// Sourced from NFO sidecars; all optional.
	Title        string   `json:"title"`
	SortTitle    string   `json:"sort_title"`
	Plot         string   `json:"plot"`
	Tagline      string   `json:"tagline"`
	Poster       string   `json:"poster"`
	Fanart       string   `json:"fanart"`
	Rating       float64  `json:"rating"`
	CriticRating int      `json:"critic_rating"`
	MPAA         string   `json:"mpaa"`
	Country      string   `json:"country"`
	Premiered    string   `json:"premiered"`
	IMDBID       string   `json:"imdb_id"`
	TMDBID       string   `json:"tmdb_id"`
	TVDBID       string   `json:"tvdb_id"`
	Collection   string   `json:"collection"`
	Genres       []string `json:"genres"`
	Studios      []string `json:"studios"`
	Tags         []string `json:"tags"`
	Directors    []string `json:"directors"`
	Writers      []string `json:"writers"`
	Cast         []Person `json:"cast"`

	// ── Sort ──────────────────────────────────────────────────────────────────
	sortKey string
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
