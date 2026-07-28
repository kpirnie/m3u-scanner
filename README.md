# KPTV m3u-scanner

[![Build Main](https://img.shields.io/github/actions/workflow/status/kpirnie/m3u-scanner/build.yaml?branch=main&label=Main&logoColor=white&logo=github&labelColor=000&style=for-the-badge)](https://github.com/kpirnie/m3u-scanner/actions?query=workflow%3A%22Build+and+Push+Docker+Image%22+branch%3Amain)
[![Build Develop](https://img.shields.io/github/actions/workflow/status/kpirnie/m3u-scanner/build.yaml?branch=develop&logoColor=white&label=Develop&logo=github&labelColor=000&style=for-the-badge)](https://github.com/kpirnie/m3u-scanner/actions?query=workflow%3A%22Build+and+Push+Docker+Image%22+branch%3Adevelop)
[![GitHub Issues](https://img.shields.io/github/issues/kpirnie/m3u-scanner?style=for-the-badge&logo=github&color=006400&logoColor=white&labelColor=000)](https://github.com/kpirnie/m3u-scanner/issues)
[![License: MIT](https://img.shields.io/badge/License-MIT-orange.svg?style=for-the-badge&logo=opensourceinitiative&logoColor=white&labelColor=000)](LICENSE)

[![Go](https://img.shields.io/badge/Go-1.26.5-00ADD8?logo=go&logoColor=white&style=for-the-badge&labelColor=000)](https://golang.org/)
[![Debian](https://img.shields.io/badge/Base-Debian%20Trixie-A81D33?logo=debian&logoColor=white&style=for-the-badge&labelColor=000)](https://www.debian.org/)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white&style=for-the-badge&labelColor=000)](https://discord.gg/bd4Qan3PaN)
[![Kevin Pirnie](https://img.shields.io/badge/www-KevinPirnie.com-000d2d?style=for-the-badge&labelColor=000&logoColor=white&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjgiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLWxpbmVqb2luPSJyb3VuZCI+CiAgPGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMTAiLz4KICA8ZWxsaXBzZSBjeD0iMTIiIGN5PSIxMiIgcng9IjQuNSIgcnk9IjEwIi8+CiAgPGxpbmUgeDE9IjIiIHkxPSIxMiIgeDI9IjIyIiB5Mj0iMTIiLz4KICA8bGluZSB4MT0iNC41IiB5MT0iNi41IiB4Mj0iMTkuNSIgeTI9IjYuNSIvPgogIDxsaW5lIHgxPSI0LjUiIHkxPSIxNy41IiB4Mj0iMTkuNSIgeTI9IjE3LjUiLz4KPC9zdmc+Cg==)](https://kevinpirnie.com/)

Recursive media library scanner that serves extended M3U playlists over HTTP,
with a built-in admin interface for managing metadata.

Point any IPTV/media player at the URL and stream directly. Metadata comes from
folder structure, embedded tags, and Jellyfin/Kodi `.nfo` sidecars — editable
in the browser and written back to disk. Subsequent scans only process new or
changed files; everything else loads instantly from the SQLite cache.

---

## Features

- **Admin UI** at `/admin/` — browse, search, and edit metadata for movies, shows, and music; poster and fanart thumbnails with full-size preview
- **NFO sidecars** — reads and writes Jellyfin/Kodi `.nfo` for plot, tagline, cast, ratings, genres, studios, artwork, and external IDs
- **Rich M3U output** — `tvg-logo`, `tvg-id`, plus extended attributes carrying the full metadata set
- **Path-first metadata** — artist, album, genre, series, season derived from folder structure
- **Embedded tag write-through** — push edited metadata and cover art into audio files themselves
- **SQLite cache** — unchanged files skip re-parsing; rescans go from minutes to seconds
- **In-memory playlists** — served directly from RAM, zero disk I/O at request time
- **Per-type playlists** — separate endpoints for movies, music, shows, images + a combined all
- **Media file serving** — files served directly over HTTP, no separate web server needed
- **Scheduled and manual scans** — cron plus full, per-type, and single-file triggers
- **ffprobe integration** — mount from host for accurate video duration
- **`go-taglib`** — Wasm-embedded TagLib for audio tags and duration (no CGO)
- **`ncruces/go-sqlite3`** — Wasm-embedded SQLite, no CGO, shares wazero with go-taglib
- **Single static binary** — UI assets embedded; `debian:trixie-slim` final image
- **Multi-arch** — `linux/amd64` and `linux/arm64`

---

## Expected folder structure

```
/media/
├── Movies/
│   ├── The.Matrix.1999.mkv
│   ├── The.Matrix.1999.nfo          ← sidecar (optional)
│   └── The.Matrix.1999-poster.jpg   ← artwork (optional)
│
├── Music/
│   └── Albums/
│       └── <Genre>/
│           └── <Artist>/
│               └── <Album>/
│                   ├── album.nfo    ← sidecar (optional, album level)
│                   ├── cover.jpg    ← artwork (optional)
│                   └── [Disk N/]
│                       └── 01 - Track Title.mp3
│
└── Shows/
    └── <Show Name>/
        ├── tvshow.nfo               ← series-level sidecar (optional)
        ├── poster.jpg
        └── Season N/
            ├── ShowName S01E01.mkv
            └── ShowName S01E01.nfo  ← episode sidecar (optional)
```

Sidecars are optional throughout — anything missing falls back to path-derived
metadata. Set `SCAN_NFO=true` to enable sidecar and artwork discovery.

---

## Quick start

```bash
# Edit docker-composer-example.yaml — set BASE_URL to your server hostname/IP
podman-compose up -d
podman-compose logs -f
```

Then open `http://your-server:8888/admin/`.

### Manual run

```bash
podman run -d \
  --name m3u-scanner \
  --restart unless-stopped \
  -p 8888:8888 \
  -v /mnt/media:/media:rw \
  -v /home/user/m3u-scanner/cache:/data \
  -v /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro \
  -e SCAN_PATH=/media \
  -e SCAN_TYPES="movies music shows" \
  -e SCAN_NFO=true \
  -e BASE_URL="http://your-server:8888" \
  -e CACHE_PATH=/data/cache.db \
  -e TZ="America/New_York" \
  localhost/m3u-scanner:latest
```

The media mount must be **read-write** — the admin UI writes `.nfo` sidecars
and, for music, embedded tags back into the library. Mount it `:ro` if you only
want playlists and never intend to edit.

---

## Admin interface

`http://your-server:8888/admin/`

Browse and filter the scanned library by type, search across titles, artists,
and series, and open any entry in a modal to edit its metadata. Saving writes a
Jellyfin/Kodi-compatible `.nfo` sidecar beside the media file (album level for
music) and immediately re-reads that one file — no full rescan needed.

Poster and fanart accept either a local path or a remote `http(s)` URL. Local
artwork is discovered automatically from sibling files (`poster.jpg`,
`folder.jpg`, `cover.jpg`, `<name>-poster.jpg`, `fanart.jpg`).

Music entries get an extra **Write File Tags** action that pushes the edited
values into the audio file's own tag container along with cover art. This
modifies the file on disk and merges rather than clears, so MusicBrainz IDs,
ReplayGain, and other existing tags are preserved.

> **No authentication.** The admin interface is intended for trusted local
> networks. Do not expose port 8888 to the internet without putting an
> authenticating reverse proxy in front of it.

---

## Endpoints

### Playlists and media

| Endpoint | Method | Description |
|---|---|---|
| `/all.m3u` | GET | All media types combined |
| `/movies.m3u` | GET | Movies only |
| `/music.m3u` | GET | Music only |
| `/shows.m3u` | GET | Shows only |
| `/images.m3u` | GET | Images only |
| `/media/...` | GET | Direct media and artwork file serving |
| `/admin/` | GET | Admin interface |
| `/health` | GET | JSON: status, entry count, last scan, ffprobe, playlists |
| `/rescan` | POST | Trigger a rescan; add `?type=` to scope it |

### API

| Endpoint | Method | Description |
|---|---|---|
| `/api/health` | GET | Same payload as `/health` |
| `/api/entries` | GET | List entries; `type=`, `q=`, `limit=`, `offset=` |
| `/api/entry/{id}` | GET | Full entry plus its sidecar path |
| `/api/entry/{id}` | PUT | Save metadata and write the `.nfo` sidecar |
| `/api/entry/{id}/rescan` | POST | Re-read one file's sidecar and artwork |
| `/api/entry/{id}/tags` | POST | Write metadata into the audio file (music only) |
| `/api/rescan` | POST | Trigger a rescan; add `?type=` to scope it |

```bash
# Health check
curl http://your-server:8888/health

# Full rescan
curl -X POST http://your-server:8888/rescan

# Rescan a single type
curl -X POST "http://your-server:8888/rescan?type=movies"

# Rescan several types
curl -X POST "http://your-server:8888/rescan?type=movies&type=shows"
```

Entry IDs are opaque and derived from the media path. Edits address entries by
ID rather than by a caller-supplied path, so a request can never resolve to a
file outside the scanned library.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SCAN_PATH` | `/media` | Root path to scan |
| `SCAN_TYPES` | `movies music shows` | Space-separated: `music` `images` `movies` `shows` |
| `SCAN_NFO` | `false` | Read `.nfo` sidecars and sibling artwork for movies, shows, and music |
| `BASE_URL` | *(empty)* | Rewrites file paths to HTTP URLs in M3U output |
| `SERVE_PORT` | `8888` | HTTP listen port |
| `CRON_SCHEDULE` | `*/30 * * * *` | 5-field cron for scheduled rescans |
| `TZ` | `UTC` | Timezone for cron scheduling |
| `FFPROBE_PATH` | `/usr/local/bin/ffprobe` | Path to ffprobe inside the container |
| `CACHE_PATH` | `/data/cache.db` | SQLite cache database path |

### `CRON_SCHEDULE` syntax

Standard 5-field: `minute hour dom month dow`

```
*/30 * * * *  → every 30 minutes (default)
0 3 * * *     → 3:00 AM daily
0 */6 * * *   → every 6 hours
@daily        → midnight daily
@hourly       → top of every hour
```

Cached scans stat unchanged files without re-parsing them, so a short interval
is inexpensive on local storage. Lengthen it if your library lives on a network
mount where per-file `stat` calls are costly.

---

## Metadata resolution

Values are filled in this order, and enrichment is strictly additive — nothing
already set is overwritten:

1. **Path structure** — artist, album, series, season, episode, movie title, year
2. **Embedded tags** (music) — via go-taglib
3. **ffprobe** — video and audio duration
4. **`.nfo` sidecar** — plot, tagline, cast, ratings, genres, studios, IDs, artwork
5. **Sibling artwork** — poster and fanart discovered from adjacent image files

Shows additionally merge series-level values from `tvshow.nfo`, walking up from
the episode's directory, so genres and artwork defined once at the series root
apply to every episode.

`.nfo` files are written atomically via a temporary file and rename, so a failed
write can never destroy an existing sidecar.

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

Schema upgrades drop and rebuild the entries table rather than migrating it —
new metadata columns cannot be back-filled for files whose `mtime` and `size`
are unchanged, so a full re-enrich is the only correct outcome. Expect one slow
scan after upgrading; the log announces it.

---

## ffprobe

Not baked into the image — mount from the host:

```yaml
volumes:
  - /usr/local/bin/ffprobe:/usr/local/bin/ffprobe:ro
```

Find your path with `which ffprobe`. Install with `sudo apt install ffmpeg` if missing.

**Without ffprobe:** audio duration comes from go-taglib (accurate), video
duration falls back to the `.nfo` runtime when present, otherwise `-1`.

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

Support for extended attributes varies. `tvg-logo`, `tvg-name`, `tvg-id`, and
`group-title` are widely honoured; the richer attributes are written for players
and tooling that read them.

---

## Building

The admin UI is compiled by Tailwind and embedded into the binary, so assets
must be built before `go build`:

```bash
npm install
npm run build
go build -o m3u-scanner .
```

The container build does this automatically in a separate stage — no local
Node installation required:

```bash
podman build -t m3u-scanner .
```

---

## Project structure

```
m3u-scanner/
├── main.go
├── go.mod
├── package.json          — Tailwind + terser asset build
├── tailwind.config.js
├── Dockerfile
├── docker-composer-example.yaml
├── LICENSE
└── internal/
    ├── cache/       — SQLite cache (ncruces/go-sqlite3, no CGO)
    ├── config/      — env var loading + validation
    ├── cron/        — minimal 5-field cron scheduler (no external dep)
    ├── extensions/  — type→extension map + known subfolder names
    ├── meta/        — ffprobe, go-taglib, NFO read/write, tag write-through
    ├── models/      — MediaEntry struct, sort key, media type enum
    ├── parser/      — path-first metadata extraction
    ├── scanner/     — recursive walker + cache-aware scan loop
    ├── server/      — net/http: playlists, media, admin UI, JSON API
    │   └── static/  — admin interface source and built assets
    └── writer/      — builds M3U []byte from sorted entries
```

---

## License

MIT © 2026 [Kevin Pirnie](https://kevinpirnie.com/)
