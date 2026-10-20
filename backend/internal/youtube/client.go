package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Elephant-on-github/simple-ytdl-ui/backend/internal/store"
)

// DefaultBaseURL is the YouTube Data API v3 endpoint.
const DefaultBaseURL = "https://www.googleapis.com/youtube/v3"

// Client talks to the YouTube Data API.
type Client struct {
	http    *http.Client
	baseURL string
	oauth   *OAuth
}

// Playlist is a playlist of the signed in user.
type Playlist struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Thumb     string `json:"thumb"`
	ItemCount int    `json:"itemCount"`
}

// PlaylistItem is a single entry of a playlist.
type PlaylistItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Thumb    string `json:"thumb"`
	Channel  string `json:"channel"`
	Duration int    `json:"duration"`
}

// NewClient builds a client for the signed in user. It fails when no refresh
// token has been stored yet.
func NewClient(ctx context.Context, st *store.Store, o *OAuth) (*Client, error) {
	refreshToken, _, ok := st.GoogleToken()
	if !ok {
		return nil, fmt.Errorf("not signed in")
	}
	return NewClientWithToken(ctx, refreshToken, o), nil
}

// NewClientWithToken builds a client from a refresh token. Access tokens are
// refreshed on demand.
func NewClientWithToken(ctx context.Context, refreshToken string, o *OAuth) *Client {
	cfg := o.Config()
	tok := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Hour), // force a refresh
	}
	// The token source outlives the request that created it.
	ts := oauth2.ReuseTokenSource(tok, cfg.TokenSource(context.WithoutCancel(ctx), tok))

	return &Client{
		http:    oauth2.NewClient(context.WithoutCancel(ctx), ts),
		baseURL: DefaultBaseURL,
		oauth:   o,
	}
}

// WithBaseURL points the client at a different API endpoint. It is used by
// the tests to talk to an httptest server.
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// Playlists lists the playlists of the signed in user.
func (c *Client) Playlists(ctx context.Context) ([]Playlist, error) {
	var doc struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title      string `json:"title"`
				Thumbnails struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
					Medium struct {
						URL string `json:"url"`
					} `json:"medium"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				ItemCount int `json:"itemCount"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := c.get(ctx, "/playlists", url.Values{
		"mine": {"true"},
		"part": {"snippet,contentDetails"},
	}, &doc); err != nil {
		return nil, err
	}

	playlists := make([]Playlist, 0, len(doc.Items))
	for _, item := range doc.Items {
		thumb := item.Snippet.Thumbnails.Default.URL
		if thumb == "" {
			thumb = item.Snippet.Thumbnails.Medium.URL
		}
		playlists = append(playlists, Playlist{
			ID:        item.ID,
			Title:     item.Snippet.Title,
			Thumb:     thumb,
			ItemCount: item.ContentDetails.ItemCount,
		})
	}
	return playlists, nil
}

// PlaylistItems lists the entries of a playlist and returns the token of the
// next page, if there is one.
func (c *Client) PlaylistItems(ctx context.Context, playlistID, pageToken string) ([]PlaylistItem, string, error) {
	var doc struct {
		NextPageToken string `json:"nextPageToken"`
		Items         []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
					Medium struct {
						URL string `json:"url"`
					} `json:"medium"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	q := url.Values{
		"playlistId": {playlistID},
		"part":       {"snippet,contentDetails"},
	}
	if pageToken != "" {
		q.Set("pageToken", pageToken)
	}

	if err := c.get(ctx, "/playlistItems", q, &doc); err != nil {
		return nil, "", err
	}

	items := make([]PlaylistItem, 0, len(doc.Items))
	for _, item := range doc.Items {
		thumb := item.Snippet.Thumbnails.Default.URL
		if thumb == "" {
			thumb = item.Snippet.Thumbnails.Medium.URL
		}
		items = append(items, PlaylistItem{
			ID:       item.ID,
			Title:    item.Snippet.Title,
			Thumb:    thumb,
			Channel:  item.Snippet.ChannelTitle,
			Duration: parseISODuration(item.ContentDetails.Duration),
		})
	}
	return items, doc.NextPageToken, nil
}

// get performs an authenticated GET against the API.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := strings.TrimRight(c.baseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("youtube api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("youtube api returned %s: %s", resp.Status, sanitizeError(string(body)))
	}

	if err := decodeJSON(resp.Body, out); err != nil {
		return fmt.Errorf("youtube api returned invalid JSON: %w", err)
	}
	return nil
}

func decodeJSON(r io.Reader, out any) error {
	return json.NewDecoder(r).Decode(out)
}

// parseISODuration turns an ISO 8601 duration such as "PT3M20S" into seconds.
func parseISODuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || !strings.HasPrefix(s, "P") {
		return 0
	}
	s = strings.TrimPrefix(s, "P")

	var (
		total int
		num   strings.Builder
	)
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			num.WriteRune(r)
		case r == 'H':
			total += atoi(num.String()) * 3600
			num.Reset()
		case r == 'M':
			total += atoi(num.String()) * 60
			num.Reset()
		case r == 'S':
			total += atoi(num.String())
			num.Reset()
		default:
			num.Reset()
		}
	}
	return total
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
