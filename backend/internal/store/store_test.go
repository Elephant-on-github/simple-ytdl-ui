package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestItemJSONFieldNames(t *testing.T) {
	it := Item{
		ID:        "yt_abc_video",
		VideoID:   "abc",
		Title:     "Title",
		Artist:    "Artist",
		Duration:  212,
		File:      "/data/video/singles/Title [abc].mp4",
		Cover:     "/data/video/singles/Title [abc].jpg",
		Mime:      "video/mp4",
		Size:      1024,
		Mode:      "video",
		SourceURL: "https://www.youtube.com/watch?v=abc",
		Playlist:  "",
		AddedAt:   time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
	}

	raw, err := json.Marshal(it)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		"id":        "yt_abc_video",
		"videoId":   "abc",
		"title":     "Title",
		"artist":    "Artist",
		"duration":  float64(212),
		"file":      "/data/video/singles/Title [abc].mp4",
		"cover":     "/data/video/singles/Title [abc].jpg",
		"mime":      "video/mp4",
		"size":      float64(1024),
		"mode":      "video",
		"sourceUrl": "https://www.youtube.com/watch?v=abc",
		"playlist":  "",
		"addedAt":   "2026-10-20T12:00:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("json[%q] = %v, want %v", k, got[k], v)
		}
	}
}

func TestJobJSONFieldNames(t *testing.T) {
	j := Job{
		ID:         "job_1",
		Title:      "Title",
		URL:        "https://youtu.be/abc",
		VideoID:    "abc",
		Mode:       "music",
		Status:     "running",
		Progress:   42.5,
		Downloaded: 100,
		Total:      200,
		Speed:      1024.5,
		ETA:        12,
		Error:      "",
		CreatedAt:  time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 10, 20, 12, 1, 0, 0, time.UTC),
	}

	raw, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		"id":         "job_1",
		"title":      "Title",
		"url":        "https://youtu.be/abc",
		"videoId":    "abc",
		"mode":       "music",
		"status":     "running",
		"progress":   42.5,
		"downloaded": float64(100),
		"total":      float64(200),
		"speed":      1024.5,
		"eta":        float64(12),
		"error":      "",
		"createdAt":  "2026-10-20T12:00:00Z",
		"updatedAt":  "2026-10-20T12:01:00Z",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("json[%q] = %v, want %v", k, got[k], v)
		}
	}
}

func TestAddItemRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	now := time.Now().UTC().Truncate(time.Second)
	if err := s.AddItem(Item{ID: "yt_1", VideoID: "1", Title: "One", AddedAt: now}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	items := s.Items()
	if len(items) != 1 {
		t.Fatalf("Items() len = %d, want 1", len(items))
	}
	if items[0].Title != "One" {
		t.Errorf("Title = %q, want One", items[0].Title)
	}
	if !items[0].AddedAt.Equal(now) {
		t.Errorf("AddedAt = %v, want %v", items[0].AddedAt, now)
	}

	// The item must survive a reload from disk.
	reloaded := New(dir)
	if got := reloaded.Items(); len(got) != 1 || got[0].ID != "yt_1" {
		t.Fatalf("reloaded Items() = %+v", got)
	}

	// library.json must exist and be valid JSON.
	raw, err := os.ReadFile(filepath.Join(dir, "library.json"))
	if err != nil {
		t.Fatalf("read library.json: %v", err)
	}
	var doc struct {
		Items []Item `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("library.json is not valid JSON: %v", err)
	}
	if len(doc.Items) != 1 {
		t.Errorf("library.json items = %d, want 1", len(doc.Items))
	}
}

func TestItemsNewestFirst(t *testing.T) {
	s := New(t.TempDir())

	base := time.Now().UTC()
	for i, title := range []string{"oldest", "middle", "newest"} {
		it := Item{
			ID:      "yt_" + title,
			VideoID: title,
			Title:   title,
			AddedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := s.AddItem(it); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
	}

	items := s.Items()
	if len(items) != 3 {
		t.Fatalf("Items() len = %d, want 3", len(items))
	}
	want := []string{"newest", "middle", "oldest"}
	for i, w := range want {
		if items[i].Title != w {
			t.Errorf("Items()[%d] = %q, want %q", i, items[i].Title, w)
		}
	}
}

func TestRemoveItemDeletesFileAndRecord(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	media := filepath.Join(dir, "Title [abc].mp4")
	if err := os.WriteFile(media, []byte("data"), 0o644); err != nil {
		t.Fatalf("write media: %v", err)
	}

	if err := s.AddItem(Item{ID: "yt_abc", VideoID: "abc", Title: "Title", File: media}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := s.RemoveItem("yt_abc"); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}

	if _, err := os.Stat(media); !os.IsNotExist(err) {
		t.Errorf("media file still present on disk (err=%v)", err)
	}
	if _, ok := s.ItemByID("yt_abc"); ok {
		t.Error("ItemByID still finds the removed item")
	}
	if len(s.Items()) != 0 {
		t.Errorf("Items() len = %d, want 0", len(s.Items()))
	}

	// The record must be gone from disk too.
	reloaded := New(dir)
	if len(reloaded.Items()) != 0 {
		t.Errorf("reloaded Items() len = %d, want 0", len(reloaded.Items()))
	}
}

func TestRemoveItemIgnoresMissingFile(t *testing.T) {
	s := New(t.TempDir())
	if err := s.AddItem(Item{ID: "yt_abc", VideoID: "abc", File: filepath.Join(t.TempDir(), "gone.mp4")}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if err := s.RemoveItem("yt_abc"); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if _, ok := s.ItemByID("yt_abc"); ok {
		t.Error("record was not removed")
	}
}

func TestRemoveUnknownItemIsNoError(t *testing.T) {
	s := New(t.TempDir())
	if err := s.RemoveItem("nope"); err != nil {
		t.Errorf("RemoveItem(unknown) = %v, want nil", err)
	}
}

func TestHasVideoDistinguishesModes(t *testing.T) {
	s := New(t.TempDir())
	if err := s.AddItem(Item{ID: "yt_abc_music", VideoID: "abc", Mode: "music"}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if !s.HasVideo("abc", "music") {
		t.Error("HasVideo(abc, music) = false, want true")
	}
	if s.HasVideo("abc", "video") {
		t.Error("HasVideo(abc, video) = true, want false")
	}
	if s.HasVideo("other", "music") {
		t.Error("HasVideo(other, music) = true, want false")
	}
}

func TestJobsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	jobs := []Job{
		{ID: "job_1", Title: "One", Status: "completed", CreatedAt: time.Now().UTC()},
		{ID: "job_2", Title: "Two", Status: "failed", Error: "boom", CreatedAt: time.Now().UTC().Add(time.Minute)},
	}
	if err := s.SetJobs(jobs); err != nil {
		t.Fatalf("SetJobs: %v", err)
	}

	got := s.Jobs()
	if len(got) != 2 {
		t.Fatalf("Jobs() len = %d, want 2", len(got))
	}
	// Newest first.
	if got[0].ID != "job_2" {
		t.Errorf("Jobs()[0].ID = %q, want job_2", got[0].ID)
	}
	if got[1].Error != "" || got[0].Error != "boom" {
		t.Errorf("errors not preserved: %+v", got)
	}

	reloaded := New(dir)
	if len(reloaded.Jobs()) != 2 {
		t.Errorf("reloaded Jobs() len = %d, want 2", len(reloaded.Jobs()))
	}
}

func TestGoogleTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	if _, _, ok := s.GoogleToken(); ok {
		t.Error("GoogleToken() ok = true before any token was stored")
	}

	if err := s.SaveGoogleToken("refresh-me", "user@example.test"); err != nil {
		t.Fatalf("SaveGoogleToken: %v", err)
	}

	rt, email, ok := s.GoogleToken()
	if !ok {
		t.Fatal("GoogleToken() ok = false, want true")
	}
	if rt != "refresh-me" {
		t.Errorf("refresh token = %q, want refresh-me", rt)
	}
	if email != "user@example.test" {
		t.Errorf("email = %q, want user@example.test", email)
	}

	reloaded := New(dir)
	rt, email, ok = reloaded.GoogleToken()
	if !ok || rt != "refresh-me" || email != "user@example.test" {
		t.Errorf("reloaded token = %q/%q/%v", rt, email, ok)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "google.json"))
		if err != nil {
			t.Fatalf("stat google.json: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("google.json mode = %v, want 0600", perm)
		}
	}

	if err := s.ClearGoogleToken(); err != nil {
		t.Fatalf("ClearGoogleToken: %v", err)
	}
	if _, _, ok := s.GoogleToken(); ok {
		t.Error("GoogleToken() ok = true after ClearGoogleToken")
	}
	if _, err := os.Stat(filepath.Join(dir, "google.json")); !os.IsNotExist(err) {
		t.Errorf("google.json still on disk (err=%v)", err)
	}
}

func TestStoreWritesAreAtomic(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.AddItem(Item{ID: "yt_1", VideoID: "1"}); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	// No leftover temp files.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file %q", e.Name())
		}
	}
}
