package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"video-server/internal/state"
	"video-server/internal/types"
	"video-server/pkg/config"
)

// WSHandler handles WebSocket connections
func WSHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := config.GetUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := &types.WSClient{
		Conn: conn,
		Mu:   sync.Mutex{},
	}

	// Add client to global map
	config.AddWSClient(client)
	defer config.RemoveWSClient(client)

	logrus.Info("New WebSocket client connected")

	// Send initial server state to the new client
	ytdlpStatus, ytdlpMessage, _ := state.GetYtdlpStatus()
	client.Mu.Lock()
	client.Conn.WriteJSON(types.WSMessage{
		Type:    "ytdlp_update",
		Message: ytdlpMessage,
	})
	client.Mu.Unlock()

	logrus.WithFields(logrus.Fields{
		"ytdlpStatus":  ytdlpStatus,
		"ytdlpMessage": ytdlpMessage,
	}).Info("Sent initial server state to new WebSocket client")

	// Send current queue status immediately to the new client
	var statuses []types.JobStatus
	for _, status := range config.GetJobStatuses() {
		statuses = append(statuses, *status)
	}

	client.Mu.Lock()
	client.Conn.WriteJSON(types.WSMessage{
		Type:  "queueStatus",
		Queue: statuses,
	})
	client.Mu.Unlock()

	logrus.WithField("queueSize", len(statuses)).Info("Sent initial queue status to new WebSocket client")

	// Handle client messages
	for {
		var msg types.WSClientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket error")
			}
			break
		}

		logrus.WithField("action", msg.Action).Info("WebSocket message received")

		switch msg.Action {
		case "subscribeAll":
			// Send current queue status
			BroadcastQueueStatus()
		case "cancelDownload":
			// Cancel download logic - forward to the cancel handler
			logrus.WithField("downloadId", msg.DownloadID).Info("Download cancellation requested via WebSocket")
			// This is handled via HTTP endpoint, so we just log it for now
			// The actual cancellation happens through the HTTP endpoint
		}
	}

	logrus.Info("WebSocket client disconnected")
}

// BroadcastToAll sends a message to all WebSocket clients
func BroadcastToAll(msg types.WSMessage) {
	config.GetWSMutex().RLock()
	clientCount := len(config.GetWSClients())
	config.GetWSMutex().RUnlock()

	logrus.WithFields(logrus.Fields{
		"message_type": msg.Type,
		"client_count": clientCount,
	}).Info("Broadcasting message to WebSocket clients")

	if clientCount == 0 {
		logrus.Warn("No WebSocket clients connected to receive message")
		return
	}

	config.GetWSMutex().RLock()
	for client := range config.GetWSClients() {
		go func(c *types.WSClient) {
			c.Mu.Lock()
			defer c.Mu.Unlock()
			err := c.Conn.WriteJSON(msg)
			if err != nil {
				logrus.WithError(err).Error("Failed to send WebSocket message to client")
			} else {
				logrus.WithFields(logrus.Fields{
					"message_type": msg.Type,
				}).Debug("Successfully sent WebSocket message to client")
			}
		}(client)
	}
	config.GetWSMutex().RUnlock()
}

// BroadcastToSubscribers sends a message to subscribers of a specific download
func BroadcastToSubscribers(downloadId string, msg types.WSMessage) {
	BroadcastToAll(msg)
}

// BroadcastQueueStatus broadcasts the current queue status to all clients
func BroadcastQueueStatus() {
	// Convert map to slice for JSON serialization
	var statuses []types.JobStatus
	for _, status := range config.GetJobStatuses() {
		statuses = append(statuses, *status)
	}

	msg := types.WSMessage{
		Type:  "queueStatus",
		Queue: statuses,
	}

	log.Printf("Broadcasting queue status to all clients")
	BroadcastToAll(msg)
}

// BroadcastAutoPlay broadcasts the current autoplay setting to all clients
func BroadcastAutoPlay(autoPlay bool) {
	msg := types.WSMessage{
		Type:     "autoPlay",
		AutoPlay: &autoPlay,
	}

	log.Printf("Broadcasting autoplay setting to all clients: %v", autoPlay)
	BroadcastToAll(msg)
}

// BroadcastVLCUrl broadcasts the current VLC URL to all clients
func BroadcastVLCUrl(vlcUrl string) {
	msg := types.WSMessage{
		Type:   "vlcUrl",
		VLCUrl: &vlcUrl,
	}

	log.Printf("Broadcasting VLC URL to all clients: %s", vlcUrl)
	BroadcastToAll(msg)
}

// BroadcastBackendUrl broadcasts the current Backend URL to all clients
func BroadcastBackendUrl(backendUrl string) {
	msg := types.WSMessage{
		Type:       "backendUrl",
		BackendUrl: &backendUrl,
	}

	log.Printf("Broadcasting Backend URL to all clients: %s", backendUrl)
	BroadcastToAll(msg)
}
