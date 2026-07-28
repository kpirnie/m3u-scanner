package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"m3u-scanner/internal/meta"
	"m3u-scanner/internal/models"
)

// defaultLimit bounds an unqualified /api/entries response.
const defaultLimit = 200

// entrySummary is the compact projection returned by the list endpoint.
type entrySummary struct {
	ID        string `json:"id"`
	MediaType string `json:"media_type"`
	Display   string `json:"display"`
	Title     string `json:"title"`
	Year      string `json:"year"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Series    string `json:"series"`
	Season    int    `json:"season"`
	Episode   int    `json:"episode"`
	Poster    string `json:"poster"`
	Fanart    string `json:"fanart"`
	HasNFO    bool   `json:"has_nfo"`
}

// entryEdit carries the user-editable subset of a MediaEntry. Fields absent
// from this struct cannot be changed through the API — path, media type and
// group are derived from the filesystem and must stay authoritative.
type entryEdit struct {
	Title        string          `json:"title"`
	SortTitle    string          `json:"sort_title"`
	Plot         string          `json:"plot"`
	Tagline      string          `json:"tagline"`
	Year         string          `json:"year"`
	Premiered    string          `json:"premiered"`
	Rating       float64         `json:"rating"`
	CriticRating int             `json:"critic_rating"`
	MPAA         string          `json:"mpaa"`
	Country      string          `json:"country"`
	Collection   string          `json:"collection"`
	IMDBID       string          `json:"imdb_id"`
	TMDBID       string          `json:"tmdb_id"`
	TVDBID       string          `json:"tvdb_id"`
	Poster       string          `json:"poster"`
	Fanart       string          `json:"fanart"`
	Genres       []string        `json:"genres"`
	Studios      []string        `json:"studios"`
	Tags         []string        `json:"tags"`
	Directors    []string        `json:"directors"`
	Writers      []string        `json:"writers"`
	Cast         []models.Person `json:"cast"`
	Artist       string          `json:"artist"`
	Album        string          `json:"album"`
	Disc         int             `json:"disc"`
	Track        int             `json:"track"`
	Series       string          `json:"series"`
	Season       int             `json:"season"`
	Episode      int             `json:"episode"`
	EpisodeTitle string          `json:"episode_title"`
}

func (s *Server) handleEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mediaType := strings.ToLower(strings.TrimSpace(q.Get("type")))
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))

	limit := defaultLimit
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 1000 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v > 0 {
		offset = v
	}

	s.mu.RLock()
	all := s.entries
	s.mu.RUnlock()

	var matched []*models.MediaEntry
	for _, e := range all {
		if mediaType != "" && e.MediaType != mediaType {
			continue
		}
		if search != "" && !entryMatches(e, search) {
			continue
		}
		matched = append(matched, e)
	}

	total := len(matched)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]entrySummary, 0, end-offset)
	for _, e := range matched[offset:end] {
		items = append(items, summarise(e))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":  total,
		"offset": offset,
		"limit":  limit,
		"items":  items,
	})
}

func (s *Server) handleEntry(w http.ResponseWriter, r *http.Request) {
	e := s.lookupEntry(r.PathValue("id"))
	if e == nil {
		writeErr(w, http.StatusNotFound, "entry not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":       EntryID(e.Path),
		"entry":    e,
		"nfo_path": meta.NFOPath(e),
	})
}

func (s *Server) handleEntrySave(w http.ResponseWriter, r *http.Request) {
	e := s.lookupEntry(r.PathValue("id"))
	if e == nil {
		writeErr(w, http.StatusNotFound, "entry not found")
		return
	}

	var edit entryEdit
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&edit); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Apply onto a copy so a failed sidecar write leaves the served entry intact
	updated := *e
	applyEdit(&updated, &edit)

	nfoPath, err := meta.WriteNFO(&updated)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "write nfo: "+err.Error())
		return
	}

	if RescanPathFunc != nil {
		if err := RescanPathFunc(updated.Path); err != nil {
			writeErr(w, http.StatusInternalServerError, "rescan: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "saved",
		"nfo_path": nfoPath,
	})
}

func (s *Server) handleEntryRescan(w http.ResponseWriter, r *http.Request) {
	e := s.lookupEntry(r.PathValue("id"))
	if e == nil {
		writeErr(w, http.StatusNotFound, "entry not found")
		return
	}
	if RescanPathFunc == nil {
		writeErr(w, http.StatusInternalServerError, "rescan not configured")
		return
	}
	if err := RescanPathFunc(e.Path); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "rescanned"})
}

func (s *Server) handleEntryTags(w http.ResponseWriter, r *http.Request) {
	e := s.lookupEntry(r.PathValue("id"))
	if e == nil {
		writeErr(w, http.StatusNotFound, "entry not found")
		return
	}
	if e.MediaType != "music" {
		writeErr(w, http.StatusBadRequest, "embedded tags are only supported for music")
		return
	}

	if err := meta.WriteAudioTags(e); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	coverErr := ""
	if err := meta.WriteAudioCover(e); err != nil {
		coverErr = err.Error()
	}

	if RescanPathFunc != nil {
		if err := RescanPathFunc(e.Path); err != nil {
			writeErr(w, http.StatusInternalServerError, "rescan: "+err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "tags written",
		"cover_note": coverErr,
	})
}

// ── Private ───────────────────────────────────────────────────────────────────

func applyEdit(e *models.MediaEntry, in *entryEdit) {
	e.Title = in.Title
	e.SortTitle = in.SortTitle
	e.Plot = in.Plot
	e.Tagline = in.Tagline
	e.Year = in.Year
	e.Premiered = in.Premiered
	e.Rating = in.Rating
	e.CriticRating = in.CriticRating
	e.MPAA = in.MPAA
	e.Country = in.Country
	e.Collection = in.Collection
	e.IMDBID = in.IMDBID
	e.TMDBID = in.TMDBID
	e.TVDBID = in.TVDBID
	e.Poster = strings.TrimSpace(in.Poster)
	e.Fanart = strings.TrimSpace(in.Fanart)
	e.Genres = cleanList(in.Genres)
	e.Studios = cleanList(in.Studios)
	e.Tags = cleanList(in.Tags)
	e.Directors = cleanList(in.Directors)
	e.Writers = cleanList(in.Writers)

	e.Cast = nil
	for _, p := range in.Cast {
		if name := strings.TrimSpace(p.Name); name != "" {
			e.Cast = append(e.Cast, models.Person{Name: name, Role: strings.TrimSpace(p.Role)})
		}
	}

	switch e.MediaType {
	case "music":
		e.Artist = in.Artist
		e.Album = in.Album
		e.Disc = in.Disc
		e.Track = in.Track
	case "shows":
		e.Series = in.Series
		e.Season = in.Season
		e.Episode = in.Episode
		e.EpisodeTitle = in.EpisodeTitle
	}
}

func cleanList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func entryMatches(e *models.MediaEntry, needle string) bool {
	for _, f := range []string{e.Display, e.Title, e.Artist, e.Album, e.Series, e.EpisodeTitle} {
		if f != "" && strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}

func summarise(e *models.MediaEntry) entrySummary {
	return entrySummary{
		ID:        EntryID(e.Path),
		MediaType: e.MediaType,
		Display:   e.Display,
		Title:     e.Title,
		Year:      e.Year,
		Artist:    e.Artist,
		Album:     e.Album,
		Series:    e.Series,
		Season:    e.Season,
		Episode:   e.Episode,
		Poster:    artURL(e.Poster),
		Fanart:    artURL(e.Fanart),
		HasNFO:    e.Title != "" || e.Plot != "",
	}
}

// artURL returns a browser-resolvable location for an artwork value. Remote
// URLs pass through; local paths are already absolute and are served verbatim
// by the /media/ file handler.
func artURL(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return path
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
