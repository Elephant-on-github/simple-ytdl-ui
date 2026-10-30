package store

import "time"

// Item is a finished download kept in the library.
type Item struct {
	ID        string    `json:"id"`      // "yt_<videoID>_<mode>" or "pl_<videoID>_<mode>"
	VideoID   string    `json:"videoId"` //
	Title     string    `json:"title"`   //
	Artist    string    `json:"artist"`  //
	Duration  int       `json:"duration"`
	File      string    `json:"file"`      // absolute path on disk
	Cover     string    `json:"cover"`     // absolute path on disk (optional)
	Mime      string    `json:"mime"`      //
	Size      int64     `json:"size"`      //
	Mode      string    `json:"mode"`      // "video" | "music"
	SourceURL string    `json:"sourceUrl"` //
	Playlist  string    `json:"playlist"`  // playlist title, empty for singles
	Folder    string    `json:"folder"`    // target output folder (from schedules)
	AddedAt   time.Time `json:"addedAt"`
}

// Job is one queued download.
type Job struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	VideoID    string    `json:"videoId"`
	Mode       string    `json:"mode"`
	Playlist   string    `json:"playlist"` // playlist title, empty for singles
	Folder     string    `json:"folder"`   // target output folder (from schedules)
	Status     string    `json:"status"`   // queued|running|completed|failed|canceled|paused
	Progress   float64   `json:"progress"`
	Downloaded int64     `json:"downloaded"`
	Total      int64     `json:"total"`
	Speed      float64   `json:"speed"`
	ETA        int       `json:"eta"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Job status values.
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"
	StatusPaused    = "paused"
)

// Schedule is a recurring download schedule.
type Schedule struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`       // YouTube playlist URL
	Title     string    `json:"title"`     // Playlist title (optional)
	Mode      string    `json:"mode"`      // "video" | "music"
	Folder    string    `json:"folder"`    // Target output folder
	Frequency string    `json:"frequency"` // "daily", "weekly", "monthly"
	NextRun   time.Time `json:"nextRun"`   // Next scheduled run time
	CreatedAt time.Time `json:"createdAt"`
}
