# simple-ytdl-ui

[![GHCR](https://ghcr.io/badge/Elephant-on-github/simple-ytdl-ui)](https://github.com/Elephant-on-github/simple-ytdl-ui/pkgs/container/simple-ytdl-ui)
[![Go](https://img.shields.io/badge/backend-Go-1.23-00ADD8?logo=go&logoColor=white)](backend/)
[![Vue](https://img.shields.io/badge/frontend-Vue_3-42b883?logo=vuedotjs&logoColor=white)](web/)
[![License](https://img.shields.io/badge/license-MIT-8b8b94)](LICENSE)

A self-hosted web app that turns YouTube URLs into files.

Paste a link, pick **Video** or **MP3**, and watch the queue fill up. Everything you
download lands in a browsable, playable library on your own disk. Sign in with Google
(optional) and your playlists show up too, ready to sync.

* **One container.** Go backend + Vue frontend + `yt-dlp` + `ffmpeg` in a single image.
* **Best-quality MP3.** The music toggle extracts the highest-bitrate audio stream to
  MP3 with cover art and tags embedded.
* **Playlists.** A playlist URL becomes one job per track, each with its own progress,
  each individually cancellable.
* **Your playlists (optional).** Google sign-in lets you sync the playlists you
  already have. The app is fully functional without it.
* **No database.** State is plain JSON on a volume you mount.

---

## Quick start

```bash
docker run -d \
  --name simple-ytdl-ui \
  -p 8080:8080 \
  -v "$PWD/data:/data" \
  ghcr.io/elephant-on-github/simple-ytdl-ui:latest
```

Open <http://localhost:8080>.

Or with compose:

```bash
docker compose pull
docker compose up -d
```

To build the image from the sources instead of pulling it:

```bash
docker compose -f docker-compose.build.yml up -d --build
```

### Image tags

| Tag | What it is |
|---|---|
| `latest` | the newest build of `main` (and the newest release) |
| `main` | the newest build of `main` |
| `1.0.0`, `1.0` | a release, cut by tagging the repository `v1.0.0` |
| `sha-abc1234` | the build of a single commit |

The image is built and published by `.github/workflows/docker.yml` on every push to
`main`, for `linux/amd64` and `linux/arm64`. It is public, so anybody can pull it
without an account:

```bash
docker pull ghcr.io/elephant-on-github/simple-ytdl-ui:latest
```

> Forking this? GitHub makes a new package private, and only the owner of the account
> can change that. Set your package to **Public** once, in its settings.

---

## Configuration

Everything is environment variables. Only `DATA_DIR` really matters; the rest have
sensible defaults.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the server listens on. |
| `DATA_DIR` | `/data` | Where the app keeps its state: the library index, the job list, the Google token. Mount this. |
| `LIBRARY_DIR` | *(empty)* | Where the downloaded media is stored. Empty means "next to the state in `DATA_DIR`". Set it to put the library on its own volume. |
| `YTDLP_BIN` | `yt-dlp` | Path to the `yt-dlp` binary. |
| `FFMPEG_BIN` | `ffmpeg` | Path to `ffmpeg` (needed for MP3 extraction and thumbnail embedding). |
| `CONCURRENT_DOWNLOADS` | `2` | How many downloads run at the same time. |
| `DEFAULT_MODE` | `video` | Initial mode of the Video/MP3 toggle: `video` or `music`. |
| `AUTH_PASSWORD` | – | Password for the app. Unset ⇒ the app is open. Set it ⇒ a login screen guards everything. |
| `AUTH_ALLOWED_EMAILS` | – | Accounts that may sign in with Google instead of the password, comma separated. |
| `GOOGLE_CLIENT_ID` | – | Google OAuth client id. Unset ⇒ Google features are off. |
| `GOOGLE_CLIENT_SECRET` | – | Google OAuth client secret. Unset ⇒ Google features are off. |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8080/api/auth/google/callback` | Must match the redirect URI registered in Google Cloud. |
| `WEB_DIR` | – | Serve the frontend from a directory instead of the embedded build (development). |

### Locking the app

Set `AUTH_PASSWORD` and the app asks for a password before it shows anything. The
login happens in the browser, the session lives in an `HttpOnly` cookie, and a
restart of the container signs everybody out again.

```yaml
environment:
  AUTH_PASSWORD: "choose-something-long"
```

Nothing else changes: the Google sign-in for playlists is a separate thing and stays
optional.

### Signing in with your Google account

`AUTH_ALLOWED_EMAILS` lets a Google account sign in to the app itself, without typing
the password. The login screen then shows a **Continue with Google** button next to
the password field.

```yaml
environment:
  AUTH_PASSWORD: "choose-something-long"
  AUTH_ALLOWED_EMAILS: "you@example.com"
```

The list is comma separated, and the comparison ignores case. Only the accounts on
the list get in — anybody else who signs in with Google can still use the playlist
sync, but not the app itself.

To use it at all you need a Google OAuth client; see [Google sign-in](#google-sign-in-optional).

### Putting the library on its own disk

The media can live somewhere else than the state. Point `LIBRARY_DIR` at a volume:

```yaml
volumes:
  - ./data:/data
  - /mnt/big-disk:/library
environment:
  LIBRARY_DIR: /library
```

The image ships with `/library` as a second volume, so this works out of the box. The
files land in `<LIBRARY_DIR>/music/…` and `<LIBRARY_DIR>/video/…`.

---

## Google sign-in (optional)

The app never requires Google. If you want your own playlists in the UI:

1. Open the [Google Cloud console](https://console.cloud.google.com/), create a project
   and enable the **YouTube Data API v3**.
2. Configure the OAuth consent screen (add your own email as a test user).
3. Create an **OAuth client ID** of type *Web application*.
4. Add the redirect URI `http://localhost:8080/api/auth/google/callback`
   (or `https://your-host/api/auth/google/callback` if you expose the app publicly).
5. Pass the credentials to the container:

   ```yaml
   environment:
     GOOGLE_CLIENT_ID: "xxxx.apps.googleusercontent.com"
     GOOGLE_CLIENT_SECRET: "xxxx"
     GOOGLE_REDIRECT_URL: "http://localhost:8080/api/auth/google/callback"
   ```

Only the refresh token is stored, in `DATA_DIR/google.json` with `0600` permissions. The
scope requested is `youtube.readonly` — the app can read playlists, it cannot modify
your account.

Signing in happens entirely in the app: press **Continue with Google** on the
Playlists page, pick an account, and you are back on the Playlists page — signed in,
with your playlists loaded. If anything goes wrong, the app tells you what, in
words. Add the address to `AUTH_ALLOWED_EMAILS` and the same button also signs you
in to the app itself.

---

## How the download works

| Mode | yt-dlp format | Result |
|---|---|---|
| Video | `bv*+ba/b`, merged to MP4 | best video + best audio, muxed into an `.mp4` |
| MP3 | `bestaudio/best`, `-x --audio-format mp3 --audio-quality 0` | highest-bitrate audio as MP3, with embedded cover art and metadata |

The MP3 is extracted with ffmpeg, so the server needs ffmpeg — it is in the image and
`FFMPEG_BIN` points at it. If the extraction fails, the download is reported as failed
with the reason instead of handing you a file that is not an MP3.

Files are named `Artist - Title [videoId].mp3` (or `Title [videoId].mp4`) so nothing
overwrites anything else. Playlist downloads are grouped into a folder per playlist.
The cover art is written next to the media file and shown in the library.

---

## Local development

Requires Go 1.23+, Bun, and `yt-dlp` + `ffmpeg` on your `PATH`.

```bash
# backend
cd backend
go test ./...
go run ./cmd/server            # serves the API on :8080

# frontend (second terminal)
cd web
bun install
bun run dev                    # http://localhost:5173, proxies /api to :8080
```

`WEB_DIR` lets the Go server serve a frontend build from disk:

```bash
cd web && bun run build
cd ../backend && WEB_DIR=../web/dist go run ./cmd/server
```

---

## Repository layout

```
backend/     Go API, yt-dlp runner, job queue, YouTube/Google client
web/         Vue 3 + TypeScript frontend (Vite, Pinia, Vue Router)
docs/        design spec and implementation plan
Dockerfile   three stages: bun build -> go build -> debian runtime
```

## API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/health` | liveness |
| `GET` | `/api/config` | feature flags and defaults |
| `GET` | `/api/auth/google` | Google authorization URL |
| `GET` | `/api/auth/google/callback` | OAuth redirect target |
| `GET` | `/api/auth/session` | current session |
| `POST` | `/api/auth/logout` | forget the stored token |
| `GET` | `/api/playlists` | your YouTube playlists |
| `GET` | `/api/playlists/{id}/items` | items of a playlist |
| `POST` | `/api/downloads` | queue a URL (single or playlist) |
| `GET` | `/api/jobs` | the job queue |
| `POST` | `/api/jobs/{id}/cancel` | cancel a job |
| `GET` | `/api/library` | everything downloaded |
| `DELETE` | `/api/library/{id}` | delete an item and its file |
| `GET` | `/api/library/{id}/stream` | stream the media (Range supported) |
| `GET` | `/api/library/{id}/cover` | cover art |

---

## Design

The interface is a near-black gallery frame, in the spirit of the
[Vesper](https://kombai.com/gallery/inspirations/vesper/) template: pure black, warm
parchment ink, and exactly one colour — a muted amber `#d4a574` — used only as a
signal. Two voices carry everything: **Instrument Serif** for the editorial lines
(headlines, plate titles, the italic) and **JetBrains Mono** for every machine label
(nav, counts, coordinates, timestamps). Frames are drawn with hairlines and corner
ticks instead of rounded cards, and depth comes from a film grain and a slow
vignette rather than shadows. Both fonts are shipped with the app, so the interface
does not need a font CDN to look right.

## Notes and limits

* `yt-dlp` is pinned to the latest release at image build time. If YouTube changes
  something and downloads start failing, pull a newer image.
* The job list is kept in memory and mirrored to `DATA_DIR/jobs.json`; the oldest
  finished jobs are dropped, so the file does not grow forever.
* A `library.json` that cannot be parsed is never overwritten — it is moved to
  `library.json.corrupt` first, so nothing is lost silently.
* This tool downloads material that YouTube's terms of service restrict you from
  downloading. Use it for content you have the right to keep. You are responsible for
  what you download.
* The API has no authentication beyond the optional Google sign-in — run it on your
  own machine or behind a reverse proxy if you expose it.

## License

MIT — see [LICENSE](LICENSE).
