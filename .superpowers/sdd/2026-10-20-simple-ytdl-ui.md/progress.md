# SDD ledger — plan: docs/superpowers/plans/2026-10-20-simple-ytdl-ui.md

Spec: docs/superpowers/specs/2026-10-20-simple-ytdl-ui-design.md
Execution: inline (executing-plans) with two parallel implementer subagents for the
independent `backend/` and `web/` trees; infra (Docker/CI/docs) done in the main session.

Pre-flight: no shared interfaces between the backend and frontend trees — the only
coupling is the frozen JSON API contract in the spec, which both sides were given
verbatim.

## Tasks

Task 1-7, 10 (backend): complete — `go vet ./...` clean, `go test ./...` 8/8 packages pass
Task 8-9 (frontend): complete — 5 test files / 24 tests pass, `bun run build` green
Task 11 (docker): complete — Dockerfile (bun -> go -> debian runtime with yt-dlp + ffmpeg), .dockerignore, docker-compose.yml
Task 12 (ci/docs): complete — .github/workflows/docker.yml, README.md, LICENSE
Task 13 (verify/publish): in progress

## Rulings

- Task 3/4: `go 1.23.0` instead of `go 1.23` — Go 1.26 refuses to build a module with
  the bare form. `golang.org/x/oauth2` pinned to v0.30.0 because v0.37.0 requires go 1.26.
  Cost if wrong: the Docker image would need a newer Go base image.
- Task 6: `oauth2` v0.30.0, YouTube Data API called over plain REST (no google api client
  library) so it can be tested with httptest. Cost if wrong: none, the wire format is the same.
- Task 7: library item ids are `yt_<videoID>_<mode>` / `pl_<videoID>_<mode>` instead of the
  spec's `yt_<videoID>`, because the same video can exist in the library twice (MP3 and
  video) and the ids would collide. Spec updated. Cost if wrong: an id shape change.
- Task 3: `--write-thumbnail` added to both download profiles (spec updated). Without it
  yt-dlp deletes the thumbnail after embedding it, and the library has no cover art to
  show. Cost if wrong: one extra .webp next to each file.
- Task 5: a yt-dlp run that exits non-zero *after* writing the media file counts as
  completed, not failed — otherwise the finished file is orphaned on disk. Pinned by
  `TestFinishedFileSurvivesAPostProcessingFailure` (verified RED without the fix).
  Cost if wrong: a genuinely broken file could be marked as done (the file is still checked for existence).
- Task 5: `FindOutputFile` only accepts media extensions, so the cover art written next to
  the media is never mistaken for the media file itself.
- Task 6: flat playlist entries carry their link in `url`, not `webpage_url`; both are read.
  Cost if wrong: none, the watch URL fallback covers it.
- Task 9: playlist previews paginate with `?pageToken=` ("Show more"), because the YouTube
  API returns 50 items per page. Cost if wrong: none.
- Task 9: the mode toggle lives in the Download hero only; the rail shows a read-only
  "Downloading as MP3/Video" chip instead of a second identical control. Cost if wrong:
  one extra click to switch mode from the Library view.
- Task 10: the fake yt-dlp lives in `internal/fakeytdlp` instead of `internal/ytdlp/testdata`
  because three test packages need it and it must be generated per platform.
- Task 10: `runner_windows.go` quotes the command line for `.bat`/`.cmd` binaries, because
  cmd.exe re-parses `|` and `%` in the frozen progress template. Cost if wrong: the Windows
  path is only used in tests.

## Verification

- `go vet ./...` clean; `go test -count=1 ./...` all packages pass.
- `bun run test` 24/24 pass; `bun run build` produces dist.
- Live end-to-end against real yt-dlp 2026.x + ffmpeg:
  - music mode -> `jawed - Me at the zoo [jNQXAC9IVRw].mp3` (524 KB) + `.webp` cover
  - video mode -> `Me at the zoo [jNQXAC9IVRw].mp4` (465 KB) + `.webp` cover
  - `GET /api/library` lists both with distinct ids, `GET .../stream` answers 206 with
    `Content-Range` on a Range request, `GET .../cover` answers 200 image/webp
  - the library grid renders the real cover in the browser, the player bar streams and
    plays the file
