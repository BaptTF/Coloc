package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"video-server/internal/state"
	"video-server/internal/types"
	"video-server/internal/vlc"
	"video-server/internal/websocket"
)

// VLCCodeHandler requests a VLC authentication challenge
func VLCCodeHandler(w http.ResponseWriter, r *http.Request) {
	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC CODE - Début demande challenge")

	// Créer un client avec jar de cookies pour maintenir la session
	jar, err := cookiejar.New(nil)
	if err != nil {
		sendError(w, "Erreur création cookie jar", http.StatusInternalServerError)
		return
	}
	client := &http.Client{Jar: jar}

	// Selon test.py, il faut faire un POST avec form data: challenge=""
	formData := url.Values{}
	formData.Set("challenge", "")
	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"data":    formData.Encode(),
	}).Info("VLC CODE - Envoi vers VLC")

	resp, err := client.Post(vlcURL+"/code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"error":   err,
		}).Error("VLC CODE - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"status":  resp.StatusCode,
	}).Info("VLC CODE - Status reçu de VLC")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	challenge := string(body)
	logrus.WithFields(logrus.Fields{
		"vlc_url":          vlcURL,
		"challenge":        challenge,
		"challenge_length": len(challenge),
	}).Info("VLC CODE - Challenge reçu de VLC")

	// Stocker la session pour cette URL VLC
	session := &types.VLCSession{
		Challenge: challenge,
		Client:    client,
		URL:       vlcURL,
	}
	vlc.SetVLCSession(vlcURL, session)

	logrus.WithField("vlc_url", vlcURL).Info("VLC CODE - Session stockée")

	// Use sendSuccess/sendError instead of proxying raw response
	if resp.StatusCode == http.StatusOK {
		sendSuccess(w, "Challenge récupéré avec succès", challenge)
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(challenge)), http.StatusBadGateway)
	}
}

// VLCVerifyHandler verifies the VLC authentication code
func VLCVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC VERIFY - Début vérification code")

	// Récupérer la session VLC stockée
	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcURL]

	if !exists {
		logrus.WithField("vlc_url", vlcURL).Error("VLC VERIFY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez redemander un code", http.StatusBadRequest)
		return
	}

	// Parser le JSON du client pour extraire le code brut (4 chiffres)
	var clientData map[string]string
	if err := json.NewDecoder(r.Body).Decode(&clientData); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"error":   err,
		}).Error("VLC VERIFY - Erreur parsing JSON")
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":     vlcURL,
		"client_data": clientData,
	}).Info("VLC VERIFY - Données reçues du client")

	rawCode, exists := clientData["code"]
	if !exists {
		logrus.WithFields(logrus.Fields{
			"vlc_url":     vlcURL,
			"client_data": clientData,
		}).Error("VLC VERIFY - Code manquant dans les données")
		sendError(w, "Code manquant", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":   vlcURL,
		"raw_code":  rawCode,
		"challenge": session.Challenge,
	}).Info("VLC VERIFY - Code brut reçu du client")

	// Calculer le hash côté serveur comme dans test.py: sha256(code + challenge)
	hasher := sha256.New()
	hasher.Write([]byte(rawCode + session.Challenge))
	hashBytes := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)

	logrus.WithFields(logrus.Fields{
		"vlc_url":       vlcURL,
		"raw_code":      rawCode,
		"challenge":     session.Challenge,
		"concatenation": rawCode + session.Challenge,
		"hash":          hashHex,
		"hash_length":   len(hashHex),
	}).Info("VLC VERIFY - Hash calculé côté serveur")

	// Selon test.py, VLC attend form data: code=<hash>
	formData := url.Values{}
	formData.Set("code", hashHex)

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"data":    formData.Encode(),
	}).Info("VLC VERIFY - Envoi vers VLC")

	resp, err := session.Client.Post(vlcURL+"/verify-code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"error":   err,
		}).Error("VLC VERIFY - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"status":  resp.StatusCode,
	}).Info("VLC VERIFY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcURL,
		"response": string(respBody),
	}).Info("VLC VERIFY - Réponse VLC")

	// Si l'authentification réussit, mettre à jour la session
	if resp.StatusCode == http.StatusOK {
		session.Authenticated = true
		session.LastActivity = time.Now()

		// Extract cookies from the HTTP client's cookie jar
		if jar, ok := session.Client.Jar.(*cookiejar.Jar); ok {
			parsedURL, _ := url.Parse(vlcURL)
			session.Cookies = jar.Cookies(parsedURL)
		}

		vlc.SetVLCSession(vlcURL, session)

		// Save cookies to disk for persistence
		if err := vlc.SaveVLCSession(vlcURL, session); err != nil {
			logrus.WithError(err).Warn("Failed to save VLC session cookies")
		}

		// Broadcast VLC authentication status to all connected clients
		websocket.BroadcastToAll(types.WSMessage{
			Type:    "vlc_authenticated",
			Message: "VLC authentifié avec succès",
		})

		logrus.WithField("vlc_url", vlcURL).Info("VLC VERIFY - Authentification réussie, session maintenue")
		sendSuccess(w, "Authentification VLC réussie", "")
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))), http.StatusBadGateway)
	}
}

// VLCPlayHandler sends play commands to authenticated VLC
func VLCPlayHandler(w http.ResponseWriter, r *http.Request) {
	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC PLAY - Début lecture vidéo")

	// Récupérer la session VLC stockée
	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcURL]

	if !exists {
		logrus.WithField("vlc_url", vlcURL).Error("VLC PLAY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez vous authentifier", http.StatusUnauthorized)
		return
	}

	// Construire l'URL avec tous les paramètres (sans le paramètre vlc)
	playURL := vlcURL + "/play?" + r.URL.RawQuery
	// Retirer le paramètre vlc de l'URL
	if queryParams := r.URL.Query(); len(queryParams) > 0 {
		queryParams.Del("vlc")
		playURL = vlcURL + "/play?" + queryParams.Encode()
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcURL,
		"play_url": playURL,
	}).Info("VLC PLAY - URL construite")

	// Créer une nouvelle requête en utilisant le client de la session
	req, err := http.NewRequest("GET", playURL, nil)
	if err != nil {
		sendError(w, "Erreur création requête", http.StatusInternalServerError)
		return
	}

	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"error":   err,
		}).Error("VLC PLAY - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"status":  resp.StatusCode,
	}).Info("VLC PLAY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcURL,
		"response": string(respBody),
	}).Info("VLC PLAY - Réponse VLC")

	// Use sendSuccess/sendError instead of proxying raw response
	if resp.StatusCode == http.StatusOK {
		sendSuccess(w, "Commande VLC envoyée avec succès", "")
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))), http.StatusBadGateway)
	}
}

// VLCStatusHandler returns VLC session status
func VLCStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcURL]

	status := map[string]interface{}{
		"authenticated": exists && session.Authenticated,
		"url":           vlcURL,
	}

	if exists {
		status["lastActivity"] = session.LastActivity
		status["cookies"] = len(session.Cookies)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// VLCStateHandler returns current VLC playback state
func VLCStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	// Get VLC state from global state
	vlcStatus, vlcQueue, vlcVolume, lastUpdate := state.GetVLCState()

	// Check if we have recent state (within last 30 seconds)
	hasRecentState := !lastUpdate.IsZero() && time.Since(lastUpdate) < 30*time.Second

	response := map[string]interface{}{
		"vlc_url":     vlcURL,
		"has_state":   hasRecentState,
		"last_update": lastUpdate,
		"vlc_status":  vlcStatus,
		"vlc_queue":   vlcQueue,
		"vlc_volume":  vlcVolume,
	}

	// Also include WebSocket connection status
	clients := vlc.GetVLCWebSocketClients()
	wsClient, wsExists := clients[vlcURL]
	response["ws_connected"] = wsExists && wsClient.IsConnected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// VLCConfigHandler manages VLC configuration
func VLCConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Return all saved VLC configurations with authentication status
		sessions := vlc.GetVLCSessions()
		configs := []map[string]interface{}{}

		for url, session := range sessions {
			configs = append(configs, map[string]interface{}{
				"url":           url,
				"authenticated": session.Authenticated,
				"lastActivity":  session.LastActivity,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			URL string `json:"url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			sendError(w, "URL requise", http.StatusBadRequest)
			return
		}

		// Save configuration (in a real implementation, this would write to a config file)
		logrus.WithField("vlc_url", req.URL).Info("VLC configuration saved")

		sendSuccess(w, "Configuration sauvegardée", "")
		return
	}

	sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
}

// VLCWebSocketConnectHandler establishes WebSocket connection to VLC
func VLCWebSocketConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC WS CONNECT - Demande de connexion WebSocket")

	// Récupérer la session VLC stockée
	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcURL]

	// Require proper authentication before WebSocket connection
	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcURL).Warn("VLC WS CONNECT - Session non authentifiée")
		sendError(w, "VLC non authentifié. Veuillez vous connecter d'abord avec le bouton 'Se connecter'", http.StatusUnauthorized)
		return
	}

	// Obtenir ou créer le client WebSocket
	wsClient := vlc.GetVLCWebSocketClient(vlcURL, session)

	// Tenter de se connecter
	if err := wsClient.Connect(); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"error":   err,
		}).Error("VLC WS CONNECT - Échec de connexion WebSocket")
		sendError(w, "Échec de connexion WebSocket VLC: "+err.Error(), http.StatusBadGateway)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC WS CONNECT - Connexion WebSocket établie")
	sendSuccess(w, "Connexion WebSocket VLC établie", "")
}

// VLCWebSocketStatusHandler returns VLC WebSocket connection status
func VLCWebSocketStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	clients := vlc.GetVLCWebSocketClients()
	wsClient, exists := clients[vlcURL]

	status := map[string]interface{}{
		"connected":     exists && wsClient.IsConnected(),
		"vlc_url":       vlcURL,
		"authenticated": false,
	}

	// Vérifier aussi la session HTTP
	sessions := vlc.GetVLCSessions()
	if session, sessionExists := sessions[vlcURL]; sessionExists {
		status["authenticated"] = session.Authenticated
		status["last_activity"] = session.LastActivity
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// VLCWebSocketControlHandler sends control commands to VLC via WebSocket
func VLCWebSocketControlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	// Parser la commande
	var req struct {
		Command     string   `json:"command"`
		ID          *int     `json:"id,omitempty"`
		FloatValue  *float64 `json:"floatValue,omitempty"`
		LongValue   *int64   `json:"longValue,omitempty"`
		StringValue *string  `json:"stringValue,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		sendError(w, "Commande requise", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"command": req.Command,
	}).Info("VLC WS CONTROL - Envoi commande WebSocket")

	// Obtenir le client WebSocket
	clients := vlc.GetVLCWebSocketClients()
	wsClient, exists := clients[vlcURL]

	if !exists || !wsClient.IsConnected() {
		logrus.WithField("vlc_url", vlcURL).Error("VLC WS CONTROL - Client WebSocket non connecté")
		sendError(w, "Client WebSocket VLC non connecté", http.StatusBadGateway)
		return
	}

	// Envoyer la commande
	if err := wsClient.SendCommand(req.Command, req.ID, req.FloatValue, req.LongValue, req.StringValue); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcURL,
			"command": req.Command,
			"error":   err,
		}).Error("VLC WS CONTROL - Échec envoi commande")
		sendError(w, "Échec envoi commande VLC: "+err.Error(), http.StatusBadGateway)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcURL,
		"command": req.Command,
	}).Info("VLC WS CONTROL - Commande envoyée avec succès")
	sendSuccess(w, "Commande VLC envoyée avec succès", "")
}

// VLCWebSocketDisconnectHandler disconnects WebSocket connection to VLC
func VLCWebSocketDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcURL := r.URL.Query().Get("vlc")
	if vlcURL == "" {
		sendError(w, "URL VLC requise", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC WS DISCONNECT - Demande de déconnexion WebSocket")

	// Supprimer le client WebSocket
	vlc.RemoveVLCWebSocketClient(vlcURL)

	logrus.WithField("vlc_url", vlcURL).Info("VLC WS DISCONNECT - Déconnexion WebSocket effectuée")
	sendSuccess(w, "Déconnexion WebSocket VLC effectuée", "")
}
