"""
Path-first metadata parser.

The folder structure is the authoritative source for all descriptive metadata.
Library calls (mutagen, pymediainfo, guessit) are used only for:
  - Duration
  - Episode number / year when not derivable from the path
  - Episode title when --nfo is enabled (handled separately in metadata/nfo.py)
"""

import re
from pathlib import Path

from .models import MediaEntry

# ── Regex patterns ────────────────────────────────────────────────────────────

# Disc subfolder: "Disk 1", "Disc 2", "CD1", "CD 2", etc.
_DISC_FOLDER_RE = re.compile(r'^(?:Dis[ck]|CD)\s*(\d+)$', re.IGNORECASE)

# Season subfolder: "Season 1", "Season 01", "Series 2", "S01"
_SEASON_FOLDER_RE = re.compile(r'^(?:Season|Series|S)\s*(\d+)$', re.IGNORECASE)

# Track filename patterns
#   "01 - Song Title"  or  "01. Song Title"
_TRACK_RE = re.compile(r'^(\d+)\s*[-\.]\s*(.+)$')
#   "1-01 Song Title"  (disc-track prefix)
_DISC_TRACK_RE = re.compile(r'^(\d+)-(\d+)\s+(.+)$')


# ── Public parsers ────────────────────────────────────────────────────────────

def parse_music(file_path: Path, music_root: Path) -> MediaEntry:
    """
    Derive metadata from path structure:
      <music_root> / Albums / <Genre> / <Artist> / <Album> / [Disk N /] <track file>

    depth (relative parts):
      0 → "Albums"   (skip)
      1 → Genre
      2 → Artist
      3 → Album
      4 → Either a Disk subfolder OR the file
      5 → File (when disc folder is present)
    """
    try:
        rel_parts = file_path.relative_to(music_root).parts
    except ValueError:
        rel_parts = file_path.parts

    genre  = rel_parts[1] if len(rel_parts) > 1 else "Unknown Genre"
    artist = rel_parts[2] if len(rel_parts) > 2 else "Unknown Artist"
    album  = rel_parts[3] if len(rel_parts) > 3 else "Unknown Album"

    # Detect optional disc subfolder at depth 4
    disc: int | None = None
    if len(rel_parts) >= 6:
        disc_match = _DISC_FOLDER_RE.match(rel_parts[4])
        if disc_match:
            disc = int(disc_match.group(1))

    # Parse track number + title from filename stem
    stem = file_path.stem
    track: int | None = None
    title = stem

    disc_track_match = _DISC_TRACK_RE.match(stem)
    track_match = _TRACK_RE.match(stem)

    if disc_track_match:
        disc  = disc or int(disc_track_match.group(1))
        track = int(disc_track_match.group(2))
        title = disc_track_match.group(3).strip()
    elif track_match:
        track = int(track_match.group(1))
        title = track_match.group(2).strip()

    display_title = f"{artist} - {title}"

    entry = MediaEntry(
        path=file_path,
        media_type="music",
        group_title="music",
        display_title=display_title,
        tvg_name=display_title,
        artist=artist,
        genre=genre,
        album=album,
        disc=disc,
        track=track,
    )
    entry.compute_sort_key()
    return entry


def parse_show(file_path: Path, shows_root: Path) -> MediaEntry:
    """
    Derive metadata from path structure:
      <shows_root> / <Show Name> / <Season N> / <filename>

    Episode number is extracted from the filename via guessit (SxxExx pattern).
    """
    try:
        rel_parts = file_path.relative_to(shows_root).parts
    except ValueError:
        rel_parts = file_path.parts

    series = rel_parts[0] if len(rel_parts) > 0 else "Unknown Show"
    season: int | None = None

    if len(rel_parts) > 1:
        season_match = _SEASON_FOLDER_RE.match(rel_parts[1])
        if season_match:
            season = int(season_match.group(1))

    # Extract episode number from filename
    episode: int | None = None
    episode_title: str | None = None

    try:
        from guessit import guessit  # type: ignore
        info = guessit(file_path.name)
        raw_ep = info.get("episode")
        if raw_ep is not None:
            episode = int(raw_ep)
        raw_title = info.get("episode_title")
        if raw_title:
            episode_title = str(raw_title).strip()
    except Exception:
        pass

    # Build display / tvg strings
    if season is not None and episode is not None:
        ep_str = f"S{season:02d}E{episode:02d}"
    else:
        ep_str = file_path.stem  # last-resort fallback

    if episode_title:
        display_title = f"{series} - {ep_str} - {episode_title}"
    else:
        display_title = f"{series} - {ep_str}"

    entry = MediaEntry(
        path=file_path,
        media_type="shows",
        group_title="shows",
        display_title=display_title,
        tvg_name=display_title,
        series=series,
        season=season,
        episode=episode,
        episode_title=episode_title,
    )
    entry.compute_sort_key()
    return entry


def parse_movie(file_path: Path) -> MediaEntry:
    """
    Derive metadata from filename alone via guessit.
    e.g. "The.Matrix.1999.mkv" → title="The Matrix", year="1999"
    """
    title = file_path.stem
    year: str | None = None

    try:
        from guessit import guessit  # type: ignore
        info = guessit(file_path.name)
        parsed_title = info.get("title")
        if parsed_title:
            title = str(parsed_title)
        parsed_year = info.get("year")
        if parsed_year:
            year = str(parsed_year)
    except Exception:
        pass

    display_title = f"{title} ({year})" if year else title

    entry = MediaEntry(
        path=file_path,
        media_type="movies",
        group_title="movies",
        display_title=display_title,
        tvg_name=display_title,
        year=year,
    )
    entry.compute_sort_key()
    return entry


def parse_image(file_path: Path) -> MediaEntry:
    """Simple image entry — title from cleaned filename stem."""
    raw = file_path.stem.replace("_", " ").replace(".", " ").replace("-", " ")
    title = " ".join(word.capitalize() for word in raw.split())

    entry = MediaEntry(
        path=file_path,
        media_type="images",
        group_title="images",
        display_title=title,
        tvg_name=title,
        duration=-1,
    )
    entry.compute_sort_key()
    return entry
