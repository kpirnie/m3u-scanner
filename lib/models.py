"""MediaEntry dataclass — the single data model used throughout the pipeline."""

from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class MediaEntry:
    # ── Required ─────────────────────────────────────────────────────────────
    path: Path
    media_type: str       # "music" | "images" | "movies" | "shows"
    group_title: str      # written as group-title= in M3U
    display_title: str    # the human-readable label after the comma on #EXTINF
    tvg_name: str         # written as tvg-name= in M3U

    # ── Common ───────────────────────────────────────────────────────────────
    duration: int = -1    # seconds; -1 = unknown / not applicable
    year: str | None = None

    # ── Music ─────────────────────────────────────────────────────────────────
    artist: str | None = None
    genre: str | None = None
    album: str | None = None
    disc: int | None = None
    track: int | None = None

    # ── Shows ─────────────────────────────────────────────────────────────────
    series: str | None = None
    season: int | None = None
    episode: int | None = None
    episode_title: str | None = None

    # ── Internal ──────────────────────────────────────────────────────────────
    sort_key: tuple = field(default_factory=tuple, compare=False, repr=False)

    # ─────────────────────────────────────────────────────────────────────────

    def compute_sort_key(self) -> None:
        """
        Populate sort_key with a tuple that produces correct ordering:
          music  → artist / album / disc / track / title
          shows  → series / season / episode
          movies → title
          images → title
        """
        match self.media_type:
            case "music":
                self.sort_key = (
                    self.group_title,
                    (self.artist or "").lower(),
                    (self.album or "").lower(),
                    self.disc or 0,
                    self.track or 0,
                    self.display_title.lower(),
                )
            case "shows":
                self.sort_key = (
                    self.group_title,
                    (self.series or "").lower(),
                    self.season or 0,
                    self.episode or 0,
                )
            case _:
                self.sort_key = (
                    self.group_title,
                    self.display_title.lower(),
                )
