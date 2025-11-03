package config

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"video-server/internal/types"
)

// Constants
const (
	VideoDir   = "/videos"
	CookieDir  = "/videos/cookie"
	CookieFile = "/videos/cookie/cookie.json"
)

// Global variables for the application
var (
	// Download queue and job management
	downloadJobs     = make(chan *types.DownloadJob, 100)
	jobStatuses      = make(map[string]*types.JobStatus) // downloadId -> status
	jobStatusesMutex sync.RWMutex

	// WebSocket management
	wsClients      = make(map[*types.WSClient]bool)
	wsClientsMutex sync.RWMutex
	upgrader       = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

// GetDownloadJobs returns the download jobs channel
func GetDownloadJobs() chan *types.DownloadJob {
	return downloadJobs
}

// GetJobStatuses returns a copy of the job statuses map (thread-safe)
func GetJobStatuses() map[string]*types.JobStatus {
	jobStatusesMutex.RLock()
	defer jobStatusesMutex.RUnlock()

	statuses := make(map[string]*types.JobStatus)
	for k, v := range jobStatuses {
		statuses[k] = v
	}
	return statuses
}

// SetJobStatus sets a job status in the global map (thread-safe)
func SetJobStatus(id string, status *types.JobStatus) {
	jobStatusesMutex.Lock()
	jobStatuses[id] = status
	jobStatusesMutex.Unlock()
}

// DeleteJobStatus removes a job status from the global map (thread-safe)
func DeleteJobStatus(id string) {
	jobStatusesMutex.Lock()
	delete(jobStatuses, id)
	jobStatusesMutex.Unlock()
}

// GetJobStatusesMutex returns the job statuses mutex
func GetJobStatusesMutex() *sync.RWMutex {
	return &jobStatusesMutex
}

// GetWSMutex returns the WebSocket clients mutex
func GetWSMutex() *sync.RWMutex {
	return &wsClientsMutex
}

// GetWSClients returns the WebSocket clients map
func GetWSClients() map[*types.WSClient]bool {
	wsClientsMutex.RLock()
	defer wsClientsMutex.RUnlock()

	clients := make(map[*types.WSClient]bool)
	for k, v := range wsClients {
		clients[k] = v
	}
	return clients
}

// AddWSClient adds a WebSocket client to the global map (thread-safe)
func AddWSClient(client *types.WSClient) {
	wsClientsMutex.Lock()
	wsClients[client] = true
	wsClientsMutex.Unlock()
}

// RemoveWSClient removes a WebSocket client from the global map (thread-safe)
func RemoveWSClient(client *types.WSClient) {
	wsClientsMutex.Lock()
	delete(wsClients, client)
	wsClientsMutex.Unlock()
}

// GetUpgrader returns the WebSocket upgrader
func GetUpgrader() websocket.Upgrader {
	return upgrader
}
