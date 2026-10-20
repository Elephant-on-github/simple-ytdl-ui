package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthURL(t *testing.T) {
	o := &OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
	}

	raw := o.AuthURL("state-123")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := u.Query()

	if u.Host != "accounts.google.com" {
		t.Errorf("host = %q, want accounts.google.com", u.Host)
	}
	if u.Path != "/o/oauth2/auth" {
		t.Errorf("path = %q, want /o/oauth2/auth", u.Path)
	}
	if got := q.Get("access_type"); got != "offline" {
		t.Errorf("access_type = %q, want offline", got)
	}
	if got := q.Get("prompt"); got != "consent" {
		t.Errorf("prompt = %q, want consent", got)
	}
	if got := q.Get("scope"); got != Scope {
		t.Errorf("scope = %q, want %q", got, Scope)
	}
	if got := q.Get("redirect_uri"); got != o.RedirectURL {
		t.Errorf("redirect_uri = %q, want %q", got, o.RedirectURL)
	}
	if got := q.Get("client_id"); got != "client-id" {
		t.Errorf("client_id = %q", got)
	}
	if got := q.Get("state"); got != "state-123" {
		t.Errorf("state = %q, want state-123", got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q, want code", got)
	}
}

func TestExchangeReturnsRefreshToken(t *testing.T) {
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	o := &OAuth{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/api/auth/google/callback",
		TokenURL:     srv.URL + "/token",
	}

	refresh, err := o.Exchange(context.Background(), "the-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if refresh != "refresh-token" {
		t.Errorf("refresh token = %q, want refresh-token", refresh)
	}
	if gotForm.Get("code") != "the-code" {
		t.Errorf("posted code = %q, want the-code", gotForm.Get("code"))
	}
	if gotForm.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", gotForm.Get("grant_type"))
	}
	if gotForm.Get("redirect_uri") != o.RedirectURL {
		t.Errorf("redirect_uri = %q", gotForm.Get("redirect_uri"))
	}
}

func TestExchangeWithoutRefreshTokenFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"access-token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb", TokenURL: srv.URL}

	_, err := o.Exchange(context.Background(), "the-code")
	if err == nil {
		t.Fatal("Exchange error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "refresh token") {
		t.Errorf("error = %q, want it to mention the refresh token", err)
	}
}

func TestExchangeFailureIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	o := &OAuth{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://x/cb", TokenURL: srv.URL}

	if _, err := o.Exchange(context.Background(), "bad-code"); err == nil {
		t.Fatal("Exchange error = nil, want an error")
	}
}
