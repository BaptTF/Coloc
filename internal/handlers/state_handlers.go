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

	// Get settings from server state
	autoPlay := state.GetAutoPlay()
	vlcUrl := state.GetVLCUrl()
	backendUrl := state.GetBackendUrl()

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
		"autoPlay":   autoPlay,
		"vlcUrl":     vlcUrl,
		"backendUrl": backendUrl,
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

// VLCUrlHandler handles getting and updating the VLC URL setting
func VLCUrlHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		vlcUrl := state.GetVLCUrl()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"vlcUrl": vlcUrl,
		})
	case http.MethodPost:
		var req struct {
			VLCUrl string `json:"vlcUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "Requête invalide", http.StatusBadRequest)
			return
		}
		state.SetVLCUrl(req.VLCUrl)

		// Broadcast the change to all connected clients
		websocket.BroadcastVLCUrl(req.VLCUrl)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"vlcUrl": req.VLCUrl,
		})
	default:
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

// BackendUrlHandler handles getting and updating the Backend URL setting
func BackendUrlHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		backendUrl := state.GetBackendUrl()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backendUrl": backendUrl,
		})
	case http.MethodPost:
		var req struct {
			BackendUrl string `json:"backendUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "Requête invalide", http.StatusBadRequest)
			return
		}
		state.SetBackendUrl(req.BackendUrl)

		// Broadcast the change to all connected clients
		websocket.BroadcastBackendUrl(req.BackendUrl)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backendUrl": req.BackendUrl,
		})
	default:
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}
