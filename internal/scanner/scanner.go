// Package scanner walks the filesystem, resolves per-type roots, and builds
// a sorted slice of MediaEntry values ready for the M3U writer.
// It uses the cache package to skip unchanged files on incremental scans.
package scanner

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/kpirnie/m3u-scanner/internal/cache"
	"github.com/kpirnie/m3u-scanner/internal/extensions"
	"github.com/kpirnie/m3u-scanner/internal/meta"
	"github.com/kpirnie/m3u-scanner/internal/models"
	"github.com/kpirnie/m3u-scanner/internal/parser"
)

// Scanner performs recursive media file discovery.
type Scanner struct {
	Root       string
	Types      []string
	EnrichMeta bool
	ParseNFO   bool
	Cache      *cache.DB
}

// New constructs a Scanner.
func New(root string, types []string, enrichMeta, parseNFO bool, c *cache.DB) *Scanner {
	return &Scanner{
		Root:       root,
		Types:      types,
		EnrichMeta: enrichMeta,
		ParseNFO:   parseNFO,
		Cache:      c,
	}
}

// Scan runs a full scan and returns a sorted slice of entries.
// Files unchanged since the last scan are loaded from the cache.
func (s *Scanner) Scan() ([]*models.MediaEntry, error) {
	// Load entire cache upfront — one DB round trip
	cached, err := s.Cache.LoadAll()
	if err != nil {
		log.Printf("[scanner] cache load error: %v — proceeding without cache", err)
		cached = make(map[string]*cache.CachedFile)
	}

	var all []*models.MediaEntry
	activePaths := make(map[string]struct{})

	for _, t := range s.Types {
		typeRoot, err := s.resolveTypeRoot(t)
		if err != nil {
			log.Printf("[scanner] %s — skipping", err)
			continue
		}

		log.Printf("[scanner] scanning %s → %s", t, typeRoot)
		entries, active, err := s.scanType(t, typeRoot, cached)
		if err != nil {
			log.Printf("[scanner] error scanning %s: %v", t, err)
			continue
		}
		log.Printf("[scanner] %s: %d file(s)", t, len(entries))
		all = append(all, entries...)
		for p := range active {
			activePaths[p] = struct{}{}
		}
	}

	// Evict deleted files from the cache
	if err := s.Cache.DeleteMissing(activePaths); err != nil {
		log.Printf("[scanner] cache eviction error: %v", err)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].SortKey() < all[j].SortKey()
	})

	return all, nil
}

// ── Private ───────────────────────────────────────────────────────────────────

func (s *Scanner) resolveTypeRoot(mediaType string) (string, error) {
	known := extensions.SubfolderNames[mediaType]

	base := filepath.Base(s.Root)
	for _, name := range known {
		if strings.EqualFold(base, name) {
			return s.Root, nil
		}
	}

	for _, name := range known {
		candidate := filepath.Join(s.Root, name)
		if isDir(candidate) {
			return candidate, nil
		}
	}

	if len(s.Types) == 1 {
		return s.Root, nil
	}

	return "", fmt.Errorf("no %q folder found under %s (known names: %v)", mediaType, s.Root, known)
}

func (s *Scanner) scanType(
	mediaType, typeRoot string,
	cached map[string]*cache.CachedFile,
) ([]*models.MediaEntry, map[string]struct{}, error) {

	exts := extensions.ByType[mediaType]

	// Collect all matching file paths first
	var files []string
	err := filepath.WalkDir(typeRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("[scanner] walk error at %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if exts[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	activePaths := make(map[string]struct{}, len(files))
	for _, f := range files {
		activePaths[f] = struct{}{}
	}

	// Split files into cache-hit (unchanged) and need-processing (new/changed)
	var (
		hits    []*models.MediaEntry
		changed []string
	)

	for _, path := range files {
		mtime, size, err := cache.StatFile(path)
		if err != nil {
			log.Printf("[scanner] stat error %s: %v", path, err)
			continue
		}

		cf := cached[path]
		if !cache.FileChanged(cf, mtime, size) {
			// Cache hit — use stored entry directly
			hits = append(hits, cf.Entry)
		} else {
			changed = append(changed, path)
		}
	}

	log.Printf("[scanner] %s: %d cached, %d new/changed", mediaType, len(hits), len(changed))

	// Process new/changed files concurrently
	workers := runtime.NumCPU() * 2
	jobs := make(chan string, workers)
	resultsCh := make(chan cache.StatEntry, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				entry := s.parseFile(path, mediaType, typeRoot)
				if entry == nil {
					continue
				}
				if s.EnrichMeta {
					meta.Enrich(entry, s.ParseNFO)
				}
				mtime, size, err := cache.StatFile(path)
				if err != nil {
					mtime, size = 0, 0
				}
				resultsCh <- cache.StatEntry{MTime: mtime, Size: size, Entry: entry}
			}
		}()
	}

	go func() {
		for _, f := range changed {
			jobs <- f
		}
		close(jobs)
		wg.Wait()
		close(resultsCh)
	}()

	var newEntries []cache.StatEntry
	for r := range resultsCh {
		newEntries = append(newEntries, r)
	}

	// Persist new/changed entries to cache in one transaction
	if len(newEntries) > 0 {
		if err := s.Cache.UpsertBatch(newEntries); err != nil {
			log.Printf("[scanner] cache write error: %v", err)
		}
	}

	// Combine cache hits + newly processed
	all := make([]*models.MediaEntry, 0, len(hits)+len(newEntries))
	all = append(all, hits...)
	for _, se := range newEntries {
		all = append(all, se.Entry)
	}

	return all, activePaths, nil
}

func (s *Scanner) parseFile(path, mediaType, typeRoot string) *models.MediaEntry {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[scanner] panic parsing %s: %v", path, r)
		}
	}()

	switch mediaType {
	case "music":
		return parser.Music(path, typeRoot)
	case "shows":
		return parser.Show(path, typeRoot)
	case "movies":
		return parser.Movie(path)
	case "images":
		return parser.Image(path)
	}
	return nil
}

func isDir(path string) bool {
	fi, err := fs.Stat(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		return false
	}
	return fi.IsDir()
}
