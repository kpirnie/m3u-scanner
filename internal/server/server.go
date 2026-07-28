// Package server provides the HTTP server that serves the M3U playlists
// and supporting endpoints. Playlists are held in memory under an RWMutex
// so scans can update them without any disk I/O on the serving path.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"m3u-scanner/internal/models"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Server holds the in-memory playlists and serves them over HTTP.
type Server struct {
	addr         string
	playlistName string

	mu         sync.RWMutex
	playlists  map[string][]byte // "all" + one key per media type
	entryCount int
	lastScan   time.Time
	ffprobe    bool

	entries   []*models.MediaEntry
	entryByID map[string]*models.MediaEntry
}

// New creates a Server.
func New(addr, playlistName string, ffprobeAvailable bool) *Server {
	return &Server{
		addr:         addr,
		playlistName: playlistName,
		ffprobe:      ffprobeAvailable,
		playlists:    make(map[string][]byte),
		entryByID:    make(map[string]*models.MediaEntry),
	}
}

// UpdatePlaylist atomically replaces all in-memory playlists.
func (s *Server) UpdatePlaylist(playlists map[string][]byte, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlists = playlists
	s.entryCount = count
	s.lastScan = time.Now()

	total := 0
	for _, b := range playlists {
		total += len(b)
	}
	log.Printf("[server] playlists updated: %d entries, %d playlists, %d bytes total",
		count, len(playlists), total)
}

// SetEntries atomically replaces the in-memory entry snapshot the admin API
// reads from, and rebuilds the id index used to resolve edit requests.
func (s *Server) SetEntries(entries []*models.MediaEntry) {
	index := make(map[string]*models.MediaEntry, len(entries))
	for _, e := range entries {
		index[EntryID(e.Path)] = e
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = entries
	s.entryByID = index
}

// EntryID returns the opaque, URL-safe identifier for a media path. Edits are
// addressed by this id rather than by a caller-supplied path, so a request can
// only ever resolve to a file already present in the scanned index.
func EntryID(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:12])
}

// lookupEntry resolves an entry id to its entry, or nil when unknown.
func (s *Server) lookupEntry(id string) *models.MediaEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entryByID[id]
}

// ListenAndServe starts the HTTP server. Blocks until the server exits.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/entries", s.handleEntries)
	mux.HandleFunc("GET /api/entry/{id}", s.handleEntry)
	mux.HandleFunc("PUT /api/entry/{id}", s.handleEntrySave)
	mux.HandleFunc("POST /api/entry/{id}/rescan", s.handleEntryRescan)
	mux.HandleFunc("POST /api/rescan", s.handleRescan)
	mux.HandleFunc("POST /api/entry/{id}/tags", s.handleEntryTags)

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /rescan", s.handleRescan)
	mux.Handle("GET /media/", http.FileServer(http.Dir("/")))
	mux.Handle("GET /admin/", http.StripPrefix("/admin/", http.FileServer(adminAssets())))
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /", s.handlePlaylist)

	log.Printf("[server] listening on %s", s.addr)
	log.Printf("[server] all media  → http://<host>%s/all.m3u", s.addr)
	log.Printf("[server] per-type   → http://<host>%s/{movies,music,shows,images}.m3u", s.addr)
	log.Printf("[server] admin ui   → http://<host>%s/admin/", s.addr)

	return http.ListenAndServe(s.addr, mux)
}

// RescanFunc is the callback the /rescan endpoint invokes for a full rescan.
// Set it after construction.
var RescanFunc func()

// RescanTypesFunc is the callback invoked for a scoped rescan of specific
// media types. Set it after construction.
var RescanTypesFunc func([]string)

// RescanPathFunc is the callback invoked to re-parse and re-enrich a single
// file after its metadata has been edited. Set it after construction.
var RescanPathFunc func(string) error

// ScanTypes is the set of configured media types, used to validate the
// ?type= parameter on /rescan. Set it after construction.
var ScanTypes []string

// ── Handlers ──────────────────────────────────────────────────────────────────

func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	// Derive playlist key from URL path:
	//   /           → "all"
	//   /all.m3u    → "all"
	//   /movies.m3u → "movies"
	name := strings.TrimPrefix(r.URL.Path, "/")
	name = strings.TrimSuffix(name, ".m3u")
	if name == "" {
		name = "all"
	}

	s.mu.RLock()
	data := s.playlists[name]
	s.mu.RUnlock()

	if len(data) == 0 {
		if name == "all" {
			http.Error(w, "playlist not ready — scan in progress", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "playlist not found: "+name+".m3u", http.StatusNotFound)
		}
		return
	}

	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="`+name+`.m3u"`)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	count := s.entryCount
	lastScan := s.lastScan
	playlists := make([]string, 0, len(s.playlists))
	for k := range s.playlists {
		playlists = append(playlists, k+".m3u")
	}
	s.mu.RUnlock()

	payload := map[string]any{
		"status":            "ok",
		"entry_count":       count,
		"last_scan":         lastScan.Format(time.RFC3339),
		"ffprobe_available": s.ffprobe,
		"playlists":         playlists,
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	requested := r.URL.Query()["type"]

	if len(requested) == 0 {
		if RescanFunc == nil {
			writeErr(w, http.StatusInternalServerError, "rescan not configured")
			return
		}
		go RescanFunc()
		writeJSON(w, http.StatusOK, map[string]any{"status": "rescan triggered", "types": "all"})
		return
	}

	if RescanTypesFunc == nil {
		writeErr(w, http.StatusInternalServerError, "rescan not configured")
		return
	}

	var types []string
	for _, t := range requested {
		t = strings.ToLower(strings.TrimSpace(t))
		for _, valid := range ScanTypes {
			if t == valid {
				types = append(types, t)
				break
			}
		}
	}
	if len(types) == 0 {
		writeErr(w, http.StatusBadRequest,
			"no valid type= values (configured: "+strings.Join(ScanTypes, ", ")+")")
		return
	}

	go RescanTypesFunc(types)
	writeJSON(w, http.StatusOK, map[string]any{"status": "rescan triggered", "types": types})
}
