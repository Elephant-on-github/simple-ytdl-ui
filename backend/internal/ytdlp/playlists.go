package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// LibraryURL is the page that holds the playlists of the signed in account.
// yt-dlp's tab extractor reads it; without a session YouTube answers with a
// 401, which is why the server needs cookies for it.
const LibraryURL = "https://www.youtube.com/feed/playlists"

// Playlist is one playlist of the account.
type Playlist struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Thumb string `json:"thumb"`
}

// Playlists lists the playlists of the account the cookies belong to.
//
// It uses yt-dlp's own tab extractor, so no Google Cloud project, no OAuth
// client and no API quota are involved — just the YouTube session in the
// cookies.
func (r *Runner) Playlists(ctx context.Context) ([]Playlist, error) {
	if !r.HasCookies() {
		return nil, fmt.Errorf("no YouTube session: set YTDLP_COOKIES or YTDLP_COOKIES_FROM_BROWSER")
	}

	args := []string{
		"-J",
		"--flat-playlist",
		"--no-warnings",
		"--extractor-args", "youtubetab:skip=authcheck",
	}
	session, cleanup, err := r.sessionArgs()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	args = append(args, session...)
	args = append(args, r.browserCookieArgs()...)
	args = append(args, LibraryURL)

	cmd := exec.CommandContext(ctx, r.Bin, args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr != "" {
			return nil, fmt.Errorf("yt-dlp could not read your playlists: %s", firstLine(stderr))
		}
		return nil, fmt.Errorf("yt-dlp could not read your playlists: %w", err)
	}

	var doc struct {
		Entries []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			URL       string `json:"url"`
			WebpageURL string `json:"webpage_url"`
			Thumbnails []struct {
				URL string `json:"url"`
			} `json:"thumbnails"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("yt-dlp returned invalid JSON: %w", err)
	}

	playlists := make([]Playlist, 0, len(doc.Entries))
	for _, entry := range doc.Entries {
		if entry.ID == "" {
			continue
		}
		url := entry.WebpageURL
		if url == "" {
			url = entry.URL
		}
		if url == "" {
			url = "https://www.youtube.com/playlist?list=" + entry.ID
		}
		thumb := ""
		if len(entry.Thumbnails) > 0 {
			thumb = entry.Thumbnails[len(entry.Thumbnails)-1].URL
		}
		playlists = append(playlists, Playlist{
			ID:    entry.ID,
			Title: entry.Title,
			URL:   url,
			Thumb: thumb,
		})
	}
	return playlists, nil
}

// firstLine keeps an error message short.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
