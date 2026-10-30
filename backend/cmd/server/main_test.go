package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/config"
)

func TestNewServerServesTheHealthEndpoint(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Port:                8080,
		DataDir:             dir,
		YtDlpBin:            "yt-dlp",
		FfmpegBin:           "ffmpeg",
		ConcurrentDownloads: 2,
		DefaultMode:         "video",
	}

	handler, mgr, err := newServer(cfg)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	if handler == nil {
		t.Fatal("newServer returned no handler")
	}
	t.Cleanup(mgr.Close)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body == "" {
		t.Error("the health endpoint returned an empty body")
	}
}

func TestNewServerFailsWithoutADataDirectory(t *testing.T) {
	if _, _, err := newServer(config.Config{}); err == nil {
		t.Fatal("newServer error = nil, want an error")
	}
}
