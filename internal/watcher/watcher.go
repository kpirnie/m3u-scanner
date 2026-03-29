// Package watcher monitors the scan root for filesystem changes and triggers
// a debounced rescan callback when events settle.
package watcher

import (
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher wraps fsnotify with:
//   - Recursive directory watching (fsnotify is non-recursive by default)
//   - Debouncing: the callback fires only after events have been quiet for
//     the configured duration, avoiding repeated scans during bulk copies
//   - Dynamic registration: newly created subdirectories are added to the watch
type Watcher struct {
	root     string
	debounce time.Duration
	onChange func()
}

// New creates a Watcher. onChange is called (in its own goroutine) after
// events have settled for the debounce duration.
func New(root string, debounce time.Duration, onChange func()) *Watcher {
	return &Watcher{
		root:     root,
		debounce: debounce,
		onChange: onChange,
	}
}

// Watch starts watching and blocks until an unrecoverable error occurs.
// It should be run in a dedicated goroutine.
func (w *Watcher) Watch() {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[watcher] failed to create watcher: %v", err)
		return
	}
	defer fw.Close()

	if err := w.addAll(fw, w.root); err != nil {
		log.Printf("[watcher] failed to add root: %v", err)
		return
	}

	log.Printf("[watcher] watching %s (debounce: %s)", w.root, w.debounce)

	var timer *time.Timer

	resetTimer := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(w.debounce, func() {
			log.Printf("[watcher] change detected — triggering rescan")
			go w.onChange()
		})
	}

	for {
		select {
		case event, ok := <-fw.Events:
			if !ok {
				return
			}

			// Dynamically add newly created directories so they are watched too
			if event.Has(fsnotify.Create) {
				fi, err := filepath.EvalSymlinks(event.Name)
				if err == nil {
					if info, err := filepath.Abs(fi); err == nil {
						_ = info
						// Check if it's a dir and add it
						_ = fw.Add(event.Name)
						// Walk in case an entire subtree was moved in
						_ = w.addAll(fw, event.Name)
					}
				}
			}

			resetTimer()

		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			log.Printf("[watcher] error: %v", err)
		}
	}
}

// addAll recursively adds root and all its subdirectories to the watcher.
func (w *Watcher) addAll(fw *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("[watcher] walk error at %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			if watchErr := fw.Add(path); watchErr != nil {
				log.Printf("[watcher] could not watch %s: %v", path, watchErr)
			}
		}
		return nil
	})
}
