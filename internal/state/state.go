package state

import (
	"sync"
	"time"
	"video-server/internal/types"
)

// ServerState holds the global server state
type ServerState struct {
	YtdlpStatus    string           `json:"ytdlpStatus"`
	YtdlpMessage   string           `json:"ytdlpMessage"`
	YtdlpUpdatedAt time.Time        `json:"ytdlpUpdatedAt"`
	VLCStatus      *types.VLCStatus `json:"vlcStatus,omitempty"`
	VLCQueue       *types.VLCQueue  `json:"vlcQueue,omitempty"`
	VLCVolume      *types.VLCVolume `json:"vlcVolume,omitempty"`
	LastVLCUpdate  time.Time        `json:"lastVlcUpdate"`
	AutoPlay       bool             `json:"autoPlay"`
	mutex          sync.RWMutex
}

var globalState = &ServerState{
	YtdlpStatus:    "unknown",
	YtdlpMessage:   "En attente...",
	YtdlpUpdatedAt: time.Now(),
	AutoPlay:       true, // Default to true (checkbox is checked by default)
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

// GetVLCState returns the current VLC state
func GetVLCState() (*types.VLCStatus, *types.VLCQueue, *types.VLCVolume, time.Time) {
	globalState.mutex.RLock()
	defer globalState.mutex.RUnlock()
	return globalState.VLCStatus, globalState.VLCQueue, globalState.VLCVolume, globalState.LastVLCUpdate
}

// SetVLCState updates the VLC state
func SetVLCState(status *types.VLCStatus, queue *types.VLCQueue, volume *types.VLCVolume) {
	globalState.mutex.Lock()
	defer globalState.mutex.Unlock()
	globalState.VLCStatus = status
	globalState.VLCQueue = queue
	globalState.VLCVolume = volume
	globalState.LastVLCUpdate = time.Now()
}

// GetAutoPlay returns the current autoplay setting
func GetAutoPlay() bool {
	globalState.mutex.RLock()
	defer globalState.mutex.RUnlock()
	return globalState.AutoPlay
}

// SetAutoPlay updates the autoplay setting
func SetAutoPlay(autoPlay bool) {
	globalState.mutex.Lock()
	defer globalState.mutex.Unlock()
	globalState.AutoPlay = autoPlay
}

// GetServerState returns the full server state
func GetServerState() map[string]interface{} {
	globalState.mutex.RLock()
	defer globalState.mutex.RUnlock()

	return map[string]interface{}{
		"ytdlpStatus":    globalState.YtdlpStatus,
		"ytdlpMessage":   globalState.YtdlpMessage,
		"ytdlpUpdatedAt": globalState.YtdlpUpdatedAt,
		"vlcStatus":      globalState.VLCStatus,
		"vlcQueue":       globalState.VLCQueue,
		"vlcVolume":      globalState.VLCVolume,
		"lastVlcUpdate":  globalState.LastVLCUpdate,
		"autoPlay":       globalState.AutoPlay,
	}
}
