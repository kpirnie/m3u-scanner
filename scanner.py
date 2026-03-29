#!/usr/bin/env python3
"""M3U Scanner - Recursive media library scanner and M3U playlist generator."""

import argparse
import sys
from pathlib import Path

from lib.scanner import MediaScanner
from lib.writer import M3UWriter


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Recursively scan a media folder and generate an M3U playlist.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Scan entire media library in one pass
  python scanner.py --path /media --types movies music shows --output /output/all.m3u

  # Targeted: scan only movies
  python scanner.py --path /media/Movies --types movies --output /output/movies.m3u

  # Targeted: scan music only, skip metadata enrichment for speed
  python scanner.py --path /media/Music --types music --output /output/music.m3u --no-meta

  # Shows with NFO sidecar episode title enrichment
  python scanner.py --path /media/Shows --types shows --output /output/shows.m3u --nfo

  # Podman (from host):
  podman run --rm \\
    -v /media:/media:ro \\
    -v /output:/output \\
    m3u-scanner:latest \\
    --path /media --types movies music shows --output /output/playlist.m3u
        """,
    )

    parser.add_argument(
        "--path", "-p",
        type=Path,
        required=True,
        help="Root path to scan",
    )
    parser.add_argument(
        "--types", "-t",
        nargs="+",
        choices=["music", "images", "movies", "shows"],
        required=True,
        metavar="TYPE",
        help="One or more media types: music, images, movies, shows",
    )
    parser.add_argument(
        "--output", "-o",
        type=Path,
        default=Path("./output.m3u"),
        help="Output M3U file path (default: ./output.m3u)",
    )
    parser.add_argument(
        "--no-meta",
        action="store_true",
        help="Skip metadata enrichment — faster, uses filenames/paths only",
    )
    parser.add_argument(
        "--nfo",
        action="store_true",
        help="Parse sibling .nfo sidecar files for episode title enrichment (shows only)",
    )
    parser.add_argument(
        "--relative",
        action="store_true",
        help="Write paths relative to the M3U file location (default: absolute)",
    )

    return parser.parse_args()


def main() -> None:
    args = parse_args()

    if not args.path.exists():
        print(f"[ERROR] Path does not exist: {args.path}", file=sys.stderr)
        sys.exit(1)

    if not args.path.is_dir():
        print(f"[ERROR] Path is not a directory: {args.path}", file=sys.stderr)
        sys.exit(1)

    print(f"[INFO] Root path  : {args.path}")
    print(f"[INFO] Types      : {', '.join(args.types)}")
    print(f"[INFO] Output     : {args.output}")
    print(f"[INFO] Enrich meta: {'no' if args.no_meta else 'yes'}")
    if args.nfo:
        print("[INFO] NFO parsing: enabled")

    scanner = MediaScanner(
        root=args.path,
        types=args.types,
        enrich_meta=not args.no_meta,
        parse_nfo=args.nfo,
    )

    entries = scanner.scan()
    print(f"[INFO] Total entries: {len(entries)}")

    if not entries:
        print("[WARN] No media files found. Check --path and --types.", file=sys.stderr)
        sys.exit(0)

    writer = M3UWriter(
        entries=entries,
        output_path=args.output,
        use_relative=args.relative,
    )
    writer.write()
    print(f"[INFO] Done → {args.output}")


if __name__ == "__main__":
    main()
