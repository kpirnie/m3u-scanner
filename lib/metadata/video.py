"""
Video metadata enrichment using pymediainfo.

Populates: duration
pymediainfo requires libmediainfo0v5 (installed in the container).
"""

from ..models import MediaEntry


def enrich_video(entry: MediaEntry) -> None:
    try:
        from pymediainfo import MediaInfo  # type: ignore

        info = MediaInfo.parse(entry.path)
        for track in info.tracks:
            if track.track_type == "General" and track.duration:
                # pymediainfo returns duration in milliseconds
                entry.duration = int(float(track.duration) / 1000)
                return

    except Exception:
        pass  # Graceful degradation — duration stays -1
