// Package config loads the server configuration from the process environment.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds every runtime knob of the server.
type Config struct {
	Port               int
	DataDir            string
	LibraryDir         string
	YtDlpBin           string
	FfmpegBin          string
	GoogleClientID     string
	GoogleClientSecret string
	// GoogleRedirectURL is the URI Google sends the browser back to. It is
	// optional: when it is empty the server derives it from the request,
	// which is what a deployment behind a proxy needs.
	GoogleRedirectURL   string
	ConcurrentDownloads int
	DefaultMode         string // "video" | "music"
	// AuthPassword guards the app. When it is empty the app is open.
	AuthPassword string
	// AuthAllowedEmails are the accounts that may sign in with Google
	// instead of the password, comma separated.
	AuthAllowedEmails string
	// CookiesFile is a Netscape cookies.txt with a YouTube session. It lets
	// the server reach private playlists and list your own library, with no
	// Google Cloud project involved.
	CookiesFile string
	// CookiesFromBrowser asks yt-dlp to read the cookies out of a browser
	// profile on this machine, e.g. "firefox" or "chrome".
	CookiesFromBrowser string
	// WebDir optionally points at a directory holding the built frontend.
	// When empty the embedded copy of the SPA is served.
	WebDir string
}

// Load reads the configuration from the environment, falling back to the
// documented defaults for anything unset or unparsable.
func Load() Config {
	return Config{
		Port:                envInt("PORT", 8080),
		DataDir:             envString("DATA_DIR", "/data"),
		LibraryDir:          os.Getenv("LIBRARY_DIR"),
		YtDlpBin:            envString("YTDLP_BIN", "yt-dlp"),
		FfmpegBin:           envString("FFMPEG_BIN", "ffmpeg"),
		GoogleClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:  os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:   os.Getenv("GOOGLE_REDIRECT_URL"),
		ConcurrentDownloads: envInt("CONCURRENT_DOWNLOADS", 2),
		DefaultMode:         envString("DEFAULT_MODE", "video"),
		AuthPassword:        os.Getenv("AUTH_PASSWORD"),
		AuthAllowedEmails:   os.Getenv("AUTH_ALLOWED_EMAILS"),
		CookiesFile:         os.Getenv("YTDLP_COOKIES"),
		CookiesFromBrowser:  os.Getenv("YTDLP_COOKIES_FROM_BROWSER"),
		WebDir:              os.Getenv("WEB_DIR"),
	}
}

// MediaDir is where the downloaded files are stored. It is the library
// directory when one is configured, so the media can live on its own volume,
// and the data directory otherwise.
func (c Config) MediaDir() string {
	if c.LibraryDir != "" {
		return c.LibraryDir
	}
	return c.DataDir
}

// AllowedEmails are the accounts that may sign in with Google.
func (c Config) AllowedEmails() []string {
	var out []string
	for _, part := range strings.Split(c.AuthAllowedEmails, ",") {
		email := strings.ToLower(strings.TrimSpace(part))
		if email != "" {
			out = append(out, email)
		}
	}
	return out
}

// AllowsEmail reports whether the address may sign in with Google.
func (c Config) AllowsEmail(email string) bool {
	if email == "" {
		return false
	}
	email = strings.ToLower(strings.TrimSpace(email))
	for _, allowed := range c.AllowedEmails() {
		if allowed == email {
			return true
		}
	}
	return false
}

// GoogleEnabled reports whether Google sign-in is configured. Both the client
// id and the client secret must be present.
func (c Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
