# m3u-scanner

Recursive media library scanner that serves extended M3U playlists over HTTP.

Drop a file in, the playlist updates automatically. Point any IPTV/media player
at the URL and stream directly. Subsequent scans only process new or changed
files — everything else loads instantly from the SQLite cache.

---

## Features

- **Path-first metadata** — artist, album, genre, series, season derived from folder structure
- **SQLite cache** — unchanged files skip re-parsing; rescans go from minutes to seconds
- **In-memory playlists** — served directly from RAM, zero disk I/O at request time
- **Per-type playlists** — separate endpoints for movies, music, shows + a combined all
- **Filesystem watcher** — debounced rescan fires automatically when files change
- **Cron heartbeat** — scheduled full rescan fallback (default: 3 AM daily)
- **ffprobe integration** — mount from host for accurate video duration
- **`go-taglib`** — Wasm-embedded TagLib for audio duration + year (no CGO)
- **`ncruces/go-sqlite3`** — Wasm-embedded SQLite, no CGO, shares wazero with go-taglib
- **Single static binary** — `debian:bookworm-slim` final image
- **`POST /rescan`** — trigger an immediate rescan from any HTTP client

---

## Expected folder structure

```
/media/
├── Movies/
│   └── The.Matrix.1999.mkv
│
├── Music/
│   └── Albums/
│       └── <Genre>/
│           └── <Artist>/
│               └── <Album>/
│                   └── [Disk N/]
│                       └── 01 - Track Title.mp3
│
└── Shows/
    └── <Show Name>/
        └── Season N/
            └── ShowName S01E01.mkv
```

---

## Quick start

```bash
# Edit compose.yaml — set BASE_URL to your server hostname/IP
podman-compose up -d
podman-compose logs -f
```

### Manual run

```bash
podman run -d \
  --name m3u-scanner \
  --restart unless-stopped \
  -p 8888:8888 \
  -v /mnt/media:/media:ro \
  -v m3u-cache:/data \
  -v /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro \
  -e SCAN_PATH=/media \
  -e SCAN_TYPES="movies music shows" \
  -e BASE_URL="http://kp-media:8888" \
  -e CACHE_PATH=/data/cache.db \
  -e TZ="America/New_York" \
  localhost/m3u-scanner:latest
```

---

## Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/all.m3u` | GET | All media types combined |
| `/movies.m3u` | GET | Movies only |
| `/music.m3u` | GET | Music only |
| `/shows.m3u` | GET | Shows only |
| `/health` | GET | JSON status, entry count, last scan, ffprobe, playlists |
| `/rescan` | POST | Trigger immediate full rescan |

```bash
curl http://kp-media:8888/health
curl -X POST http://kp-media:8888/rescan
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SCAN_PATH` | `/media` | Root path to scan |
| `SCAN_TYPES` | `movies music shows` | Space-separated: `music` `images` `movies` `shows` |
| `SCAN_NFO` | `false` | Parse Jellyfin `.nfo` sidecars for episode titles |
| `BASE_URL` | *(empty)* | Rewrites file paths to HTTP URLs in M3U output |
| `PLAYLIST_NAME` | `playlist.m3u` | Base name (unused — endpoints are fixed per type) |
| `SERVE_PORT` | `8888` | HTTP listen port |
| `DEBOUNCE_SECONDS` | `30` | Seconds of quiet after fs events before rescanning |
| `CRON_SCHEDULE` | `0 3 * * *` | 5-field cron for scheduled rescans |
| `TZ` | `UTC` | Timezone for cron scheduling |
| `FFPROBE_PATH` | `/usr/local/bin/ffprobe` | Path to ffprobe inside the container |
| `CACHE_PATH` | `/data/cache.db` | SQLite cache database path |

---

## Cache

The SQLite cache stores every scanned entry with its `mtime` and `size`.
On each rescan, only files whose `mtime` or `size` has changed are re-parsed
and re-enriched. Everything else loads from the database instantly.

The cache lives at `CACHE_PATH` (default `/data/cache.db`). Mount a named
volume at `/data` to persist it across container restarts and rebuilds:

```yaml
volumes:
  - m3u-cache:/data
```

To force a full cold rescan, delete the database file and restart:

```bash
podman exec m3u-scanner rm /data/cache.db
curl -X POST http://kp-media:8888/rescan
```

---

## ffprobe

Mount from host — not baked into the image:

```yaml
volumes:
  - /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro
```

**Without ffprobe:** audio duration from go-taglib (accurate), video duration = `-1`.

---

## Compatible players

| Player | Platform |
|---|---|
| VLC | Windows / Mac / Linux / iOS / Android |
| Kodi + IPTV Simple Client | Any |
| TiviMate | Android / Fire TV |
| IPTV Smarters | Android / iOS / Fire TV |
| Infuse | Apple TV / iOS |
| mpv | Desktop / CLI |

---

## Project structure

```
m3u-scanner/
├── main.go
├── go.mod
├── Containerfile
├── compose.yaml
└── internal/
    ├── cache/       — SQLite cache (ncruces/go-sqlite3, no CGO)
    ├── config/      — env var loading + validation
    ├── cron/        — minimal 5-field cron scheduler
    ├── extensions/  — type→extension map + known subfolder names
    ├── meta/        — ffprobe + go-taglib + NFO dispatcher
    ├── models/      — MediaEntry struct, sort key, media type enum
    ├── parser/      — path-first metadata extraction
    ├── scanner/     — recursive walker + cache-aware scan loop
    ├── server/      — net/http server, RWMutex in-memory playlists
    ├── watcher/     — fsnotify + debounce + recursive dir tracking
    └── writer/      — builds M3U []byte from sorted entries
```
