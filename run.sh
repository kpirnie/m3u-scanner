#!/usr/bin/env bash
# Example run script — edit MEDIA_PATH and OUTPUT_PATH to suit your setup.
set -euo pipefail

IMAGE="m3u-scanner:latest"
MEDIA_PATH="/media"          # mounted read-only inside the container
OUTPUT_PATH="$(pwd)/output"  # where the .m3u file will be written

mkdir -p "$OUTPUT_PATH"

podman run --rm \
  -v "${MEDIA_PATH}:/media:ro" \
  -v "${OUTPUT_PATH}:/output" \
  "$IMAGE" \
  --path /media \
  --types movies music shows \
  --output /output/playlist.m3u

# ── Other usage examples ───────────────────────────────────────────────────────

# Targeted scan — movies only
# podman run --rm \
#   -v /media/Movies:/media/Movies:ro \
#   -v "$(pwd)/output":/output \
#   "$IMAGE" \
#   --path /media/Movies --types movies --output /output/movies.m3u

# Targeted scan — music only, no metadata enrichment
# podman run --rm \
#   -v /media/Music:/media/Music:ro \
#   -v "$(pwd)/output":/output \
#   "$IMAGE" \
#   --path /media/Music --types music --output /output/music.m3u --no-meta

# Shows with NFO sidecar episode titles
# podman run --rm \
#   -v /media/Shows:/media/Shows:ro \
#   -v "$(pwd)/output":/output \
#   "$IMAGE" \
#   --path /media/Shows --types shows --output /output/shows.m3u --nfo

# All types + NFO + relative paths
# podman run --rm \
#   -v /media:/media:ro \
#   -v "$(pwd)/output":/output \
#   "$IMAGE" \
#   --path /media --types movies music shows images \
#   --output /output/all.m3u --nfo --relative
