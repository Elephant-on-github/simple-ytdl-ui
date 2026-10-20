# simple-ytdl-ui Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a single-container web app that downloads YouTube videos/playlists as video or best-quality MP3, with optional Google sign-in for playlist sync, a modern Vue UI, and a GHCR-published Docker image.

**Architecture:** One Go binary (stdlib HTTP + `golang.org/x/oauth2`) that embeds the built Vue SPA and owns a bounded yt-dlp job queue; JSON persistence under `DATA_DIR`; a 3-stage Alpine Dockerfile publishing to GHCR via GitHub Actions.

**Tech Stack:** Go 1.23 (stdlib, `x/oauth2`), Vue 3 + TypeScript + Vite + Pinia + Vue Router, Bun, Vitest, Docker/Buildx, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-10-20-simple-ytdl-ui-design.md` — read it before implementing; it pins the API contract, the `yt-dlp` argument lists, the JSON shapes and the design language.

## Global Constraints

- Go module path: `github.com/Elephant-on-github/simple-ytdl-ui/backend`; Go version `1.23`; only third-party import allowed: `golang.org/x/oauth2`.
- Only stdlib HTTP routing (`net/http` with `GET /api/x/{id}` patterns). No web framework, no ORM, no SQLite.
- JSON persistence only: `DATA_DIR/library.json`, `DATA_DIR/jobs.json`, `DATA_DIR/google.json` (token file `0600`). Writes are atomic (temp file + `os.Rename`).
- Frontend: Vue 3 `<script setup>` + TypeScript, Vite, Pinia, Vue Router, Vitest. No UI component library, no Tailwind.
- The API contract in the spec is frozen. Both the Go handlers and the TS client must match it exactly (field names are camelCase in JSON).
- `yt-dlp` argument lists are frozen verbatim in the spec — copy them, do not improvise.
- Commit after every task using the `:gitmoji:` convention from `AGENTS.md`.
- Never commit secrets, never force-push.

## Review Focus

Inputs the spec implies but that are easy to get wrong — each line must be pinned by a test in the owning task:

1. A pasted URL that contains **both** a video id and a `list=` parameter → must be treated as a *single video* download (YouTube's watch URL always has `list=`), not a playlist.
2. A URL with **no** YouTube id at all (e.g. `https://example.com/song.mp3`) → must be rejected with 400 and a clear message, not queued and failed 5 minutes later.
3. Music mode with `--audio-quality 0` must produce an `.mp3` in the output directory and must **not** leave a `.webm`/`.m4a` intermediate behind (the extraction must actually happen).
4. `DELETE /api/library/{id}` must remove the file from disk *and* the record — no orphan rows pointing at missing files.
5. Re-syncing the same Google playlist twice must **not** create duplicate library items or re-download (dedupe on video id + mode).
6. A download that fails (yt-dlp exits non-zero) must surface yt-dlp's stderr in the job's `error` field, visible in the UI.
7. `GET /api/library/{id}/stream` must honour a `Range: bytes=0-1023` request and return `206` with `Content-Range`, otherwise the browser audio player cannot seek.
8. Cancelling a running job must actually kill the yt-dlp process and leave the job in status `canceled`, not `running` forever.
9. `GET /api/playlists` without a Google session must return 401, not an empty list that looks like "no playlists".
10. A video id that already exists in the library but in the *other* mode (mp3 exists, user now wants video) must still be downloadable.

---

### Task 1: Repository scaffold, Go module, config package

**Files:**
- Create: `backend/go.mod`, `backend/internal/config/config.go`, `backend/internal/config/config_test.go`
- Create: `.gitignore`

**Interfaces:**
- Produces: `config.Config` struct and `config.Load() Config`.

```go
type Config struct {
    Port           int
    DataDir        string
    YtDlpBin       string
    FfmpegBin      string
    GoogleClientID string
    GoogleClientSecret string
    GoogleRedirectURL  string
    ConcurrentDownloads int
    DefaultMode    string // "video" | "music"
}
func Load() Config
func (c Config) GoogleEnabled() bool // true only if both client id and secret are non-empty
```

- [ ] **Step 1: Write the failing test** `backend/internal/config/config_test.go` — table test over env: defaults (`PORT=8080`, `DATA_DIR=/data`, `CONCURRENT_DOWNLOADS=2`, `DEFAULT_MODE=video`), overrides, and `GoogleEnabled()` false when only one of id/secret is set, true when both are.
- [ ] **Step 2: Run** `cd backend; go test ./internal/config/...` → Expected: FAIL (`undefined: Load`).
- [ ] **Step 3: Implement** `config.go`: read env with `os.Getenv`, parse ints with a fallback on parse error, `GoogleEnabled()` as specified. `Load()` takes no arguments and reads the process environment directly (so tests use `t.Setenv`).
- [ ] **Step 4: Run** `go test ./internal/config/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: scaffold backend module and config package`.

### Task 2: JSON store package

**Files:**
- Create: `backend/internal/store/store.go`, `backend/internal/store/store_test.go`
- Create: `backend/internal/store/types.go` (the `Item` and `Job` structs from the spec)

**Interfaces:**
- Produces:
```go
func New(dir string) *Store
func (s *Store) Items() []Item            // newest first (AddedAt desc)
func (s *Store) AddItem(it Item) error
func (s *Store) ItemByID(id string) (Item, bool)
func (s *Store) RemoveItem(id string) error   // removes record AND the file on disk
func (s *Store) HasVideo(videoID, mode string) bool
func (s *Store) SetJobs(jobs []Job) error
func (s *Store) Jobs() []Job
func (s *Store) SaveGoogleToken(refreshToken, email string) error
func (s *Store) GoogleToken() (refreshToken, email string, ok bool)
func (s *Store) ClearGoogleToken() error
```
`Item` and `Job` field names/JSON tags come verbatim from the spec.

- [ ] **Step 1: Write the failing tests**: round-trip a store in `t.TempDir()` — add an item, read it back, assert JSON field names via `json.Marshal` (e.g. `addedAt`, `videoId`); `RemoveItem` deletes the file on disk; `HasVideo` distinguishes modes; `SaveGoogleToken`/`GoogleToken` round-trip; the token file has mode `0600` on unix (skip the mode assertion on Windows).
- [ ] **Step 2: Run** `go test ./internal/store/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: `Store` holds a `sync.RWMutex`, in-memory slices, and a `save()` that writes `<name>.tmp` then `os.Rename`. `Items()` returns a copy sorted by `AddedAt` descending. `RemoveItem` removes the file (ignore `os.IsNotExist`) then the record.
- [ ] **Step 4: Run** `go test ./internal/store/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add JSON store with atomic writes`.

### Task 3: yt-dlp argument building, URL classification, progress parsing

**Files:**
- Create: `backend/internal/ytdlp/args.go`, `backend/internal/ytdlp/args_test.go`
- Create: `backend/internal/ytdlp/progress.go`, `backend/internal/ytdlp/progress_test.go`
- Create: `backend/internal/ytdlp/url.go`, `backend/internal/ytdlp/url_test.go`

**Interfaces:**
- Produces:
```go
type Mode string
const (ModeVideo Mode = "video"; ModeMusic Mode = "music")

func ParseMode(s string) (Mode, error)          // "" -> ModeVideo
func ClassifyURL(raw string) (Kind, VideoID, PlaylistID string, err error)
// Kind is one of KindVideo, KindPlaylist, KindInvalid
func BuildDownloadArgs(mode Mode, outDir string, url string) []string
func ParseProgress(line string) (Progress, bool)
type Progress struct { Status string; Downloaded, Total int64; Speed float64; ETA int }
```

- [ ] **Step 1: Write the failing tests**:
  - `url_test.go`: `https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123` → `KindVideo`, videoID `dQw4w9WgXcQ`; `https://www.youtube.com/playlist?list=PL123` → `KindPlaylist`; `https://youtu.be/dQw4w9WgXcQ` → `KindVideo`; `https://example.com/x.mp3` → `KindInvalid`; empty string → `KindInvalid`.
  - `args_test.go`: `BuildDownloadArgs(ModeMusic, "/data/music", "URL")` contains `-x`, `--audio-format`, `mp3`, `--audio-quality`, `0`, `-f`, `bestaudio/best`, `--embed-thumbnail`, `--embed-metadata`, and `-o` with a template containing `%(artist,creator,channel,uploader)s` and `%(id)s`; `BuildDownloadArgs(ModeVideo, …)` contains `-f`, `bv*+ba/b`, `--merge-output-format`, `mp4`, and **must not** contain `-x`. Both must contain the progress template from the spec and `--no-playlist`.
  - `progress_test.go`: the line `dl|downloading|5242880|10485760|1048576.5|3` → `Progress{Status:"downloading", Downloaded:5242880, Total:10485760, Speed:1048576.5, ETA:3}`; a line without the `dl|` prefix → `ok == false`; `dl|finished|10485760|10485760|0|0` parses.
- [ ] **Step 2: Run** `go test ./internal/ytdlp/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: `ClassifyURL` parses with `net/url`, looks for `v=` in the query or a `youtu.be` path, and `list=` in the query; a URL with **both** is a video (rule 1 in Review Focus). `BuildDownloadArgs` returns the exact argument list from the spec. `ParseProgress` splits on `|` and parses with `strconv`.
- [ ] **Step 4: Run** `go test ./internal/ytdlp/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add yt-dlp argument, url and progress parsing`.

### Task 4: yt-dlp runner (probe + download) against a fake binary

**Files:**
- Create: `backend/internal/ytdlp/runner.go`, `backend/internal/ytdlp/runner_test.go`
- Create: `backend/internal/ytdlp/testdata/fake-ytdlp.sh`

**Interfaces:**
- Produces:
```go
type Runner struct { Bin string }
func NewRunner(bin string) *Runner
type Info struct {
    ID string `json:"id"`
    Title string `json:"title"`
    Uploader string `json:"uploader"`
    Duration float64 `json:"duration"`
    Thumbnail string `json:"thumbnail"`
    WebpageURL string `json:"webpage_url"`
    Ext string `json:"ext"`
    Filesize int64 `json:"filesize"`
    Entries []Info `json:"entries"`
    PlaylistTitle string `json:"playlist_title"`
}
func (r *Runner) Probe(ctx context.Context, rawURL string, flat bool) (Info, error)
func (r *Runner) Download(ctx context.Context, mode Mode, outDir, rawURL string, onProgress func(Progress)) (string, error)
```
`Download` returns the absolute path of the produced file (found by globbing `outDir` for `*[{videoID}].*` after the process exits) and calls `onProgress` for each parsed progress line.

- [ ] **Step 1: Write the failing test** using `testdata/fake-ytdlp.sh` (a POSIX shell script that: with `-J` prints a canned info JSON; otherwise prints one `dl|downloading|…` line, then copies a fixture file into the directory named after the `-o` template's `%(id)s`, then exits 0). Tests: `Probe` returns the canned title/duration; `Probe` with `flat=true` returns entries; `Download` reports progress and returns a path that exists; `Download` returns the error text of a failing fake binary (non-zero exit → error contains stderr, Review Focus rule 6).
- [ ] **Step 2: Run** `go test ./internal/ytdlp/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: `exec.CommandContext` with `--newline`; scan stdout line by line, feed lines to `onProgress`; collect stderr into the error. The fake script must be executable (`chmod +x` in the test setup when on unix; on Windows the test uses a `fake-ytdlp.bat` variant — or, to stay portable, the test writes the fake binary at runtime into `t.TempDir()` with the right extension for `runtime.GOOS`).
- [ ] **Step 4: Run** `go test ./internal/ytdlp/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add yt-dlp runner with probe and download`.

### Task 5: Download manager (job queue)

**Files:**
- Create: `backend/internal/download/manager.go`, `backend/internal/download/manager_test.go`

**Interfaces:**
- Consumes: `ytdlp.Runner`, `ytdlp.Mode`, `store.Store`.
- Produces:
```go
type Manager struct { /* … */ }
func NewManager(runner *ytdlp.Runner, st *store.Store, workDir string, concurrent int) *Manager
func (m *Manager) Enqueue(ctx context.Context, rawURL string, mode ytdlp.Mode, playlist bool) ([]store.Job, error)
func (m *Manager) Jobs() []store.Job
func (m *Manager) Cancel(id string) error
func (m *Manager) WaitIdle()  // test helper
```

- [ ] **Step 1: Write the failing tests**: enqueueing a single video creates one job in `queued` then `completed`; enqueueing a playlist URL with `playlist=true` creates one job per entry (fake `yt-dlp` returns 3 entries); a job that fails records a non-empty `Error`; `Cancel` on a running job results in status `canceled` (fake binary sleeps) — the manager kills the child process (Review Focus rule 8); `HasVideo` dedupe: enqueueing a video id already in the library with the same mode returns an existing job instead of a new download.
- [ ] **Step 2: Run** `go test ./internal/download/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: a buffered channel of jobs and `concurrent` workers; each worker calls `runner.Download` into `workDir/<mode>/<sanitised playlist or "singles">`; on success add a `store.Item` and set the job to `completed`; on context cancellation set `canceled`; on error set `failed` + `Error`. Keep a `map[string]*exec.Cmd`/`context.CancelFunc` per running job for `Cancel`.
- [ ] **Step 4: Run** `go test ./internal/download/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add download job manager with worker pool`.

### Task 6: Google OAuth + YouTube Data API client

**Files:**
- Create: `backend/internal/youtube/oauth.go`, `backend/internal/youtube/oauth_test.go`
- Create: `backend/internal/youtube/client.go`, `backend/internal/youtube/client_test.go`

**Interfaces:**
- Produces:
```go
type OAuth struct { ClientID, ClientSecret, RedirectURL string }
func (o *OAuth) AuthURL(state string) string
func (o *OAuth) Exchange(ctx context.Context, code string) (refreshToken string, err error)

type Client struct { http *http.Client; baseURL string }  // baseURL overridable in tests
func NewClient(ctx context.Context, st *store.Store, o *OAuth) (*Client, error)
func (c *Client) Playlists(ctx context.Context) ([]Playlist, error)
func (c *Client) PlaylistItems(ctx context.Context, playlistID, pageToken string) ([]PlaylistItem, string, error)
type Playlist struct { ID, Title, Thumb string; ItemCount int }
type PlaylistItem struct { ID, Title, Thumb, Channel string; Duration int }
```
- [ ] **Step 1: Write the failing tests** with `httptest`: `AuthURL` contains `access_type=offline`, `prompt=consent`, the youtube.readonly scope and the redirect URL; `Exchange` posts to a fake token endpoint and returns the refresh token; `Playlists` parses a canned `playlists.list` JSON (asserting `items[0].id`, `snippet.title`, `contentDetails.itemCount`); `PlaylistItems` returns items and the `nextPageToken`; a 401 from the API surfaces an error; `NewClient` with no stored token returns an error ("not signed in").
- [ ] **Step 2: Run** `go test ./internal/youtube/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: use `golang.org/x/oauth2` (`oauth2.Config` with `Endpoint` pointing at Google's token URL) for the token exchange and refresh; call the REST API with `http.NewRequest` + `oauth2.NewClient`; the base URL is a struct field so tests point it at `httptest.Server`.
- [ ] **Step 4: Run** `go test ./internal/youtube/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add google oauth and youtube data api client`.

### Task 7: HTTP API + static SPA serving

**Files:**
- Create: `backend/internal/api/server.go`, `backend/internal/api/server_test.go`
- Create: `backend/internal/api/static.go`

**Interfaces:**
- Produces:
```go
type Server struct { /* cfg, store, manager, youtube, state */ }
func New(cfg config.Config, st *store.Store, mgr *download.Manager, yt *youtube.Client) *Server
func (s *Server) Handler() http.Handler
```
Routes exactly as in the spec's API contract table, using `net/http` `ServeMux` patterns (`"GET /api/jobs/{id}/cancel"`).

- [ ] **Step 1: Write the failing tests** (httptest server + fake yt-dlp on `PATH`): `GET /api/health` → 200 `{"status":"ok"}`; `GET /api/config` → `googleEnabled:false` without credentials; `POST /api/downloads` with a bad URL → 400; with a good URL → 200 and a job list; `GET /api/jobs` → the job; `POST /api/jobs/{id}/cancel` → 200; unknown id → 404; `GET /api/library` → items; `DELETE /api/library/{id}` → 200 and the file is gone from disk (Review Focus rule 4); `GET /api/library/{id}/stream` with `Range: bytes=0-3` → 206 + `Content-Range` (rule 7); `GET /api/playlists` without a session → 401 (rule 9); `GET /api/auth/google` without credentials → 503.
- [ ] **Step 2: Run** `go test ./internal/api/...` → Expected: FAIL.
- [ ] **Step 3: Implement**: a `mux` with the routes; `writeJSON`/`writeError` helpers; `state` is a random string generated at startup, compared on the OAuth callback; `static.go` serves the embedded SPA (or the `web/dist` directory when `WEB_DIR` is set) with an index.html fallback for unknown non-`/api` paths.
- [ ] **Step 4: Run** `go test ./internal/api/...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: add http api and static spa serving`.

### Task 8: Frontend scaffold + design system

**Files:**
- Create: `web/package.json`, `web/vite.config.ts`, `web/tsconfig.json`, `web/index.html`, `web/src/main.ts`, `web/src/App.vue`, `web/src/styles/theme.css`, `web/src/styles/base.css`, `web/src/router/index.ts`
- Create: `web/src/lib/api.ts` (typed client for the frozen API contract), `web/src/lib/format.ts`
- Create: `web/src/stores/mode.ts`, `web/src/stores/mode.test.ts`
- Create: `web/src/lib/validate.ts`, `web/src/lib/validate.test.ts`

- [ ] **Step 1: Write the failing tests**: `mode.test.ts` — the mode store defaults to `video`, `toggle()` flips it, and it persists to `localStorage` under `simple-ytdl-ui:mode`; `validate.test.ts` — `isValidYouTubeURL` accepts `https://www.youtube.com/watch?v=abc`, `https://youtu.be/abc`, `https://www.youtube.com/playlist?list=PL1` and rejects `https://example.com/x.mp3` and `""`.
- [ ] **Step 2: Run** `bun run test` (vitest) → Expected: FAIL.
- [ ] **Step 3: Implement**: `theme.css` holds the CSS custom properties from the spec (canvas `#0a0a0b`, panel `#131316`, hairline `rgba(255,255,255,.08)`, text `#f5f5f6`, muted `#8b8b94`, amber `#f5a524`, blue `#5b8def`); `base.css` holds resets, the type scale (12/13/15/20/34px), tabular numerals, and the shared button/card primitives. `api.ts` is a typed `fetch` wrapper with the exact JSON shapes from the spec. `vite.config.ts` proxies `/api` to `http://localhost:8080` in dev.
- [ ] **Step 4: Run** `bun run test` and `bun run build` → Expected: PASS and a successful bundle.
- [ ] **Step 5: Commit**: `:sparkles: scaffold vue app with design tokens and api client`.

### Task 9: Frontend views — Download, Library, Playlists, player

**Files:**
- Create: `web/src/views/DownloadView.vue`, `web/src/views/LibraryView.vue`, `web/src/views/PlaylistsView.vue`
- Create: `web/src/components/ModeToggle.vue`, `web/src/components/JobQueue.vue`, `web/src/components/PlayerBar.vue`, `web/src/components/LibraryGrid.vue`, `web/src/components/PlaylistList.vue`, `web/src/components/TopBar.vue`
- Create: `web/src/stores/jobs.ts`, `web/src/stores/library.ts`, `web/src/stores/session.ts`

- [ ] **Step 1: Write the failing test**: `stores/jobs.test.ts` — the jobs store merges a polled `GET /api/jobs` payload, keeps jobs sorted newest-first, and exposes `activeCount`.
- [ ] **Step 2: Run** `bun run test` → Expected: FAIL.
- [ ] **Step 3: Implement** the views per the spec's design language: the mode toggle swaps the accent colour via a `data-mode` attribute on `<html>`; the queue shows title, a determinate progress bar, speed, ETA and a cancel button; the library grid shows cover art, title, artist, duration, play button, and delete; the player bar is fixed at the bottom with play/pause, seek and time; the playlists view shows the Google sign-in card when signed out and the playlist list with a per-playlist "Sync" button that calls `POST /api/downloads` with `playlist:true` and the current mode.
- [ ] **Step 4: Run** `bun run test` and `bun run build` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: build download library and playlist views`.

### Task 10: `main.go`, embedding, and end-to-end wiring

**Files:**
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/api/dist/.gitkeep` (placeholder so `go:embed` compiles; the real `dist` is copied in at image build time)

**Interfaces:**
- Consumes: `config.Load`, `store.New`, `ytdlp.NewRunner`, `download.NewManager`, `youtube.NewClient`, `api.New`.
- Produces: a `main` that loads config, opens the store, starts the manager, builds the server, and `http.ListenAndServe` on `cfg.Port` with graceful shutdown on SIGINT/SIGTERM.

- [ ] **Step 1: Write the failing test**: `backend/cmd/server/main_test.go` — a test that builds the `http.Handler` through a `newServer(cfg)` helper and asserts `GET /api/health` returns 200 (the helper is what `main` calls, so the wiring is exercised).
- [ ] **Step 2: Run** `go test ./cmd/...` → Expected: FAIL.
- [ ] **Step 3: Implement** `main.go` and the `newServer` helper; `api.StaticHandler` picks the embedded FS when `WEB_DIR` is unset, otherwise the directory.
- [ ] **Step 4: Run** `go build ./... && go test ./...` → Expected: PASS.
- [ ] **Step 5: Commit**: `:sparkles: wire server main and embed frontend`.

### Task 11: Docker image, compose file, ignore rules

**Files:**
- Create: `Dockerfile`, `.dockerignore`, `docker-compose.yml`

- [ ] **Step 1: Write the "test"** — there is no unit test for a Dockerfile; the check is the build itself: `docker build -t simple-ytdl-ui:local .` must succeed. (If the local Docker daemon is unavailable, the build is verified by the CI workflow in Task 13 and the Dockerfile is reviewed line by line.)
- [ ] **Step 2: Create the Dockerfile**: three stages — `web` (`node:22-alpine`, `bun install --frozen-lockfile`, `bun run build`), `build` (`golang:1.23-alpine`, copy `web/dist` into `backend/internal/api/dist`, `CGO_ENABLED=0 go build -o /out/server ./cmd/server`), `runtime` (`alpine:3.20`, `apk add --no-cache ffmpeg yt-dlp ca-certificates tzdata`, non-root user `ytdl` uid 1000, `VOLUME /data`, `EXPOSE 8080`, `ENTRYPOINT ["/usr/local/bin/server"]`).
- [ ] **Step 3: `.dockerignore`**: `node_modules`, `web/dist`, `.git`, `data`, `docs`, `*.md`.
- [ ] **Step 4: `docker-compose.yml`**: service `app`, `build: .`, `ports: ["8080:8080"]`, `volumes: ["./data:/data"]`, `environment` listing the supported env vars with defaults, `restart: unless-stopped`.
- [ ] **Step 5: Verify** `docker build -t simple-ytdl-ui:local .` → Expected: image builds; `docker run --rm -p 8080:8080 simple-ytdl-ui:local` → `curl localhost:8080/api/health` returns `{"status":"ok"}`. If the daemon is unavailable, record that in the commit body and rely on CI.
- [ ] **Step 6: Commit**: `:package: add multi-stage docker image and compose file`.

### Task 12: CI workflow, README, LICENSE

**Files:**
- Create: `.github/workflows/docker.yml`
- Create: `README.md`, `LICENSE`

- [ ] **Step 1: Create the workflow**: `on: push` to `main` and tags `v*`; `permissions: {packages: write, contents: read}`; `docker/setup-buildx-action`, `docker/login-action` against `ghcr.io` with `${{ secrets.GITHUB_TOKEN }}`, `docker/metadata-action` (tags: `latest`, `type=semver`, `type=raw,value=main`), `docker/build-push-action` with `context: .`, `file: ./Dockerfile`, `push: true`, `tags:` + `labels:` from the metadata action, `cache-from/to: type=gha`.
- [ ] **Step 2: Write the README**: title, one-line description, GHCR pull badge, feature list, `docker run` and compose snippets, the full env var table, Google OAuth setup (create OAuth client → web app → redirect URI `http://localhost:8080/api/auth/google/callback` → set `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`), local development instructions, a note that yt-dlp may need updating, and a legal note that the user is responsible for what they download.
- [ ] **Step 3: Add the MIT `LICENSE`** (2026, Elephant-on-github).
- [ ] **Step 4: Verify** `docker compose config` parses.
- [ ] **Step 5: Commit**: `:memo: add ghcr publishing workflow readme and license`.

### Task 13: End-to-end verification, GitHub publication, GHCR release

**Files:** none (verification only)

- [ ] **Step 1: Run the full suite**: `cd backend; go vet ./... && go test ./...`; `cd web; bun run test && bun run build` → Expected: all green.
- [ ] **Step 2: Run the app locally** (`go run ./cmd/server` with `DATA_DIR` pointing at a temp dir) and smoke-test `curl /api/health`, `curl /api/config`, and a `POST /api/downloads` round trip → Expected: JSON as specified.
- [ ] **Step 3: `git init`** (if needed), add a `main` branch, and commit the tree.
- [ ] **Step 4: Create the GitHub repo** `Elephant-on-github/simple-ytdl-ui` (public, so GHCR packages are free to pull) and push `main` — or, if it already exists, push to it.
- [ ] **Step 5: Watch the CI run**: `gh run watch` → Expected: the build-and-push job succeeds and `ghcr.io/elephant-on-github/simple-ytdl-ui:latest` exists (`gh api /user/packages/container/simple-ytdl-ui/versions`).
- [ ] **Step 6: Verify the published image**: `docker pull ghcr.io/elephant-on-github/simple-ytdl-ui:latest` and `docker run` it → `curl` health check passes.
- [ ] **Step 7: Commit** any fix-ups discovered above, and record the final state in the README badge section.
