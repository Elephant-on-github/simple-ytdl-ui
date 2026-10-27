# Quick Start

## 1. Pull and run

```bash
docker compose up -d
```

Open http://localhost:8080

## 2. Add a YouTube session (optional)

**Option A — Paste cookies file:**
1. Export cookies from your browser (use "Get cookies.txt" extension)
2. In the app, go to Google → paste cookies → Save

**Option B — Mount cookies file:**
```yaml
volumes:
  - ./cookies.txt:/config/cookies.txt:ro
environment:
  YTDLP_COOKIES: /config/cookies.txt
```

## 3. Download a video

Paste any YouTube URL in the Download page → Download

## 4. Schedule recurring downloads

Go to Schedule → add a playlist URL + folder + frequency

## 5. Password protection (optional)

```yaml
environment:
  AUTH_PASSWORD: your-secret
```

## Volumes

| Volume | Purpose |
|--------|---------|
| `./data` | App state (library index, jobs, token) |
| `./library` | Downloaded media files |

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `DATA_DIR` | /data | State directory |
| `LIBRARY_DIR` | (empty) | Media directory (defaults to DATA_DIR) |
| `CONCURRENT_DOWNLOADS` | 2 | Parallel downloads |
| `DEFAULT_MODE` | video | Default download mode |
| `AUTH_PASSWORD` | (empty) | App password lock |
| `YTDLP_COOKIES` | (empty) | Path to cookies file |