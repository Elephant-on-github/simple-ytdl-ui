package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/fakeytdlp"
)

func fakeRunner(t *testing.T, mode string) *Runner {
	t.Helper()
	bin, err := fakeytdlp.Write(t.TempDir(), mode)
	if err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return NewRunner(bin)
}

func TestResolveFfmpegTurnsANameIntoAPath(t *testing.T) {
	// yt-dlp expects a path. A bare program name would be looked up relative
	// to the working directory instead of the PATH.
	dir := t.TempDir()
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	binary := filepath.Join(dir, name)
	if err := os.WriteFile(binary, []byte("not a real ffmpeg"), 0o755); err != nil {
		t.Fatalf("write the fake ffmpeg: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got := resolveFfmpeg("ffmpeg")
	if got == "ffmpeg" {
		t.Errorf("resolveFfmpeg(ffmpeg) = %q, want an absolute path", got)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("resolveFfmpeg(ffmpeg) = %q, want an absolute path", got)
	}
}

func TestResolveFfmpegKeepsAPath(t *testing.T) {
	got := resolveFfmpeg("/usr/bin/ffmpeg")
	if got != "/usr/bin/ffmpeg" {
		t.Errorf("resolveFfmpeg = %q, want the path unchanged", got)
	}
	if resolveFfmpeg("") != "" {
		t.Error("resolveFfmpeg(\"\") must stay empty")
	}
}

func TestCookieArgsAreHandedToYtDlp(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)

	if r.HasCookies() {
		t.Error("HasCookies() = true without a session")
	}
	if got := r.cookieArgs(); len(got) != 0 {
		t.Errorf("cookieArgs() = %v, want none", got)
	}

	r.WithCookies("/data/cookies.txt", "")
	if !r.HasCookies() {
		t.Error("HasCookies() = false with a cookies file")
	}
	if got := strings.Join(r.cookieArgs(), " "); got != "--cookies /data/cookies.txt" {
		t.Errorf("cookieArgs() = %q", got)
	}

	r.WithCookies("", "firefox")
	if got := strings.Join(r.cookieArgs(), " "); got != "--cookies-from-browser firefox" {
		t.Errorf("cookieArgs() = %q", got)
	}
}

func TestPlaylistsWithoutASession(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)

	if _, err := r.Playlists(context.Background()); err == nil {
		t.Error("Playlists() succeeded without a session, want an error")
	}
}

func TestProbeReturnsCannedInfo(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)

	info, err := r.Probe(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", false)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.Title != "Fake Video" {
		t.Errorf("Title = %q, want Fake Video", info.Title)
	}
	if info.Uploader != "Fake Artist" {
		t.Errorf("Uploader = %q, want Fake Artist", info.Uploader)
	}
	if info.Duration != 212 {
		t.Errorf("Duration = %v, want 212", info.Duration)
	}
	if info.ID != "dQw4w9WgXcQ" {
		t.Errorf("ID = %q, want dQw4w9WgXcQ", info.ID)
	}
	if info.Thumbnail == "" {
		t.Error("Thumbnail is empty")
	}
}

func TestProbeFlatPlaylistReturnsEntries(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)

	info, err := r.Probe(context.Background(), "https://www.youtube.com/playlist?list=PLfake", true)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.PlaylistTitle != "Fake Playlist" {
		t.Errorf("PlaylistTitle = %q, want Fake Playlist", info.PlaylistTitle)
	}
	if len(info.Entries) != 3 {
		t.Fatalf("len(Entries) = %d, want 3", len(info.Entries))
	}
	if info.Entries[0].ID != "vid001" || info.Entries[0].Title != "First Fake Video" {
		t.Errorf("Entries[0] = %+v", info.Entries[0])
	}
	if info.Entries[2].ID != "vid003" {
		t.Errorf("Entries[2].ID = %q, want vid003", info.Entries[2].ID)
	}
	// A flat playlist listing carries the link of an entry in "url".
	if info.Entries[0].URL != "https://www.youtube.com/watch?v=vid001" {
		t.Errorf("Entries[0].URL = %q, want the url of the entry", info.Entries[0].URL)
	}
}

func TestProbeFailureSurfacesStderr(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeFail)

	_, err := r.Probe(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", false)
	if err == nil {
		t.Fatal("Probe error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "Video unavailable") {
		t.Errorf("error = %q, want it to contain the yt-dlp stderr", err)
	}
}

func TestDownloadReportsProgressAndReturnsFile(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)
	out := t.TempDir()

	var seen []Progress
	path, err := r.Download(context.Background(), ModeVideo, out, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", func(p Progress) {
		seen = append(seen, p)
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}

	if len(seen) == 0 {
		t.Fatal("no progress reported")
	}
	if seen[0].Status != "downloading" {
		t.Errorf("first progress status = %q, want downloading", seen[0].Status)
	}
	if seen[0].Total != 10485760 {
		t.Errorf("first progress total = %d, want 10485760", seen[0].Total)
	}
	if seen[len(seen)-1].Status != "finished" {
		t.Errorf("last progress status = %q, want finished", seen[len(seen)-1].Status)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("path %q is not absolute", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded file does not exist: %v", err)
	}
	if !strings.Contains(filepath.Base(path), "[dQw4w9WgXcQ]") {
		t.Errorf("file name %q does not carry the video id", filepath.Base(path))
	}
}

func TestDownloadMusicModeProducesMp3(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)
	out := t.TempDir()

	path, err := r.Download(context.Background(), ModeMusic, out, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if !strings.HasSuffix(path, ".mp3") {
		t.Errorf("path = %q, want an .mp3 file", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded file does not exist: %v", err)
	}
}

func TestDownloadWithListParameterStillUsesTheVideoID(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeOK)
	out := t.TempDir()

	path, err := r.Download(context.Background(), ModeVideo, out, "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLfake", nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if !strings.Contains(filepath.Base(path), "[dQw4w9WgXcQ]") {
		t.Errorf("file name %q does not carry the video id", filepath.Base(path))
	}
}

func TestDownloadFailureSurfacesStderr(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeFail)
	out := t.TempDir()

	path, err := r.Download(context.Background(), ModeVideo, out, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
	if err == nil {
		t.Fatal("Download error = nil, want an error")
	}
	if path != "" {
		t.Errorf("path = %q, want empty on failure", path)
	}
	if !strings.Contains(err.Error(), "Video unavailable") {
		t.Errorf("error = %q, want it to contain the yt-dlp stderr", err)
	}
}

func TestDownloadCancellationStopsTheProcess(t *testing.T) {
	r := fakeRunner(t, fakeytdlp.ModeSleep)
	out := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := r.Download(ctx, ModeVideo, out, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
		done <- err
	}()

	// Give the fake time to start, then cancel.
	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Download error = nil after cancel, want an error")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Download did not return after the context was canceled")
	}
}
