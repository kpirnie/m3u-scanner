// Package parser derives metadata from file paths.
// The folder structure is always the authoritative source for descriptive
// fields (artist, album, series, season, etc.).
// Tag libraries and ffprobe are enrichment only, handled in internal/meta.
package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kpirnie/m3u-scanner/internal/models"
)

// ── Compiled regexes ──────────────────────────────────────────────────────────

// Disc subfolder: "Disk 1", "Disc 2", "CD1", "CD 2", etc.
var discFolderRE = regexp.MustCompile(`(?i)^(?:Dis[ck]|CD)\s*(\d+)$`)

// Season folder: "Season 1", "Season 01", "Series 2", "S01"
var seasonFolderRE = regexp.MustCompile(`(?i)^(?:Season|Series|S)\s*(\d+)$`)

// Track filename:
//
//	"01 - Song Title"   or   "01. Song Title"
var trackRE = regexp.MustCompile(`^(\d+)\s*[-\.]\s*(.+)$`)

// Disc-track prefix: "1-01 Song Title"
var discTrackRE = regexp.MustCompile(`^(\d+)-(\d+)\s+(.+)$`)

// Episode: SxxExx (case-insensitive, anywhere in the string)
var episodeRE = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,2})`)

// Movie year: (YYYY) or .YYYY. anywhere in filename
var movieYearRE = regexp.MustCompile(`[\.\s\(](\d{4})[\.\s\)]`)

// ── Public parsers ────────────────────────────────────────────────────────────

// Music parses a music file using the known path structure:
//
//	<musicRoot> / Albums / <Genre> / <Artist> / <Album> / [Disk N /] <file>
func Music(filePath, musicRoot string) *models.MediaEntry {
	rel, _ := filepath.Rel(musicRoot, filePath)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	// parts[0] = "Albums" (skip), [1]=Genre, [2]=Artist, [3]=Album, [4]=Disk|File, [5]=File

	genre := safeGet(parts, 1, "Unknown Genre")
	artist := safeGet(parts, 2, "Unknown Artist")
	album := safeGet(parts, 3, "Unknown Album")

	disc := 0
	// detect optional disc subfolder at depth 4
	if len(parts) >= 6 {
		if m := discFolderRE.FindStringSubmatch(parts[4]); m != nil {
			disc, _ = strconv.Atoi(m[1])
		}
	}

	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	track := 0
	title := stem

	if m := discTrackRE.FindStringSubmatch(stem); m != nil {
		if disc == 0 {
			disc, _ = strconv.Atoi(m[1])
		}
		track, _ = strconv.Atoi(m[2])
		title = strings.TrimSpace(m[3])
	} else if m := trackRE.FindStringSubmatch(stem); m != nil {
		track, _ = strconv.Atoi(m[1])
		title = strings.TrimSpace(m[2])
	}

	display := fmt.Sprintf("%s - %s", artist, title)

	return &models.MediaEntry{
		Path:       filePath,
		MediaType:  "music",
		GroupTitle: "Music/" + genre + "/" + artist,
		TVGName:    display,
		Display:    display,
		Duration:   -1,
		Artist:     artist,
		Genre:      genre,
		Album:      album,
		Disc:       disc,
		Track:      track,
	}
}

// Show parses a show file using the known path structure:
//
//	<showsRoot> / <Show Name> / <Season N> / <file>
func Show(filePath, showsRoot string) *models.MediaEntry {
	rel, _ := filepath.Rel(showsRoot, filePath)
	parts := strings.Split(filepath.ToSlash(rel), "/")

	series := safeGet(parts, 0, "Unknown Show")
	season := 0
	if len(parts) > 1 {
		if m := seasonFolderRE.FindStringSubmatch(parts[1]); m != nil {
			season, _ = strconv.Atoi(m[1])
		}
	}

	// Extract SxxExx from filename
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	episode := 0
	if m := episodeRE.FindStringSubmatch(stem); m != nil {
		if season == 0 {
			season, _ = strconv.Atoi(m[1])
		}
		episode, _ = strconv.Atoi(m[2])
	}

	epStr := buildEpStr(season, episode, stem)
	display := fmt.Sprintf("%s - %s", series, epStr)

	return &models.MediaEntry{
		Path:       filePath,
		MediaType:  "shows",
		GroupTitle: "Shows/" + series,
		TVGName:    display,
		Display:    display,
		Duration:   -1,
		Series:     series,
		Season:     season,
		Episode:    episode,
	}
}

// Movie parses a movie file using the filename only.
// Title and optional year are extracted from the filename stem.
func Movie(filePath string) *models.MediaEntry {
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Grab year before cleaning the stem
	year := ""
	if m := movieYearRE.FindStringSubmatch(stem); m != nil {
		year = m[1]
	}

	title := cleanMovieTitle(stem)
	display := title
	if year != "" {
		display = fmt.Sprintf("%s (%s)", title, year)
	}

	return &models.MediaEntry{
		Path:       filePath,
		MediaType:  "movies",
		GroupTitle: "Movies",
		TVGName:    display,
		Display:    display,
		Duration:   -1,
		Year:       year,
	}
}

// Image parses an image file using the filename only.
func Image(filePath string) *models.MediaEntry {
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	title := cleanGenericTitle(stem)

	return &models.MediaEntry{
		Path:       filePath,
		MediaType:  "images",
		GroupTitle: "Images",
		TVGName:    title,
		Display:    title,
		Duration:   -1,
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func safeGet(parts []string, idx int, fallback string) string {
	if idx < len(parts) && parts[idx] != "" {
		return parts[idx]
	}
	return fallback
}

func buildEpStr(season, episode int, fallbackStem string) string {
	if season > 0 && episode > 0 {
		return fmt.Sprintf("S%02dE%02d", season, episode)
	}
	return fallbackStem
}

// cleanMovieTitle converts dot/underscore-separated filenames into title case
// and strips the year and common release-group suffixes.
func cleanMovieTitle(stem string) string {
	// Strip trailing year and everything after: "The.Matrix.1999.BluRay" → "The Matrix"
	cleaned := movieYearRE.ReplaceAllString(stem, " ")
	// Replace dots and underscores with spaces
	cleaned = strings.NewReplacer(".", " ", "_", " ").Replace(cleaned)
	// Collapse whitespace
	fields := strings.Fields(cleaned)
	// Title-case each word
	for i, f := range fields {
		if len(f) > 0 {
			fields[i] = strings.ToUpper(f[:1]) + f[1:]
		}
	}
	return strings.Join(fields, " ")
}

func cleanGenericTitle(stem string) string {
	cleaned := strings.NewReplacer("_", " ", ".", " ", "-", " ").Replace(stem)
	fields := strings.Fields(cleaned)
	for i, f := range fields {
		if len(f) > 0 {
			fields[i] = strings.ToUpper(f[:1]) + strings.ToLower(f[1:])
		}
	}
	return strings.Join(fields, " ")
}
