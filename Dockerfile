# ── Stage 1: Assets ───────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM docker.io/library/node:26-alpine AS assets

WORKDIR /build

COPY package.json tailwind.config.js ./
RUN npm install

COPY internal/server/static ./internal/server/static

RUN npm run build && npm run clean

# ── Stage 2: Build ────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.26.5-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git

WORKDIR /build

COPY . .

COPY --from=assets /build/internal/server/static/admin.css ./internal/server/static/admin.css
COPY --from=assets /build/internal/server/static/admin.js  ./internal/server/static/admin.js

RUN go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o m3u-scanner .

# ── Stage 3: Runtime ───────────────────────────────────────────────────────────
FROM docker.io/library/debian:trixie-slim

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
