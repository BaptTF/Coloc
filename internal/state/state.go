package state

import (
	"sync"
	"time"
)

// ServerState holds the global server state
type ServerState struct {
	YtdlpStatus     string    `json:"ytdlpStatus"`
	YtdlpMessage    string    `json:"ytdlpMessage"`
	YtdlpUpdatedAt  time.Time `json:"ytdlpUpdatedAt"`
	mutex           sync.RWMutex
}

var globalState = &ServerState{
	YtdlpStatus:  "unknown",
	YtdlpMessage: "En attente...",
	YtdlpUpdatedAt: time.Now(),
}

// GetYtdlpStatus returns the current yt-dlp status
func GetYtdlpStatus() (string, string, time.Time) {
	globalState.mutex.RLock()
	defer globalState.mutex.RUnlock()
	return globalState.YtdlpStatus, globalState.YtdlpMessage, globalState.YtdlpUpdatedAt
}

// SetYtdlpStatus updates the yt-dlp status
func SetYtdlpStatus(status, message string) {
	globalState.mutex.Lock()
	defer globalState.mutex.Unlock()
	globalState.YtdlpStatus = status
	globalState.YtdlpMessage = message
	globalState.YtdlpUpdatedAt = time.Now()
}

// GetServerState returns the full server state
func GetServerState() map[string]interface{} {
	globalState.mutex.RLock()
	defer globalState.mutex.RUnlock()
	
	return map[string]interface{}{
		"ytdlpStatus":    globalState.YtdlpStatus,
		"ytdlpMessage":   globalState.YtdlpMessage,
		"ytdlpUpdatedAt": globalState.YtdlpUpdatedAt,
	}
}
