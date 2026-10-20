package ytdlp

import (
	"fmt"
	"net/url"
	"strings"
)

// Kind describes what a pasted URL points at.
type Kind string

const (
	KindVideo    Kind = "video"
	KindPlaylist Kind = "playlist"
	KindInvalid  Kind = "invalid"
)

// youtubeHosts are the hosts a YouTube URL may live on.
var youtubeHosts = map[string]bool{
	"youtube.com":              true,
	"www.youtube.com":          true,
	"m.youtube.com":            true,
	"music.youtube.com":        true,
	"gaming.youtube.com":       true,
	"youtube-nocookie.com":     true,
	"www.youtube-nocookie.com": true,
	"youtu.be":                 true,
}

// videoPathPrefixes are path segments that carry a bare video id.
var videoPathPrefixes = []string{"/shorts/", "/embed/", "/live/", "/v/"}

// ClassifyURL decides whether raw points at a single video, a playlist or
// nothing usable.
//
// A watch URL that also carries a list= parameter is a *video*: YouTube adds
// list= to every watch URL when the video belongs to a playlist.
func ClassifyURL(raw string) (Kind, string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return KindInvalid, "", "", fmt.Errorf("no URL provided")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return KindInvalid, "", "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return KindInvalid, "", "", fmt.Errorf("invalid URL %q: expected an http(s) URL", raw)
	}

	host := strings.ToLower(u.Hostname())
	if !youtubeHosts[host] {
		return KindInvalid, "", "", fmt.Errorf("invalid URL %q: not a YouTube URL", raw)
	}

	query := u.Query()

	// A video id always wins over a playlist id.
	if v := strings.TrimSpace(query.Get("v")); v != "" {
		if !isValidID(v) {
			return KindInvalid, "", "", fmt.Errorf("invalid URL %q: %q is not a video id", raw, v)
		}
		return KindVideo, v, "", nil
	}

	if host == "youtu.be" {
		id := strings.Trim(strings.TrimSpace(u.Path), "/")
		if id == "" || !isValidID(id) {
			return KindInvalid, "", "", fmt.Errorf("invalid URL %q: no video id", raw)
		}
		return KindVideo, id, "", nil
	}

	for _, prefix := range videoPathPrefixes {
		if strings.HasPrefix(u.Path, prefix) {
			id := strings.Trim(strings.TrimSpace(strings.TrimPrefix(u.Path, prefix)), "/")
			if id == "" || !isValidID(id) {
				return KindInvalid, "", "", fmt.Errorf("invalid URL %q: no video id", raw)
			}
			return KindVideo, id, "", nil
		}
	}

	if list := strings.TrimSpace(query.Get("list")); list != "" {
		if !isValidID(list) {
			return KindInvalid, "", "", fmt.Errorf("invalid URL %q: %q is not a playlist id", raw, list)
		}
		return KindPlaylist, "", list, nil
	}

	return KindInvalid, "", "", fmt.Errorf("invalid URL %q: no video or playlist id found", raw)
}

// isValidID keeps obviously bogus ids out.
func isValidID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
