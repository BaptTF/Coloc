package types

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// VLCSession represents a VLC session with authentication state
type VLCSession struct {
	Challenge     string
	Client        *http.Client
	URL           string
	Authenticated bool
	LastActivity  time.Time
	Cookies       []*http.Cookie
}

// VLCCookie represents a VLC cookie for JSON serialization
type VLCCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Path   string `json:"path"`
	Domain string `json:"domain"`
}

// VLCConfig represents persistent VLC configuration
type VLCConfig struct {
	URL           string      `json:"url"`
	Authenticated bool        `json:"authenticated"`
	LastActivity  string      `json:"last_activity"`
	Cookies       []VLCCookie `json:"cookies"`
}

// URLRequest represents a download request
type URLRequest struct {
	URL        string `json:"url"`
	AutoPlay   bool   `json:"autoPlay,omitempty"`
	VLCUrl     string `json:"vlcUrl,omitempty"`
	BackendUrl string `json:"backendUrl,omitempty"`
	Mode       string `json:"mode,omitempty"` // "stream" or "download"
}

// Response represents an API response
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type       string      `json:"type"`
	DownloadID string      `json:"downloadId,omitempty"`
	Line       string      `json:"line,omitempty"`
	Percent    float64     `json:"percent,omitempty"`
	File       string      `json:"file,omitempty"`
	Message    string      `json:"message,omitempty"`
	Videos     []string    `json:"videos,omitempty"`
	Queue      []JobStatus `json:"queue,omitempty"`
	// VLC-specific fields
	VLCStatus *VLCStatus `json:"vlcStatus,omitempty"`
	VLCQueue  *VLCQueue  `json:"vlcQueue,omitempty"`
	VLCError  *VLCError  `json:"vlcError,omitempty"`
	VLCVolume *VLCVolume `json:"vlcVolume,omitempty"`
	VLCAuth   *VLCAuth   `json:"vlcAuth,omitempty"`
}

// VLCStatus represents the current VLC playback status
type VLCStatus struct {
	Title              string        `json:"title"`
	Artist             string        `json:"artist"`
	Playing            bool          `json:"playing"`
	IsVideoPlaying     bool          `json:"isVideoPlaying"`
	Progress           int64         `json:"progress"`
	Duration           int64         `json:"duration"`
	ID                 int64         `json:"id"`
	ArtworkURL         string        `json:"artworkURL"`
	URI                string        `json:"uri"`
	Volume             int           `json:"volume"`
	Speed              float64       `json:"speed"`
	SleepTimer         int64         `json:"sleepTimer"`
	WaitForMediaEnd    bool          `json:"waitForMediaEnd"`
	ResetOnInteraction bool          `json:"resetOnInteraction"`
	Shuffle            bool          `json:"shuffle"`
	Repeat             int           `json:"repeat"`
	ShouldShow         bool          `json:"shouldShow"`
	Bookmarks          []VLCBookmark `json:"bookmarks"`
	Chapters           []VLCChapter  `json:"chapters"`
}

// VLCBookmark represents a bookmark in VLC
type VLCBookmark struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Time  int64  `json:"time"`
}

// VLCChapter represents a chapter in VLC
type VLCChapter struct {
	Title string `json:"title"`
	Time  int64  `json:"time"`
}

// VLCQueue represents the VLC play queue
type VLCQueue struct {
	Medias []VLCMedia `json:"medias"`
}

// VLCMedia represents a media item in VLC queue
type VLCMedia struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Duration   int64  `json:"duration"`
	ArtworkURL string `json:"artworkURL"`
	Playing    bool   `json:"playing"`
	Resolution string `json:"resolution"`
	Path       string `json:"path"`
	IsFolder   bool   `json:"isFolder"`
	Progress   int64  `json:"progress"`
	Played     bool   `json:"played"`
	FileType   string `json:"fileType"`
	Favorite   bool   `json:"favorite"`
}

// VLCError represents an error message from VLC
type VLCError struct {
	Text string `json:"text"`
}

// VLCVolume represents a volume update from VLC
type VLCVolume struct {
	Volume int `json:"volume"`
}

// VLCAuth represents an authentication message from VLC
type VLCAuth struct {
	Status         string `json:"status"`
	InitialMessage string `json:"initialMessage"`
}

// VLCLoginNeeded represents a login requirement from VLC
type VLCLoginNeeded struct {
	DialogOpened bool `json:"dialogOpened"`
}

// VLCResumeConfirmation represents a resume confirmation from VLC
type VLCResumeConfirmation struct {
	MediaTitle string `json:"mediaTitle"`
	Consumed   bool   `json:"consumed"`
}

// VLCBrowserDescription represents browser description from VLC
type VLCBrowserDescription struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

// VLCPlaybackControlForbidden represents playback control restriction from VLC
type VLCPlaybackControlForbidden struct {
	Forbidden bool `json:"forbidden"`
}

// VLCMLRefreshNeeded represents media library refresh request from VLC
type VLCMLRefreshNeeded struct {
	RefreshNeeded bool `json:"refreshNeeded"`
}

// VLCNetworkShares represents network shares from VLC
type VLCNetworkShares struct {
	Shares []VLCMedia `json:"shares"`
}

// WSClientMessage represents a message from the WebSocket client
type WSClientMessage struct {
	Action     string `json:"action"`
	DownloadID string `json:"downloadId,omitempty"`
}

// WSClient represents a WebSocket client connection
type WSClient struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

// DownloadJob represents a download task
type DownloadJob struct {
	ID             string             `json:"id"`
	URL            string             `json:"url"`
	OutputTemplate string             `json:"outputTemplate"`
	AutoPlay       bool               `json:"autoPlay"`
	VLCUrl         string             `json:"vlcUrl"`
	BackendUrl     string             `json:"backendUrl"`
	Mode           string             `json:"mode"` // "stream" or "download"
	CreatedAt      time.Time          `json:"createdAt"`
	CancelContext  context.Context    `json:"-"`
	CancelFunc     context.CancelFunc `json:"-"`
}

// JobStatus represents the status of a download job
type JobStatus struct {
	Job         *DownloadJob `json:"job"`
	Status      string       `json:"status"`                // "queued", "processing", "completed", "error", "cancelled"
	Progress    string       `json:"progress"`              // Current progress message
	Error       string       `json:"error,omitempty"`       // Error message if any
	CompletedAt *time.Time   `json:"completedAt,omitempty"` // Completion timestamp
	StreamURL   string       `json:"streamUrl,omitempty"`   // Final stream URL
	Cancelled   bool         `json:"cancelled"`             // Whether the job was cancelled
}
