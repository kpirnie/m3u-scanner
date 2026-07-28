// Package config loads all runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the scanner/server.
type Config struct {
	// Scanning
	ScanPath  string
	ScanTypes []string
	ScanNFO   bool

	// Output
	BaseURL      string
	PlaylistName string

	// Server
	ServePort int

	// Cron
	CronSchedule string

	// ffprobe
	FFProbePath string

	// Cache
	CachePath string

	// Derived
	ServeAddr string
}

var validTypes = map[string]bool{
	"music": true, "images": true, "movies": true, "shows": true,
}

func Load() (*Config, error) {
	c := &Config{}

	c.ScanPath = envOr("SCAN_PATH", "/media")
	if c.ScanPath == "" {
		return nil, fmt.Errorf("SCAN_PATH must not be empty")
	}
	c.ScanPath = filepath.Clean(c.ScanPath)

	rawTypes := envOr("SCAN_TYPES", "movies music shows")
	for _, t := range strings.Fields(rawTypes) {
		t = strings.ToLower(strings.TrimSpace(t))
		if !validTypes[t] {
			return nil, fmt.Errorf("invalid SCAN_TYPES value %q (valid: music, images, movies, shows)", t)
		}
		c.ScanTypes = append(c.ScanTypes, t)
	}
	if len(c.ScanTypes) == 0 {
		return nil, fmt.Errorf("SCAN_TYPES must contain at least one valid type")
	}

	c.ScanNFO = envBool("SCAN_NFO", false)

	c.BaseURL = strings.TrimRight(envOr("BASE_URL", ""), "/")
	c.PlaylistName = envOr("PLAYLIST_NAME", "playlist.m3u")
	if !strings.HasSuffix(c.PlaylistName, ".m3u") {
		c.PlaylistName += ".m3u"
	}

	port, err := envInt("SERVE_PORT", 8888)
	if err != nil {
		return nil, fmt.Errorf("SERVE_PORT: %w", err)
	}
	c.ServePort = port
	c.ServeAddr = fmt.Sprintf(":%d", c.ServePort)

	c.CronSchedule = envOr("CRON_SCHEDULE", "*/30 * * * *")
	c.FFProbePath = envOr("FFPROBE_PATH", "/usr/local/bin/ffprobe")
	c.CachePath = envOr("CACHE_PATH", "/data/cache.db")

	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", v)
	}
	return n, nil
}
