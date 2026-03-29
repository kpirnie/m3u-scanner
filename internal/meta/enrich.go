package meta

import "github.com/kpirnie/m3u-scanner/internal/models"

// Enrich dispatches to the correct enrichment path for the entry's media type.
// parseNFO controls whether Jellyfin .nfo sidecars are read for show entries.
//
// Enrichment is strictly additive — path-derived fields are never overwritten.
func Enrich(e *models.MediaEntry, parseNFO bool) {
	switch e.MediaType {
	case "music":
		EnrichAudio(e)
	case "movies", "shows":
		EnrichVideo(e)
		if e.MediaType == "shows" && parseNFO {
			EnrichNFO(e)
		}
	case "images":
		EnrichImage(e)
	}
}
