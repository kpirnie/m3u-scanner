"""
Audio metadata enrichment using mutagen.

Populates: duration, year
Never touches: artist, album, genre, track (all from path).
"""

from ..models import MediaEntry


def enrich_audio(entry: MediaEntry) -> None:
    try:
        from mutagen import File as MutagenFile  # type: ignore

        audio = MutagenFile(entry.path)
        if audio is None:
            return

        # ── Duration ──────────────────────────────────────────────────────────
        if hasattr(audio, "info") and hasattr(audio.info, "length"):
            entry.duration = int(audio.info.length)

        # ── Year (enrichment only — never overwrites path-derived value) ──────
        if entry.year is not None:
            return

        tags = audio.tags
        if not tags:
            return

        # ID3 tags (MP3)
        for id3_tag in ("TDRC", "TYER", "TDOR"):
            val = tags.get(id3_tag)
            if val:
                entry.year = str(val)[:4]
                return

        # Vorbis comment / FLAC
        for key in ("date", "year", "DATE", "YEAR"):
            val = tags.get(key)
            if val:
                entry.year = str(val[0])[:4]
                return

        # MP4 / M4A
        mp4_date = tags.get("\xa9day")
        if mp4_date:
            entry.year = str(mp4_date[0])[:4]

    except Exception:
        pass  # Graceful degradation — duration stays -1, year stays None
