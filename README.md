# simple-ytdl-ui

[![GHCR](https://ghcr.io/badge/Elephant-on-github/simple-ytdl-ui)](https://github.com/Elephant-on-github/simple-ytdl-ui/pkgs/container/simple-ytdl-ui)
[![Go](https://img.shields.io/badge/backend-Go-1.23-00ADD8?logo=go&logoColor=white)](backend/)
[![Vue](https://img.shields.io/badge/frontend-Vue_3-42b883?logo=vuedotjs&logoColor=white)](web/)
[![License](https://img.shields.io/badge/license-MIT-8b8b94)](LICENSE)

A self-hosted web app that turns YouTube URLs into files.

Paste a link, pick **Video** or **MP3**, and watch the queue fill up. Everything you
download lands in a browsable, playable library on your own disk.

- **One container.** Go backend, Vue frontend, `yt-dlp` and `ffmpeg` in a single image.
- **Best-quality MP3.** The music toggle extracts the highest-bitrate audio stream to
  MP3, with the cover art and the tags baked in.
- **Playlists.** A playlist URL becomes one job per track, each with its own progress
  and each individually cancellable.
- **Your own playlists, without a Google Cloud project.** Hand the server a YouTube
  session — a `cookies.txt` — and it lists your library and syncs from it. No OAuth
  client, no API key, no quota.
- **No database.** The state is plain JSON on a volume you mount.

---

## Quick start

```bash
docker run -d \
  --name simple-ytdl-ui \
  -p 8080:8080 \
  -v "$PWD/data:/data" \
  -v "$PWD/library:/library" \
  ghcr.io/elephant-on-github/simple-ytdl-ui:latest
```

Open <http://localhost:8080>, paste a link, press **Download**.

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

## Your own playlists

To see the playlists of *your* account — and to reach private ones — the server needs
a **YouTube session**. The cheapest one is a cookies file, and it needs nothing else:
no Google Cloud project, no OAuth client, no API key, no quota. This is the same
session `yt-dlp` itself uses.

**1. Get a `cookies.txt` for your account.** It is a Netscape cookie file with the
session of a signed-in YouTube browser. Two ways to get one:

*From the browser, in one command* — no extension needed, if you have `yt-dlp` on
your machine:

```bash
yt-dlp --cookies-from-browser firefox --cookies cookies.txt "https://www.youtube.com/robots.txt"
```

`--cookies-from-browser` takes `brave`, `chrome`, `chromium`, `edge`, `firefox`,
`opera`, `safari`, `vivaldi` or `whale`; add `:ProfileName` for a specific profile.
`--cookies` is where the file is written. On Linux, Chromium browsers may need their
keyring: `--cookies-from-browser chrome+gnomekeyring`.

*With a browser extension* — install **Get cookies.txt LOCALLY** (Chrome and Firefox),
open it while you are signed in on `youtube.com`, and export the domain
`.youtube.com`.

> **Export from a private window.** YouTube rotates the session cookies whenever a
> YouTube tab is open, and the rotation invalidates the file you just exported. So:
> sign in to YouTube in a private/incognito window, go to
> `https://www.youtube.com/robots.txt` in that same window, export the cookies, and
> close the private window right away. The
> [yt-dlp wiki](https://github.com/yt-dlp/yt-dlp/wiki/Extractors#exporting-youtube-cookies)
> has the details.

**2. Mount the file and point the server at it.**

```yaml
services:
  app:
    image: ghcr.io/elephant-on-github/simple-ytdl-ui:latest
    volumes:
      - ./data:/data
      - ./library:/library
      - ./cookies.txt:/config/cookies.txt:ro   # your YouTube session
    environment:
      YTDLP_COOKIES: /config/cookies.txt
```

**Or let yt-dlp read the cookies out of a browser profile** on the machine that runs
the server — handy when the server is your own machine:

```yaml
environment:
  YTDLP_COOKIES_FROM_BROWSER: firefox   # or chrome, edge, brave, …
```

With a session in place the Playlists page lists `youtube.com/feed/playlists`, the
**Sync** button pulls every track of a playlist into your library, and private
playlists and age-restricted videos become reachable. Paste a playlist link on the
Download page and it syncs the same way — that part never needed a session at all.

> **Keep the cookies file private.** It is a full session of your YouTube account:
> anyone holding it is you. Mount it read-only, do not put it in a public bucket, and
> replace it when you sign out of YouTube.

If the Playlists page complains about the session ("401 Unauthorized" or "could not
read your playlists"), the file is stale or was never a signed-in session — export it
again.

### Google sign-in, if you would rather have a button

The OAuth way still works and is the alternative to cookies. It needs a project in
the [Google Cloud console](https://console.cloud.google.com/):

1. Create a project and enable the **YouTube Data API v3**.
2. Configure the OAuth consent screen and add your own address as a test user.
3. Create an **OAuth client ID** of type *Web application*.
4. Add the **redirect URI**: the exact address your app answers on, for example
   `https://ytdl.example.com/api/auth/google/callback`.
   Google matches the scheme, the host and the trailing slash *exactly*, and only
   `http://localhost` may skip HTTPS — anything else has to be a real `https://`
   address.
5. Pass the credentials to the container:

   ```yaml
   environment:
     GOOGLE_CLIENT_ID: "xxxx.apps.googleusercontent.com"
     GOOGLE_CLIENT_SECRET: "xxxx"
   ```

Leave `GOOGLE_REDIRECT_URL` unset and the server works out the address from the
request it came in through, which is what you want behind a proxy or on a cloud
platform. Set it only when the app answers on a different address than the browser
used.

Only the refresh token is stored, in `DATA_DIR/google.json` with `0600` permissions.
The scope asked for is `youtube.readonly`: the app can read your playlists, it cannot
touch your account.

---

## Locking the app

The app is open by default — anyone who can reach the port can use it. Set a password
and a login screen guards everything:

```yaml
environment:
  AUTH_PASSWORD: "choose-something-long"
```

The login happens in the browser; the session lives in an `HttpOnly` cookie, marked
secure when the app is served over HTTPS. A restart of the container signs everybody
out again.

`AUTH_ALLOWED_EMAILS` lets a Google account sign in *instead of* typing the password.
The login screen then shows a **Continue with Google** button next to the password
field:

```yaml
environment:
  AUTH_PASSWORD: "choose-something-long"
  AUTH_ALLOWED_EMAILS: "you@example.com"
```

The list is comma separated and the comparison ignores case. Only the accounts on the
list get in that way — anybody else who signs in with Google can still use the
playlist sync, but not the app itself.

---

## Configuration

Everything is an environment variable.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the server listens on. |
| `DATA_DIR` | `/data` | Where the app keeps its state: the library index, the job list, the Google token. Mount this. |
| `LIBRARY_DIR` | *(empty)* | Where the downloaded media is stored. Empty means "next to the state in `DATA_DIR`". Set it to put the library on its own disk. |
| `YTDLP_BIN` | `yt-dlp` | Path to the `yt-dlp` binary. |
| `FFMPEG_BIN` | `ffmpeg` | Path to `ffmpeg` (needed for the MP3 extraction and the cover art). |
| `YTDLP_COOKIES` | – | A `cookies.txt` with a YouTube session. With it the server lists your playlists and reaches private ones. |
| `YTDLP_COOKIES_FROM_BROWSER` | – | Read the YouTube session from a browser profile instead, e.g. `firefox` or `chrome`. |
| `AUTH_PASSWORD` | – | Password for the app. Unset ⇒ the app is open. |
| `AUTH_ALLOWED_EMAILS` | – | Accounts that may sign in with Google instead of the password, comma separated. |
| `GOOGLE_CLIENT_ID` | – | Google OAuth client id. Unset ⇒ the Google sign-in is off. |
| `GOOGLE_CLIENT_SECRET` | – | Google OAuth client secret. Unset ⇒ the Google sign-in is off. |
| `GOOGLE_REDIRECT_URL` | *(from the request)* | The address Google sends the browser back to. Must match the registered URI exactly; unset means the server takes it from the request. |
| `CONCURRENT_DOWNLOADS` | `2` | How many downloads run at the same time. |
| `DEFAULT_MODE` | `video` | The mode the Video/MP3 toggle starts in: `video` or `music`. |
| `WEB_DIR` | – | Serve the frontend from a directory instead of the embedded build (development). |

### The library on its own disk

The media can live somewhere else than the state, which is what you want when the
files outgrow the disk the app runs on:

```yaml
volumes:
  - ./data:/data
  - /mnt/big-disk:/library
environment:
  LIBRARY_DIR: /library
```

The image ships `/library` as a second volume, so this works out of the box. The files
land in `<LIBRARY_DIR>/music/…` and `<LIBRARY_DIR>/video/…`, one folder per playlist.

---

## Deploying it somewhere public

The app is a single container that listens on one port, so any of the usual hosts
works. A few things to get right:

- **A public address.** The browser reaches the app on `https://your-host` — put a
  reverse proxy (Caddy, nginx, Traefik) or the platform's own HTTPS in front of it.
  The proxy has to pass `X-Forwarded-Proto` and `X-Forwarded-Host`; the app reads
  them to work out the address it is reachable at, which is what the Google redirect
  URI and the secure cookie flag are built from.
- **Two volumes.** `data` for the state, `library` for the media. Both must be
  persistent — a container without them comes back empty.
- **The cookies file**, if you want your own playlists. Mount it read-only.
- **A password**, if the app is reachable from outside. `AUTH_PASSWORD` and a
  reverse proxy with TLS is the sensible pair.
- **Disk space.** A 4K video is a few hundred megabytes; the MP3 of the same video is
  a few megabytes. The MP3 mode is the cheap one.

```yaml
services:
  app:
    image: ghcr.io/elephant-on-github/simple-ytdl-ui:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - /mnt/big-disk:/library
      - ./cookies.txt:/config/cookies.txt:ro
    environment:
      LIBRARY_DIR: /library
      YTDLP_COOKIES: /config/cookies.txt
      AUTH_PASSWORD: "choose-something-long"
    restart: unless-stopped
```

---

## How the download works

| Mode | yt-dlp format | Result |
|---|---|---|
| Video | `bv*+ba/b`, merged to MP4 | the best video and the best audio, muxed into an `.mp4` |
| MP3 | `bestaudio/best`, `-x --audio-format mp3 --audio-quality 0` | the highest-bitrate audio as an MP3, with the cover art and the tags embedded |

The MP3 is extracted with ffmpeg, which is in the image and pointed at by
`FFMPEG_BIN`. If the extraction fails the download is reported as failed with the
reason, instead of handing you a file that is not an MP3.

Files are named `Artist - Title [videoId].mp3` (or `Title [videoId].mp4`), so nothing
overwrites anything else. The cover art is written next to the media file and shown
in the library.

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

Everything below `/api` needs the session, unless a password is set — then only the
rows marked *public* answer without one.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/health` | liveness — *public* |
| `GET` | `/api/auth/session` | who is signed in — *public* |
| `POST` | `/api/auth/login` | sign in with the password — *public* |
| `POST` | `/api/auth/logout` | sign out of the app — *public* |
| `GET` | `/api/auth/google` | the Google authorization URL — *public* |
| `GET` | `/api/auth/google/callback` | the OAuth redirect target — *public* |
| `POST` | `/api/auth/google/logout` | forget the stored Google token |
| `GET` | `/api/config` | feature flags and defaults |
| `GET` | `/api/playlists` | your playlists — from the session or from Google |
| `GET` | `/api/playlists/{id}/items` | the tracks of a playlist |
| `POST` | `/api/downloads` | queue a URL (single or playlist) |
| `GET` | `/api/jobs` | the job queue |
| `POST` | `/api/jobs/{id}/cancel` | cancel a job |
| `GET` | `/api/library` | everything downloaded |
| `DELETE` | `/api/library/{id}` | delete an item and its file |
| `GET` | `/api/library/{id}/stream` | stream the media (Range supported) |
| `GET` | `/api/library/{id}/cover` | the cover art |

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

---

## Notes and limits

- `yt-dlp` is pinned to the latest release at image build time. If YouTube changes
  something and downloads start failing, pull a newer image.
- The job list is kept in memory and mirrored to `DATA_DIR/jobs.json`; the oldest
  finished jobs are dropped, so the file does not grow forever.
- A `library.json` that cannot be parsed is never overwritten — it is moved to
  `library.json.corrupt` first, so nothing is lost silently.
- With a cookies session the number of tracks in a playlist is not known up front, so
  the library page does not show a count for it. The tracks themselves are all there.
- The session cookie is marked secure when the app is reached over HTTPS, and the
  app trusts `X-Forwarded-Proto` — so only put it behind a proxy you trust.
- This tool downloads material that YouTube's terms of service restrict you from
  downloading. Use it for content you have the right to keep. You are responsible for
  what you download.

## License

MIT — see [LICENSE](LICENSE).
