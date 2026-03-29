package meta

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kpirnie/m3u-scanner/internal/models"
)

// nfoEpisode mirrors the relevant fields from a Jellyfin episode .nfo file.
type nfoEpisode struct {
	Title     string `xml:"title"`
	Aired     string `xml:"aired"`
	Premiered string `xml:"premiered"`
	Year      string `xml:"year"`
}

// EnrichNFO looks for a sibling <filename>.nfo file and, if found, parses it
// to fill in EpisodeTitle (and Year if missing). It then rebuilds Display and
// TVGName to include the episode title.
//
// Silently no-ops if the .nfo is absent or malformed.
func EnrichNFO(e *models.MediaEntry) {
	nfoPath := strings.TrimSuffix(e.Path, filepath.Ext(e.Path)) + ".nfo"

	data, err := os.ReadFile(nfoPath)
	if err != nil {
		return // .nfo not present — this is the common case, not an error
	}

	var ep nfoEpisode
	if err := xml.Unmarshal(data, &ep); err != nil {
		return
	}

	// ── Episode title ─────────────────────────────────────────────────────────
	if e.EpisodeTitle == "" && ep.Title != "" {
		e.EpisodeTitle = strings.TrimSpace(ep.Title)
		rebuildShowDisplay(e)
	}

	// ── Year ──────────────────────────────────────────────────────────────────
	if e.Year == "" {
		for _, v := range []string{ep.Aired, ep.Premiered, ep.Year} {
			if len(v) >= 4 {
				e.Year = v[:4]
				break
			}
		}
	}
}

// rebuildShowDisplay reconstructs Display and TVGName after episode title enrichment.
func rebuildShowDisplay(e *models.MediaEntry) {
	epStr := ""
	if e.Season > 0 && e.Episode > 0 {
		epStr = fmt.Sprintf("S%02dE%02d", e.Season, e.Episode)
	}

	if epStr != "" && e.EpisodeTitle != "" {
		e.Display = fmt.Sprintf("%s - %s - %s", e.Series, epStr, e.EpisodeTitle)
	} else if epStr != "" {
		e.Display = fmt.Sprintf("%s - %s", e.Series, epStr)
	} else {
		e.Display = e.Series
	}
	e.TVGName = e.Display
}
