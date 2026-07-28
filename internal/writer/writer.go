// Package writer builds extended M3U playlists as byte slices held in memory.
package writer

import (
	"fmt"
	"strconv"
	"strings"

	"m3u-scanner/internal/models"
)

// maxAttrLen bounds any single attribute value in runes. Plot synopses are
// unbounded in a .nfo and some players truncate or reject very long #EXTINF
// lines outright.
const maxAttrLen = 600

// maxCast bounds how many credited actors are written to tvg-cast.
const maxCast = 8

// BuildAll returns a map of playlist name -> M3U bytes.
// Keys are each media type present + "all" containing everything.
func BuildAll(entries []*models.MediaEntry, baseURL string) map[string][]byte {
	byType := make(map[string][]*models.MediaEntry)
	for _, e := range entries {
		byType[e.MediaType] = append(byType[e.MediaType], e)
	}

	out := make(map[string][]byte)
	out["all"] = Build(entries, baseURL)
	for t, te := range byType {
		out[t] = Build(te, baseURL)
	}
	return out
}

// Build converts a sorted slice of entries into a complete #EXTM3U playlist.
func Build(entries []*models.MediaEntry, baseURL string) []byte {
	var b strings.Builder
	b.WriteString("#EXTM3U\n\n")
	for _, e := range entries {
		b.WriteString(formatEntry(e, baseURL))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func formatEntry(e *models.MediaEntry, baseURL string) string {
	attrs := buildAttrs(e, baseURL)
	extinf := fmt.Sprintf("#EXTINF:%d %s,%s", e.Duration, attrs, sanitize(e.Display))
	return extinf + "\n" + formatPath(e.Path, baseURL) + "\n"
}

func buildAttrs(e *models.MediaEntry, baseURL string) string {
	type kv struct{ k, v string }
	var pairs []kv

	add := func(k, v string) {
		if v = sanitize(v); v != "" {
			pairs = append(pairs, kv{k, v})
		}
	}

	// Standard attributes first — lenient parsers that stop at the first
	// unrecognised key still receive the ones they actually render.
	add("tvg-id", tvgID(e))
	add("tvg-name", e.TVGName)
	add("group-title", e.GroupTitle)
	if e.Poster != "" {
		add("tvg-logo", formatPath(e.Poster, baseURL))
	}

	switch e.MediaType {
	case "music":
		add("tvg-artist", e.Artist)
		add("tvg-album", e.Album)
		if e.Disc > 0 {
			add("tvg-disc", strconv.Itoa(e.Disc))
		}
		if e.Track > 0 {
			add("tvg-track", strconv.Itoa(e.Track))
		}
	case "shows":
		add("tvg-show", e.Series)
		if e.Season > 0 {
			add("tvg-season", strconv.Itoa(e.Season))
		}
		if e.Episode > 0 {
			add("tvg-episode", strconv.Itoa(e.Episode))
		}
		add("tvg-episode-title", e.EpisodeTitle)
	}

	add("tvg-genre", strings.Join(e.Genres, ", "))
	add("tvg-year", e.Year)

	// Non-standard extended attributes. Ignored by most players today; they
	// cost one line each and are what a future Xtream layer reads from.
	add("tvg-title", e.Title)
	add("tvg-tagline", e.Tagline)
	add("tvg-plot", e.Plot)
	if e.Rating > 0 {
		add("tvg-rating", strconv.FormatFloat(e.Rating, 'f', 1, 64))
	}
	if e.CriticRating > 0 {
		add("tvg-critic-rating", strconv.Itoa(e.CriticRating))
	}
	add("tvg-mpaa", e.MPAA)
	add("tvg-director", strings.Join(e.Directors, ", "))
	add("tvg-cast", castNames(e))
	add("tvg-studio", strings.Join(e.Studios, ", "))
	add("tvg-country", e.Country)
	add("tvg-premiered", e.Premiered)
	add("tvg-collection", e.Collection)
	if e.Fanart != "" {
		add("tvg-fanart", formatPath(e.Fanart, baseURL))
	}

	var parts []string
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, p.k, escape(p.v)))
	}
	return strings.Join(parts, " ")
}

// tvgID returns a stable external identifier for the entry, or an empty string
// when none is known. Identifiers are never synthesised: a fabricated tvg-id
// can collide with a real EPG channel id in a merged playlist.
func tvgID(e *models.MediaEntry) string {
	switch {
	case e.IMDBID != "":
		return e.IMDBID
	case e.TMDBID != "":
		return "tmdb-" + e.TMDBID
	case e.TVDBID != "":
		return "tvdb-" + e.TVDBID
	}
	return ""
}

func castNames(e *models.MediaEntry) string {
	if len(e.Cast) == 0 {
		return ""
	}
	n := len(e.Cast)
	if n > maxCast {
		n = maxCast
	}
	names := make([]string, 0, n)
	for _, p := range e.Cast[:n] {
		if p.Name != "" {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ", ")
}

// sanitize collapses all whitespace to single spaces and bounds the result.
// An #EXTINF line is newline-terminated, so an embedded newline in a plot or
// tagline would truncate the attribute list and corrupt every entry that
// follows it in the playlist.
func sanitize(s string) string {
	if s == "" {
		return ""
	}

	s = strings.Join(strings.Fields(s), " ")

	r := []rune(s)
	if len(r) > maxAttrLen {
		s = strings.TrimSpace(string(r[:maxAttrLen])) + "…"
	}
	return s
}

func formatPath(absPath, baseURL string) string {
	if baseURL == "" {
		return absPath
	}
	return baseURL + absPath
}

func escape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
