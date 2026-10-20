// Package api serves the JSON API and the single page application.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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

	mu       sync.Mutex
	oauth    *youtube.OAuth
	state    string
	stateEnd time.Time
	cached   *youtube.Client

	mux *http.ServeMux
}

// New creates the API server.
func New(cfg config.Config, st *store.Store, mgr *download.Manager, yt *youtube.Client) *Server {
	s := &Server{
		cfg:     cfg,
		store:   st,
		manager: mgr,
		yt:      yt,
		oauth: &youtube.OAuth{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
		},
		mux: http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/config", s.handleConfig)
	s.mux.HandleFunc("GET /api/auth/google", s.handleAuthGoogle)
	s.mux.HandleFunc("GET /api/auth/google/callback", s.handleAuthCallback)
	s.mux.HandleFunc("GET /api/auth/session", s.handleSession)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/playlists", s.handlePlaylists)
	s.mux.HandleFunc("GET /api/playlists/{id}/items", s.handlePlaylistItems)
	s.mux.HandleFunc("POST /api/downloads", s.handleDownloads)
	s.mux.HandleFunc("GET /api/jobs", s.handleJobs)
	s.mux.HandleFunc("POST /api/jobs/{id}/cancel", s.handleCancel)
	s.mux.HandleFunc("GET /api/library", s.handleLibrary)
	s.mux.HandleFunc("DELETE /api/library/{id}", s.handleDeleteItem)
	s.mux.HandleFunc("GET /api/library/{id}/stream", s.handleStream)
	s.mux.HandleFunc("GET /api/library/{id}/cover", s.handleCover)

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
	root.Handle("/api/", s.mux)
	root.Handle("/", StaticHandler(s.cfg.WebDir))
	return root
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

func (s *Server) handleAuthGoogle(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.GoogleEnabled() {
		writeError(w, http.StatusServiceUnavailable, "google sign-in is not configured on this server")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"url": s.oauthConfig().AuthURL(s.stateToken()),
	})
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !s.validState(query.Get("state")) {
		http.Redirect(w, r, "/?error=invalid_state", http.StatusFound)
		return
	}
	if query.Get("code") == "" {
		http.Redirect(w, r, "/?error=missing_code", http.StatusFound)
		return
	}

	token, err := s.oauthConfig().ExchangeToken(r.Context(), query.Get("code"))
	if err != nil {
		log.Printf("google oauth exchange failed: %v", err)
		http.Redirect(w, r, "/?error=exchange_failed", http.StatusFound)
		return
	}
	if token.RefreshToken == "" {
		http.Redirect(w, r, "/?error=missing_refresh_token", http.StatusFound)
		return
	}

	email := s.oauthConfig().Email(r.Context(), token)
	if err := s.store.SaveGoogleToken(token.RefreshToken, email); err != nil {
		log.Printf("could not store the google token: %v", err)
		http.Redirect(w, r, "/?error=storage_failed", http.StatusFound)
		return
	}
	s.resetClient()

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	_, email, ok := s.store.GoogleToken()
	writeJSON(w, http.StatusOK, map[string]any{
		"signedIn": ok,
		"email":    email,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearGoogleToken(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.forgetClient()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlaylists(w http.ResponseWriter, r *http.Request) {
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
	client, err := s.client(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sign in with Google to see your playlists")
		return
	}

	id := r.PathValue("id")
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

	jobs, err := s.manager.Enqueue(r.Context(), body.URL, parsed, body.Playlist)
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

func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	items := s.store.Items()
	if items == nil {
		items = []store.Item{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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

// stateToken returns the OAuth state value, generating a new one when the
// current one expired.
func (s *Server) stateToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == "" || time.Now().After(s.stateEnd) {
		s.state = randomToken()
		s.stateEnd = time.Now().Add(stateTTL)
	}
	return s.state
}

// validState compares the state of the callback with the one of this server.
func (s *Server) validState(state string) bool {
	if state == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == "" || time.Now().After(s.stateEnd) {
		return false
	}
	return state == s.state
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
