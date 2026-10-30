package config

import "testing"

// clearEnv unsets every variable Load() looks at so each test starts from a
// known state. t.Setenv registers a cleanup that restores the old value.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PORT", "DATA_DIR", "YTDLP_BIN", "FFMPEG_BIN",
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URL",
		"CONCURRENT_DOWNLOADS", "DEFAULT_MODE", "WEB_DIR",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	c := Load()

	if c.Port != 8080 {
		t.Errorf("Port = %d, want 8080", c.Port)
	}
	if c.DataDir != "/data" {
		t.Errorf("DataDir = %q, want /data", c.DataDir)
	}
	if c.YtDlpBin != "yt-dlp" {
		t.Errorf("YtDlpBin = %q, want yt-dlp", c.YtDlpBin)
	}
	if c.FfmpegBin != "ffmpeg" {
		t.Errorf("FfmpegBin = %q, want ffmpeg", c.FfmpegBin)
	}
	// The redirect URL is empty by default: the server takes it from the
	// request, which is what a deployment behind a proxy needs.
	if c.GoogleRedirectURL != "" {
		t.Errorf("GoogleRedirectURL = %q, want empty", c.GoogleRedirectURL)
	}
	if c.ConcurrentDownloads != 2 {
		t.Errorf("ConcurrentDownloads = %d, want 2", c.ConcurrentDownloads)
	}
	if c.DefaultMode != "video" {
		t.Errorf("DefaultMode = %q, want video", c.DefaultMode)
	}
	if c.GoogleClientID != "" || c.GoogleClientSecret != "" {
		t.Errorf("google credentials = %q/%q, want empty", c.GoogleClientID, c.GoogleClientSecret)
	}
}

func TestLoadOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "9999")
	t.Setenv("DATA_DIR", "/tmp/data")
	t.Setenv("YTDLP_BIN", "/usr/local/bin/yt-dlp")
	t.Setenv("FFMPEG_BIN", "/usr/local/bin/ffmpeg")
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://example.test/cb")
	t.Setenv("CONCURRENT_DOWNLOADS", "7")
	t.Setenv("DEFAULT_MODE", "music")

	c := Load()

	if c.Port != 9999 {
		t.Errorf("Port = %d, want 9999", c.Port)
	}
	if c.DataDir != "/tmp/data" {
		t.Errorf("DataDir = %q", c.DataDir)
	}
	if c.YtDlpBin != "/usr/local/bin/yt-dlp" {
		t.Errorf("YtDlpBin = %q", c.YtDlpBin)
	}
	if c.FfmpegBin != "/usr/local/bin/ffmpeg" {
		t.Errorf("FfmpegBin = %q", c.FfmpegBin)
	}
	if c.GoogleClientID != "cid" || c.GoogleClientSecret != "secret" {
		t.Errorf("google credentials = %q/%q", c.GoogleClientID, c.GoogleClientSecret)
	}
	if c.GoogleRedirectURL != "http://example.test/cb" {
		t.Errorf("GoogleRedirectURL = %q", c.GoogleRedirectURL)
	}
	if c.ConcurrentDownloads != 7 {
		t.Errorf("ConcurrentDownloads = %d, want 7", c.ConcurrentDownloads)
	}
	if c.DefaultMode != "music" {
		t.Errorf("DefaultMode = %q, want music", c.DefaultMode)
	}
}

func TestLoadInvalidNumbersFallBackToDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "not-a-number")
	t.Setenv("CONCURRENT_DOWNLOADS", "0")

	c := Load()

	if c.Port != 8080 {
		t.Errorf("Port = %d, want fallback 8080", c.Port)
	}
	if c.ConcurrentDownloads != 2 {
		t.Errorf("ConcurrentDownloads = %d, want fallback 2", c.ConcurrentDownloads)
	}
}

func TestLibraryDirIsItsOwnVolume(t *testing.T) {
	t.Setenv("DATA_DIR", "/data")
	t.Setenv("LIBRARY_DIR", "/library")

	cfg := Load()

	if cfg.LibraryDir != "/library" {
		t.Errorf("LibraryDir = %q, want /library", cfg.LibraryDir)
	}
	if got := cfg.MediaDir(); got != "/library" {
		t.Errorf("MediaDir = %q, want /library", got)
	}
}

func TestMediaDirFallsBackToTheDataDir(t *testing.T) {
	t.Setenv("DATA_DIR", "/data")
	t.Setenv("LIBRARY_DIR", "")

	cfg := Load()

	if got := cfg.MediaDir(); got != "/data" {
		t.Errorf("MediaDir = %q, want /data", got)
	}
}

func TestAllowedEmails(t *testing.T) {
	t.Setenv("AUTH_ALLOWED_EMAILS", " Demo@example.com , other@example.com ")

	cfg := Load()

	emails := cfg.AllowedEmails()
	if len(emails) != 2 {
		t.Fatalf("AllowedEmails() = %v, want two addresses", emails)
	}
	if !cfg.AllowsEmail("demo@example.com") {
		t.Error("AllowsEmail(demo@example.com) = false, want true")
	}
	if !cfg.AllowsEmail("DEMO@EXAMPLE.COM") {
		t.Error("the comparison must not care about the case")
	}
	if cfg.AllowsEmail("stranger@example.com") {
		t.Error("AllowsEmail(stranger@example.com) = true, want false")
	}
	if cfg.AllowsEmail("") {
		t.Error("AllowsEmail(\"\") = true, want false")
	}
}

func TestNoAllowedEmailsMeansNobody(t *testing.T) {
	t.Setenv("AUTH_ALLOWED_EMAILS", "")

	cfg := Load()

	if len(cfg.AllowedEmails()) != 0 {
		t.Errorf("AllowedEmails() = %v, want none", cfg.AllowedEmails())
	}
	if cfg.AllowsEmail("demo@example.com") {
		t.Error("AllowsEmail(demo@example.com) = true, want false")
	}
}

func TestGoogleEnabled(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		secret string
		want   bool
	}{
		{"both set", "cid", "secret", true},
		{"only id", "cid", "", false},
		{"only secret", "", "secret", false},
		{"neither", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("GOOGLE_CLIENT_ID", tt.id)
			t.Setenv("GOOGLE_CLIENT_SECRET", tt.secret)

			if got := Load().GoogleEnabled(); got != tt.want {
				t.Errorf("GoogleEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
