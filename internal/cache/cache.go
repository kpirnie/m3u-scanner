// Package cache provides a SQLite-backed persistent cache for scanned media
// entries. On each scan, only files whose mtime or size has changed since the
// last scan are re-parsed and re-enriched — everything else is loaded directly
// from the database, making subsequent scans dramatically faster.
//
// Driver: github.com/ncruces/go-sqlite3 (Wasm, no CGO, shares wazero runtime
// with go-taglib).
package cache

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/kpirnie/m3u-scanner/internal/models"
)

const schemaVersion = 1

// DB wraps a SQLite connection and exposes cache operations.
type DB struct {
	db *sql.DB
}

// CachedFile holds the staleness-check fields for a single file.
type CachedFile struct {
	MTime int64 // Unix timestamp seconds
	Size  int64
	Entry *models.MediaEntry
}

// Open opens (or creates) the cache database at path and ensures the schema
// is up to date. Returns a ready-to-use DB.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// WAL mode — reads don't block writes during scan updates
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
		return nil, err
	}

	c := &DB{db: db}
	if err := c.migrate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Close closes the underlying database connection.
func (c *DB) Close() error {
	return c.db.Close()
}

// LoadAll returns every cached entry keyed by absolute file path.
func (c *DB) LoadAll() (map[string]*CachedFile, error) {
	rows, err := c.db.Query(`SELECT
		path, mtime, size, media_type_id,
		duration, year, display, tvg_name, group_title,
		artist, genre, album, disc, track,
		series, season, episode, episode_title
	FROM entries`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*CachedFile)
	for rows.Next() {
		var (
			path, display, tvgName, groupTitle string
			mtime, size                        int64
			mediaTypeID                        int
			duration                           int
			year                               sql.NullString
			artist, genre, album               sql.NullString
			disc, track                        sql.NullInt64
			series, episodeTitle               sql.NullString
			season, episode                    sql.NullInt64
		)
		if err := rows.Scan(
			&path, &mtime, &size, &mediaTypeID,
			&duration, &year, &display, &tvgName, &groupTitle,
			&artist, &genre, &album, &disc, &track,
			&series, &season, &episode, &episodeTitle,
		); err != nil {
			log.Printf("[cache] scan error: %v", err)
			continue
		}

		mediaType := models.MediaTypeFromInt[mediaTypeID]
		e := &models.MediaEntry{
			Path:         path,
			MediaType:    mediaType,
			GroupTitle:   groupTitle,
			TVGName:      tvgName,
			Display:      display,
			Duration:     duration,
			Year:         year.String,
			Artist:       artist.String,
			Genre:        genre.String,
			Album:        album.String,
			Disc:         int(disc.Int64),
			Track:        int(track.Int64),
			Series:       series.String,
			Season:       int(season.Int64),
			Episode:      int(episode.Int64),
			EpisodeTitle: episodeTitle.String,
		}

		result[path] = &CachedFile{
			MTime: mtime,
			Size:  size,
			Entry: e,
		}
	}
	return result, rows.Err()
}

// Upsert inserts or replaces a single entry in the cache.
func (c *DB) Upsert(mtime, size int64, e *models.MediaEntry) error {
	mediaTypeID := models.MediaTypeToInt[e.MediaType]
	_, err := c.db.Exec(`INSERT OR REPLACE INTO entries (
		path, mtime, size, media_type_id,
		duration, year, display, tvg_name, group_title,
		artist, genre, album, disc, track,
		series, season, episode, episode_title
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.Path, mtime, size, mediaTypeID,
		e.Duration, nullStr(e.Year), e.Display, e.TVGName, e.GroupTitle,
		nullStr(e.Artist), nullStr(e.Genre), nullStr(e.Album),
		nullInt(e.Disc), nullInt(e.Track),
		nullStr(e.Series), nullInt(e.Season), nullInt(e.Episode),
		nullStr(e.EpisodeTitle),
	)
	return err
}

// UpsertBatch writes a slice of entries in a single transaction.
func (c *DB) UpsertBatch(files []StatEntry) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO entries (
		path, mtime, size, media_type_id,
		duration, year, display, tvg_name, group_title,
		artist, genre, album, disc, track,
		series, season, episode, episode_title
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range files {
		mediaTypeID := models.MediaTypeToInt[f.Entry.MediaType]
		if _, err = stmt.Exec(
			f.Entry.Path, f.MTime, f.Size, mediaTypeID,
			f.Entry.Duration, nullStr(f.Entry.Year), f.Entry.Display,
			f.Entry.TVGName, f.Entry.GroupTitle,
			nullStr(f.Entry.Artist), nullStr(f.Entry.Genre), nullStr(f.Entry.Album),
			nullInt(f.Entry.Disc), nullInt(f.Entry.Track),
			nullStr(f.Entry.Series), nullInt(f.Entry.Season), nullInt(f.Entry.Episode),
			nullStr(f.Entry.EpisodeTitle),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteMissing removes any cached entries whose paths are not in the
// provided set. Called after a scan to evict deleted files.
func (c *DB) DeleteMissing(activePaths map[string]struct{}) error {
	encoded, err := json.Marshal(keys(activePaths))
	if err != nil {
		return err
	}

	// SQLite JSON1 — delete rows whose path isn't in the active set
	_, err = c.db.Exec(
		`DELETE FROM entries WHERE path NOT IN (
			SELECT value FROM json_each(?)
		)`, string(encoded))
	return err
}

// SetMeta stores a key/value pair in the meta table.
func (c *DB) SetMeta(key, value string) error {
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetMeta retrieves a value from the meta table. Returns "" if not found.
func (c *DB) GetMeta(key string) string {
	var v string
	_ = c.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	return v
}

// StatEntry pairs a MediaEntry with its filesystem stat values.
type StatEntry struct {
	MTime int64
	Size  int64
	Entry *models.MediaEntry
}

// ── Schema migration ──────────────────────────────────────────────────────────

func (c *DB) migrate() error {
	// Create tables if they don't exist
	_, err := c.db.Exec(`
	CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS entries (
		path          TEXT PRIMARY KEY,
		mtime         INTEGER NOT NULL,
		size          INTEGER NOT NULL,
		media_type_id INTEGER NOT NULL,

		-- shared
		duration      INTEGER NOT NULL DEFAULT -1,
		year          TEXT,
		display       TEXT NOT NULL,
		tvg_name      TEXT NOT NULL,
		group_title   TEXT NOT NULL,

		-- music
		artist        TEXT,
		genre         TEXT,
		album         TEXT,
		disc          INTEGER DEFAULT 0,
		track         INTEGER DEFAULT 0,

		-- shows
		series        TEXT,
		season        INTEGER DEFAULT 0,
		episode       INTEGER DEFAULT 0,
		episode_title TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_entries_media_type ON entries(media_type_id);
	CREATE INDEX IF NOT EXISTS idx_entries_mtime      ON entries(mtime);
	`)
	if err != nil {
		return err
	}

	// Store/verify schema version
	existing := c.GetMeta("schema_version")
	if existing == "" {
		if err := c.SetMeta("schema_version", "1"); err != nil {
			return err
		}
		if err := c.SetMeta("created_at", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		log.Printf("[cache] database initialised (schema v%d)", schemaVersion)
	} else {
		log.Printf("[cache] database opened (schema v%s)", existing)
	}

	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt(n int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(n), Valid: n != 0}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// FileChanged returns true if the file at path has a different mtime or size
// than what is cached.
func FileChanged(cached *CachedFile, mtime, size int64) bool {
	if cached == nil {
		return true
	}
	return cached.MTime != mtime || cached.Size != size
}

// StatFile returns the mtime (Unix seconds) and size of a file.
func StatFile(path string) (mtime, size int64, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	return fi.ModTime().Unix(), fi.Size(), nil
}
