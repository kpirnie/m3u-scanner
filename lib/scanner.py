"""Recursive media file scanner."""

import sys
from pathlib import Path

from .extensions import EXTENSIONS, TYPE_SUBFOLDERS
from .models import MediaEntry
from .path_parser import parse_music, parse_show, parse_movie, parse_image


class MediaScanner:
    def __init__(
        self,
        root: Path,
        types: list[str],
        enrich_meta: bool = True,
        parse_nfo: bool = False,
    ) -> None:
        self.root = root.resolve()
        self.types = types
        self.enrich_meta = enrich_meta
        self.parse_nfo = parse_nfo

    # ── Public ────────────────────────────────────────────────────────────────

    def scan(self) -> list[MediaEntry]:
        entries: list[MediaEntry] = []

        for media_type in self.types:
            type_root = self._resolve_type_root(media_type)
            if type_root is None:
                print(
                    f"[WARN] No '{media_type}' folder found under {self.root} "
                    f"— skipping (known names: {TYPE_SUBFOLDERS[media_type]})",
                    file=sys.stderr,
                )
                continue

            print(f"[INFO] Scanning {media_type} → {type_root}")
            files = self._collect_files(type_root, media_type)
            print(f"[INFO]   {len(files)} file(s) found")

            for file_path in files:
                entry = self._parse_file(file_path, media_type, type_root)
                if entry is None:
                    continue
                if self.enrich_meta:
                    self._enrich(entry)
                entries.append(entry)

        return entries

    # ── Private ───────────────────────────────────────────────────────────────

    def _resolve_type_root(self, media_type: str) -> Path | None:
        """
        Determine the actual root directory for a given media type.

        Targeted mode:
          The supplied --path folder name matches a known subfolder name for
          this type (e.g. --path /media/Movies --types movies).
          → return root as-is.

        Broad mode:
          Look for a direct child folder of root whose name is in the known
          list (e.g. --path /media → find /media/Movies).
          → return that subfolder.

        Single-type fallback:
          If only one type was requested and nothing matched, assume the caller
          aimed --path directly at the content.
          → return root as-is.
        """
        known = TYPE_SUBFOLDERS[media_type]

        # Targeted: root itself is the type folder
        if self.root.name in known:
            return self.root

        # Broad: look for a known direct child
        for name in known:
            candidate = self.root / name
            if candidate.is_dir():
                return candidate

        # Single-type fallback
        if len(self.types) == 1:
            return self.root

        return None

    def _collect_files(self, type_root: Path, media_type: str) -> list[Path]:
        exts = EXTENSIONS[media_type]
        return sorted(
            f for f in type_root.rglob("*")
            if f.is_file() and f.suffix.lower() in exts
        )

    def _parse_file(
        self,
        file_path: Path,
        media_type: str,
        type_root: Path,
    ) -> MediaEntry | None:
        try:
            match media_type:
                case "music":
                    return parse_music(file_path, type_root)
                case "shows":
                    return parse_show(file_path, type_root)
                case "movies":
                    return parse_movie(file_path)
                case "images":
                    return parse_image(file_path)
        except Exception as exc:
            print(f"[WARN] Could not parse {file_path}: {exc}", file=sys.stderr)
        return None

    def _enrich(self, entry: MediaEntry) -> None:
        try:
            from .metadata import enrich
            enrich(entry, parse_nfo=self.parse_nfo)
        except Exception as exc:
            print(f"[WARN] Metadata enrichment failed for {entry.path}: {exc}", file=sys.stderr)
