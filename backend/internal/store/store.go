// Package store persists the library, the job list and the Google token as
// JSON documents under DATA_DIR. Every write is atomic (temp file + rename).
package store

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	libraryFile = "library.json"
	jobsFile    = "jobs.json"
	googleFile  = "google.json"
)

// Store is a JSON-backed data store. It is safe for concurrent use.
type Store struct {
	mu  sync.RWMutex
	dir string

	items []Item
	jobs  []Job

	googleRefreshToken string
	googleEmail        string
}

// New opens (and creates if needed) the store rooted at dir. Existing files
// are loaded on a best effort basis; a corrupt file simply starts empty.
func New(dir string) *Store {
	s := &Store{dir: dir}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return s
	}
	s.load()
	return s
}

func (s *Store) load() {
	var library struct {
		Items []Item `json:"items"`
	}
	if s.loadJSON(libraryFile, &library) {
		s.items = library.Items
	}

	var jobs struct {
		Jobs []Job `json:"jobs"`
	}
	if s.loadJSON(jobsFile, &jobs) {
		s.jobs = jobs.Jobs
	}

	var google struct {
		RefreshToken string `json:"refreshToken"`
		Email        string `json:"email"`
	}
	if s.loadJSON(googleFile, &google) {
		s.googleRefreshToken = google.RefreshToken
		s.googleEmail = google.Email
	}
}

// loadJSON reads a JSON document into out. A file that cannot be parsed is
// moved aside as <name>.corrupt: the next write must never destroy data that
// could not be read. It reports whether the document was read.
func (s *Store) loadJSON(name string, out any) bool {
	path := filepath.Join(s.dir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(raw, out); err != nil {
		log.Printf("the %s file could not be parsed (%v); it was kept as %s.corrupt", name, err, name)
		if err := os.Rename(path, path+".corrupt"); err != nil {
			log.Printf("could not move the broken %s file aside: %v", name, err)
		}
		return false
	}
	return true
}

// writeAtomic writes v as JSON to <name> inside the store directory by way of
// a temp file that is flushed and renamed into place.
func (s *Store) writeAtomic(name string, v any, perm os.FileMode) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, name+".tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	// Flush before the rename: a power cut must not leave a half written file
	// behind under the real name.
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Items returns a copy of the library, newest first.
func (s *Store) Items() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Item, len(s.items))
	copy(out, s.items)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].AddedAt.After(out[j].AddedAt)
	})
	return out
}

// AddItem appends an item to the library and persists it.
func (s *Store) AddItem(it Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, it)
	return s.writeAtomic(libraryFile, struct {
		Items []Item `json:"items"`
	}{Items: s.items}, 0o644)
}

// ItemByID returns the library item with the given id.
func (s *Store) ItemByID(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}

// RemoveItem deletes the record and the file (and cover) it points at.
func (s *Store) RemoveItem(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, it := range s.items {
		if it.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}

	it := s.items[idx]
	for _, path := range []string{it.File, it.Cover} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	s.items = append(s.items[:idx], s.items[idx+1:]...)
	return s.writeAtomic(libraryFile, struct {
		Items []Item `json:"items"`
	}{Items: s.items}, 0o644)
}

// HasVideo reports whether the library already holds a download of the given
// video in the given mode.
func (s *Store) HasVideo(videoID, mode string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, it := range s.items {
		if it.VideoID == videoID && it.Mode == mode {
			return true
		}
	}
	return false
}

// SetJobs replaces the persisted job list.
func (s *Store) SetJobs(jobs []Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = make([]Job, len(jobs))
	copy(s.jobs, jobs)
	return s.writeAtomic(jobsFile, struct {
		Jobs []Job `json:"jobs"`
	}{Jobs: s.jobs}, 0o644)
}

// Jobs returns a copy of the job list, newest first.
func (s *Store) Jobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Job, len(s.jobs))
	copy(out, s.jobs)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// SaveGoogleToken stores the OAuth refresh token and the account email.
func (s *Store) SaveGoogleToken(refreshToken, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.googleRefreshToken = refreshToken
	s.googleEmail = email
	return s.writeAtomic(googleFile, struct {
		RefreshToken string `json:"refreshToken"`
		Email        string `json:"email"`
	}{RefreshToken: refreshToken, Email: email}, 0o600)
}

// GoogleToken returns the stored refresh token and email.
func (s *Store) GoogleToken() (refreshToken, email string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.googleRefreshToken == "" {
		return "", "", false
	}
	return s.googleRefreshToken, s.googleEmail, true
}

// ClearGoogleToken forgets the stored token and removes the token file.
func (s *Store) ClearGoogleToken() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.googleRefreshToken = ""
	s.googleEmail = ""
	err := os.Remove(filepath.Join(s.dir, googleFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
