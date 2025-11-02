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
