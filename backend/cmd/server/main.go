// Command server runs the simple-ytdl-ui backend: the JSON API and the
// single page application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/api"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/config"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/download"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/ytdlp"
)

func main() {
	cfg := config.Load()

	handler, err := newServer(cfg)
	if err != nil {
		log.Fatalf("could not start the server: %v", err)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		log.Println("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown failed: %v", err)
		}
	}()

	log.Printf("simple-ytdl-ui listening on http://localhost:%d (data dir %s)", cfg.Port, cfg.DataDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

// newServer wires the whole application together and returns its handler.
func newServer(cfg config.Config) (http.Handler, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("no data directory configured")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create the data directory %q: %w", cfg.DataDir, err)
	}

	st := store.New(cfg.DataDir)
	runner := ytdlp.NewRunner(cfg.YtDlpBin).WithFfmpeg(cfg.FfmpegBin)

	// The YouTube session. A session pasted into the app wins; otherwise the
	// file the container was started with is copied into the data directory.
	// The copy matters: yt-dlp rewrites the cookie jar after every run, so it
	// needs a file it can write to, and a mounted one is often read-only.
	if !st.HasCookies() && cfg.CookiesFile != "" {
		if err := st.SaveCookiesFrom(cfg.CookiesFile); err != nil {
			log.Printf("could not use the youtube session in %s: %v", cfg.CookiesFile, err)
		}
	}
	switch {
	case st.HasCookies():
		runner.WithCookies(st.CookiesPath(), "")
	case cfg.CookiesFromBrowser != "":
		runner.WithCookies("", cfg.CookiesFromBrowser)
	}

	// The media can live on its own volume: LIBRARY_DIR if it is set,
	// the data directory otherwise.
	mgr := download.NewManager(runner, st, cfg.MediaDir(), cfg.ConcurrentDownloads)

	return api.New(cfg, st, mgr, nil).Handler(), nil
}
