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
# Stage 3 — fetch a static ffmpeg (no shared library dependencies)
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS ffmpeg
ARG TARGETARCH

RUN apk add --no-cache curl ca-certificates

# A fully static build, so the final image needs no shared libraries at all.
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) asset=ffmpeg-master-latest-linux64-gpl ;; \
      arm64) asset=ffmpeg-master-latest-linuxarm64-gpl ;; \
      *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/ffmpeg.tar.xz \
      "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/${asset}.tar.xz"; \
    mkdir -p /tmp/ffmpeg; \
    tar -xf /tmp/ffmpeg.tar.xz -C /tmp/ffmpeg; \
    FF=$(find /tmp/ffmpeg -name 'ffmpeg' -type f | head -n 1); \
    cp "$FF" /usr/local/bin/ffmpeg; \
    chmod 0755 /usr/local/bin/ffmpeg; \
    /usr/local/bin/ffmpeg -version | head -n 1


# ---------------------------------------------------------------------------
# Stage 4 — runtime: Alpine, three static binaries, nothing else
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
 && rm -rf /var/cache/apk/*

# yt-dlp is pinned to a fresh release: YouTube breaks older versions regularly.
# The standalone binary bundles its own Python, so no interpreter is needed.
ARG TARGETARCH
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) asset=yt-dlp_linux ;; \
      arm64) asset=yt-dlp_linux_aarch64 ;; \
      *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    wget -qO /usr/local/bin/yt-dlp \
      "https://github.com/yt-dlp/yt-dlp/releases/latest/download/${asset}"; \
    chmod 0755 /usr/local/bin/yt-dlp; \
    yt-dlp --version

RUN adduser -D -u 10001 ytdl \
 && mkdir -p /data /library \
 && chown -R ytdl:ytdl /data /library

# Only three files make up the application.
COPY --from=build /out/server /usr/local/bin/server
COPY --from=ffmpeg /usr/local/bin/ffmpeg /usr/bin/ffmpeg

ENV DATA_DIR=/data \
    LIBRARY_DIR=/library \
    PORT=8080 \
    YTDLP_BIN=/usr/local/bin/yt-dlp \
    FFMPEG_BIN=/usr/bin/ffmpeg \
    TZ=UTC

USER ytdl
WORKDIR /data
VOLUME ["/data", "/library"]
EXPOSE 8080

# Alpine's busybox wget handles the health check without a separate package.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/usr/local/bin/server"]
