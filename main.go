package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"m3u-scanner/internal/cache"
	"m3u-scanner/internal/config"
	"m3u-scanner/internal/cron"
	"m3u-scanner/internal/meta"
	"m3u-scanner/internal/models"
	"m3u-scanner/internal/scanner"
	"m3u-scanner/internal/server"
	"m3u-scanner/internal/writer"
)

func main() {
	log.SetFlags(log.LstdFlags)

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[main] config error: %v", err)
	}

	log.Printf("[main] scan path  : %s", cfg.ScanPath)
	log.Printf("[main] scan types : %v", cfg.ScanTypes)
	log.Printf("[main] base url   : %q", cfg.BaseURL)
	log.Printf("[main] playlist   : /%s", cfg.PlaylistName)
	log.Printf("[main] port       : %d", cfg.ServePort)
	log.Printf("[main] cron       : %s", cfg.CronSchedule)
	log.Printf("[main] nfo        : %v", cfg.ScanNFO)
	log.Printf("[main] cache      : %s", cfg.CachePath)

	// ── ffprobe ───────────────────────────────────────────────────────────────
	meta.InitFFProbe(cfg.FFProbePath)

	// ── Cache ─────────────────────────────────────────────────────────────────
	if err := os.MkdirAll(filepath.Dir(cfg.CachePath), 0755); err != nil {
		log.Fatalf("[main] cache dir error: %v", err)
	}
	cacheDB, err := cache.Open(cfg.CachePath)
	if err != nil {
		log.Fatalf("[main] cache open error: %v", err)
	}
	defer cacheDB.Close()

	// ── Scanner ───────────────────────────────────────────────────────────────
	sc := scanner.New(cfg.ScanPath, cfg.ScanTypes, true, cfg.ScanNFO, cacheDB)

	// ── Server ────────────────────────────────────────────────────────────────
	srv := server.New(cfg.ServeAddr, cfg.PlaylistName, meta.FFProbeAvailable())

	// ── Entry store ───────────────────────────────────────────────────────────
	// Entries are held per media type so a partial rescan can replace only the
	// types it scanned while the playlists are still rebuilt from the full set.
	var (
		entryMu sync.Mutex
		byType  = make(map[string][]*models.MediaEntry)
	)

	doScanTypes := func(types []string) {
		log.Printf("[main] scan started: %v", types)
		start := time.Now()

		entries, err := sc.ScanTypes(types)
		if err != nil {
			log.Printf("[main] scan error: %v", err)
			return
		}

		entryMu.Lock()
		for _, t := range types {
			byType[t] = nil
		}
		for _, e := range entries {
			byType[e.MediaType] = append(byType[e.MediaType], e)
		}

		var all []*models.MediaEntry
		for _, te := range byType {
			all = append(all, te...)
		}
		entryMu.Unlock()

		sort.Slice(all, func(i, j int) bool {
			return all[i].SortKey() < all[j].SortKey()
		})

		playlists := writer.BuildAll(all, cfg.BaseURL)
		srv.UpdatePlaylist(playlists, len(all))
		srv.SetEntries(all)

		_ = cacheDB.SetMeta("last_scan", time.Now().UTC().Format(time.RFC3339))

		log.Printf("[main] scan done in %s — %d entries, %d playlists",
			time.Since(start).Round(time.Millisecond), len(all), len(playlists))
	}

	doScan := func() { doScanTypes(cfg.ScanTypes) }

	doScanPath := func(path string) error {
		entry, err := sc.ScanPath(path)
		if err != nil {
			log.Printf("[main] path scan error: %v", err)
			return err
		}

		entryMu.Lock()
		list := byType[entry.MediaType]
		replaced := false
		for i, existing := range list {
			if existing.Path == entry.Path {
				list[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			list = append(list, entry)
		}
		byType[entry.MediaType] = list

		var all []*models.MediaEntry
		for _, te := range byType {
			all = append(all, te...)
		}
		entryMu.Unlock()

		sort.Slice(all, func(i, j int) bool {
			return all[i].SortKey() < all[j].SortKey()
		})

		playlists := writer.BuildAll(all, cfg.BaseURL)
		srv.UpdatePlaylist(playlists, len(all))
		srv.SetEntries(all)

		log.Printf("[main] re-enriched %s", entry.Path)
		return nil
	}

	server.RescanFunc = doScan
	server.RescanPathFunc = doScanPath
	server.RescanTypesFunc = doScanTypes
	server.ScanTypes = cfg.ScanTypes

	// ── Initial scan ──────────────────────────────────────────────────────────
	doScan()

	// ── Cron ──────────────────────────────────────────────────────────────────
	cr, err := cron.New(cfg.CronSchedule, doScan)
	if err != nil {
		log.Printf("[main] cron disabled: %v", err)
	} else {
		go cr.Run()
	}

	// ── HTTP server ───────────────────────────────────────────────────────────
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("[main] server: %v", err)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[main] received %s — shutting down", sig)
}
