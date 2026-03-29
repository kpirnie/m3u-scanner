// Package meta handles metadata enrichment from tag libraries and ffprobe.
package meta

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

// ffprobeAvailable is set once at startup and shared across all calls.
var ffprobeAvailable bool
var ffprobePath string

// InitFFProbe checks whether ffprobe exists at path and is executable.
// Must be called once during startup before any enrichment runs.
func InitFFProbe(path string) {
	ffprobePath = path
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("[meta] ffprobe not found at %s — duration will use tag libraries only", path)
		return
	}
	if info.Mode()&0111 == 0 {
		log.Printf("[meta] ffprobe at %s is not executable — skipping", path)
		return
	}
	ffprobeAvailable = true
	log.Printf("[meta] ffprobe available at %s", path)
}

// FFProbeAvailable reports whether ffprobe was found and is executable.
func FFProbeAvailable() bool {
	return ffprobeAvailable
}

// DurationViaFFProbe returns the duration in seconds for the given file.
// Returns -1 and an error if ffprobe is unavailable or fails.
func DurationViaFFProbe(filePath string) (int, error) {
	if !ffprobeAvailable {
		return -1, fmt.Errorf("ffprobe not available")
	}

	out, err := exec.Command(
		ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_entries", "format=duration",
		filePath,
	).Output()
	if err != nil {
		return -1, fmt.Errorf("ffprobe exec: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return -1, fmt.Errorf("ffprobe parse: %w", err)
	}

	secs, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return -1, fmt.Errorf("ffprobe duration value: %w", err)
	}

	return int(secs), nil
}
