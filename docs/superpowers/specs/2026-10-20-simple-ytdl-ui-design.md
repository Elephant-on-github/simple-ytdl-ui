# simple-ytdl-ui — Design

**Date:** 2026-10-20
**Status:** approved by requester (build-first, no questions)

## Intent

A self-hosted, single-container web app that turns YouTube URLs (single videos and
playlists) into downloadable files, with an optional Google sign-in that lets the user
browse and sync their own YouTube playlists. A prominent **Music** toggle switches
downloads to "best quality MP3" (audio-only, extracted at highest bitrate, with cover art
and tags embedded). The UI must look modern and deliberate, not like a template.

The user (a developer) distributes the app as a Docker image on **GHCR**, pulling it
straight from `ghcr.io/elephant-on-github/simple-ytdl-ui`.

## Why

The existing tools for this are either CLI-only (no UI, no library view) or heavy
all-in-one media managers. This app is deliberately narrow: paste a URL, get a file, keep
a browsable local library. Google sign-in is *optional* — the app is fully useful with no
Google credentials configured at all.

## Goals

- One Docker image that contains the Go backend, the built Vue frontend, `yt-dlp` and
  `ffmpeg`. One container, one volume, one port.
- Download a single YouTube video or an entire playlist, as video or as best-quality MP3.
- A live queue with per-item progress, speed, ETA, cancel.
- A browsable, playable library of everything downloaded (HTML5 audio player).
- Optional Google OAuth; when signed in, list the user's own playlists, preview their
  items, and sync a playlist into the library. Re-syncing must not duplicate files.
- Modern, dark, editorial UI that a developer would happily run on a second monitor.

## Non-goals

- No transcoding beyond ffmpeg's MP3 extraction; no video re-encode.
- No multi-user accounts, no permissions, no public sharing links.
- No support for non-YouTube sources beyond what `yt-dlp` handles generically
  (the code is source-agnostic, but the UI copy is YouTube-centric).
- No metadata scraping/fixup service (no Spotify/Discogs lookups).

## Anti-goals (things that would make this a failure)

- A Docker image that cannot pull from GHCR with a single `docker pull`.
- A UI that requires a Google account to be useful.
- Silent failures: a failed download must say *why* in the UI.
- Copy-paste-unfriendly output: downloaded files must be named `Artist - Title [id].mp3`.

## Constraints

- Backend: Go (stdlib HTTP server, `net/http` with Go 1.22+ pattern routing). Only one
  third-party dependency: `golang.org/x/oauth2`. YouTube Data API v3 is called over
  plain REST (no heavy Google API client library) so it can be tested with `httptest`.
- Frontend: Vue 3 (`<script setup>`, TypeScript), Vite, Pinia, Vue Router. No UI kit.
- Persistence: JSON files under `DATA_DIR`, written atomically. No database.
- Target image size matters: use `alpine`, install `yt-dlp`/`ffmpeg` from Alpine
  packages, run as a non-root user.

## Architecture

Three stages in one `Dockerfile`:

1. `web` stage — `node:22-alpine` runs `bun install` + `bun run build` in `web/`,
   producing `web/dist`.
2. `build` stage — `golang:1.23-alpine` copies `web/dist` into `backend/internal/api/dist/`
   and builds a static binary with the frontend embedded (`//go:embed`).
3. `runtime` stage — `alpine:3.20` with `ffmpeg`, `yt-dlp`, `ca-certificates`, `tzdata`;
   the single static binary; user `ytdl`; `/data` volume.

The Go binary serves the API under `/api/*` and the embedded SPA for everything else
(so a deep link like `/library` works on refresh).

### Backend packages

| Package | Responsibility |
|---|---|
| `internal/config` | Parse env vars into a `Config` struct. `GoogleEnabled()` is true only when both client id and secret are set. |
| `internal/store` | JSON-backed persistence for `library.json`, `jobs.json`, `google.json`. Atomic writes (temp file + rename), `0600` for the token file. |
| `internal/ytdlp` | Thin wrapper around the `yt-dlp` binary: `Probe` (metadata JSON), `Download` (runs with a progress template, reports progress through a channel, returns the produced file path). Also owns argument construction per mode. |
| `internal/download` | Job manager: bounded worker pool, job state machine, cancellation, fan-out of playlists into individual jobs. |
| `internal/youtube` | Google OAuth2 (authorization URL, code exchange, token refresh) and YouTube Data API v3 (`playlists.list`, `playlistItems.list`). |
| `internal/api` | HTTP handlers, JSON helpers, static SPA serving with range requests for media. |

### Data model

The state lives in `DATA_DIR`; the media itself can live in `LIBRARY_DIR` (its own
volume) or next to the state when that is not set.

`library.json` → `{"items":[Item]}` where

```go
type Item struct {
    ID          string    `json:"id"`          // "yt_<videoID>_<mode>" or "pl_<videoID>_<mode>"
    VideoID     string    `json:"videoId"`
    Title       string    `json:"title"`
    Artist      string    `json:"artist"`
    Duration    int       `json:"duration"`     // seconds
    File        string    `json:"file"`         // absolute path on disk
    Cover       string    `json:"cover"`        // absolute path on disk (optional)
    Mime        string    `json:"mime"`
    Size        int64     `json:"size"`
    Mode        string    `json:"mode"`         // "video" | "music"
    SourceURL   string    `json:"sourceUrl"`
    Playlist    string    `json:"playlist"`     // playlist title, empty for singles
    AddedAt     time.Time `json:"addedAt"`
}
```

The id carries the mode because the same video can be in the library twice — once as
MP3, once as a video — and the two must not collide.

`jobs.json` → `{"jobs":[Job]}` where

```go
type Job struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    URL       string    `json:"url"`
    VideoID   string    `json:"videoId"`
    Mode      string    `json:"mode"`
    Status    string    `json:"status"`   // queued|running|completed|failed|canceled
    Progress  float64   `json:"progress"` // 0..100
    Downloaded int64    `json:"downloaded"`
    Total     int64     `json:"total"`
    Speed     float64   `json:"speed"`     // bytes/sec
    ETA       int       `json:"eta"`       // seconds
    Error     string    `json:"error"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

`google.json` → `{"refreshToken":"...","email":"..."}` (mode `0600`).

### API contract

All responses are JSON except media endpoints (which stream bytes).

| Method | Path | Behaviour |
|---|---|---|
| GET | `/api/health` | `{"status":"ok","version":"…"}` |
| GET | `/api/config` | `{"googleEnabled":bool,"defaultMode":"video","concurrent":int}` |
| GET | `/api/auth/google` | `{"url":"https://accounts.google.com/…"}`; 503 if Google is not configured |
| GET | `/api/auth/google/callback?code=&state=` | Exchanges the code, stores the refresh token, 302 → `/` |
| GET | `/api/auth/session` | `{"signedIn":bool,"email":"…"}` |
| POST | `/api/auth/logout` | `{"ok":true}` — deletes the stored token |
| GET | `/api/playlists` | `{"items":[{"id","title","itemCount","thumb"}]}`; 401 if not signed in |
| GET | `/api/playlists/{id}/items` | `{"items":[{"id","title","thumb","channel","duration"}],"nextPageToken":"…"}`; accepts `?pageToken=` |
| POST | `/api/downloads` | Body `{"url":"…","mode":"video|music","playlist":true|false}` → `{"jobs":[Job,…]}`; 400 on bad URL |
| GET | `/api/jobs` | `{"items":[Job]}` (newest first) |
| POST | `/api/jobs/{id}/cancel` | `{"ok":true}`; 404 unknown job |
| GET | `/api/library` | `{"items":[Item]}` (newest first) |
| DELETE | `/api/library/{id}` | `{"ok":true}` — deletes the file from disk and the record |
| GET | `/api/library/{id}/stream` | Audio/video bytes with `Range` support |
| GET | `/api/library/{id}/cover` | Cover image bytes |

### `yt-dlp` invocation

Probe (single): `yt-dlp -J --no-playlist --no-warnings <url>`
Probe (playlist): `yt-dlp -J --flat-playlist --no-warnings <url>`

Download, video mode:

```
yt-dlp --no-playlist --newline --no-warnings \
  --progress-template "dl|%(progress.status)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s" \
  -f "bv*+ba/b" --merge-output-format mp4 --write-thumbnail \
  --ffmpeg-location <ffmpeg> \
  -o "<dir>/%(title).150s [%(id)s].%(ext)s" <url>
```

Download, music mode:

```
yt-dlp --no-playlist --newline --no-warnings \
  --progress-template "dl|%(progress.status)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s" \
  -f "bestaudio/best" -x --audio-format mp3 --audio-quality 0 \
  --embed-thumbnail --embed-metadata --write-thumbnail \
  --ffmpeg-location <ffmpeg> \
  -o "<dir>/%(artist,creator,channel,uploader)s - %(title).150s [%(id)s].%(ext)s" <url>
```

`--audio-quality 0` means best VBR quality for the MP3 encoder — this is the "best
quality MP3" requirement. `--embed-thumbnail`/`--embed-metadata` require ffmpeg, which
is in the image. `--write-thumbnail` keeps the cover image on disk next to the media
file, so the library can show it (`--embed-thumbnail` alone deletes it again after
embedding it into the audio). `--ffmpeg-location` points yt-dlp at the ffmpeg binary
the server was configured with, so the extraction does not depend on the PATH.

The produced file is located after the process exits by globbing the output directory
for `*[{videoID}].*` — deterministic because the output template always embeds the id.
Only files that can hold media count; the cover art next to them shares the same base
name and is never mistaken for the media file.

**In music mode only `.mp3` counts as media.** If the extraction fails — ffmpeg
missing, for instance — yt-dlp keeps the source audio and the job is reported as
failed with the reason. A leftover `.m4a` is never handed over as a finished MP3.

If yt-dlp exits non-zero *after* the media file was written — a thumbnail that cannot
be embedded, for instance — the job is still reported as completed and the file is
added to the library. The file is never orphaned on disk.

### Playlists and sync

* Pasting a playlist URL (`youtube.com/playlist?list=…` or `watch?v=…&list=…`) is probed
  with `--flat-playlist`; every entry becomes its own job so each has its own progress,
  and a single failure does not kill the batch.
* A playlist is a batch: `POST /api/downloads` with `playlist:true` returns one job per
  entry, all sharing a `playlist` title. The request returns as soon as the entries
  are known — a thousand track playlist must not hold the browser waiting.
* "Sync" from the Google playlists view re-queues only the videos that are **not**
  already in the library (dedupe key = video id + mode), so pressing sync twice is safe
  and does not download anything twice.
* Files of a playlist are stored in `DATA_DIR/music/<Playlist Title>/` (or
  `DATA_DIR/video/<Playlist Title>/`), sanitised for filesystem safety.

### Durability

* Every JSON document is written to a temp file, flushed and renamed into place.
* A document that cannot be parsed is moved aside as `<name>.corrupt` before anything
  is written to it — a broken file is never silently overwritten.
* The job list is pruned to the last 200 jobs, so `jobs.json` does not grow forever.

### Google OAuth

* Authorization URL: `https://accounts.google.com/o/oauth2/auth` with
  `access_type=offline`, `prompt=consent`, scope `https://www.googleapis.com/auth/youtube.readonly`.
* The callback exchanges the code with `https://oauth2.googleapis.com/token` and stores
  the **refresh token** only; access tokens are refreshed in memory per request.
* `state` is a random, one time value the server keeps for ten minutes (CSRF
  protection). It also remembers the page the sign-in was started from, so the
  browser lands back where it left off — the whole sign-in happens in the app, no
  URL has to be copied by hand.
* The callback redirects to that page with `?signedIn=1` or `?error=<code>`. The
  playlists view reads the result, shows it, and clears the query string again.
* A redirect to another site is ignored: only a path of the app itself is accepted.
* If `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET` are absent, the backend reports
  `googleEnabled:false` and the UI hides the sign-in affordance entirely.

## Frontend

Vue 3 + TS + Vite + Pinia + Vue Router. The interface follows the Vesper design
reference (a near-black gallery frame):

* **Colour** — the page is pure black `#000`, plates are `#0a0a0a`, and the ink ladder
  is `#e8e6e1` / `#8a8780` / `#4a4843`. There is exactly one accent, a muted amber
  `#d4a574`, and it is only ever a signal: a label, a dot, an underline, a rule — never
  a fill. Every hairline is the ink-faint colour; there is no separate hairline token.
* **Type** — two voices. Instrument Serif (regular and italic) carries the editorial
  lines: the display headline `clamp(42px, 7vw, 96px)`, the italic plate titles
  `clamp(36px, 4.2vw, 64px)`. JetBrains Mono carries every machine label: `11px`,
  uppercase, tracked `0.28–0.32em`. Both are served from the app itself
  (`web/public/fonts`), not from a font CDN — a self-hosted tool should not need the
  internet to look right.
* **Shape** — nothing is rounded except the 6px status dots. Frames are drawn with
  1px hairlines and 18px corner ticks, never with a filled box.
* **Depth** — no shadows. An always-on film grain (an SVG turbulence at 6% opacity,
  blended over) and a breathing vignette give the darkroom texture.
* **Motion** — grain shift 7s, vignette breath 9s, accent dot pulse 2.6s, all on
  `cubic-bezier(0.22, 1, 0.36, 1)`.

Three views:

* **Download** (`/`) — the download bar is the main thing on the page: a large serif
  field, the source label, and the download action. The settings sit underneath it:
  the output toggle (Video / MP3) and a line describing what the current mode does.
  The queue follows below. The toggle starts in the mode the server announces
  (`DEFAULT_MODE`) until the user picks one; the choice is remembered in the browser.
* **Library** (`/library`) — the downloads as plates: the cover art in a black frame
  that fades into the page, a roman numeral in the corner, mono meta, and an italic
  serif title. Audio plays in the player bar; a video download opens in a new tab,
  where the browser shows it with its own controls.
* **Playlists** (`/playlists`) — the Google sign-in card, then the user's playlists as
  a hairline-separated list with a "Sync →" action per playlist.

The shell is a three-row frame: a topbar (wordmark · edition marker · nav), the stage,
and a footbar with a live UTC clock, the current mode and the build. The player bar
docks at the bottom.

## Testing

* Go unit tests for: URL classification (single vs playlist), `yt-dlp` argument
  construction per mode, progress-line parsing, the store's atomic write/read round
  trip, the job manager's state machine, and every API handler (via `httptest` with a
  fake `yt-dlp` binary on `PATH`).
* Frontend unit tests (Vitest) for the URL validator and the mode-toggle store logic.
* A fake `yt-dlp` shell script that emits a canned JSON and a canned progress line is
  used by the handler tests, so the suite never touches the network.

## Delivery

* `Dockerfile` (3 stages), `docker-compose.yml`, `.dockerignore`.
* `.github/workflows/docker.yml`: buildx build on push to `main` and on tags, push to
  `ghcr.io/${{ github.repository }}` with `packages: write` permission, plus
  `latest` and semver tags.
* `README.md` with a GHCR pull badge, `docker run` and compose examples, the full env
  var table, and step-by-step Google OAuth setup.
* Commits follow `:gitmoji:` convention; every task ends with a commit.
