// Package ytdlp wraps the yt-dlp binary: argument construction, URL
// classification, progress parsing and the process runner.
package ytdlp

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Mode selects the download profile.
type Mode string

const (
	// ModeVideo downloads the best video+audio and merges it into mp4.
	ModeVideo Mode = "video"
	// ModeMusic extracts the best audio into a high quality mp3.
	ModeMusic Mode = "music"
)

// progressTemplate is the frozen --progress-template value from the spec.
// Field order: status, downloaded_bytes, total_bytes, total_bytes_estimate,
// speed, eta.
const progressTemplate = "dl|%(progress.status)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s"

// ParseMode converts a user supplied mode into a Mode. An empty mode means
// video; anything else that is not a known mode is an error.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ModeVideo, nil
	case string(ModeVideo):
		return ModeVideo, nil
	case string(ModeMusic):
		return ModeMusic, nil
	default:
		return "", fmt.Errorf("unknown mode %q (want %q or %q)", s, ModeVideo, ModeMusic)
	}
}

// BuildDownloadArgs returns the frozen yt-dlp argument list for the given
// mode. The lists are copied verbatim from the design spec.
//
// ffmpegLocation is the path to the ffmpeg binary (or its directory). When it
// is empty yt-dlp looks for ffmpeg on its own, which is what happens when the
// server runs outside the container.
func BuildDownloadArgs(mode Mode, outDir string, url string, ffmpegLocation string) []string {
	var args []string

	if mode == ModeMusic {
		args = []string{
			"--no-playlist",
			"--newline",
			"--no-warnings",
			"--progress-template", progressTemplate,
			"-f", "bestaudio/best",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "0",
			"--embed-thumbnail",
			"--embed-metadata",
			"--write-thumbnail",
			"-o", filepath.Join(outDir, "%(artist,creator,channel,uploader)s - %(title).150s [%(id)s].%(ext)s"),
		}
	} else {
		args = []string{
			"--no-playlist",
			"--newline",
			"--no-warnings",
			"--progress-template", progressTemplate,
			"-f", "bv*+ba/b",
			"--merge-output-format", "mp4",
			"--write-thumbnail",
			"-o", filepath.Join(outDir, "%(title).150s [%(id)s].%(ext)s"),
		}
	}

	if ffmpegLocation != "" {
		args = append(args, "--ffmpeg-location", ffmpegLocation)
	}
	return append(args, url)
}
