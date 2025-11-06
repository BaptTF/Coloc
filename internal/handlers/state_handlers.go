package handlers

import (
	"encoding/json"
	"net/http"

	"video-server/internal/state"
	"video-server/internal/vlc"
	"video-server/internal/websocket"
)

// ServerStateHandler returns the global server state
func ServerStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Get yt-dlp status
	ytdlpStatus, ytdlpMessage, ytdlpUpdatedAt := state.GetYtdlpStatus()

	// Get VLC sessions status
	sessions := vlc.GetVLCSessions()
	vlcSessions := make([]map[string]interface{}, 0)
	for url, session := range sessions {
		vlcSessions = append(vlcSessions, map[string]interface{}{
			"url":           url,
			"authenticated": session.Authenticated,
			"lastActivity":  session.LastActivity,
		})
	}

	// Get autoplay setting
	autoPlay := state.GetAutoPlay()

	// Build response
	response := map[string]interface{}{
		"ytdlp": map[string]interface{}{
			"status":    ytdlpStatus,
			"message":   ytdlpMessage,
			"updatedAt": ytdlpUpdatedAt,
		},
		"vlc": map[string]interface{}{
			"sessions": vlcSessions,
		},
		"autoPlay": autoPlay,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AutoPlayHandler handles getting and updating the autoplay setting
func AutoPlayHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		autoPlay := state.GetAutoPlay()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"autoPlay": autoPlay,
		})
	case http.MethodPost:
		var req struct {
			AutoPlay bool `json:"autoPlay"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "Requête invalide", http.StatusBadRequest)
			return
		}
		state.SetAutoPlay(req.AutoPlay)

		// Broadcast the change to all connected clients
		websocket.BroadcastAutoPlay(req.AutoPlay)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"autoPlay": req.AutoPlay,
		})
	default:
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}
