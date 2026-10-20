package ytdlp

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// wantProgressTemplate is the frozen --progress-template value from the spec.
const wantProgressTemplate = "dl|%(progress.status)s|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s"

func TestBuildDownloadArgsVideo(t *testing.T) {
	got := BuildDownloadArgs(ModeVideo, filepath.Join("/data", "video", "singles"), "URL")

	want := []string{
		"--no-playlist",
		"--newline",
		"--no-warnings",
		"--progress-template", wantProgressTemplate,
		"-f", "bv*+ba/b",
		"--merge-output-format", "mp4",
		"--write-thumbnail",
		"-o", filepath.Join("/data", "video", "singles", "%(title).150s [%(id)s].%(ext)s"),
		"URL",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildDownloadArgs(video) =\n%#v\nwant\n%#v", got, want)
	}
}

func TestBuildDownloadArgsMusic(t *testing.T) {
	got := BuildDownloadArgs(ModeMusic, filepath.Join("/data", "music", "singles"), "URL")

	want := []string{
		"--no-playlist",
		"--newline",
		"--no-warnings",
		"--progress-template", wantProgressTemplate,
		"-f", "bestaudio/best",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--embed-thumbnail",
		"--embed-metadata",
		"--write-thumbnail",
		"-o", filepath.Join("/data", "music", "singles", "%(artist,creator,channel,uploader)s - %(title).150s [%(id)s].%(ext)s"),
		"URL",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildDownloadArgs(music) =\n%#v\nwant\n%#v", got, want)
	}
}

func TestBuildDownloadArgsMusicExtractsMP3(t *testing.T) {
	args := BuildDownloadArgs(ModeMusic, "/data/music", "URL")

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-x") {
		t.Error("music args must extract audio (-x)")
	}
	if !strings.Contains(joined, "--audio-format mp3") {
		t.Error("music args must set --audio-format mp3")
	}
	if !strings.Contains(joined, "--audio-quality 0") {
		t.Error("music args must set --audio-quality 0 (best quality VBR)")
	}
	if !strings.Contains(joined, "bestaudio/best") {
		t.Error("music args must select bestaudio/best")
	}
	if !strings.Contains(joined, "--embed-thumbnail") || !strings.Contains(joined, "--embed-metadata") {
		t.Error("music args must embed thumbnail and metadata")
	}
}

func TestBuildDownloadArgsWritesThumbnailForCoverArt(t *testing.T) {
	// The library grid shows the cover art next to every file, so the
	// thumbnail has to stay on disk next to the media file.
	for _, mode := range []Mode{ModeVideo, ModeMusic} {
		args := BuildDownloadArgs(mode, "/data", "URL")
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--write-thumbnail") {
			t.Errorf("%s args must contain --write-thumbnail", mode)
		}
	}
}

func TestBuildDownloadArgsVideoDoesNotExtractAudio(t *testing.T) {
	args := BuildDownloadArgs(ModeVideo, "/data/video", "URL")

	for _, a := range args {
		if a == "-x" {
			t.Error("video args must not contain -x")
		}
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "bv*+ba/b") {
		t.Error("video args must use the bv*+ba/b format")
	}
	if !strings.Contains(joined, "--merge-output-format mp4") {
		t.Error("video args must merge to mp4")
	}
}

func TestBuildDownloadArgsAlwaysSingleAndStreamed(t *testing.T) {
	for _, mode := range []Mode{ModeVideo, ModeMusic} {
		args := BuildDownloadArgs(mode, "/data", "URL")
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--no-playlist") {
			t.Errorf("%s args must contain --no-playlist", mode)
		}
		if !strings.Contains(joined, "--newline") {
			t.Errorf("%s args must contain --newline", mode)
		}
		if !strings.Contains(joined, wantProgressTemplate) {
			t.Errorf("%s args must contain the frozen progress template", mode)
		}
		if args[len(args)-1] != "URL" {
			t.Errorf("%s args must end with the url, got %q", mode, args[len(args)-1])
		}
	}
}

func TestParseMode(t *testing.T) {
	if m, err := ParseMode(""); err != nil || m != ModeVideo {
		t.Errorf("ParseMode(\"\") = %q, %v; want video, nil", m, err)
	}
	if m, err := ParseMode("video"); err != nil || m != ModeVideo {
		t.Errorf("ParseMode(video) = %q, %v", m, err)
	}
	if m, err := ParseMode("music"); err != nil || m != ModeMusic {
		t.Errorf("ParseMode(music) = %q, %v", m, err)
	}
	if _, err := ParseMode("bogus"); err == nil {
		t.Error("ParseMode(bogus) error = nil, want error")
	}
}
