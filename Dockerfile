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

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server


# ---------------------------------------------------------------------------
# Stage 3 — runtime
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime

# No default value on purpose: declaring "ARG TARGETARCH=amd64" would override
# the architecture BuildKit injects, and the arm64 image would get the amd64
# yt-dlp binary.
ARG TARGETARCH

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates ffmpeg tzdata curl \
 && rm -rf /var/lib/apt/lists/*

# yt-dlp is pinned to a fresh release: YouTube breaks older versions regularly.
# The standalone binary is published per architecture.
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) asset=yt-dlp_linux ;; \
      arm64) asset=yt-dlp_linux_aarch64 ;; \
      *) echo "unsupported arch ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /usr/local/bin/yt-dlp \
      "https://github.com/yt-dlp/yt-dlp/releases/latest/download/${asset}"; \
    chmod 0755 /usr/local/bin/yt-dlp; \
    yt-dlp --version

RUN useradd --uid 10001 --create-home --shell /usr/sbin/nologin ytdl \
 && mkdir -p /data \
 && chown -R ytdl:ytdl /data

COPY --from=build /out/server /usr/local/bin/server

ENV DATA_DIR=/data \
    PORT=8080 \
    YTDLP_BIN=/usr/local/bin/yt-dlp \
    FFMPEG_BIN=/usr/bin/ffmpeg \
    TZ=UTC

USER ytdl
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS "http://127.0.0.1:${PORT}/api/health" || exit 1

ENTRYPOINT ["/usr/local/bin/server"]
