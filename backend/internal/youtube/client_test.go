package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
)

// testServer serves the YouTube Data API and the OAuth token endpoint.
type testServer struct {
	*httptest.Server
	requests []string
}

func newTestServer(t *testing.T, handler http.HandlerFunc) *testServer {
	t.Helper()

	ts := &testServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.requests = append(ts.requests, r.URL.String())
		if strings.HasSuffix(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func signedInStore(t *testing.T, srv *testServer) *store.Store {
	t.Helper()
	st := store.New(t.TempDir())
	if err := st.SaveGoogleToken("refresh-token", "user@example.test"); err != nil {
		t.Fatalf("SaveGoogleToken: %v", err)
	}
	return st
}

func TestNewClientWithoutTokenFails(t *testing.T) {
	st := store.New(t.TempDir())
	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb"}

	_, err := NewClient(context.Background(), st, o)
	if err == nil {
		t.Fatal("NewClient error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("error = %q, want it to mention that the user is not signed in", err)
	}
}

func TestPlaylists(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "mine=true") {
			t.Errorf("query = %q, want mine=true", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "part=snippet") {
			t.Errorf("query = %q, want part=snippet", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"kind":"youtube#playlistListResponse",
			"items":[
				{"id":"PL1","snippet":{"title":"First Playlist","thumbnails":{"default":{"url":"http://img/1.jpg"}}},"contentDetails":{"itemCount":42}},
				{"id":"PL2","snippet":{"title":"Second Playlist","thumbnails":{"default":{"url":"http://img/2.jpg"}}},"contentDetails":{"itemCount":7}}
			]
		}`))
	})

	st := signedInStore(t, srv)
	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb", TokenURL: srv.URL + "/token"}
	c, err := NewClient(context.Background(), st, o)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.baseURL = srv.URL

	playlists, err := c.Playlists(context.Background())
	if err != nil {
		t.Fatalf("Playlists: %v", err)
	}
	if len(playlists) != 2 {
		t.Fatalf("len(playlists) = %d, want 2", len(playlists))
	}
	if playlists[0].ID != "PL1" {
		t.Errorf("playlists[0].ID = %q, want PL1", playlists[0].ID)
	}
	if playlists[0].Title != "First Playlist" {
		t.Errorf("playlists[0].Title = %q", playlists[0].Title)
	}
	if playlists[0].ItemCount != 42 {
		t.Errorf("playlists[0].ItemCount = %d, want 42", playlists[0].ItemCount)
	}
	if playlists[0].Thumb != "http://img/1.jpg" {
		t.Errorf("playlists[0].Thumb = %q", playlists[0].Thumb)
	}
	if playlists[1].ID != "PL2" {
		t.Errorf("playlists[1].ID = %q, want PL2", playlists[1].ID)
	}
}

func TestPlaylistItems(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("playlistId"); got != "PL1" {
			t.Errorf("playlistId = %q, want PL1", got)
		}
		if got := r.URL.Query().Get("pageToken"); got != "page-2" {
			t.Errorf("pageToken = %q, want page-2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"kind":"youtube#playlistItemListResponse",
			"nextPageToken":"page-3",
			"items":[
				{"id":"vid1","snippet":{"title":"First Video","channelTitle":"Fake Channel","thumbnails":{"default":{"url":"http://img/v1.jpg"}}},"contentDetails":{"duration":"PT3M20S"}},
				{"id":"vid2","snippet":{"title":"Second Video","channelTitle":"Fake Channel","thumbnails":{"default":{"url":"http://img/v2.jpg"}}},"contentDetails":{"duration":"PT1M1S"}}
			]
		}`))
	})

	st := signedInStore(t, srv)
	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb", TokenURL: srv.URL + "/token"}
	c, err := NewClient(context.Background(), st, o)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.baseURL = srv.URL

	items, next, err := c.PlaylistItems(context.Background(), "PL1", "page-2")
	if err != nil {
		t.Fatalf("PlaylistItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != "vid1" {
		t.Errorf("items[0].ID = %q, want vid1", items[0].ID)
	}
	if items[0].Title != "First Video" {
		t.Errorf("items[0].Title = %q", items[0].Title)
	}
	if items[0].Channel != "Fake Channel" {
		t.Errorf("items[0].Channel = %q", items[0].Channel)
	}
	if items[0].Thumb != "http://img/v1.jpg" {
		t.Errorf("items[0].Thumb = %q", items[0].Thumb)
	}
	if items[0].Duration != 200 {
		t.Errorf("items[0].Duration = %d, want 200", items[0].Duration)
	}
	if items[1].Duration != 61 {
		t.Errorf("items[1].Duration = %d, want 61", items[1].Duration)
	}
	if next != "page-3" {
		t.Errorf("nextPageToken = %q, want page-3", next)
	}
}

func TestAPIErrorIsSurfaced(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":401,"message":"Invalid Credentials"}}`))
	})

	st := signedInStore(t, srv)
	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb", TokenURL: srv.URL + "/token"}
	c, err := NewClient(context.Background(), st, o)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.baseURL = srv.URL

	if _, err := c.Playlists(context.Background()); err == nil {
		t.Fatal("Playlists error = nil, want an error")
	}
	if _, _, err := c.PlaylistItems(context.Background(), "PL1", ""); err == nil {
		t.Fatal("PlaylistItems error = nil, want an error")
	}
}

func TestParseISODuration(t *testing.T) {
	tests := map[string]int{
		"PT3M20S":  200,
		"PT1M1S":   61,
		"PT45S":    45,
		"PT1H2M3S": 3723,
		"":         0,
		"bogus":    0,
	}
	for in, want := range tests {
		if got := parseISODuration(in); got != want {
			t.Errorf("parseISODuration(%q) = %d, want %d", in, got, want)
		}
	}
}
