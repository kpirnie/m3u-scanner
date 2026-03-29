# m3u-scanner

Recursively scans a local media library and writes an extended M3U playlist file.

Supports **music**, **movies**, **shows**, and **images**. Metadata is derived
primarily from the folder structure, enriched with duration and supplementary
tags from `mutagen`, `pymediainfo`, and `guessit`.

---

## Folder structure expectations

```
/media/
├── Movies/
│   └── The.Matrix.1999.mkv
├── Music/
│   └── Albums/
│       └── <Genre>/
│           └── <Artist>/
│               └── <Album>/
│                   └── [Disk N/]
│                       └── 01 - Track Title.mp3
└── Shows/
    └── <Show Name>/
        └── Season N/
            └── ShowName S01E01.mkv
```

---

## Requirements

### System (host or container)
- `libmediainfo0v5` — required by `pymediainfo`

### Python packages (auto-installed in the container)
| Package | Purpose |
|---|---|
| `mutagen` | Audio tag reading (duration, year) |
| `pymediainfo` | Video/audio duration via libmediainfo |
| `guessit` | Filename parsing for movies and episode numbers |
| `Pillow` | Image EXIF reading |

---

## Podman usage

### Build
```bash
chmod +x build.sh && ./build.sh
```

### Run — broad scan (all types in one pass)
```bash
podman run --rm \
  -v /media:/media:ro \
  -v /output:/output \
  m3u-scanner:latest \
  --path /media \
  --types movies music shows \
  --output /output/playlist.m3u
```

### Run — targeted scan (one type)
```bash
# Movies only
podman run --rm \
  -v /media/Movies:/media/Movies:ro \
  -v /output:/output \
  m3u-scanner:latest \
  --path /media/Movies --types movies --output /output/movies.m3u

# Music only
podman run --rm \
  -v /media/Music:/media/Music:ro \
  -v /output:/output \
  m3u-scanner:latest \
  --path /media/Music --types music --output /output/music.m3u
```

---

## CLI reference

```
usage: scanner.py [-h] --path PATH --types TYPE [TYPE ...] [--output OUTPUT]
                  [--no-meta] [--nfo] [--relative]
```

| Flag | Description |
|---|---|
| `--path` / `-p` | Root directory to scan *(required)* |
| `--types` / `-t` | One or more of: `music` `images` `movies` `shows` *(required)* |
| `--output` / `-o` | Output `.m3u` path (default: `./output.m3u`) |
| `--no-meta` | Skip library-based enrichment — faster, paths/filenames only |
| `--nfo` | Parse sibling `.nfo` XML sidecars for episode titles (shows only) |
| `--relative` | Write relative paths in the M3U instead of absolute |

---

## M3U output format

```m3u
#EXTM3U

#EXTINF:214 tvg-name="Pink Floyd - Another Brick In The Wall (pt 2)" group-title="music" tvg-artist="Pink Floyd" tvg-album="The Wall (Soundtrack)" tvg-genre="Classic Rock",Pink Floyd - Another Brick In The Wall (pt 2)
/media/Music/Albums/Classic Rock/Pink Floyd/The Wall (Soundtrack)/Disk 1/08 - Another Brick In The Wall (pt 2).mp3

#EXTINF:1380 tvg-name="Bluey - S01E02" group-title="shows" tvg-show="Bluey" tvg-season="1" tvg-episode="2",Bluey - S01E02
/media/Shows/Bluey/Season 1/Bluey S01E02.mkv

#EXTINF:8280 tvg-name="The Matrix (1999)" group-title="movies" tvg-year="1999",The Matrix (1999)
/media/Movies/The.Matrix.1999.mkv
```

### Sorting
- **Music** → artist → album → disc → track number
- **Shows** → series → season → episode
- **Movies** → title
- **Images** → title

---

## Project structure

```
m3u-scanner/
├── Containerfile          # Podman build file
├── requirements.txt
├── scanner.py             # CLI entrypoint
├── build.sh               # podman build helper
├── run.sh                 # example run commands
└── lib/
    ├── extensions.py      # type → extension map + known subfolder names
    ├── models.py          # MediaEntry dataclass
    ├── path_parser.py     # path-first metadata extraction (primary source)
    ├── scanner.py         # recursive file walker
    ├── writer.py          # M3U file builder
    └── metadata/
        ├── __init__.py    # dispatcher
        ├── audio.py       # mutagen (duration + year for music)
        ├── video.py       # pymediainfo (duration for video)
        ├── image.py       # Pillow EXIF (year for images)
        └── nfo.py         # Jellyfin .nfo XML sidecar parser (opt-in)
```
