"""
Image metadata enrichment using Pillow.

Populates: year (from EXIF DateTimeOriginal)
Duration stays -1 for images — M3U players that support images treat -1 correctly.
"""

from ..models import MediaEntry


def enrich_image(entry: MediaEntry) -> None:
    try:
        from PIL import Image  # type: ignore
        from PIL.ExifTags import TAGS  # type: ignore

        with Image.open(entry.path) as img:
            exif_raw = img._getexif()
            if not exif_raw:
                return

            for tag_id, value in exif_raw.items():
                tag_name = TAGS.get(tag_id, "")
                if tag_name == "DateTimeOriginal" and entry.year is None:
                    # Format: "YYYY:MM:DD HH:MM:SS"
                    entry.year = str(value)[:4]
                    break

    except Exception:
        pass
