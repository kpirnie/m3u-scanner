package meta

import (
	"log"

	"github.com/kpirnie/m3u-scanner/internal/models"
)

// EnrichVideo populates Duration on a movie or show entry.
// ffprobe is the only source for video duration — if unavailable, stays -1.
func EnrichVideo(e *models.MediaEntry) {
	if !ffprobeAvailable {
		// Duration stays -1 — M3U spec allows this and all players handle it.
		return
	}
	dur, err := DurationViaFFProbe(e.Path)
	if err != nil {
		log.Printf("[meta/video] ffprobe failed for %s: %v", e.Path, err)
		return
	}
	e.Duration = dur
}
