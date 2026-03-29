package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kpirnie/m3u-scanner/internal/cache"
	"github.com/kpirnie/m3u-scanner/internal/config"
	"github.com/kpirnie/m3u-scanner/internal/cron"
	"github.com/kpirnie/m3u-scanner/internal/meta"
	"github.com/kpirnie/m3u-scanner/internal/scanner"
	"github.com/kpirnie/m3u-scanner/internal/server"
	"github.com/kpirnie/m3u-scanner/internal/watcher"
	"github.com/kpirnie/m3u-scanner/internal/writer"
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
	log.Printf("[main] debounce   : %s", cfg.Debounce())
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

	// ── Shared scan function ──────────────────────────────────────────────────
	doScan := func() {
		log.Println("[main] scan started")
		start := time.Now()

		entries, err := sc.Scan()
		if err != nil {
			log.Printf("[main] scan error: %v", err)
			return
		}

		playlists := writer.BuildAll(entries, cfg.BaseURL)
		srv.UpdatePlaylist(playlists, len(entries))

		_ = cacheDB.SetMeta("last_scan", time.Now().UTC().Format(time.RFC3339))

		log.Printf("[main] scan done in %s — %d entries, %d playlists",
			time.Since(start).Round(time.Millisecond), len(entries), len(playlists))
	}

	server.RescanFunc = doScan

	// ── Initial scan ──────────────────────────────────────────────────────────
	doScan()

	// ── Filesystem watcher ────────────────────────────────────────────────────
	w := watcher.New(cfg.ScanPath, cfg.Debounce(), doScan)
	go w.Watch()

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
