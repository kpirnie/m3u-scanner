package meta

import "m3u-scanner/internal/models"

// Enrich dispatches to the correct enrichment path for the entry's media type.
// parseNFO controls whether .nfo sidecars and sibling artwork are read.
//
// Enrichment is strictly additive — path-derived fields are never overwritten.
func Enrich(e *models.MediaEntry, parseNFO bool) {
	switch e.MediaType {
	case "music":
		EnrichAudio(e)
		if parseNFO {
			EnrichNFO(e)
		}
	case "movies", "shows":
		EnrichVideo(e)
		if parseNFO {
			EnrichNFO(e)
		}
	case "images":
		EnrichImage(e)
	}
}
