# KPTV m3u-scanner

[![Build Main](https://img.shields.io/github/actions/workflow/status/kpirnie/m3u-scanner/build.yaml?branch=main&label=Main&logoColor=white&logo=github&style=for-the-badge)](https://github.com/kpirnie/m3u-scanner/actions?query=workflow%3A%22Build+and+Push+Docker+Image%22+branch%3Amain)
[![Build Develop](https://img.shields.io/github/actions/workflow/status/kpirnie/m3u-scanner/build.yaml?branch=develop&logoColor=white&label=Develop&logo=github&style=for-the-badge)](https://github.com/kpirnie/m3u-scanner/actions?query=workflow%3A%22Build+and+Push+Docker+Image%22+branch%3Adevelop)
[![GitHub Issues](https://img.shields.io/github/issues/kpirnie/m3u-scanner?style=for-the-badge&logo=github&color=006400&logoColor=white)](https://github.com/kpirnie/m3u-scanner/issues)
[![License: MIT](https://img.shields.io/badge/License-MIT-orange.svg?style=for-the-badge&logo=opensourceinitiative)](LICENSE)

[![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go&logoColor=white&style=for-the-badge)](https://golang.org/)
[![Debian](https://img.shields.io/badge/Base-Debian%20Trixie-A81D33?logo=debian&logoColor=white&style=for-the-badge)](https://www.debian.org/)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/bd4Qan3PaN)
[![Kevin Pirnie](https://img.shields.io/badge/www-KevinPirnie.com-000d2d?style=for-the-badge&labelColor=555&logoColor=white&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjgiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+CiAgPGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMTAiLz4KICA8ZWxsaXBzZSBjeD0iMTIiIGN5PSIxMiIgcng9IjQuNSIgcnk9IjEwIi8+CiAgPGxpbmUgeDE9IjIiIHkxPSIxMiIgeDI9IjIyIiB5Mj0iMTIiLz4KICA8bGluZSB4MT0iNC41IiB5MT0iNi41IiB4Mj0iMTkuNSIgeTI9IjYuNSIvPgogIDxsaW5lIHgxPSI0LjUiIHkxPSIxNy41IiB4Mj0iMTkuNSIgeTI9IjE3LjUiLz4KPC9zdmc+Cg==)](https://kevinpirnie.com/)

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
- **Media file serving** — files served directly over HTTP, no separate web server needed
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
  -v /home/user/m3u-scanner/cache:/data \
  -v /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro \
  -e SCAN_PATH=/media \
  -e SCAN_TYPES="movies music shows" \
  -e BASE_URL="http://your-server:8888" \
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
| `/media/...` | GET | Direct media file serving |
| `/health` | GET | JSON: status, entry count, last scan, ffprobe, playlists |
| `/rescan` | POST | Trigger immediate full rescan |

```bash
# Health check
curl http://your-server:8888/health

# Force rescan
curl -X POST http://your-server:8888/rescan
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SCAN_PATH` | `/media` | Root path to scan |
| `SCAN_TYPES` | `movies music shows` | Space-separated: `music` `images` `movies` `shows` |
| `SCAN_NFO` | `false` | Parse Jellyfin `.nfo` sidecars for episode titles |
| `BASE_URL` | *(empty)* | Rewrites file paths to HTTP URLs in M3U output |
| `SERVE_PORT` | `8888` | HTTP listen port |
| `DEBOUNCE_SECONDS` | `30` | Seconds of quiet after fs events before rescanning |
| `CRON_SCHEDULE` | `0 3 * * *` | 5-field cron for scheduled rescans |
| `TZ` | `UTC` | Timezone for cron scheduling |
| `FFPROBE_PATH` | `/usr/local/bin/ffprobe` | Path to ffprobe inside the container |
| `CACHE_PATH` | `/data/cache.db` | SQLite cache database path |

### `CRON_SCHEDULE` syntax

Standard 5-field: `minute hour dom month dow`

```
0 3 * * *     → 3:00 AM daily (default)
0 */6 * * *   → every 6 hours
@daily        → midnight daily
@hourly       → top of every hour
```

---

## Cache

The SQLite cache stores every scanned entry with its `mtime` and `size`.
On each rescan, only files whose `mtime` or `size` has changed are re-parsed
and re-enriched. Everything else loads from the database instantly.

Mount a host directory at `/data` to persist the cache across restarts:

```yaml
volumes:
  - /home/user/m3u-scanner/cache:/data
```

To force a full cold rescan:

```bash
rm /home/user/m3u-scanner/cache/cache.db
curl -X POST http://your-server:8888/rescan
```

---

## ffprobe

Not baked into the image — mount from the host:

```yaml
volumes:
  - /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro
```

Find your path with `which ffprobe`. Install with `sudo apt install ffmpeg` if missing.

**Without ffprobe:** audio duration comes from go-taglib (accurate), video duration = `-1`.

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
├── LICENSE
└── internal/
    ├── cache/       — SQLite cache (ncruces/go-sqlite3, no CGO)
    ├── config/      — env var loading + validation
    ├── cron/        — minimal 5-field cron scheduler (no external dep)
    ├── extensions/  — type→extension map + known subfolder names
    ├── meta/        — ffprobe + go-taglib + NFO dispatcher
    ├── models/      — MediaEntry struct, sort key, media type enum
    ├── parser/      — path-first metadata extraction
    ├── scanner/     — recursive walker + cache-aware scan loop
    ├── server/      — net/http: playlist serving, media file serving, health, rescan
    ├── watcher/     — fsnotify + debounce + recursive dir tracking
    └── writer/      — builds M3U []byte from sorted entries
```

---

## License

MIT © 2026 [Kevin Pirnie](https://kevinpirnie.com/)