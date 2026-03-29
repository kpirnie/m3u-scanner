// Package server provides the HTTP server that serves the M3U playlists
// and supporting endpoints. Playlists are held in memory under an RWMutex
// so scans can update them without any disk I/O on the serving path.
package server

import (
	"encoding/json"
	"log"
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
}

// New creates a Server.
func New(addr, playlistName string, ffprobeAvailable bool) *Server {
	return &Server{
		addr:         addr,
		playlistName: playlistName,
		ffprobe:      ffprobeAvailable,
		playlists:    make(map[string][]byte),
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

// ListenAndServe starts the HTTP server. Blocks until the server exits.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/rescan", s.handleRescan)
	mux.HandleFunc("/", s.handlePlaylist)
	mux.Handle("/media/", http.FileServer(http.Dir("/")))

	log.Printf("[server] listening on %s", s.addr)
	log.Printf("[server] all media  → http://<host>%s/all.m3u", s.addr)
	log.Printf("[server] per-type   → http://<host>%s/{movies,music,shows,images}.m3u", s.addr)

	return http.ListenAndServe(s.addr, mux)
}

// RescanFunc is the callback the /rescan endpoint invokes.
// Set it after construction.
var RescanFunc func()

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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if RescanFunc != nil {
		go RescanFunc()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"rescan triggered"}`))
	} else {
		http.Error(w, "rescan not configured", http.StatusInternalServerError)
	}
}
