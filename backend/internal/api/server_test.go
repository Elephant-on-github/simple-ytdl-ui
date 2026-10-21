package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/auth"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/config"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/download"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/fakeytdlp"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/youtube"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/ytdlp"
)

type harness struct {
	server *Server
	store  *store.Store
	dir    string
}

func newHarness(t *testing.T, mode string) *harness {
	t.Helper()

	dir := t.TempDir()
	bin, err := fakeytdlp.Write(t.TempDir(), mode)
	if err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}

	cfg := config.Config{
		Port:                8080,
		DataDir:             dir,
		YtDlpBin:            bin,
		FfmpegBin:           "ffmpeg",
		ConcurrentDownloads: 2,
		DefaultMode:         "video",
		WebDir:              t.TempDir(),
	}

	st := store.New(dir)
	mgr := download.NewManager(ytdlp.NewRunner(bin), st, dir, cfg.ConcurrentDownloads)
	t.Cleanup(mgr.Close)

	srv := New(cfg, st, mgr, nil)
	return &harness{server: srv, store: st, dir: dir}
}

func (h *harness) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, reader)
	rec := httptest.NewRecorder()
	h.server.Handler().ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not JSON (%q): %v", rec.Body.String(), err)
	}
	return doc
}

// newGuardedHarness returns a harness whose server needs a password.
func newGuardedHarness(t *testing.T, password string) *harness {
	t.Helper()
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.AuthPassword = password
	h.server.auth = auth.New(password)
	return h
}

// doWithCookie is like do, but sends the given cookies.
func (h *harness) doWithCookie(t *testing.T, method, target string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, reader)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.server.Handler().ServeHTTP(rec, req)
	return rec
}

func TestWithoutAPasswordTheAppIsOpen(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/library", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	doc := decode(t, h.do(t, http.MethodGet, "/api/auth/session", nil))
	if doc["authRequired"] != false {
		t.Errorf("authRequired = %v, want false", doc["authRequired"])
	}
	if doc["authenticated"] != true {
		t.Errorf("authenticated = %v, want true", doc["authenticated"])
	}
}

func TestAPasswordGuardsTheApp(t *testing.T) {
	h := newGuardedHarness(t, "hunter2")

	// Without a session everything but the login is closed.
	rec := h.do(t, http.MethodGet, "/api/library", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	// The wrong password is refused.
	rec = h.do(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	// The right password opens a session.
	rec = h.do(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "hunter2"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("the login did not set a session cookie")
	}
	if !cookie.HttpOnly {
		t.Error("the session cookie is not HttpOnly")
	}

	// With the cookie the app opens up.
	rec = h.doWithCookie(t, http.MethodGet, "/api/library", nil, []*http.Cookie{cookie})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	doc := decode(t, h.doWithCookie(t, http.MethodGet, "/api/auth/session", nil, []*http.Cookie{cookie}))
	if doc["authenticated"] != true {
		t.Errorf("authenticated = %v, want true", doc["authenticated"])
	}
	if doc["authRequired"] != true {
		t.Errorf("authRequired = %v, want true", doc["authRequired"])
	}

	// After the logout the cookie is worthless.
	rec = h.doWithCookie(t, http.MethodPost, "/api/auth/logout", nil, []*http.Cookie{cookie})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	rec = h.doWithCookie(t, http.MethodGet, "/api/library", nil, []*http.Cookie{cookie})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d after the logout, want 401", rec.Code)
	}
}

func TestTheHealthCheckWorksWithoutASession(t *testing.T) {
	// The container health check has no cookie.
	h := newGuardedHarness(t, "hunter2")

	rec := h.do(t, http.MethodGet, "/api/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAnAllowedGoogleAccountSignsInToTheApp(t *testing.T) {
	h := newGuardedHarness(t, "hunter2")
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"
	h.server.cfg.AuthAllowedEmails = "demo@example.com, other@example.com"

	// The token and the user info endpoint both answer with the same server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/userinfo" {
			w.Write([]byte(`{"email":"demo@example.com"}`))
			return
		}
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL + "/userinfo",
	})

	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	state := stateFromURL(decode(t, rec)["url"].(string))

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}

	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			session = c
		}
	}
	if session == nil {
		t.Fatal("the google sign-in did not open a session")
	}

	// The session gets into the app without the password.
	rec = h.doWithCookie(t, http.MethodGet, "/api/library", nil, []*http.Cookie{session})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAGoogleAccountThatIsNotAllowedGetsNoSession(t *testing.T) {
	h := newGuardedHarness(t, "hunter2")
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"
	h.server.cfg.AuthAllowedEmails = "demo@example.com"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/userinfo" {
			w.Write([]byte(`{"email":"stranger@example.com"}`))
			return
		}
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL + "/userinfo",
	})

	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	state := stateFromURL(decode(t, rec)["url"].(string))

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			t.Error("a google account that is not on the list got a session")
		}
	}
}

func TestHealth(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	doc := decode(t, rec)
	if doc["status"] != "ok" {
		t.Errorf("status = %v, want ok", doc["status"])
	}
	if _, ok := doc["version"]; !ok {
		t.Error("the response has no version")
	}
}

func TestConfig(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/config", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	doc := decode(t, rec)
	if doc["googleEnabled"] != false {
		t.Errorf("googleEnabled = %v, want false", doc["googleEnabled"])
	}
	if doc["defaultMode"] != "video" {
		t.Errorf("defaultMode = %v, want video", doc["defaultMode"])
	}
	if doc["concurrent"] != float64(2) {
		t.Errorf("concurrent = %v, want 2", doc["concurrent"])
	}
}

func TestDownloadsRejectsBadURL(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{"url": "https://example.com/song.mp3"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if _, ok := decode(t, rec)["error"]; !ok {
		t.Error("the response has no error message")
	}
}

func TestDownloadsRejectsUnknownMode(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url":  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"mode": "bogus",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDownloadsRejectsBlankMode(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url":  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"mode": "",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestDownloadsWithoutModeDefaultsToVideo(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	doc := decode(t, rec)
	jobs, ok := doc["jobs"].([]any)
	if !ok || len(jobs) != 1 {
		t.Fatalf("jobs = %v, want one job", doc["jobs"])
	}
	job := jobs[0].(map[string]any)
	if job["mode"] != "video" {
		t.Errorf("mode = %v, want video", job["mode"])
	}
	h.server.manager.WaitIdle()
}

func TestDownloadsReturnsJobsAndTheyShowUpInTheJobList(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url":  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"mode": "music",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	doc := decode(t, rec)
	jobs := doc["jobs"].([]any)
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	job := jobs[0].(map[string]any)
	for _, key := range []string{"id", "title", "url", "videoId", "mode", "status", "progress", "createdAt"} {
		if _, ok := job[key]; !ok {
			t.Errorf("the job has no %q field", key)
		}
	}

	rec = h.do(t, http.MethodGet, "/api/jobs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	items, ok := decode(t, rec)["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want one job", decode(t, rec)["items"])
	}
}

func TestCancelJob(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	jobs := decode(t, rec)["jobs"].([]any)
	id := jobs[0].(map[string]any)["id"].(string)

	rec = h.do(t, http.MethodPost, "/api/jobs/"+id+"/cancel", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if decode(t, rec)["ok"] != true {
		t.Error("the response does not report ok")
	}
	h.server.manager.WaitIdle()
}

func TestCancelUnknownJob(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodPost, "/api/jobs/nope/cancel", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestLibraryLifecycle(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	// Download something so the library has an item.
	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("download failed: %s", rec.Body.String())
	}
	h.server.manager.WaitIdle()

	rec = h.do(t, http.MethodGet, "/api/library", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	items, ok := decode(t, rec)["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want one item", decode(t, rec)["items"])
	}
	item := items[0].(map[string]any)
	for _, key := range []string{"id", "videoId", "title", "artist", "duration", "file", "mime", "size", "mode", "sourceUrl", "addedAt"} {
		if _, ok := item[key]; !ok {
			t.Errorf("the item has no %q field", key)
		}
	}
	id := item["id"].(string)
	file := item["file"].(string)

	// The stream endpoint honours Range requests.
	req := httptest.NewRequest(http.MethodGet, "/api/library/"+id+"/stream", nil)
	req.Header.Set("Range", "bytes=0-3")
	rec = httptest.NewRecorder()
	h.server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206 (%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Range"); got == "" {
		t.Error("no Content-Range header")
	}
	if got := rec.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Errorf("Accept-Ranges = %q, want bytes", got)
	}
	if rec.Body.Len() != 4 {
		t.Errorf("body length = %d, want 4", rec.Body.Len())
	}

	// Without a Range header the whole file is returned.
	rec = h.do(t, http.MethodGet, "/api/library/"+id+"/stream", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Deleting removes the file from disk and the record.
	rec = h.do(t, http.MethodDelete, "/api/library/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if decode(t, rec)["ok"] != true {
		t.Error("the response does not report ok")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the file %q is still on disk (err=%v)", file, err)
	}

	rec = h.do(t, http.MethodGet, "/api/library", nil)
	items = decode(t, rec)["items"].([]any)
	if len(items) != 0 {
		t.Errorf("the library still has %d items", len(items))
	}

	// The stream endpoint reports a missing item.
	rec = h.do(t, http.MethodGet, "/api/library/"+id+"/stream", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteUnknownLibraryItem(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodDelete, "/api/library/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPlaylistsWithoutSessionAreUnauthorized(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/playlists", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	rec = h.do(t, http.MethodGet, "/api/playlists/PL1/items", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAuthGoogleWithoutCredentials(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestAuthGoogleWithCredentials(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	raw, _ := decode(t, rec)["url"].(string)
	if !strings.Contains(raw, "access_type=offline") {
		t.Errorf("url = %q, want it to ask for an offline token", raw)
	}
	if !strings.Contains(raw, "prompt=consent") {
		t.Errorf("url = %q, want it to ask for consent", raw)
	}
}

func TestSessionAndLogout(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/auth/session", nil)
	doc := decode(t, rec)
	if doc["signedIn"] != false {
		t.Errorf("signedIn = %v, want false", doc["signedIn"])
	}

	if err := h.store.SaveGoogleToken("refresh-token", "user@example.test"); err != nil {
		t.Fatalf("SaveGoogleToken: %v", err)
	}

	rec = h.do(t, http.MethodGet, "/api/auth/session", nil)
	doc = decode(t, rec)
	if doc["signedIn"] != true {
		t.Errorf("signedIn = %v, want true", doc["signedIn"])
	}
	if doc["email"] != "user@example.test" {
		t.Errorf("email = %v", doc["email"])
	}

	rec = h.do(t, http.MethodPost, "/api/auth/google/logout", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if decode(t, rec)["ok"] != true {
		t.Error("the response does not report ok")
	}

	rec = h.do(t, http.MethodGet, "/api/auth/session", nil)
	if decode(t, rec)["signedIn"] != false {
		t.Error("the session is still signed in after the logout")
	}
}

func TestOAuthCallbackStoresTheTokenAndRedirects(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL,
	})

	// The state is handed out by the auth endpoint.
	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	authURL := decode(t, rec)["url"].(string)
	state := authURL[strings.Index(authURL, "state=")+len("state="):]
	if idx := strings.Index(state, "&"); idx >= 0 {
		state = state[:idx]
	}
	if state == "" {
		t.Fatal("no state in the auth url")
	}

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/playlists?signedIn=1" {
		t.Errorf("Location = %q, want /playlists?signedIn=1", got)
	}

	refresh, _, ok := h.store.GoogleToken()
	if !ok {
		t.Fatal("the refresh token was not stored")
	}
	if refresh != "refresh-token" {
		t.Errorf("refresh token = %q", refresh)
	}
}

func TestOAuthCallbackReturnsToThePageTheSignInStartedFrom(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL,
	})

	rec := h.do(t, http.MethodGet, "/api/auth/google?redirect=%2Flibrary", nil)
	authURL := decode(t, rec)["url"].(string)
	state := stateFromURL(authURL)

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/library?signedIn=1" {
		t.Errorf("Location = %q, want /library?signedIn=1", got)
	}
}

func TestOAuthStateCanOnlyBeUsedOnce(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL,
	})

	rec := h.do(t, http.MethodGet, "/api/auth/google", nil)
	state := stateFromURL(decode(t, rec)["url"].(string))

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	if got := rec.Header().Get("Location"); got != "/playlists?signedIn=1" {
		t.Fatalf("Location = %q, want /playlists?signedIn=1", got)
	}

	// The same state a second time must not sign anybody in.
	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	if got := rec.Header().Get("Location"); !strings.Contains(got, "error=invalid_state") {
		t.Errorf("Location = %q, want an error about the state", got)
	}
}

func TestOAuthRedirectStaysOnThisSite(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	h.server.SetOAuth(&youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL,
		UserInfoURL:  srv.URL,
	})

	rec := h.do(t, http.MethodGet, "/api/auth/google?redirect=https%3A%2F%2Fevil.example%2Fsteal", nil)
	state := stateFromURL(decode(t, rec)["url"].(string))

	rec = h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state="+state, nil)
	got := rec.Header().Get("Location")
	if !strings.HasPrefix(got, "/") || strings.HasPrefix(got, "//") {
		t.Errorf("Location = %q, want a path on this site", got)
	}
}

// stateFromURL pulls the state value out of an authorization URL.
func stateFromURL(authURL string) string {
	idx := strings.Index(authURL, "state=")
	if idx < 0 {
		return ""
	}
	state := authURL[idx+len("state="):]
	if end := strings.Index(state, "&"); end >= 0 {
		state = state[:end]
	}
	return state
}

func TestOAuthCallbackRejectsAWrongState(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	rec := h.do(t, http.MethodGet, "/api/auth/google/callback?code=the-code&state=wrong", nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if _, _, ok := h.store.GoogleToken(); ok {
		t.Error("a token was stored even though the state did not match")
	}
}

func TestPlaylistsWithASession(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer token.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"id":"PL1","snippet":{"title":"First Playlist","thumbnails":{"default":{"url":"http://img/1.jpg"}}},"contentDetails":{"itemCount":42}}]}`))
	}))
	defer api.Close()

	client := youtube.NewClientWithToken(context.Background(), "refresh-token", &youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     token.URL,
	}).WithBaseURL(api.URL)
	h.server.yt = client

	rec := h.do(t, http.MethodGet, "/api/playlists", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	items, ok := decode(t, rec)["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v", decode(t, rec)["items"])
	}
	playlist := items[0].(map[string]any)
	if playlist["id"] != "PL1" {
		t.Errorf("id = %v", playlist["id"])
	}
	if playlist["title"] != "First Playlist" {
		t.Errorf("title = %v", playlist["title"])
	}
	if playlist["itemCount"] != float64(42) {
		t.Errorf("itemCount = %v", playlist["itemCount"])
	}
}

func TestPlaylistItems(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.GoogleClientID = "client-id"
	h.server.cfg.GoogleClientSecret = "client-secret"

	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer token.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"nextPageToken":"page-2","items":[{"id":"vid1","snippet":{"title":"First Video","channelTitle":"Fake Channel","thumbnails":{"default":{"url":"http://img/v1.jpg"}}},"contentDetails":{"duration":"PT3M20S"}}]}`))
	}))
	defer api.Close()

	client := youtube.NewClientWithToken(context.Background(), "refresh-token", &youtube.OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     token.URL,
	}).WithBaseURL(api.URL)
	h.server.yt = client

	rec := h.do(t, http.MethodGet, "/api/playlists/PL1/items", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	doc := decode(t, rec)
	items, ok := doc["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v", doc["items"])
	}
	item := items[0].(map[string]any)
	if item["id"] != "vid1" {
		t.Errorf("id = %v", item["id"])
	}
	if item["duration"] != float64(200) {
		t.Errorf("duration = %v, want 200", item["duration"])
	}
	if doc["nextPageToken"] != "page-2" {
		t.Errorf("nextPageToken = %v, want page-2", doc["nextPageToken"])
	}
}

func TestStaticFallsBackToIndexHTML(t *testing.T) {
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><title>app</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	h := newHarness(t, fakeytdlp.ModeOK)
	h.server.cfg.WebDir = webDir

	rec := h.do(t, http.MethodGet, "/library", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>app</title>") {
		t.Errorf("body = %q, want the index page", rec.Body.String())
	}

	// An existing file is served as it is.
	if err := os.WriteFile(filepath.Join(webDir, "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	rec = h.do(t, http.MethodGet, "/app.js", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log(1)" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestAPIRoutesAreNotShadowedByTheStaticHandler(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)

	rec := h.do(t, http.MethodGet, "/api/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("content type = %q, want JSON", ct)
	}
}

func TestFailedDownloadIsVisibleInTheJob(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeFail)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	h.server.manager.WaitIdle()

	rec = h.do(t, http.MethodGet, "/api/jobs", nil)
	items := decode(t, rec)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	job := items[0].(map[string]any)
	if job["status"] != "failed" {
		t.Errorf("status = %v, want failed", job["status"])
	}
	message, _ := job["error"].(string)
	if !strings.Contains(message, "Video unavailable") {
		t.Errorf("error = %q, want the yt-dlp stderr", message)
	}
}

func TestDownloadsAreDeduplicated(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeOK)
	body := map[string]any{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}

	rec := h.do(t, http.MethodPost, "/api/downloads", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	h.server.manager.WaitIdle()

	rec = h.do(t, http.MethodPost, "/api/downloads", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	h.server.manager.WaitIdle()

	rec = h.do(t, http.MethodGet, "/api/library", nil)
	items := decode(t, rec)["items"].([]any)
	if len(items) != 1 {
		t.Errorf("the library has %d items, want 1", len(items))
	}
}

func TestHealthEndpointDoesNotBlockWhileDownloadsRun(t *testing.T) {
	h := newHarness(t, fakeytdlp.ModeSleep)

	rec := h.do(t, http.MethodPost, "/api/downloads", map[string]any{
		"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	id := decode(t, rec)["jobs"].([]any)[0].(map[string]any)["id"].(string)

	done := make(chan int, 1)
	go func() {
		r := h.do(t, http.MethodGet, "/api/health", nil)
		done <- r.Code
	}()
	select {
	case code := <-done:
		if code != http.StatusOK {
			t.Errorf("status = %d, want 200", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the health endpoint blocked while a download was running")
	}

	rec = h.do(t, http.MethodPost, "/api/jobs/"+id+"/cancel", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("cancel status = %d, want 200", rec.Code)
	}
	h.server.manager.WaitIdle()
}
