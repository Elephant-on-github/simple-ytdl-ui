package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/fakeytdlp"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/ytdlp"
)

type fixture struct {
	manager *Manager
	store   *store.Store
	dir     string
	bin     string
	count   string
}

func newFixture(t *testing.T, mode string) *fixture {
	t.Helper()

	dir := t.TempDir()
	bin, err := fakeytdlp.Write(t.TempDir(), mode)
	if err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}

	// Every download creates a directory in this folder so the tests can
	// count them. A directory is used because two downloads that run at the
	// same time would overwrite a line in a file.
	count := filepath.Join(t.TempDir(), "downloads")
	if err := os.MkdirAll(count, 0o755); err != nil {
		t.Fatalf("create the counter directory: %v", err)
	}
	t.Setenv("FAKE_YTDLP_COUNT", count)

	st := store.New(dir)
	mgr := NewManager(ytdlp.NewRunner(bin), st, dir, 2)
	t.Cleanup(mgr.Close)

	return &fixture{manager: mgr, store: st, dir: dir, bin: bin, count: count}
}

func (f *fixture) downloads(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(f.count)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read the download counter: %v", err)
	}
	return len(entries)
}

func jobByID(jobs []store.Job, id string) (store.Job, bool) {
	for _, j := range jobs {
		if j.ID == id {
			return j, true
		}
	}
	return store.Job{}, false
}

func TestEnqueueSingleVideoCompletes(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if jobs[0].Status != store.StatusQueued {
		t.Errorf("status = %q, want queued", jobs[0].Status)
	}
	if jobs[0].VideoID != "dQw4w9WgXcQ" {
		t.Errorf("videoId = %q, want dQw4w9WgXcQ", jobs[0].VideoID)
	}

	f.manager.WaitIdle()

	final, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if final.Status != store.StatusCompleted {
		t.Fatalf("status = %q (%s), want completed", final.Status, final.Error)
	}
	if final.Progress != 100 {
		t.Errorf("progress = %v, want 100", final.Progress)
	}

	items := f.store.Items()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.VideoID != "dQw4w9WgXcQ" {
		t.Errorf("item videoId = %q", item.VideoID)
	}
	if item.Mode != "video" {
		t.Errorf("item mode = %q, want video", item.Mode)
	}
	if item.Title != "Fake Video" {
		t.Errorf("item title = %q, want Fake Video", item.Title)
	}
	if item.Artist != "Fake Artist" {
		t.Errorf("item artist = %q, want Fake Artist", item.Artist)
	}
	if item.Duration != 212 {
		t.Errorf("item duration = %d, want 212", item.Duration)
	}
	if item.Mime != "video/mp4" {
		t.Errorf("item mime = %q, want video/mp4", item.Mime)
	}
	if _, err := os.Stat(item.File); err != nil {
		t.Errorf("item file %q missing: %v", item.File, err)
	}
	if item.SourceURL != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("item sourceUrl = %q", item.SourceURL)
	}
}

func TestEnqueuePlaylistCreatesOneJobPerEntry(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/playlist?list=PLfake", ytdlp.ModeMusic, true)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("len(jobs) = %d, want 3", len(jobs))
	}

	f.manager.WaitIdle()

	for _, j := range f.manager.Jobs() {
		if j.Status != store.StatusCompleted {
			t.Errorf("job %s status = %q (%s), want completed", j.ID, j.Status, j.Error)
		}
	}

	items := f.store.Items()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	for _, item := range items {
		if item.Playlist != "Fake Playlist" {
			t.Errorf("item playlist = %q, want Fake Playlist", item.Playlist)
		}
		if item.Mode != "music" {
			t.Errorf("item mode = %q, want music", item.Mode)
		}
		if !strings.HasPrefix(item.ID, "pl_") {
			t.Errorf("item id = %q, want a pl_ prefix", item.ID)
		}
	}
}

func TestWatchURLWithListParameterIsASingleVideo(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLfake", ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	if jobs[0].VideoID != "dQw4w9WgXcQ" {
		t.Errorf("videoId = %q, want dQw4w9WgXcQ", jobs[0].VideoID)
	}
	f.manager.WaitIdle()
}

func TestInvalidURLIsRejected(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	if _, err := f.manager.Enqueue(context.Background(), "https://example.com/song.mp3", ytdlp.ModeVideo, false); err == nil {
		t.Fatal("Enqueue error = nil, want an error")
	}
	if len(f.manager.Jobs()) != 0 {
		t.Errorf("jobs = %d, want 0", len(f.manager.Jobs()))
	}
}

func TestFailedDownloadRecordsStderr(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeFail)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	f.manager.WaitIdle()

	final, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if final.Status != store.StatusFailed {
		t.Fatalf("status = %q, want failed", final.Status)
	}
	if !strings.Contains(final.Error, "Video unavailable") {
		t.Errorf("error = %q, want it to contain the yt-dlp stderr", final.Error)
	}
	if len(f.store.Items()) != 0 {
		t.Errorf("library has %d items, want 0", len(f.store.Items()))
	}
}

func TestCancelRunningJobLeavesItCanceled(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeSleep)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait until the job is actually running.
	deadline := time.Now().Add(10 * time.Second)
	for {
		j, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
		if ok && j.Status == store.StatusRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("job never started")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := f.manager.Cancel(jobs[0].ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	f.manager.WaitIdle()

	final, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if final.Status != store.StatusCanceled {
		t.Fatalf("status = %q, want canceled", final.Status)
	}
	if len(f.store.Items()) != 0 {
		t.Errorf("library has %d items, want 0", len(f.store.Items()))
	}
}

func TestCancelUnknownJobFails(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)
	if err := f.manager.Cancel("nope"); err == nil {
		t.Fatal("Cancel error = nil, want an error")
	}
}

func TestSameVideoAndModeIsNotDownloadedTwice(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)
	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	first, err := f.manager.Enqueue(context.Background(), url, ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	f.manager.WaitIdle()

	second, err := f.manager.Enqueue(context.Background(), url, ytdlp.ModeVideo, false)
	if err != nil {
		t.Fatalf("Enqueue (again): %v", err)
	}
	f.manager.WaitIdle()

	if got := f.downloads(t); got != 1 {
		t.Errorf("downloads = %d, want 1", got)
	}
	if len(f.store.Items()) != 1 {
		t.Errorf("library has %d items, want 1", len(f.store.Items()))
	}
	if len(second) != 1 {
		t.Fatalf("len(second) = %d, want 1", len(second))
	}
	if second[0].ID != first[0].ID {
		t.Errorf("second job id = %q, want the existing job %q", second[0].ID, first[0].ID)
	}
	if second[0].Status != store.StatusCompleted {
		t.Errorf("second job status = %q, want completed", second[0].Status)
	}
}

func TestSameVideoInAnotherModeIsDownloaded(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)
	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	if _, err := f.manager.Enqueue(context.Background(), url, ytdlp.ModeMusic, false); err != nil {
		t.Fatalf("Enqueue (music): %v", err)
	}
	f.manager.WaitIdle()

	if _, err := f.manager.Enqueue(context.Background(), url, ytdlp.ModeVideo, false); err != nil {
		t.Fatalf("Enqueue (video): %v", err)
	}
	f.manager.WaitIdle()

	if got := f.downloads(t); got != 2 {
		t.Errorf("downloads = %d, want 2", got)
	}
	if len(f.store.Items()) != 2 {
		t.Errorf("library has %d items, want 2", len(f.store.Items()))
	}

	ids := map[string]bool{}
	for _, item := range f.store.Items() {
		if ids[item.ID] {
			t.Errorf("duplicate item id %q", item.ID)
		}
		ids[item.ID] = true
	}
}

func TestMusicDownloadProducesAnMp3(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	if _, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeMusic, false); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	f.manager.WaitIdle()

	items := f.store.Items()
	if len(items) != 1 {
		t.Fatalf("library has %d items, want 1", len(items))
	}
	if !strings.HasSuffix(items[0].File, ".mp3") {
		t.Errorf("file = %q, want an .mp3", items[0].File)
	}
	if items[0].Mime != "audio/mpeg" {
		t.Errorf("mime = %q, want audio/mpeg", items[0].Mime)
	}

	// The extraction must not leave an intermediate behind.
	dir := filepath.Dir(items[0].File)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the output directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != filepath.Base(items[0].File) {
			t.Errorf("the output directory still holds %q", entry.Name())
		}
	}
}

func TestJobsAreReturnedNewestFirst(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)

	if _, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=vid001", ytdlp.ModeVideo, false); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	f.manager.WaitIdle()
	if _, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=vid002", ytdlp.ModeVideo, false); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	f.manager.WaitIdle()

	jobs := f.manager.Jobs()
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}
	if jobs[0].VideoID != "vid002" {
		t.Errorf("jobs[0].videoId = %q, want vid002", jobs[0].VideoID)
	}
}

func TestFinishedFileSurvivesAPostProcessingFailure(t *testing.T) {
	// yt-dlp can exit non-zero after the media file was written, for example
	// when a thumbnail cannot be embedded. The file is there, so the download
	// must not be reported as failed and the file must not be orphaned.
	f := newFixture(t, fakeytdlp.ModeFailAfterWrite)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeMusic, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	f.manager.WaitIdle()

	final, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if final.Status != store.StatusCompleted {
		t.Fatalf("status = %q (%s), want completed", final.Status, final.Error)
	}

	items := f.store.Items()
	if len(items) != 1 {
		t.Fatalf("library has %d items, want 1", len(items))
	}
	if _, err := os.Stat(items[0].File); err != nil {
		t.Errorf("item file %q missing: %v", items[0].File, err)
	}
}

func TestMusicModeWithoutAnMp3IsNotReportedAsFinished(t *testing.T) {
	// Without ffmpeg yt-dlp keeps the source audio. The music toggle promises
	// an MP3, so the job must fail instead of handing over a file that is not
	// an MP3.
	f := newFixture(t, fakeytdlp.ModeNoMusic)

	jobs, err := f.manager.Enqueue(context.Background(), "https://www.youtube.com/watch?v=dQw4w9WgXcQ", ytdlp.ModeMusic, false)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	f.manager.WaitIdle()

	final, ok := jobByID(f.manager.Jobs(), jobs[0].ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if final.Status != store.StatusFailed {
		t.Fatalf("status = %q, want failed", final.Status)
	}
	if len(f.store.Items()) != 0 {
		t.Errorf("library has %d items, want 0", len(f.store.Items()))
	}
}

func TestEnqueueingManyEntriesDoesNotBlock(t *testing.T) {
	// A big playlist must return at once: the browser waits for this request
	// and would otherwise show "Starting…" for the length of the whole batch.
	f := newFixture(t, fakeytdlp.ModeSleep)

	entries := make([]ytdlp.Info, 0, 1500)
	for i := 0; i < 1500; i++ {
		entries = append(entries, ytdlp.Info{
			ID:    fmt.Sprintf("vid%04d", i),
			Title: fmt.Sprintf("Video %04d", i),
		})
	}

	done := make(chan []store.Job, 1)
	go func() {
		jobs, err := f.manager.enqueueEntries(ytdlp.ModeVideo, entries, "Big Playlist")
		if err != nil {
			t.Errorf("enqueueEntries: %v", err)
		}
		done <- jobs
	}()

	select {
	case jobs := <-done:
		if len(jobs) != len(entries) {
			t.Errorf("len(jobs) = %d, want %d", len(jobs), len(entries))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("enqueueEntries blocked while the workers were busy")
	}
}

func TestSyncingTheSamePlaylistTwiceDoesNotDownloadTwice(t *testing.T) {
	f := newFixture(t, fakeytdlp.ModeOK)
	url := "https://www.youtube.com/playlist?list=PLfake"

	for i := 0; i < 2; i++ {
		if _, err := f.manager.Enqueue(context.Background(), url, ytdlp.ModeMusic, true); err != nil {
			t.Fatalf("Enqueue (round %d): %v", i, err)
		}
		f.manager.WaitIdle()
	}

	if got := f.downloads(t); got != 3 {
		t.Errorf("downloads = %d, want 3 (one per track, not two)", got)
	}
	if len(f.store.Items()) != 3 {
		t.Errorf("library has %d items, want 3", len(f.store.Items()))
	}
}

func TestSanitisePlaylistTitle(t *testing.T) {
	tests := map[string]string{
		"":                 "singles",
		"My Playlist":      "My Playlist",
		"a/b\\c:d*e?f":     "a_b_c_d_e_f",
		"  spaced  ":       "spaced",
		"trailing dots...": "trailing dots",
	}
	for in, want := range tests {
		if got := sanitise(in); got != want {
			t.Errorf("sanitise(%q) = %q, want %q", in, got, want)
		}
	}
}
