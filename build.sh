#!/usr/bin/env bash
# Build the m3u-scanner container image with Podman.
set -euo pipefail

IMAGE="m3u-scanner:latest"

echo "[INFO] Building image: $IMAGE"
podman build -t "$IMAGE" .
echo "[INFO] Done — image: $IMAGE"
