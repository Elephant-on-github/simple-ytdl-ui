// Package api serves the JSON API and the single page application.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/auth"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/config"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/download"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/youtube"
	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/ytdlp"
)

// Version is stamped into the build.
var Version = "dev"

// stateTTL is how long an OAuth state value stays valid.
const stateTTL = 10 * time.Minute

var errNotSignedIn = errors.New("not signed in")

// Server implements the HTTP API.
type Server struct {
	cfg     config.Config
	store   *store.Store
	manager *download.Manager
	yt      *youtube.Client
	auth    *auth.Manager

	mu     sync.Mutex
	oauth  *youtube.OAuth
	states map[string]authState
	cached *youtube.Client

	mux *http.ServeMux
}

// authState is a one time OAuth state value. It remembers where the browser
// was sent from, so the sign-in ends up back where it started.
type authState struct {
	redirect string
	expires  time.Time
}

// New creates the API server.
func New(cfg config.Config, st *store.Store, mgr *download.Manager, yt *youtube.Client) *Server {
	s := &Server{
		cfg:     cfg,
		store:   st,
		manager: mgr,
		yt:      yt,
		auth:    auth.New(cfg.AuthPassword),
		states:  map[string]authState{},
		oauth: &youtube.OAuth{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
		},
		mux: http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/config", s.handleConfig)
	s.mux.HandleFunc("GET /api/auth/session", s.handleSession)
	s.mux.HandleFunc("POST /api/auth/google/logout", s.handleGoogleLogout)
	s.mux.HandleFunc("GET /api/playlists", s.handlePlaylists)
	s.mux.HandleFunc("POST /api/session/cookies", s.handleSaveCookies)
	s.mux.HandleFunc("DELETE /api/session/cookies", s.handleClearCookies)
	s.mux.HandleFunc("GET /api/playlists/{id}/items", s.handlePlaylistItems)
	s.mux.HandleFunc("POST /api/downloads", s.handleDownloads)
	s.mux.HandleFunc("GET /api/jobs", s.handleJobs)
	s.mux.HandleFunc("POST /api/jobs/{id}/cancel", s.handleCancel)
	s.mux.HandleFunc("POST /api/jobs/{id}/pause", s.handlePause)
	s.mux.HandleFunc("POST /api/jobs/{id}/resume", s.handleResume)
	s.mux.HandleFunc("DELETE /api/jobs/{id}", s.handleRemoveJob)
	s.mux.HandleFunc("POST /api/jobs/pause-all", s.handlePauseAll)
	s.mux.HandleFunc("POST /api/jobs/resume-all", s.handleResumeAll)
	s.mux.HandleFunc("POST /api/jobs/cancel-all", s.handleCancelAll)
	s.mux.HandleFunc("DELETE /api/jobs/finished", s.handleClearFinished)
	s.mux.HandleFunc("GET /api/library", s.handleLibrary)
	s.mux.HandleFunc("DELETE /api/library/{id}", s.handleDeleteItem)
	s.mux.HandleFunc("GET /api/library/{id}/stream", s.handleStream)
	s.mux.HandleFunc("GET /api/library/{id}/cover", s.handleCover)
	s.mux.HandleFunc("GET /api/schedules", s.handleSchedules)
	s.mux.HandleFunc("POST /api/schedules", s.handleCreateSchedule)
	s.mux.HandleFunc("POST /api/schedules/{id}/run", s.handleRunSchedule)
	s.mux.HandleFunc("DELETE /api/schedules/{id}", s.handleRemoveSchedule)

	return s
}

// SetOAuth replaces the OAuth configuration. It exists so the tests can point
// the callback at a fake token endpoint.
func (s *Server) SetOAuth(o *youtube.OAuth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.oauth = o
	s.cached = nil
}

// Handler returns the HTTP handler of the whole application: the API under
// /api and the single page application everywhere else.
func (s *Server) Handler() http.Handler {
	root := http.NewServeMux()

	// The routes that work without a session: the health check the container
	// uses, the login itself, and the Google sign-in — the callback is what
	// establishes the session, so it cannot ask for one.
	root.Handle("GET /api/health", http.HandlerFunc(s.handleHealth))
	root.Handle("GET /api/auth/session", http.HandlerFunc(s.handleSession))
	root.Handle("POST /api/auth/login", http.HandlerFunc(s.handleLogin))
	root.Handle("POST /api/auth/logout", http.HandlerFunc(s.handleLogout))
	root.Handle("GET /api/auth/google", http.HandlerFunc(s.handleAuthGoogle))
	root.Handle("GET /api/auth/google/callback", http.HandlerFunc(s.handleAuthCallback))

	// Everything else needs a session when a password is set.
	root.Handle("/api/", s.requireAuth(s.mux))
	root.Handle("/", StaticHandler(s.cfg.WebDir))
	return root
}

// sessionCookie is the name of the cookie that holds the session token.
const sessionCookie = "sid"

// requireAuth guards the handlers behind a session. Without a password every
// request goes straight through.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Required() {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || !s.auth.Valid(cookie.Value) {
			writeError(w, http.StatusUnauthorized, "sign in to use this server")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// setSessionCookie opens a session for the browser. Over HTTPS the cookie is
// marked secure, so it never travels in the clear.
func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   publicScheme(r) == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.SessionTTL.Seconds()),
	})
}

// handleLogin checks the password and opens a session.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Required() {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "required": false})
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "the request body must be JSON")
		return
	}

	token, ok := s.auth.Login(body.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "that password is not right")
		return
	}

	s.setSessionCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "required": true})
}

// handleLogout closes the session of the browser.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		s.auth.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": Version,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	mode := s.cfg.DefaultMode
	if mode == "" {
		mode = string(ytdlp.ModeVideo)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"googleEnabled": s.cfg.GoogleEnabled(),
		"defaultMode":   mode,
		"concurrent":    s.cfg.ConcurrentDownloads,
	})
}

// publicScheme is the scheme the browser used to reach this server. Behind a
// proxy the connection to us is plain HTTP, so the headers are what count.
func publicScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(strings.TrimSpace(strings.Split(proto, ",")[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return "https"
	}
	return "http"
}

// publicHost is the host the browser used to reach this server.
func publicHost(r *http.Request) string {
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		return strings.TrimSpace(strings.Split(host, ",")[0])
	}
	return r.Host
}

// redirectURL is the address Google sends the browser back to. The configured
// one wins; without it the address is taken from the request, so the app works
// behind a proxy or on a cloud platform without any configuration.
func (s *Server) redirectURL(r *http.Request) string {
	if s.cfg.GoogleRedirectURL != "" {
		return s.cfg.GoogleRedirectURL
	}
	return publicScheme(r) + "://" + publicHost(r) + "/api/auth/google/callback"
}

// oauthFor returns the OAuth configuration for this request, with the redirect
// URI the request came in through.
func (s *Server) oauthFor(r *http.Request) *youtube.OAuth {
	o := s.oauthConfig()
	if o.RedirectURL == "" {
		o = o.WithRedirect(s.redirectURL(r))
	}
	return o
}

// handleAuthGoogle hands out the URL of Google's consent screen. The optional
// "redirect" parameter says where the browser should end up afterwards, so the
// sign-in can be started and finished inside the app.
func (s *Server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.GoogleEnabled() {
		writeError(w, http.StatusServiceUnavailable, "google sign-in is not configured on this server")
		return
	}
	redirect := safeRedirect(r.URL.Query().Get("redirect"))
	state := s.newState(redirect)
	writeJSON(w, http.StatusOK, map[string]any{
		"url": s.oauthFor(r).AuthURL(state),
	})
}

// handleAuthCallback finishes the sign-in and sends the browser back to the
// page it came from.
func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirect, ok := s.consumeState(query.Get("state"))
	if !ok {
		http.Redirect(w, r, withQuery(defaultRedirect, "error", "invalid_state"), http.StatusFound)
		return
	}
	if query.Get("code") == "" {
		http.Redirect(w, r, withQuery(redirect, "error", "missing_code"), http.StatusFound)
		return
	}

	token, err := s.oauthFor(r).ExchangeToken(r.Context(), query.Get("code"))
	if err != nil {
		log.Printf("google oauth exchange failed: %v", err)
		http.Redirect(w, r, withQuery(redirect, "error", "exchange_failed"), http.StatusFound)
		return
	}
	if token.RefreshToken == "" {
		http.Redirect(w, r, withQuery(redirect, "error", "missing_refresh_token"), http.StatusFound)
		return
	}

	email := s.oauthFor(r).Email(r.Context(), token)
	if err := s.store.SaveGoogleToken(token.RefreshToken, email); err != nil {
		log.Printf("could not store the google token: %v", err)
		http.Redirect(w, r, withQuery(redirect, "error", "storage_failed"), http.StatusFound)
		return
	}
	s.resetClient()

	// The account is on the allowed list: it may sign in to the app itself.
	if s.cfg.AllowsEmail(email) {
		s.setSessionCookie(w, r, s.auth.OpenSession())
	}

	http.Redirect(w, r, withQuery(redirect, "signedIn", "1"), http.StatusFound)
}

// handleSession reports who is signed in: the app itself and Google.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	_, email, ok := s.store.GoogleToken()
	authenticated := true
	if s.auth.Required() {
		cookie, err := r.Cookie(sessionCookie)
		authenticated = err == nil && s.auth.Valid(cookie.Value)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": authenticated,
		"authRequired":  s.auth.Required(),
		// Can a Google account sign in to the app itself?
		"googleCanAuth": s.cfg.GoogleEnabled() && len(s.cfg.AllowedEmails()) > 0,
		// Can the playlists be listed at all, and how?
		"playlistsAvailable": s.manager.HasCookies() || ok,
		"playlistSource":     s.playlistSource(ok),
		"signedIn":           ok,
		"email":              email,
	})
}

// playlistSource says where the playlists come from: the YouTube session in
// the cookies, or the YouTube Data API.
func (s *Server) playlistSource(googleSignedIn bool) string {
	if s.manager.HasCookies() {
		return "cookies"
	}
	if googleSignedIn {
		return "google"
	}
	return ""
}

// handleGoogleLogout forgets the stored Google token.
func (s *Server) handleGoogleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearGoogleToken(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.forgetClient()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSaveCookies takes a YouTube session that was pasted into the app and
// checks it right away: a session that cannot list the playlists is worthless,
// and the user should know before they leave.
//
// The body is the contents of a cookies.txt.
func (s *Server) handleSaveCookies(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the cookies")
		return
	}
	if !looksLikeCookies(body) {
		writeError(w, http.StatusBadRequest, "that does not look like a cookies.txt (it should start with the Netscape cookie header)")
		return
	}

	if err := s.store.SaveCookies(body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.manager.SetCookies(s.store.CookiesPath())

	// Check it right away: a session that cannot list the playlists is
	// worthless, and the user should know now. The check gets a deadline of
	// its own — yt-dlp retries three times, which is longer than a proxy
	// waits for an answer.
	ctx, cancel := context.WithTimeout(r.Context(), sessionCheckTimeout)
	defer cancel()

	lists, err := s.manager.Playlists(ctx)
	if err != nil {
		message := err.Error()
		if ctx.Err() != nil {
			message = "the session was stored, but the check did not finish in time — try again in a moment"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": message,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"playlists": len(lists),
	})
}

// handleClearCookies forgets the pasted YouTube session.
func (s *Server) handleClearCookies(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearCookies(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Fall back to the session the server was started with, if there is one.
	if s.cfg.CookiesFile != "" {
		if err := s.store.SaveCookiesFrom(s.cfg.CookiesFile); err != nil {
			log.Printf("could not use the youtube session in %s: %v", s.cfg.CookiesFile, err)
		}
	}
	if s.store.HasCookies() {
		s.manager.SetCookies(s.store.CookiesPath())
	} else {
		s.manager.SetCookies("")
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// looksLikeCookies is a cheap check that the body is a Netscape cookie file.
func looksLikeCookies(body []byte) bool {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Every cookie line has seven tab separated fields.
		return len(strings.Split(line, "\t")) >= 7
	}
	return false
}

// handlePlaylists lists the playlists of the account.
//
// A YouTube session in the cookies is enough — no Google Cloud project, no
// OAuth client, no API quota. Without one the YouTube Data API is used, which
// needs the Google sign-in.
func (s *Server) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	if s.manager.HasCookies() {
		lists, err := s.manager.Playlists(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		items := make([]map[string]any, 0, len(lists))
		for _, p := range lists {
			items = append(items, map[string]any{
				"id":        p.ID,
				"title":     p.Title,
				"thumb":     p.Thumb,
				"itemCount": 0,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
		return
	}

	client, err := s.client(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sign in with Google to see your playlists")
		return
	}

	playlists, err := client.Playlists(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if playlists == nil {
		playlists = []youtube.Playlist{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": playlists})
}

func (s *Server) handlePlaylistItems(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// When only a YouTube session (cookies) is available, probe the playlist
	// directly with yt-dlp — no Google Cloud project or OAuth needed.
	if s.manager.HasCookies() {
		items, err := s.manager.PlaylistItems(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		out := make([]map[string]any, 0, len(items))
		for _, it := range items {
			out = append(out, map[string]any{
				"id":          it.ID,
				"title":       it.Title,
				"artist":      it.Uploader,
				"duration":    it.Duration,
				"thumbnail":   it.Thumbnail,
				"webpage_url": it.WebpageURL,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
		return
	}

	client, err := s.client(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sign in with Google to see your playlists")
		return
	}

	items, next, err := client.PlaylistItems(r.Context(), id, r.URL.Query().Get("pageToken"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []youtube.PlaylistItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":         items,
		"nextPageToken": next,
	})
}

// downloadRequest is the body of POST /api/downloads.
type downloadRequest struct {
	URL      string  `json:"url"`
	Mode     *string `json:"mode"`
	Playlist bool    `json:"playlist"`
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request) {
	var body downloadRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "the request body must be JSON")
		return
	}

	mode := s.cfg.DefaultMode
	if mode == "" {
		mode = string(ytdlp.ModeVideo)
	}
	if body.Mode != nil {
		if strings.TrimSpace(*body.Mode) == "" {
			writeError(w, http.StatusBadRequest, "mode must be either video or music")
			return
		}
		mode = *body.Mode
	}
	parsed, err := ytdlp.ParseMode(mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, _, _, err := ytdlp.ClassifyURL(body.URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	jobs, err := s.manager.Enqueue(r.Context(), body.URL, parsed, body.Playlist, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	jobs := s.manager.Jobs()
	if jobs == nil {
		jobs = []store.Job{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": jobs})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Cancel(id); err != nil {
		writeError(w, http.StatusNotFound, "unknown job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Pause(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Resume(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRemoveJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.Remove(id); err != nil {
		writeError(w, http.StatusNotFound, "unknown job")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePauseAll(w http.ResponseWriter, r *http.Request) {
	s.manager.PauseAll()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleResumeAll(w http.ResponseWriter, r *http.Request) {
	s.manager.ResumeAll()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCancelAll(w http.ResponseWriter, r *http.Request) {
	s.manager.CancelAll()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleClearFinished(w http.ResponseWriter, r *http.Request) {
	s.manager.ClearFinished()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSchedules lists all scheduled downloads.
func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	schedules := s.manager.Schedules()
	writeJSON(w, http.StatusOK, map[string]any{"schedules": schedules})
}

// handleCreateSchedule creates a new scheduled download.
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL       string `json:"url"`
		Mode      string `json:"mode"`
		Folder    string `json:"folder"`
		Frequency string `json:"frequency"`
		Title     string `json:"title"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "the request body must be JSON")
		return
	}

	// Validate required fields.
	if strings.TrimSpace(body.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if _, _, _, err := ytdlp.ClassifyURL(body.URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mode := body.Mode
	if mode == "" {
		mode = string(ytdlp.ModeVideo)
	}
	if _, err := ytdlp.ParseMode(mode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	freq := body.Frequency
	if freq == "" {
		freq = "daily"
	}
	if freq != "daily" && freq != "weekly" && freq != "monthly" {
		writeError(w, http.StatusBadRequest, "frequency must be daily, weekly, or monthly")
		return
	}

	sched := store.Schedule{
		URL:       strings.TrimSpace(body.URL),
		Mode:      mode,
		Folder:    strings.TrimSpace(body.Folder),
		Frequency: freq,
		Title:     strings.TrimSpace(body.Title),
		NextRun:   time.Now(), // schedule for now
	}

	saved, err := s.manager.SaveSchedule(sched)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "schedule": saved})
}

// handleRemoveSchedule removes a scheduled download.
func (s *Server) handleRemoveSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.RemoveSchedule(id); err != nil {
		writeError(w, http.StatusNotFound, "unknown schedule")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleRunSchedule triggers a scheduled download immediately.
func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.manager.TriggerSchedule(id); err != nil {
		if errors.Is(err, download.ErrUnknownSchedule) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	items := s.store.Items()
	if items == nil {
		items = []store.Item{}
	}
	// The file and cover paths are absolute server paths. The client never
	// needs them: it streams and fetches the cover by id.
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"id":        it.ID,
			"videoId":   it.VideoID,
			"title":     it.Title,
			"artist":    it.Artist,
			"duration":  it.Duration,
			"mime":      it.Mime,
			"size":      it.Size,
			"mode":      it.Mode,
			"sourceUrl": it.SourceURL,
			"playlist":  it.Playlist,
			"hasCover":  it.Cover != "",
			"addedAt":   it.AddedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.store.ItemByID(id); !ok {
		writeError(w, http.StatusNotFound, "unknown library item")
		return
	}
	if err := s.store.RemoveItem(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	item, ok := s.store.ItemByID(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown library item")
		return
	}
	if item.File == "" {
		writeError(w, http.StatusNotFound, "this item has no file")
		return
	}

	file, err := os.Open(item.File)
	if err != nil {
		writeError(w, http.StatusNotFound, "the file of this item is gone")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item.Mime != "" {
		w.Header().Set("Content-Type", item.Mime)
	}
	// ServeContent handles Range requests and answers 206 with a
	// Content-Range header.
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (s *Server) handleCover(w http.ResponseWriter, r *http.Request) {
	item, ok := s.store.ItemByID(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown library item")
		return
	}
	if item.Cover == "" {
		writeError(w, http.StatusNotFound, "this item has no cover")
		return
	}

	file, err := os.Open(item.Cover)
	if err != nil {
		writeError(w, http.StatusNotFound, "the cover of this item is gone")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// client returns a YouTube client for the signed in user.
func (s *Server) client(ctx context.Context) (*youtube.Client, error) {
	s.mu.Lock()
	if s.yt != nil {
		client := s.yt
		s.mu.Unlock()
		return client, nil
	}
	if s.cached != nil {
		client := s.cached
		s.mu.Unlock()
		return client, nil
	}
	oauth := s.oauth
	s.mu.Unlock()

	if !s.cfg.GoogleEnabled() {
		return nil, errNotSignedIn
	}
	client, err := youtube.NewClient(ctx, s.store, oauth)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cached = client
	s.mu.Unlock()
	return client, nil
}

// resetClient drops the cached YouTube client.
func (s *Server) resetClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
}

// oauthConfig returns the OAuth configuration of this server.
func (s *Server) oauthConfig() *youtube.OAuth {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.oauth
}

// forgetClient drops the cached client and the injected one. It is used when
// the user signs out.
func (s *Server) forgetClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
	s.yt = nil
}

// defaultRedirect is where the browser lands when nothing else is known.
const defaultRedirect = "/playlists"

// sessionCheckTimeout is how long the check of a pasted session may take.
// yt-dlp retries three times, which is longer than a proxy waits.
const sessionCheckTimeout = 20 * time.Second

// newState issues a one time state value that remembers the page the sign-in
// was started from.
func (s *Server) newState(redirect string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneStatesLocked()
	token := randomToken()
	s.states[token] = authState{redirect: redirect, expires: time.Now().Add(stateTTL)}
	return token
}

// consumeState takes a state value out of the list. A state can only be used
// once, so a captured callback cannot be replayed.
func (s *Server) consumeState(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.states[token]
	if !ok {
		return "", false
	}
	delete(s.states, token)
	if time.Now().After(st.expires) {
		return "", false
	}
	return st.redirect, true
}

// pruneStatesLocked drops the expired state values. The lock must be held.
func (s *Server) pruneStatesLocked() {
	now := time.Now()
	for token, st := range s.states {
		if now.After(st.expires) {
			delete(s.states, token)
		}
	}
}

// safeRedirect keeps the browser on this site: only a path of the app itself
// is accepted, never an absolute address.
//
// A backslash is rejected too. WHATWG URL parsing reads "\" as "/" for
// special schemes, so "/\evil.example" is a different origin even though it
// starts with a slash.
func safeRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return defaultRedirect
	}
	if strings.ContainsAny(raw, "\\\r\n") {
		return defaultRedirect
	}
	if parsed, err := url.Parse(raw); err != nil || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") {
		return defaultRedirect
	}
	return raw
}

// withQuery adds a parameter to a local path.
func withQuery(path, key, value string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + url.QueryEscape(key) + "=" + url.QueryEscape(value)
}

func randomToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf[:])
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("could not write the response: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
