"""
Metadata enrichment dispatcher.

Enrichment is additive only — path-derived fields are never overwritten.
Each enricher is responsible for:
  - Duration (all types except images)
  - Year (music, movies when guessit missed it)
  - Episode title via NFO (shows, opt-in)
"""

from ..models import MediaEntry


def enrich(entry: MediaEntry, parse_nfo: bool = False) -> MediaEntry:
    match entry.media_type:
        case "music":
            from .audio import enrich_audio
            enrich_audio(entry)
        case "movies" | "shows":
            from .video import enrich_video
            enrich_video(entry)
            if entry.media_type == "shows" and parse_nfo:
                from .nfo import enrich_nfo
                enrich_nfo(entry)
        case "images":
            from .image import enrich_image
            enrich_image(entry)
    return entry
