// Package youtube implements the optional Google sign-in and the YouTube
// Data API v3 calls the playlists view needs.
package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// Scope is the only permission the app asks for: read the user's playlists.
const Scope = "https://www.googleapis.com/auth/youtube.readonly"

// Default endpoints. They can be overridden so the tests can point the
// client at an httptest server.
const (
	DefaultAuthURL     = "https://accounts.google.com/o/oauth2/auth"
	DefaultTokenURL    = "https://oauth2.googleapis.com/token"
	DefaultUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

// OAuth holds the Google OAuth client configuration.
type OAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string

	// AuthURLBase, TokenURL and UserInfoURL override the Google endpoints.
	AuthURLBase string
	TokenURL    string
	UserInfoURL string
}

// Config returns the oauth2 configuration for the Google endpoints.
func (o *OAuth) Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Scopes:       []string{Scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  o.authURL(),
			TokenURL: o.tokenURL(),
		},
	}
}

func (o *OAuth) authURL() string {
	if o.AuthURLBase != "" {
		return o.AuthURLBase
	}
	return DefaultAuthURL
}

func (o *OAuth) tokenURL() string {
	if o.TokenURL != "" {
		return o.TokenURL
	}
	return DefaultTokenURL
}

func (o *OAuth) userInfoURL() string {
	if o.UserInfoURL != "" {
		return o.UserInfoURL
	}
	return DefaultUserInfoURL
}

// AuthURL builds the URL the browser is sent to in order to sign in.
//
// The parameters follow the spec: access_type=offline asks for a refresh
// token and prompt=consent makes Google hand one out again.
func (o *OAuth) AuthURL(state string) string {
	q := url.Values{
		"client_id":     {o.ClientID},
		"redirect_uri":  {o.RedirectURL},
		"response_type": {"code"},
		"scope":         {Scope},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return o.authURL() + "?" + q.Encode()
}

// ExchangeToken swaps an authorization code for a token.
func (o *OAuth) ExchangeToken(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "" {
		return nil, errors.New("no authorization code provided")
	}
	return o.Config().Exchange(ctx, code)
}

// Exchange swaps an authorization code for a token and returns the refresh
// token, which is the only thing that is stored.
func (o *OAuth) Exchange(ctx context.Context, code string) (string, error) {
	tok, err := o.ExchangeToken(ctx, code)
	if err != nil {
		return "", fmt.Errorf("could not exchange the authorization code: %w", err)
	}
	if tok.RefreshToken == "" {
		return "", errors.New("google did not return a refresh token (access_type=offline and prompt=consent are required)")
	}
	return tok.RefreshToken, nil
}

// Email asks the user info endpoint for the address of the signed in user.
// It is best effort: a failure only means the UI shows no address.
func (o *OAuth) Email(ctx context.Context, tok *oauth2.Token) string {
	if tok == nil {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.userInfoURL(), nil)
	if err != nil {
		return ""
	}
	tok.SetAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var doc struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(resp.Body, &doc); err != nil {
		return ""
	}
	return doc.Email
}

// sanitizeError trims a Google error message.
func sanitizeError(msg string) string {
	return strings.TrimSpace(msg)
}
