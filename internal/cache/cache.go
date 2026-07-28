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
	"strconv"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"m3u-scanner/internal/models"
)

const schemaVersion = 2

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
		artist, album, disc, track,
		series, season, episode, episode_title,
		title, sort_title, plot, tagline, poster, fanart,
		rating, critic_rating, mpaa, country, premiered,
		imdb_id, tmdb_id, tvdb_id, collection,
		genres, studios, tags, cast_members, directors, writers
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
			artist, album                      sql.NullString
			disc, track                        sql.NullInt64
			series, episodeTitle               sql.NullString
			season, episode                    sql.NullInt64

			title, sortTitle, plot, tagline    sql.NullString
			poster, fanart                     sql.NullString
			rating                             sql.NullFloat64
			criticRating                       sql.NullInt64
			mpaa, country, premiered           sql.NullString
			imdbID, tmdbID, tvdbID, collection sql.NullString
			genresJSON, studiosJSON, tagsJSON  sql.NullString
			castJSON, directorsJSON            sql.NullString
			writersJSON                        sql.NullString
		)
		if err := rows.Scan(
			&path, &mtime, &size, &mediaTypeID,
			&duration, &year, &display, &tvgName, &groupTitle,
			&artist, &album, &disc, &track,
			&series, &season, &episode, &episodeTitle,
			&title, &sortTitle, &plot, &tagline, &poster, &fanart,
			&rating, &criticRating, &mpaa, &country, &premiered,
			&imdbID, &tmdbID, &tvdbID, &collection,
			&genresJSON, &studiosJSON, &tagsJSON, &castJSON, &directorsJSON, &writersJSON,
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
			Album:        album.String,
			Disc:         int(disc.Int64),
			Track:        int(track.Int64),
			Series:       series.String,
			Season:       int(season.Int64),
			Episode:      int(episode.Int64),
			EpisodeTitle: episodeTitle.String,
			Title:        title.String,
			SortTitle:    sortTitle.String,
			Plot:         plot.String,
			Tagline:      tagline.String,
			Poster:       poster.String,
			Fanart:       fanart.String,
			Rating:       rating.Float64,
			CriticRating: int(criticRating.Int64),
			MPAA:         mpaa.String,
			Country:      country.String,
			Premiered:    premiered.String,
			IMDBID:       imdbID.String,
			TMDBID:       tmdbID.String,
			TVDBID:       tvdbID.String,
			Collection:   collection.String,
		}

		decodeJSON(genresJSON, &e.Genres)
		decodeJSON(studiosJSON, &e.Studios)
		decodeJSON(tagsJSON, &e.Tags)
		decodeJSON(castJSON, &e.Cast)
		decodeJSON(directorsJSON, &e.Directors)
		decodeJSON(writersJSON, &e.Writers)

		result[path] = &CachedFile{
			MTime: mtime,
			Size:  size,
			Entry: e,
		}
	}
	return result, rows.Err()
}

// upsertSQL is shared by Upsert and UpsertBatch so the column list and its
// argument order can only ever drift together.
const upsertSQL = `INSERT OR REPLACE INTO entries (
	path, mtime, size, media_type_id,
	duration, year, display, tvg_name, group_title,
	artist, album, disc, track,
	series, season, episode, episode_title,
	title, sort_title, plot, tagline, poster, fanart,
	rating, critic_rating, mpaa, country, premiered,
	imdb_id, tmdb_id, tvdb_id, collection,
	genres, studios, tags, cast_members, directors, writers
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

// upsertArgs builds the argument list for upsertSQL in column order.
func upsertArgs(mtime, size int64, mediaTypeID int, e *models.MediaEntry) []any {
	return []any{
		e.Path, mtime, size, mediaTypeID,
		e.Duration, nullStr(e.Year), e.Display, e.TVGName, e.GroupTitle,
		nullStr(e.Artist), nullStr(e.Album), nullInt(e.Disc), nullInt(e.Track),
		nullStr(e.Series), nullInt(e.Season), nullInt(e.Episode), nullStr(e.EpisodeTitle),
		nullStr(e.Title), nullStr(e.SortTitle), nullStr(e.Plot), nullStr(e.Tagline),
		nullStr(e.Poster), nullStr(e.Fanart),
		e.Rating, e.CriticRating, nullStr(e.MPAA), nullStr(e.Country), nullStr(e.Premiered),
		nullStr(e.IMDBID), nullStr(e.TMDBID), nullStr(e.TVDBID), nullStr(e.Collection),
		nullJSON(e.Genres), nullJSON(e.Studios), nullJSON(e.Tags),
		nullJSON(e.Cast), nullJSON(e.Directors), nullJSON(e.Writers),
	}
}

// Upsert inserts or replaces a single entry in the cache.
func (c *DB) Upsert(mtime, size int64, e *models.MediaEntry) error {
	mediaTypeID := models.MediaTypeToInt[e.MediaType]
	_, err := c.db.Exec(upsertSQL, upsertArgs(mtime, size, mediaTypeID, e)...)
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

	stmt, err := tx.Prepare(upsertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range files {
		mediaTypeID := models.MediaTypeToInt[f.Entry.MediaType]
		if _, err = stmt.Exec(upsertArgs(f.MTime, f.Size, mediaTypeID, f.Entry)...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteMissing removes cached entries whose paths are not in the provided
// set, scoped to the given media type IDs. Called after a scan to evict
// deleted files. Scoping is required so a partial (per-type) scan does not
// evict entries belonging to types that were not scanned.
func (c *DB) DeleteMissing(activePaths map[string]struct{}, mediaTypeIDs []int) error {
	if len(mediaTypeIDs) == 0 {
		return nil
	}

	encodedPaths, err := json.Marshal(keys(activePaths))
	if err != nil {
		return err
	}

	encodedTypes, err := json.Marshal(mediaTypeIDs)
	if err != nil {
		return err
	}

	// SQLite JSON1 — delete rows in the scanned types whose path isn't active
	_, err = c.db.Exec(
		`DELETE FROM entries WHERE media_type_id IN (
			SELECT value FROM json_each(?)
		) AND path NOT IN (
			SELECT value FROM json_each(?)
		)`, string(encodedTypes), string(encodedPaths))
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
	if _, err := c.db.Exec(`
	CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`); err != nil {
		return err
	}

	existing := c.GetMeta("schema_version")
	current := strconv.Itoa(schemaVersion)

	// The entries table is a pure cache. On a schema change it is dropped and
	// rebuilt rather than ALTERed: new metadata columns cannot be back-filled
	// for files whose mtime and size are unchanged, so an additive migration
	// would leave every existing row permanently missing the new fields.
	if existing != "" && existing != current {
		log.Printf("[cache] schema v%s → v%s — dropping cache, a full re-scan will follow",
			existing, current)
		if _, err := c.db.Exec(`DROP TABLE IF EXISTS entries`); err != nil {
			return err
		}
	}

	if _, err := c.db.Exec(`
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
		album         TEXT,
		disc          INTEGER DEFAULT 0,
		track         INTEGER DEFAULT 0,

		-- shows
		series        TEXT,
		season        INTEGER DEFAULT 0,
		episode       INTEGER DEFAULT 0,
		episode_title TEXT,

		-- extended metadata
		title         TEXT,
		sort_title    TEXT,
		plot          TEXT,
		tagline       TEXT,
		poster        TEXT,
		fanart        TEXT,
		rating        REAL    DEFAULT 0,
		critic_rating INTEGER DEFAULT 0,
		mpaa          TEXT,
		country       TEXT,
		premiered     TEXT,
		imdb_id       TEXT,
		tmdb_id       TEXT,
		tvdb_id       TEXT,
		collection    TEXT,

		-- extended metadata, JSON-encoded arrays
		genres        TEXT,
		studios       TEXT,
		tags          TEXT,
		cast_members  TEXT,
		directors     TEXT,
		writers       TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_entries_media_type ON entries(media_type_id);
	CREATE INDEX IF NOT EXISTS idx_entries_mtime      ON entries(mtime);
	`); err != nil {
		return err
	}

	if err := c.SetMeta("schema_version", current); err != nil {
		return err
	}

	if existing == "" {
		if err := c.SetMeta("created_at", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		log.Printf("[cache] database initialised (schema v%d)", schemaVersion)
	} else {
		log.Printf("[cache] database opened (schema v%s)", current)
	}

	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// nullJSON marshals a slice to JSON for storage, returning NULL when empty
// so absent values stay distinguishable from empty arrays.
func nullJSON(v any) sql.NullString {
	switch s := v.(type) {
	case []string:
		if len(s) == 0 {
			return sql.NullString{}
		}
	case []models.Person:
		if len(s) == 0 {
			return sql.NullString{}
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// decodeJSON unmarshals a stored JSON column into out, ignoring NULL and
// malformed values so one bad row cannot fail an entire cache load.
func decodeJSON(s sql.NullString, out any) {
	if !s.Valid || s.String == "" {
		return
	}
	if err := json.Unmarshal([]byte(s.String), out); err != nil {
		log.Printf("[cache] json decode error: %v", err)
	}
}

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
