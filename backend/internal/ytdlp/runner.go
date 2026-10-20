package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Runner executes the yt-dlp binary.
type Runner struct {
	Bin string
	// FfmpegBin is the path to the ffmpeg binary. When it is set it is handed
	// to yt-dlp, so the server does not depend on ffmpeg being on the PATH.
	FfmpegBin string
}

// NewRunner returns a runner that uses the given yt-dlp binary.
func NewRunner(bin string) *Runner {
	if bin == "" {
		bin = "yt-dlp"
	}
	return &Runner{Bin: bin}
}

// WithFfmpeg points the runner at an ffmpeg binary and returns it.
func (r *Runner) WithFfmpeg(path string) *Runner {
	r.FfmpegBin = resolveFfmpeg(path)
	return r
}

// resolveFfmpeg turns the configured ffmpeg into something yt-dlp can use.
// yt-dlp expects a path, not a program name: a bare "ffmpeg" would be looked
// up relative to the working directory. A name is therefore resolved against
// the PATH, and a directory is passed through as it is.
func resolveFfmpeg(path string) string {
	if path == "" {
		return ""
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	if strings.ContainsRune(path, os.PathSeparator) || strings.ContainsRune(path, '/') {
		return path
	}
	if abs, err := exec.LookPath(path); err == nil {
		return abs
	}
	return path
}

// Info is the subset of yt-dlp's JSON output the server cares about.
type Info struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Uploader      string  `json:"uploader"`
	Duration      float64 `json:"duration"`
	Thumbnail     string  `json:"thumbnail"`
	WebpageURL    string  `json:"webpage_url"`
	URL           string  `json:"url"`
	Ext           string  `json:"ext"`
	Filesize      int64   `json:"filesize"`
	Entries       []Info  `json:"entries"`
	PlaylistTitle string  `json:"playlist_title"`
}

// Probe asks yt-dlp for metadata. With flat set the playlist listing is
// requested, otherwise a single video is described.
func (r *Runner) Probe(ctx context.Context, rawURL string, flat bool) (Info, error) {
	var info Info

	args := []string{"-J"}
	if flat {
		args = append(args, "--flat-playlist")
	} else {
		args = append(args, "--no-playlist")
	}
	args = append(args, "--no-warnings", rawURL)

	cmd := prepareCmdPlatform(ctx, r.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = 2 * time.Second

	if err := cmd.Run(); err != nil {
		return info, fmt.Errorf("yt-dlp probe failed: %s", errorMessage(err, stderr.String()))
	}

	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		return info, fmt.Errorf("yt-dlp probe returned invalid JSON: %w", err)
	}
	return info, nil
}

// Download runs yt-dlp and returns the absolute path of the produced file.
// Progress lines are handed to onProgress while the process runs.
func (r *Runner) Download(ctx context.Context, mode Mode, outDir, rawURL string, onProgress func(Progress)) (string, error) {
	absDir, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve output directory %q: %w", outDir, err)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create output directory %q: %w", absDir, err)
	}

	args := BuildDownloadArgs(mode, absDir, rawURL, r.FfmpegBin)
	cmd := prepareCmdPlatform(ctx, r.Bin, args...)
	cmd.WaitDelay = 2 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("yt-dlp: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("yt-dlp: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		if p, ok := ParseProgress(scanner.Text()); ok && onProgress != nil {
			onProgress(p)
		}
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %s", errorMessage(err, stderr.String()))
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("yt-dlp: %w", err)
	}

	_, videoID, _, err := ClassifyURL(rawURL)
	if err != nil {
		return "", err
	}
	path, err := FindOutputFile(absDir, videoID, mode)
	if err != nil {
		return "", err
	}
	return path, nil
}

// errorMessage prefers the stderr of the failed process over the generic
// exec error, because that is where yt-dlp explains what went wrong.
func errorMessage(err error, stderr string) string {
	if msg := strings.TrimSpace(stderr); msg != "" {
		return msg
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return err.Error()
}

// FindOutputFile locates the media file yt-dlp produced. The output template
// always embeds the video id, so the file is found by looking for "[<id>]" in
// the file name. Only files that can hold media are considered: the cover art
// that is written next to the media shares the same base name.
func FindOutputFile(outDir, videoID string, mode Mode) (string, error) {
	if videoID == "" {
		return "", errors.New("yt-dlp produced no file: unknown video id")
	}

	matches, err := filepath.Glob(filepath.Join(outDir, "*"+videoID+"]*"))
	if err != nil {
		return "", err
	}

	media := mediaExtensions(mode)
	var candidates []string
	for _, m := range matches {
		base := filepath.Base(m)
		if !strings.Contains(base, "["+videoID+"]") {
			continue
		}
		if isIntermediate(base) {
			continue
		}
		if !media[strings.ToLower(filepath.Ext(base))] {
			continue
		}
		candidates = append(candidates, m)
	}
	if len(candidates) == 0 {
		if mode == ModeMusic {
			return "", fmt.Errorf("yt-dlp produced no MP3 for %s in %s (the extraction needs ffmpeg)", videoID, outDir)
		}
		return "", fmt.Errorf("yt-dlp produced no file for %s in %s", videoID, outDir)
	}

	// Prefer the extension the mode asks for, then the most recent file.
	preferred := preferredExtensions(mode)
	sort.SliceStable(candidates, func(i, j int) bool {
		pi, pj := preferred[strings.ToLower(filepath.Ext(candidates[i]))], preferred[strings.ToLower(filepath.Ext(candidates[j]))]
		if pi != pj {
			return pi
		}
		ti, tj := modTime(candidates[i]), modTime(candidates[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return candidates[i] < candidates[j]
	})
	return candidates[0], nil
}

// isIntermediate reports whether a file name belongs to a partial download or
// to a fragment yt-dlp merges into the final file.
func isIntermediate(name string) bool {
	lower := strings.ToLower(name)
	for _, suffix := range []string{".part", ".ytdl", ".temp", ".tmp"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	// Fragment files look like "Title [id].f137.mp4".
	if dot := strings.LastIndex(lower, "."); dot >= 0 {
		frag := lower[dot+1:]
		if len(frag) > 1 && frag[0] == 'f' {
			allDigits := true
			for _, r := range frag[1:] {
				if r < '0' || r > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				return true
			}
		}
	}
	return false
}

// mediaExtensions are the extensions that can hold the downloaded media.
// The cover art written next to the media is deliberately not in this list.
//
// In music mode only the MP3 counts: the promise of the music toggle is a real
// MP3, so a leftover source file (an .m4a that yt-dlp keeps when the
// extraction fails) must never be reported as a finished download.
func mediaExtensions(mode Mode) map[string]bool {
	if mode == ModeMusic {
		return map[string]bool{".mp3": true}
	}
	return map[string]bool{
		".mp4": true, ".mkv": true, ".webm": true,
		".mov": true, ".avi": true, ".m4v": true,
	}
}

func preferredExtensions(mode Mode) map[string]bool {
	if mode == ModeMusic {
		return map[string]bool{".mp3": true}
	}
	return map[string]bool{".mp4": true, ".mkv": true, ".webm": true, ".mov": true}
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
