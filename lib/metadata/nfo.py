"""
NFO sidecar parser — opt-in via --nfo flag.

Jellyfin writes XML .nfo files alongside each episode file.
This enricher looks for a sibling .nfo, parses it, and fills in
episode_title (and year if missing).  It then rebuilds display_title
and tvg_name to include the episode title.

The .nfo is never required — missing or malformed files are silently ignored.
"""

import xml.etree.ElementTree as ET
from ..models import MediaEntry


def enrich_nfo(entry: MediaEntry) -> None:
    nfo_path = entry.path.with_suffix(".nfo")
    if not nfo_path.exists():
        return

    try:
        tree = ET.parse(nfo_path)
        root = tree.getroot()

        # ── Episode title ─────────────────────────────────────────────────────
        if entry.episode_title is None:
            title_el = root.find("title")
            if title_el is not None and title_el.text:
                entry.episode_title = title_el.text.strip()

                # Rebuild display strings with the new title
                if entry.season is not None and entry.episode is not None:
                    ep_str = f"S{entry.season:02d}E{entry.episode:02d}"
                    entry.display_title = f"{entry.series} - {ep_str} - {entry.episode_title}"
                    entry.tvg_name = entry.display_title
                    entry.compute_sort_key()

        # ── Year from aired date ──────────────────────────────────────────────
        if entry.year is None:
            for tag in ("aired", "premiered", "year"):
                el = root.find(tag)
                if el is not None and el.text:
                    entry.year = el.text.strip()[:4]
                    break

    except Exception:
        pass
