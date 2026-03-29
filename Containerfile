# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM docker.io/library/golang:1.26.1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY . .

RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o m3u-scanner .

# ── Stage 2: Runtime ───────────────────────────────────────────────────────────
FROM docker.io/library/debian:bookworm-slim

LABEL org.opencontainers.image.title="m3u-scanner"
LABEL org.opencontainers.image.description="Recursive media library M3U playlist server"
LABEL org.opencontainers.image.source="https://github.com/kpirnie/m3u-scanner"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/m3u-scanner .

# /media  — bind-mount your media library (read-only recommended)
# /data   — persistent volume for the SQLite cache database
VOLUME ["/media", "/data"]

EXPOSE 8888

ENTRYPOINT ["/app/m3u-scanner"]
