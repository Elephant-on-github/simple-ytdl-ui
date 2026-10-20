package ytdlp

import "testing"

func TestClassifyURL(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantKind     Kind
		wantVideo    string
		wantPlaylist string
		wantErr      bool
	}{
		{
			name:      "watch url with list parameter is a single video",
			raw:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "plain watch url",
			raw:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "short youtu.be url",
			raw:       "https://youtu.be/dQw4w9WgXcQ",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:         "playlist url",
			raw:          "https://www.youtube.com/playlist?list=PL123",
			wantKind:     KindPlaylist,
			wantPlaylist: "PL123",
		},
		{
			name:         "watch url with only a list parameter is a playlist",
			raw:          "https://www.youtube.com/watch?list=PL123",
			wantKind:     KindPlaylist,
			wantPlaylist: "PL123",
		},
		{
			name:      "shorts url",
			raw:       "https://www.youtube.com/shorts/dQw4w9WgXcQ",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "embed url",
			raw:       "https://www.youtube.com/embed/dQw4w9WgXcQ",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "music youtube url",
			raw:       "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
			wantKind:  KindVideo,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:     "url without a youtube id",
			raw:      "https://example.com/x.mp3",
			wantKind: KindInvalid,
			wantErr:  true,
		},
		{
			name:     "empty url",
			raw:      "",
			wantKind: KindInvalid,
			wantErr:  true,
		},
		{
			name:     "garbage url",
			raw:      "not a url at all",
			wantKind: KindInvalid,
			wantErr:  true,
		},
		{
			name:     "youtube url without id",
			raw:      "https://www.youtube.com/",
			wantKind: KindInvalid,
			wantErr:  true,
		},
		{
			name:     "watch url with empty video id",
			raw:      "https://www.youtube.com/watch?v=",
			wantKind: KindInvalid,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, videoID, playlistID, err := ClassifyURL(tt.raw)

			if kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", kind, tt.wantKind)
			}
			if videoID != tt.wantVideo {
				t.Errorf("videoID = %q, want %q", videoID, tt.wantVideo)
			}
			if playlistID != tt.wantPlaylist {
				t.Errorf("playlistID = %q, want %q", playlistID, tt.wantPlaylist)
			}
			if tt.wantErr && err == nil {
				t.Error("err = nil, want an error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("err = %v, want nil", err)
			}
		})
	}
}

func TestClassifyURLVideoIgnoresPlaylistID(t *testing.T) {
	// A watch url always carries a list= parameter; it must never be treated
	// as a playlist.
	kind, videoID, playlistID, err := ClassifyURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if kind != KindVideo {
		t.Errorf("kind = %q, want %q", kind, KindVideo)
	}
	if videoID != "dQw4w9WgXcQ" {
		t.Errorf("videoID = %q", videoID)
	}
	if playlistID != "" {
		t.Errorf("playlistID = %q, want empty for a video url", playlistID)
	}
}
