"""M3U extended playlist writer."""

from pathlib import Path

from .models import MediaEntry


class M3UWriter:
    def __init__(
        self,
        entries: list[MediaEntry],
        output_path: Path,
        use_relative: bool = False,
    ) -> None:
        self.entries = sorted(entries, key=lambda e: e.sort_key)
        self.output_path = output_path.resolve()
        self.use_relative = use_relative

    # ── Public ────────────────────────────────────────────────────────────────

    def write(self) -> None:
        self.output_path.parent.mkdir(parents=True, exist_ok=True)

        with self.output_path.open("w", encoding="utf-8") as fh:
            fh.write("#EXTM3U\n\n")
            for entry in self.entries:
                fh.write(self._format_entry(entry))
                fh.write("\n")

    # ── Private ───────────────────────────────────────────────────────────────

    def _format_entry(self, entry: MediaEntry) -> str:
        """
        Produce an #EXTINF line + path line for one entry.

        Format:
          #EXTINF:<duration> tvg-name="..." group-title="..." [extra attrs],<display title>
          /path/to/file.ext
        """
        attrs: dict[str, str] = {
            "tvg-name":    entry.tvg_name,
            "group-title": entry.group_title,
        }

        # Type-specific extra attributes
        match entry.media_type:
            case "music":
                if entry.artist:
                    attrs["tvg-artist"] = entry.artist
                if entry.album:
                    attrs["tvg-album"] = entry.album
                if entry.genre:
                    attrs["tvg-genre"] = entry.genre
                if entry.year:
                    attrs["tvg-year"] = entry.year
            case "shows":
                if entry.series:
                    attrs["tvg-show"] = entry.series
                if entry.season is not None:
                    attrs["tvg-season"] = str(entry.season)
                if entry.episode is not None:
                    attrs["tvg-episode"] = str(entry.episode)
            case "movies":
                if entry.year:
                    attrs["tvg-year"] = entry.year

        attrs_str = " ".join(
            f'{k}="{self._esc(v)}"' for k, v in attrs.items()
        )
        extinf = f"#EXTINF:{entry.duration} {attrs_str},{entry.display_title}"
        path_str = self._fmt_path(entry.path)

        return f"{extinf}\n{path_str}\n"

    def _fmt_path(self, path: Path) -> str:
        if self.use_relative:
            try:
                return str(path.resolve().relative_to(self.output_path.parent))
            except ValueError:
                pass  # paths on different drives / outside output dir
        return str(path.resolve())

    @staticmethod
    def _esc(value: str) -> str:
        """Escape double-quotes inside attribute values."""
        return value.replace('"', '\\"')
