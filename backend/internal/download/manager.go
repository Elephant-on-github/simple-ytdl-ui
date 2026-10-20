// Package download owns the job queue: it fans playlists out into single
// jobs, runs a bounded number of yt-dlp downloads in parallel and keeps the
// job list and the library in sync.
package download

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/ytdlp"
)

// Manager runs downloads.
type Manager struct {
	runner     *ytdlp.Runner
	store      *store.Store
	workDir    string
	concurrent int

	mu   sync.Mutex
	cond *sync.Cond

	jobs    []store.Job // oldest first
	cancels map[string]context.CancelFunc
	active  int
	persist map[string]time.Time

	queue chan *task
}

// task is the unit of work handed to the workers.
type task struct {
	id       string
	url      string
	videoID  string
	title    string
	artist   string
	duration int
	playlist string
	mode     ytdlp.Mode
	fromList bool
}

// NewManager starts the worker pool. Persisted jobs are restored so the UI
// keeps its history across restarts.
func NewManager(runner *ytdlp.Runner, st *store.Store, workDir string, concurrent int) *Manager {
	if concurrent < 1 {
		concurrent = 1
	}
	m := &Manager{
		runner:     runner,
		store:      st,
		workDir:    workDir,
		concurrent: concurrent,
		cancels:    map[string]context.CancelFunc{},
		persist:    map[string]time.Time{},
		queue:      make(chan *task, 1024),
	}
	m.cond = sync.NewCond(&m.mu)

	// Restore the persisted jobs, oldest first. Jobs that were still running
	// when the process stopped can never finish, so they are marked failed.
	stored := st.Jobs()
	for i := len(stored) - 1; i >= 0; i-- {
		j := stored[i]
		if j.Status == store.StatusQueued || j.Status == store.StatusRunning {
			j.Status = store.StatusFailed
			j.Error = "interrupted by a server restart"
		}
		m.jobs = append(m.jobs, j)
	}

	for i := 0; i < concurrent; i++ {
		go m.worker()
	}
	return m
}

// Enqueue turns a URL into one or more jobs. A playlist URL is expanded into
// one job per entry so every entry gets its own progress and a single failure
// does not kill the batch.
func (m *Manager) Enqueue(ctx context.Context, rawURL string, mode ytdlp.Mode, playlist bool) ([]store.Job, error) {
	kind, videoID, _, err := ytdlp.ClassifyURL(rawURL)
	if err != nil {
		return nil, err
	}

	tasks, err := m.plan(ctx, rawURL, mode, kind, videoID, playlist)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	var (
		out     []store.Job
		toQueue []*task
	)
	for _, t := range tasks {
		if m.store.HasVideo(t.videoID, string(mode)) {
			// Already in the library: never download the same video and mode
			// twice.
			if j := m.findJobLocked(t.videoID, string(mode)); j != nil {
				out = append(out, *j)
			} else {
				out = append(out, m.addFinishedJobLocked(t))
			}
			continue
		}
		if j := m.findActiveJobLocked(t.videoID, string(mode)); j != nil {
			out = append(out, *j)
			continue
		}

		now := time.Now().UTC()
		job := store.Job{
			ID:        newJobID(),
			Title:     t.title,
			URL:       t.url,
			VideoID:   t.videoID,
			Mode:      string(mode),
			Status:    store.StatusQueued,
			CreatedAt: now,
			UpdatedAt: now,
		}
		t.id = job.ID
		m.jobs = append(m.jobs, job)
		m.active++
		out = append(out, job)
		toQueue = append(toQueue, t)
	}
	m.persistLocked()
	m.mu.Unlock()

	for _, t := range toQueue {
		m.queue <- t
	}
	return out, nil
}

// plan resolves the URL into the tasks that have to be run.
func (m *Manager) plan(ctx context.Context, rawURL string, mode ytdlp.Mode, kind ytdlp.Kind, videoID string, playlist bool) ([]*task, error) {
	if playlist || kind == ytdlp.KindPlaylist {
		info, err := m.runner.Probe(ctx, rawURL, true)
		if err != nil {
			return nil, fmt.Errorf("cannot read playlist: %w", err)
		}
		title := info.PlaylistTitle
		if title == "" {
			title = info.Title
		}
		if title == "" {
			title = "Playlist"
		}
		tasks := make([]*task, 0, len(info.Entries))
		for _, e := range info.Entries {
			if e.ID == "" {
				continue
			}
			url := e.WebpageURL
			if url == "" {
				// A flat playlist listing carries the link in "url".
				url = e.URL
			}
			if url == "" {
				url = "https://www.youtube.com/watch?v=" + e.ID
			}
			tasks = append(tasks, &task{
				url:      url,
				videoID:  e.ID,
				title:    e.Title,
				artist:   e.Uploader,
				duration: int(e.Duration),
				playlist: title,
				mode:     mode,
				fromList: true,
			})
		}
		if len(tasks) == 0 {
			return nil, errors.New("the playlist is empty")
		}
		return tasks, nil
	}

	t := &task{url: rawURL, videoID: videoID, mode: mode}
	if info, err := m.runner.Probe(ctx, rawURL, false); err == nil {
		t.title = info.Title
		t.artist = info.Uploader
		t.duration = int(info.Duration)
		if info.ID != "" {
			t.videoID = info.ID
		}
	}
	if t.title == "" {
		t.title = t.videoID
	}
	return []*task{t}, nil
}

// worker pulls tasks off the queue until the manager is closed.
func (m *Manager) worker() {
	for t := range m.queue {
		m.run(t)
	}
}

// run executes a single download.
func (m *Manager) run(t *task) {
	m.mu.Lock()
	idx := m.indexOf(t.id)
	if idx < 0 {
		m.active--
		m.cond.Broadcast()
		m.mu.Unlock()
		return
	}
	if m.jobs[idx].Status == store.StatusCanceled {
		// Canceled while it was queued; Cancel already accounted for it.
		m.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[t.id] = cancel
	m.jobs[idx].Status = store.StatusRunning
	m.jobs[idx].UpdatedAt = time.Now().UTC()
	m.persistLocked()
	m.mu.Unlock()

	outDir := filepath.Join(m.workDir, string(t.mode), sanitise(t.playlist))
	path, err := m.runner.Download(ctx, t.mode, outDir, t.url, func(p ytdlp.Progress) {
		m.onProgress(t.id, p)
	})

	m.mu.Lock()
	defer m.mu.Unlock()

	canceled := ctx.Err() != nil
	delete(m.cancels, t.id)
	cancel()
	delete(m.persist, t.id)

	idx = m.indexOf(t.id)
	if idx < 0 {
		m.active--
		m.cond.Broadcast()
		return
	}

	job := &m.jobs[idx]
	job.UpdatedAt = time.Now().UTC()
	switch {
	case canceled:
		job.Status = store.StatusCanceled
		job.Error = ""
		job.Speed = 0
		job.ETA = 0
	case err != nil:
		// yt-dlp can fail after the media file was written — a thumbnail that
		// cannot be embedded for example. The file is there, so the download
		// counts as finished instead of leaving the file orphaned.
		if path, ferr := ytdlp.FindOutputFile(outDir, t.videoID, t.mode); ferr == nil {
			job.Status = store.StatusCompleted
			job.Progress = 100
			job.Total = job.Downloaded
			job.Speed = 0
			job.ETA = 0
			m.addItemLocked(t, path)
		} else {
			job.Status = store.StatusFailed
			job.Error = err.Error()
		}
	default:
		job.Status = store.StatusCompleted
		job.Progress = 100
		job.Total = job.Downloaded
		job.Speed = 0
		job.ETA = 0
		m.addItemLocked(t, path)
	}

	m.active--
	m.persistLocked()
	m.cond.Broadcast()
}

// onProgress records the progress of a running download.
func (m *Manager) onProgress(id string, p ytdlp.Progress) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.indexOf(id)
	if idx < 0 {
		return
	}
	job := &m.jobs[idx]
	job.Downloaded = p.Downloaded
	if p.Total > 0 {
		job.Total = p.Total
		job.Progress = float64(p.Downloaded) / float64(p.Total) * 100
		if job.Progress > 100 {
			job.Progress = 100
		}
	}
	job.Speed = p.Speed
	job.ETA = p.ETA
	job.UpdatedAt = time.Now().UTC()

	// Writing the job list on every progress line would be wasteful.
	if last, ok := m.persist[id]; !ok || time.Since(last) >= time.Second {
		m.persist[id] = time.Now()
		m.persistLocked()
	}
}

// addItemLocked stores a finished download in the library.
func (m *Manager) addItemLocked(t *task, path string) {
	var size int64
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}

	prefix := "yt"
	if t.fromList {
		prefix = "pl"
	}
	item := store.Item{
		ID:        fmt.Sprintf("%s_%s_%s", prefix, t.videoID, t.mode),
		VideoID:   t.videoID,
		Title:     t.title,
		Artist:    t.artist,
		Duration:  t.duration,
		File:      path,
		Cover:     findCover(path),
		Mime:      mimeFor(path),
		Size:      size,
		Mode:      string(t.mode),
		SourceURL: t.url,
		Playlist:  t.playlist,
		AddedAt:   time.Now().UTC(),
	}
	m.store.AddItem(item)
}

// addFinishedJobLocked records that a video is already in the library.
func (m *Manager) addFinishedJobLocked(t *task) store.Job {
	now := time.Now().UTC()
	job := store.Job{
		ID:        newJobID(),
		Title:     t.title,
		URL:       t.url,
		VideoID:   t.videoID,
		Mode:      string(t.mode),
		Status:    store.StatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.jobs = append(m.jobs, job)
	return job
}

// Jobs returns the job list, newest first.
func (m *Manager) Jobs() []store.Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]store.Job, 0, len(m.jobs))
	for i := len(m.jobs) - 1; i >= 0; i-- {
		out = append(out, m.jobs[i])
	}
	return out
}

// Cancel stops a queued or running job. A running download is killed.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := m.indexOf(id)
	if idx < 0 {
		return fmt.Errorf("unknown job %q", id)
	}

	switch m.jobs[idx].Status {
	case store.StatusQueued:
		m.jobs[idx].Status = store.StatusCanceled
		m.jobs[idx].UpdatedAt = time.Now().UTC()
		m.active--
		m.persistLocked()
		m.cond.Broadcast()
		return nil
	case store.StatusRunning:
		if cancel, ok := m.cancels[id]; ok {
			cancel()
		}
		return nil
	default:
		// Already finished, nothing to cancel.
		return nil
	}
}

// WaitIdle blocks until no job is queued or running. It is used by the tests.
func (m *Manager) WaitIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for m.active > 0 {
		m.cond.Wait()
	}
}

// Close stops the worker pool.
func (m *Manager) Close() {
	close(m.queue)
}

func (m *Manager) indexOf(id string) int {
	for i := range m.jobs {
		if m.jobs[i].ID == id {
			return i
		}
	}
	return -1
}

// findJobLocked returns the most recent job for a video and mode.
func (m *Manager) findJobLocked(videoID, mode string) *store.Job {
	for i := len(m.jobs) - 1; i >= 0; i-- {
		if m.jobs[i].VideoID == videoID && m.jobs[i].Mode == mode {
			return &m.jobs[i]
		}
	}
	return nil
}

// findActiveJobLocked returns a queued or running job for a video and mode.
func (m *Manager) findActiveJobLocked(videoID, mode string) *store.Job {
	for i := len(m.jobs) - 1; i >= 0; i-- {
		j := &m.jobs[i]
		if j.VideoID == videoID && j.Mode == mode && (j.Status == store.StatusQueued || j.Status == store.StatusRunning) {
			return j
		}
	}
	return nil
}

func (m *Manager) persistLocked() {
	jobs := make([]store.Job, len(m.jobs))
	copy(jobs, m.jobs)
	m.store.SetJobs(jobs)
}

func newJobID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("job_%d", time.Now().UnixNano())
	}
	return "job_" + hex.EncodeToString(buf[:])
}

// sanitise turns a playlist title into a safe directory name.
func sanitise(name string) string {
	if name == "" {
		return "singles"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 32, r == '/', r == '\\', r == ':', r == '*', r == '?', r == '"', r == '<', r == '>':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(strings.TrimRight(b.String(), ". "))
	if out == "" {
		return "singles"
	}
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

// findCover looks for a cover image next to the media file.
func findCover(media string) string {
	base := strings.TrimSuffix(media, filepath.Ext(media))
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		candidate := base + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// mimeFor guesses the mime type of a media file from its extension.
func mimeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".opus":
		return "audio/opus"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".mp4":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
