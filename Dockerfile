# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------------
# Stage 1 — build the Vue frontend
# ---------------------------------------------------------------------------
FROM oven/bun:1-alpine AS web
WORKDIR /src/web

COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile || bun install

COPY web/ ./
RUN bun run build


# ---------------------------------------------------------------------------
# Stage 2 — build the Go binary with the frontend embedded
# ---------------------------------------------------------------------------
FROM golang:1.23-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
COPY --from=web /src/web/dist ./internal/api/dist

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/api.Version=${VERSION}" \
    -o /out/server ./cmd/server


# ---------------------------------------------------------------------------
# Stage 3 - runtime: Alpine with ffmpeg and yt-dlp from packages
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

# ffmpeg from Alpine includes the codecs yt-dlp needs (libx264, libmp3lame,
# libvpx, libopus, ...). yt-dlp is installed with pip so the latest release
# is used rather than the (often stale) Alpine package.
RUN apk add --no-cache \
      ca-certificates \
      tzdata \
      ffmpeg \
      python3 \
      py3-pip \
 && pip install --no-cache-dir --break-system-packages yt-dlp \
 && ln -sf /usr/bin/yt-dlp /usr/local/bin/yt-dlp \
 && rm -rf /var/cache/apk/* /root/.cache/pip

RUN adduser -D -u 10001 ytdl \
 && mkdir -p /data /library

# su-exec drops privileges the way gosu does, but it is a tiny static binary
# that fits in an Alpine image.
RUN apk add --no-cache su-exec

# Only two files make up the application.
COPY --from=build /out/server /usr/local/bin/server
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

ENV DATA_DIR=/data \
    LIBRARY_DIR=/library \
    PORT=8080 \
    YTDLP_BIN=/usr/local/bin/yt-dlp \
    FFMPEG_BIN=/usr/bin/ffmpeg \
    TZ=UTC

WORKDIR /data
VOLUME ["/data", "/library"]
EXPOSE 8080

# Alpine's busybox wget handles the health check without a separate package.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
