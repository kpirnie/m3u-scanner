// Package writer builds extended M3U playlists as byte slices held in memory.
package writer

import (
	"fmt"
	"strings"

	"github.com/kpirnie/m3u-scanner/internal/models"
)

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
	attrs := buildAttrs(e)
	extinf := fmt.Sprintf("#EXTINF:%d %s,%s", e.Duration, attrs, e.Display)
	return extinf + "\n" + formatPath(e.Path, baseURL) + "\n"
}

func buildAttrs(e *models.MediaEntry) string {
	type kv struct{ k, v string }
	var pairs []kv

	add := func(k, v string) {
		if v != "" {
			pairs = append(pairs, kv{k, v})
		}
	}

	add("tvg-name", e.TVGName)
	add("group-title", e.GroupTitle)

	switch e.MediaType {
	case "music":
		add("tvg-artist", e.Artist)
		add("tvg-album", e.Album)
		add("tvg-genre", e.Genre)
		add("tvg-year", e.Year)
	case "shows":
		add("tvg-show", e.Series)
		if e.Season > 0 {
			add("tvg-season", fmt.Sprintf("%d", e.Season))
		}
		if e.Episode > 0 {
			add("tvg-episode", fmt.Sprintf("%d", e.Episode))
		}
		add("tvg-year", e.Year)
	case "movies":
		add("tvg-year", e.Year)
	}

	var parts []string
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, p.k, escape(p.v)))
	}
	return strings.Join(parts, " ")
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
