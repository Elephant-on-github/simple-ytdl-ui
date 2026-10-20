// Package fakeytdlp writes a stand-in for the yt-dlp binary so the test suite
// can exercise the runner, the job manager and the HTTP API without ever
// touching the network.
//
// The fake understands the two ways the server calls yt-dlp:
//
//   - with -J it prints a canned info JSON (a playlist listing when
//     --flat-playlist is present),
//   - otherwise it prints progress lines and creates the output file named
//     after the -o template.
//
// Three behaviours are available through the FAKE_YTDLP_MODE environment
// variable: "ok" (default), "fail" (writes to stderr and exits 1) and "sleep"
// (sleeps before finishing, so a running job can be cancelled).
package fakeytdlp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Modes understood by the fake.
const (
	ModeOK = "ok"
	// ModeFail writes to stderr and exits 1 without producing a file.
	ModeFail = "fail"
	// ModeSleep sleeps before finishing, so a running job can be cancelled.
	ModeSleep = "sleep"
	// ModeFailAfterWrite produces the media file and then exits 1, the way
	// yt-dlp does when a post processor (thumbnail embedding) fails.
	ModeFailAfterWrite = "failafterwrite"
	// ModeNoMusic keeps the source audio instead of extracting an MP3, the
	// way yt-dlp does when ffmpeg is missing.
	ModeNoMusic = "nomusic"
)

// Write creates a fake yt-dlp binary inside dir and returns its path.
func Write(dir, mode string) (string, error) {
	if mode == "" {
		mode = ModeOK
	}
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "fake-ytdlp.bat")
		if err := os.WriteFile(path, []byte(windowsScript(mode)), 0o755); err != nil {
			return "", err
		}
		return path, nil
	}
	path := filepath.Join(dir, "fake-ytdlp.sh")
	if err := os.WriteFile(path, []byte(shellScript(mode)), 0o755); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// shellScript is the POSIX version of the fake.
const shellScriptTemplate = `#!/bin/sh
# Fake yt-dlp used by the test suite. It never touches the network.
mode="{{MODE}}"
is_json=0
flat=0
music=0
out=""
url=""
for arg in "$@"; do
  url="$arg"
done
for arg in "$@"; do
  case "$arg" in
    -J) is_json=1 ;;
    --flat-playlist) flat=1 ;;
    -x) music=1 ;;
  esac
done
while [ $# -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    out="$1"
  fi
  shift
done

if [ "$mode" = "fail" ]; then
  echo "ERROR: [youtube] dQw4w9WgXcQ: Video unavailable. This video is private." 1>&2
  exit 1
fi

id=""
case "$url" in
  *v=*)
    id="${url#*v=}"
    id="${id%%&*}"
    id="${id%%#*}"
    ;;
  *)
    id="${url##*/}"
    ;;
esac
id="${id%%\?*}"

if [ "$is_json" = "1" ]; then
  if [ "$flat" = "1" ]; then
    printf '%s\n' '{"_type":"playlist","id":"PLfake","title":"Fake Playlist","playlist_count":3,"entries":[{"id":"vid001","title":"First Fake Video","url":"https://www.youtube.com/watch?v=vid001","duration":61},{"id":"vid002","title":"Second Fake Video","url":"https://www.youtube.com/watch?v=vid002","duration":62},{"id":"vid003","title":"Third Fake Video","url":"https://www.youtube.com/watch?v=vid003","duration":63}]}'
  else
    printf '%s\n' '{"id":"'"$id"'","title":"Fake Video","uploader":"Fake Artist","duration":212,"thumbnail":"https://example.test/thumb.jpg","webpage_url":"https://www.youtube.com/watch?v='"$id"'","ext":"mp4","filesize":123456}'
  fi
  exit 0
fi

echo "dl|downloading|5242880|10485760|1048576.5|3"
echo "dl|finished|10485760|10485760|0|0"

if not "%FAKE_YTDLP_COUNT%"=="" echo download>>"%FAKE_YTDLP_COUNT%"

if [ "$mode" = "sleep" ]; then
  sleep 30
fi

dir=$(dirname "$out")
mkdir -p "$dir"
if [ "$music" = "1" ]; then
  if [ "$mode" = "nomusic" ]; then
    printf 'fake m4a' > "$dir/Fake Artist - Fake Video [$id].m4a"
  else
    printf 'fake mp3' > "$dir/Fake Artist - Fake Video [$id].mp3"
  fi
else
  printf 'fake mp4' > "$dir/Fake Video [$id].mp4"
fi
if [ "$mode" = "failafterwrite" ]; then
  echo "ERROR: [EmbedThumbnail] Unable to embed thumbnail" 1>&2
  exit 1
fi
exit 0
`

// windowsScriptTemplate is the Windows batch version of the fake.
const windowsScriptTemplate = `@echo off
setlocal EnableExtensions
set "MODE={{MODE}}"
set "ISJSON=0"
set "ISFLAT=0"
set "ISMUSIC=0"
set "OUT="
set "URL="
:parse
if "%~1"=="" goto afterparse
if /I "%~1"=="-o" goto getout
if /I "%~1"=="-J" set "ISJSON=1"
if /I "%~1"=="--flat-playlist" set "ISFLAT=1"
if /I "%~1"=="-x" set "ISMUSIC=1"
set "URL=%~1"
shift
goto parse
:getout
shift
set "OUT=%~1"
shift
goto parse
:afterparse
if not "%MODE%"=="fail" goto notfail
echo ERROR: [youtube] dQw4w9WgXcQ: Video unavailable. This video is private. 1>&2
exit /b 1
:notfail
set "TMP=%URL%"
for /f "tokens=1* delims=&#" %%a in ("%TMP%") do set "TMP=%%a"
set "ID=%TMP:*v=%"
if "%ID%"=="%TMP%" goto idfrompath
set "ID=%ID:~1%"
goto idready
:idfrompath
for %%F in ("%URL%") do set "ID=%%~nxF"
:idready
if "%ISJSON%"=="1" goto json
echo dl^|downloading^|5242880^|10485760^|1048576.5^|3
echo dl^|finished^|10485760^|10485760^|0^|0
if not "%FAKE_YTDLP_COUNT%"=="" echo download>>"%FAKE_YTDLP_COUNT%"
if not "%MODE%"=="sleep" goto writefile
ping -n 11 127.0.0.1 >nul 2>&1
:writefile
for %%A in ("%OUT%") do set "DIR=%%~dpA"
if not exist "%DIR%" mkdir "%DIR%"
if "%ISMUSIC%"=="1" goto writemp3
echo fake mp4>"%DIR%Fake Video [%ID%].mp4"
goto done
:writemp3
if not "%MODE%"=="nomusic" goto writemp3file
echo fake m4a>"%DIR%Fake Artist - Fake Video [%ID%].m4a"
goto done
:writemp3file
echo fake mp3>"%DIR%Fake Artist - Fake Video [%ID%].mp3"
:done
if not "%MODE%"=="failafterwrite" exit /b 0
echo ERROR: [EmbedThumbnail] Unable to embed thumbnail 1>&2
exit /b 1
:json
if "%ISFLAT%"=="1" goto jsonplaylist
echo {"id":"%ID%","title":"Fake Video","uploader":"Fake Artist","duration":212,"thumbnail":"https://example.test/thumb.jpg","webpage_url":"https://www.youtube.com/watch?v=%ID%","ext":"mp4","filesize":123456}
exit /b 0
:jsonplaylist
echo {"_type":"playlist","id":"PLfake","title":"Fake Playlist","playlist_title":"Fake Playlist","playlist_count":3,"entries":[{"id":"vid001","title":"First Fake Video","url":"https://www.youtube.com/watch?v=vid001","duration":61},{"id":"vid002","title":"Second Fake Video","url":"https://www.youtube.com/watch?v=vid002","duration":62},{"id":"vid003","title":"Third Fake Video","url":"https://www.youtube.com/watch?v=vid003","duration":63}]}
exit /b 0
`

func shellScript(mode string) string {
	return strings.ReplaceAll(shellScriptTemplate, "{{MODE}}", mode)
}

func windowsScript(mode string) string {
	return strings.ReplaceAll(windowsScriptTemplate, "{{MODE}}", mode)
}
