package meta

import (
	"log"

	"github.com/kpirnie/m3u-scanner/internal/models"
	"go.senan.xyz/taglib"
)

// EnrichAudio populates Duration and Year on a music entry.
//
// Priority:
//  1. ffprobe (if available) for duration — most accurate across all formats
//  2. go-taglib ReadProperties — pure Go fallback, covers all common audio formats
//  3. go-taglib ReadTags — year only, when not already set
func EnrichAudio(e *models.MediaEntry) {
	// ── Duration ──────────────────────────────────────────────────────────────
	if ffprobeAvailable {
		if dur, err := DurationViaFFProbe(e.Path); err == nil {
			e.Duration = dur
		} else {
			log.Printf("[meta/audio] ffprobe failed for %s: %v — falling back to taglib", e.Path, err)
			taglibDuration(e)
		}
	} else {
		taglibDuration(e)
	}

	// ── Year ──────────────────────────────────────────────────────────────────
	if e.Year == "" {
		taglibYear(e)
	}
}

func taglibDuration(e *models.MediaEntry) {
	props, err := taglib.ReadProperties(e.Path)
	if err != nil {
		log.Printf("[meta/audio] taglib ReadProperties failed for %s: %v", e.Path, err)
		return
	}
	e.Duration = int(props.Length.Seconds())
}

func taglibYear(e *models.MediaEntry) {
	tags, err := taglib.ReadTags(e.Path)
	if err != nil {
		return
	}
	if years := tags["YEAR"]; len(years) > 0 && years[0] != "" {
		// Year tags are often "YYYY" or "YYYY-MM-DD" — take the first 4 chars
		y := years[0]
		if len(y) >= 4 {
			e.Year = y[:4]
		} else {
			e.Year = y
		}
	}
}
