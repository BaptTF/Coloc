package handlers

import (
	"encoding/json"
	"net/http"

	"video-server/internal/state"
	"video-server/internal/vlc"
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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
