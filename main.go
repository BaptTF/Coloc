package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lrstanley/go-ytdlp"
	"github.com/sirupsen/logrus"
	"github.com/u2takey/ffmpeg-go"
)

//go:embed index.html
var indexHTML []byte

//go:embed styles.css
var stylesCSS []byte

//go:embed app.js
var appJS []byte

const videoDir = "/videos"

// Structure pour maintenir les sessions VLC
type VLCSession struct {
	Challenge     string
	Client        *http.Client
	URL           string
	Authenticated bool
	LastActivity  time.Time
}

// Map pour stocker les sessions VLC par URL
var vlcSessions = make(map[string]*VLCSession)
var vlcSessionsMutex sync.RWMutex

// Configuration VLC persistante
type VLCConfig struct {
	URL           string `json:"url"`
	Authenticated bool   `json:"authenticated"`
	LastActivity  string `json:"last_activity"`
}

type URLRequest struct {
	URL        string `json:"url"`
	AutoPlay   bool   `json:"autoPlay,omitempty"`
	VLCUrl     string `json:"vlcUrl,omitempty"`
	BackendUrl string `json:"backendUrl,omitempty"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// WebSocket structures
type WSMessage struct {
	Type       string   `json:"type"`
	DownloadID string   `json:"downloadId,omitempty"`
	Line       string   `json:"line,omitempty"`
	Percent    float64  `json:"percent,omitempty"`
	File       string   `json:"file,omitempty"`
	Message    string   `json:"message,omitempty"`
	Videos     []string `json:"videos,omitempty"`
}

type WSClientMessage struct {
	Action     string `json:"action"`
	DownloadID string `json:"downloadId,omitempty"`
}

type WSClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// Download queue structures
type DownloadJob struct {
	ID             string
	URL            string
	OutputTemplate string
	AutoPlay       bool
	VLCUrl         string
	BackendUrl     string
	CreatedAt      time.Time
}

// Global variables for download system
var (
	downloadJobs     = make(chan *DownloadJob, 100)
	wsClients        = make(map[*WSClient]bool)
	wsClientsMutex   sync.RWMutex
	subscribers      = make(map[string]map[*WSClient]bool) // downloadId -> clients
	subscribersMutex sync.RWMutex
	upgrader         = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

func main() {
	// Check for install-only mode
	if len(os.Args) > 1 && os.Args[1] == "install-tools" {
		logrus.Info("Installing yt-dlp and ffmpeg...")
		ytdlp.MustInstall(context.TODO(), nil)
		ytdlp.MustInstallFFmpeg(context.TODO(), nil)
		ytdlp.MustInstallFFprobe(context.TODO(), nil)
		logrus.Info("Tools installed successfully")
		return
	}

	// Configure logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// Add ffmpeg to PATH before installation
	ffmpegPath := "/root/.cache/go-ytdlp"
	if currentPath := os.Getenv("PATH"); currentPath != "" {
		os.Setenv("PATH", ffmpegPath+":"+currentPath)
	} else {
		os.Setenv("PATH", ffmpegPath)
	}

	// Install yt-dlp and ffmpeg if needed
	logrus.Info("Installing yt-dlp and ffmpeg if needed...")
	ytdlp.MustInstallAll(context.TODO())
	logrus.Info("yt-dlp and ffmpeg ready")

	// Crée le dossier videos s'il n'existe pas
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		logrus.Fatal(err)
	}

	// Create segments directory
	segmentsDir := filepath.Join(videoDir, "segments")
	if err := os.MkdirAll(segmentsDir, 0755); err != nil {
		logrus.Fatal(err)
	}

	// Start download worker
	go downloadWorker()

	// Servir les vidéos en static
	fs := http.FileServer(http.Dir(videoDir))
	http.Handle("/videos/", http.StripPrefix("/videos/", fs))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/styles.css", stylesHandler)
	http.HandleFunc("/app.js", appHandler)
	http.HandleFunc("/url", downloadURLHandler)
	http.HandleFunc("/urlyt", downloadYouTubeHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/vlc/code", vlcCodeHandler)
	http.HandleFunc("/vlc/verify-code", vlcVerifyHandler)
	http.HandleFunc("/vlc/play", vlcPlayHandler)
	http.HandleFunc("/vlc/status", vlcStatusHandler)
	http.HandleFunc("/vlc/config", vlcConfigHandler)

	logrus.Info("Serveur démarré sur http://localhost:8080")
	logrus.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func stylesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(stylesCSS)
}

func appHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(appJS)
}

// downloadURLHandler télécharge une vidéo depuis une URL directe
func downloadURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req URLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		sendError(w, "URL manquante", http.StatusBadRequest)
		return
	}

	logrus.WithField("url", req.URL).Info("Début de téléchargement direct")

	// Télécharger le fichier
	resp, err := http.Get(req.URL)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur de téléchargement: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sendError(w, fmt.Sprintf("Erreur HTTP: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	// Générer un nom de fichier unique
	filename := fmt.Sprintf("video_%d.mp4", time.Now().Unix())
	filePath := filepath.Join(videoDir, filename)

	// Créer le fichier
	out, err := os.Create(filePath)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur de création du fichier: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Copier le contenu
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		sendError(w, fmt.Sprintf("Erreur d'écriture: %v", err), http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"url":      req.URL,
		"autoplay": req.AutoPlay,
	}).Info("Vidéo téléchargée avec succès")

	// Auto-play si demandé
	if req.AutoPlay && req.VLCUrl != "" && req.BackendUrl != "" {
		go autoPlayVideo(filename, req.VLCUrl, req.BackendUrl)
	}

	sendSuccess(w, "Vidéo téléchargée avec succès", filename)
	pruneVideos()
}

// downloadYouTubeHandler télécharge une vidéo avec yt-dlp via queue system
func downloadYouTubeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req URLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		sendError(w, "URL manquante", http.StatusBadRequest)
		return
	}

	logrus.WithField("url", req.URL).Info("Ajout téléchargement YouTube à la file")

	// Generate unique download ID
	downloadID := generateDownloadID()

	// Create download job
	job := &DownloadJob{
		ID:             downloadID,
		URL:            req.URL,
		OutputTemplate: filepath.Join(videoDir, "%(title)s.%(ext)s"),
		AutoPlay:       req.AutoPlay,
		VLCUrl:         req.VLCUrl,
		BackendUrl:     req.BackendUrl,
		CreatedAt:      time.Now(),
	}

	// Add job to queue (non-blocking)
	select {
	case downloadJobs <- job:
		logrus.WithFields(logrus.Fields{
			"downloadId": downloadID,
			"url":        req.URL,
			"autoplay":   req.AutoPlay,
		}).Info("Download job added to queue")

		// Return immediately with download ID
		sendSuccess(w, fmt.Sprintf("Téléchargement ajouté à la file d'attente (ID: %s)", downloadID), downloadID)
	default:
		sendError(w, "File d'attente des téléchargements pleine, veuillez réessayer plus tard", http.StatusServiceUnavailable)
	}
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: message,
	})
}

func sendSuccess(w http.ResponseWriter, message string, filename string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: message,
		File:    filename,
	})
}

// verifyVideoAccessible vérifie qu'une vidéo est accessible via HTTP avant de lancer VLC
func verifyVideoAccessible(videoPath string, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		logrus.WithFields(logrus.Fields{
			"video_path":  videoPath,
			"attempt":     i + 1,
			"max_retries": maxRetries,
		}).Info("AUTO-PLAY - Vérification accessibilité vidéo")

		resp, err := http.Head(videoPath)
		if err == nil && resp.StatusCode == http.StatusOK {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
			}).Info("AUTO-PLAY - Vidéo accessible via HTTP")
			return true
		}

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
				"error":      err.Error(),
			}).Warn("AUTO-PLAY - Erreur vérification HTTP")
		} else {
			logrus.WithFields(logrus.Fields{
				"video_path": videoPath,
				"attempt":    i + 1,
				"status":     resp.StatusCode,
			}).Warn("AUTO-PLAY - Vidéo pas encore accessible")
		}

		// Attendre avant le prochain essai (backoff exponentiel)
		waitTime := time.Duration(500*(i+1)) * time.Millisecond
		time.Sleep(waitTime)
	}

	logrus.WithFields(logrus.Fields{
		"video_path":  videoPath,
		"max_retries": maxRetries,
	}).Error("AUTO-PLAY - Vidéo toujours pas accessible après tous les essais")
	return false
}

// autoPlayVideo lance automatiquement une vidéo sur VLC si une session authentifiée existe
func autoPlayVideo(filename string, vlcUrl string, backendUrl string) {
	if filename == "" || vlcUrl == "" {
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
	}).Info("AUTO-PLAY - Tentative de lecture automatique")

	// Vérifier si une session VLC authentifiée existe
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcUrl).Warn("AUTO-PLAY - Pas de session VLC authentifiée")
		return
	}

	// Construire l'URL de la vidéo avec proper path encoding
	// Utiliser PathEscape pour les chemins URL (pas QueryEscape)
	encodedFilename := url.PathEscape(filename)
	videoPath := backendUrl + "/videos/" + encodedFilename

	// Vérifier que la vidéo est accessible via HTTP avant de contacter VLC
	// Extended wait time for debugging timing issues
	if !verifyVideoAccessible(videoPath, 60) {
		logrus.WithFields(logrus.Fields{
			"filename":   filename,
			"video_path": videoPath,
		}).Error("AUTO-PLAY - Vidéo non accessible, annulation auto-play")
		return
	}

	baseUrl, _ := url.Parse(vlcUrl + "/play")
	queryParams := baseUrl.Query()
	queryParams.Set("id", "-1")
	queryParams.Set("path", videoPath)
	queryParams.Set("type", "stream")
	baseUrl.RawQuery = queryParams.Encode()
	playUrl := baseUrl.String()

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("AUTO-PLAY - Envoi commande lecture à VLC")

	// Créer la requête
	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		logrus.WithError(err).Error("AUTO-PLAY - Erreur création requête")
		return
	}

	// Utiliser le client de la session pour maintenir l'authentification
	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("AUTO-PLAY - Erreur connexion VLC")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"vlc_url":  vlcUrl,
		}).Info("AUTO-PLAY - Lecture automatique réussie")
	} else {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"vlc_url":  vlcUrl,
			"status":   resp.StatusCode,
		}).Warn("AUTO-PLAY - Échec lecture automatique")
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		sendError(w, "Impossible de lister les vidéos", http.StatusInternalServerError)
		return
	}

	// Structure pour trier par date de modification
	type fileWithTime struct {
		name    string
		modTime time.Time
	}

	var filesWithTime []fileWithTime
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			filesWithTime = append(filesWithTime, fileWithTime{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// Trier par date de modification (plus récent en premier)
	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.After(filesWithTime[j].modTime)
	})

	// Extraire juste les noms de fichiers
	var files []string
	for _, f := range filesWithTime {
		files = append(files, f.name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func pruneVideos() {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		logrus.WithError(err).Error("Erreur pruneVideos")
		return
	}
	type fileInfo struct {
		name    string
		modTime time.Time
	}
	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
	}
	if len(files) <= 10 {
		return
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})
	for _, fi := range files[:len(files)-10] {
		os.Remove(filepath.Join(videoDir, fi.name))
	}
}

func vlcCodeHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC CODE - Début demande challenge")

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
		"vlc_url": vlcUrl,
		"data":    formData.Encode(),
	}).Info("VLC CODE - Envoi vers VLC")

	resp, err := client.Post(vlcUrl+"/code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC CODE - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC CODE - Status reçu de VLC")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	challenge := string(body)
	logrus.WithFields(logrus.Fields{
		"vlc_url":          vlcUrl,
		"challenge":        challenge,
		"challenge_length": len(challenge),
	}).Info("VLC CODE - Challenge reçu de VLC")

	// Stocker la session pour cette URL VLC
	vlcSessionsMutex.Lock()
	vlcSessions[vlcUrl] = &VLCSession{
		Challenge: challenge,
		Client:    client,
		URL:       vlcUrl,
	}
	vlcSessionsMutex.Unlock()

	logrus.WithField("vlc_url", vlcUrl).Info("VLC CODE - Session stockée")

	// Use sendSuccess/sendError instead of proxying raw response
	if resp.StatusCode == http.StatusOK {
		sendSuccess(w, "Challenge récupéré avec succès", challenge)
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(challenge)), http.StatusBadGateway)
	}
}

func vlcVerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC VERIFY - Début vérification code")

	// Récupérer la session VLC stockée
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists {
		logrus.WithField("vlc_url", vlcUrl).Error("VLC VERIFY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez redemander un code", http.StatusBadRequest)
		return
	}

	// Parser le JSON du client pour extraire le code brut (4 chiffres)
	var clientData map[string]string
	if err := json.NewDecoder(r.Body).Decode(&clientData); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC VERIFY - Erreur parsing JSON")
		sendError(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":     vlcUrl,
		"client_data": clientData,
	}).Info("VLC VERIFY - Données reçues du client")

	rawCode, exists := clientData["code"]
	if !exists {
		logrus.WithFields(logrus.Fields{
			"vlc_url":     vlcUrl,
			"client_data": clientData,
		}).Error("VLC VERIFY - Code manquant dans les données")
		sendError(w, "Code manquant", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":   vlcUrl,
		"raw_code":  rawCode,
		"challenge": session.Challenge,
	}).Info("VLC VERIFY - Code brut reçu du client")

	// Calculer le hash côté serveur comme dans test.py: sha256(code + challenge)
	hasher := sha256.New()
	hasher.Write([]byte(rawCode + session.Challenge))
	hashBytes := hasher.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)

	logrus.WithFields(logrus.Fields{
		"vlc_url":       vlcUrl,
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
		"vlc_url": vlcUrl,
		"data":    formData.Encode(),
	}).Info("VLC VERIFY - Envoi vers VLC")

	resp, err := session.Client.Post(vlcUrl+"/verify-code", "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC VERIFY -  coErreurnnexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC VERIFY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"response": string(respBody),
	}).Info("VLC VERIFY - Réponse VLC")

	// Si l'authentification réussit, mettre à jour la session
	if resp.StatusCode == http.StatusOK {
		vlcSessionsMutex.Lock()
		session.Authenticated = true
		session.LastActivity = time.Now()
		vlcSessionsMutex.Unlock()
		logrus.WithField("vlc_url", vlcUrl).Info("VLC VERIFY - Authentification réussie, session maintenue")
		sendSuccess(w, "Authentification VLC réussie", "")
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))), http.StatusBadGateway)
	}
}

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	client := &WSClient{conn: conn}

	// Add client to global list
	wsClientsMutex.Lock()
	wsClients[client] = true
	wsClientsMutex.Unlock()

	// Remove client when done
	defer func() {
		wsClientsMutex.Lock()
		delete(wsClients, client)
		wsClientsMutex.Unlock()

		// Remove from all subscriptions
		subscribersMutex.Lock()
		for downloadId := range subscribers {
			if clientMap, exists := subscribers[downloadId]; exists {
				delete(clientMap, client)
				if len(clientMap) == 0 {
					delete(subscribers, downloadId)
				}
			}
		}
		subscribersMutex.Unlock()
	}()

	logrus.Info("WebSocket client connected")

	// Handle incoming messages
	for {
		var msg WSClientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket read error")
			}
			break
		}

		logrus.WithFields(logrus.Fields{
			"action":     msg.Action,
			"downloadId": msg.DownloadID,
		}).Info("WebSocket message received")

		switch msg.Action {
		case "subscribe":
			if msg.DownloadID != "" {
				subscribersMutex.Lock()
				if subscribers[msg.DownloadID] == nil {
					subscribers[msg.DownloadID] = make(map[*WSClient]bool)
				}
				subscribers[msg.DownloadID][client] = true
				subscribersMutex.Unlock()
				logrus.WithField("downloadId", msg.DownloadID).Info("Client subscribed to download")
			}
		case "unsubscribe":
			if msg.DownloadID != "" {
				subscribersMutex.Lock()
				if clientMap, exists := subscribers[msg.DownloadID]; exists {
					delete(clientMap, client)
					if len(clientMap) == 0 {
						delete(subscribers, msg.DownloadID)
					}
				}
				subscribersMutex.Unlock()
				logrus.WithField("downloadId", msg.DownloadID).Info("Client unsubscribed from download")
			}
		case "list":
			// Send current video list
			files := getVideoList()
			response := WSMessage{
				Type:   "list",
				Videos: files,
			}
			client.send(response)
		case "subscribeAll":
			// Subscribe to all future downloads - we'll implement this by sending to all clients
			logrus.Info("Client subscribed to all downloads")
		}
	}
}

// Helper method to send message to WebSocket client
func (c *WSClient) send(msg WSMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.conn.WriteJSON(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to send WebSocket message")
	}
}

// Broadcast message to subscribers of a specific download
func broadcastToSubscribers(downloadId string, msg WSMessage) {
	subscribersMutex.RLock()
	if clientMap, exists := subscribers[downloadId]; exists {
		for client := range clientMap {
			go client.send(msg)
		}
	}
	subscribersMutex.RUnlock()
}

// Broadcast message to all WebSocket clients
func broadcastToAll(msg WSMessage) {
	wsClientsMutex.RLock()
	for client := range wsClients {
		go client.send(msg)
	}
	wsClientsMutex.RUnlock()
}

// Get current video list (helper function)
func getVideoList() []string {
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		return []string{}
	}

	type fileWithTime struct {
		name    string
		modTime time.Time
	}

	var filesWithTime []fileWithTime
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			filesWithTime = append(filesWithTime, fileWithTime{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.After(filesWithTime[j].modTime)
	})

	var files []string
	for _, f := range filesWithTime {
		files = append(files, f.name)
	}
	return files
}

// Download worker that processes the stream queue
func downloadWorker() {
	logrus.Info("Stream worker started")
	for job := range downloadJobs {
		logrus.WithFields(logrus.Fields{
			"downloadId": job.ID,
			"url":        job.URL,
		}).Info("Processing stream job")

		// Notify subscribers that streaming is starting
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "queued",
			DownloadID: job.ID,
			Message:    "Streaming en file d'attente",
		})

		// Get video and audio URLs using yt-dlp
		dl := ytdlp.New().
			GetTitle().
			GetURL().
			Format("bestvideo[ext=mp4]").
			NoPlaylist()

		output, err := dl.Run(context.TODO(), job.URL)
		if err != nil {
			logrus.WithError(err).Error("yt-dlp video URL extraction failed")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur extraction URL vidéo: %v", err),
			})
			continue
		}

		// Parse the video output: title, then video URL
		lines := strings.Split(strings.TrimSpace(output.Stdout), "\n")
		logrus.WithFields(logrus.Fields{
			"urlCount": len(lines),
			"stdout":   output.Stdout[:200] + "...", // Truncate for logging
		}).Info("yt-dlp video output received")

		var title string
		var videoURL string

		if len(lines) == 2 {
			// Title + video URL
			title = strings.TrimSpace(lines[0])
			videoURL = strings.TrimSpace(lines[1])
		} else if len(lines) == 1 {
			// No title, just video URL
			videoURL = strings.TrimSpace(lines[0])
			title = "" // Will fallback to job ID
		} else {
			logrus.Error("Unexpected number of video output lines from yt-dlp")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    "Nombre inattendu de lignes vidéo de yt-dlp",
			})
			continue
		}

		// Get audio URL
		dlAudio := ytdlp.New().
			GetURL().
			Format("bestaudio[ext=m4a]").
			NoPlaylist()

		outputAudio, err := dlAudio.Run(context.TODO(), job.URL)
		if err != nil {
			logrus.WithError(err).Error("yt-dlp audio URL extraction failed")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur extraction URL audio: %v", err),
			})
			continue
		}

		audioURL := strings.TrimSpace(outputAudio.Stdout)
		logrus.WithFields(logrus.Fields{
			"audioURL": audioURL[:50] + "...",
		}).Info("yt-dlp audio URL extracted successfully")

		// Sanitize title for filename
		sanitizedTitle := sanitizeFilename(title)
		if sanitizedTitle == "" {
			sanitizedTitle = job.ID // fallback to ID if title is empty
		}

		logrus.WithFields(logrus.Fields{
			"title":    title,
			"sanitizedTitle": sanitizedTitle,
			"videoURL": videoURL[:50] + "...",
			"audioURL": audioURL[:50] + "...",
		}).Info("Title and URLs extracted successfully")

		// Create segments directory for this video
		segmentsDir := filepath.Join(videoDir, "segments", job.ID)
		if err := os.MkdirAll(segmentsDir, 0755); err != nil {
			logrus.WithError(err).Error("Failed to create segments directory")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur création dossier segments: %v", err),
			})
			continue
		}

		// Generate HLS stream name
		streamName := fmt.Sprintf("%s.m3u8", sanitizedTitle)
		segmentPattern := filepath.Join("/videos", "segments", job.ID, job.ID, "segment_%03d.ts")

		// Ensure the segment subfolder exists
		segmentSubDir := filepath.Join(segmentsDir, job.ID, job.ID)
		if err := os.MkdirAll(segmentSubDir, 0755); err != nil {
			logrus.WithError(err).Error("Failed to create segment subfolder")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur création dossier segments: %v", err),
			})
			continue
		}

		// Start HLS conversion with ffmpeg
		videoInput := ffmpeg_go.Input(videoURL)
		audioInput := ffmpeg_go.Input(audioURL)

		// Explicitly map video from first input and audio from second input
		videoStream := videoInput.Video()
		audioStream := audioInput.Audio()

		// Change to videos directory for ffmpeg execution
		oldDir, err := os.Getwd()
		if err != nil {
			logrus.WithError(err).Error("Failed to get current directory")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur répertoire: %v", err),
			})
			continue
		}

		if err := os.Chdir(videoDir); err != nil {
			logrus.WithError(err).Error("Failed to change to videos directory")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur changement répertoire: %v", err),
			})
			continue
		}

		err = ffmpeg_go.Output([]*ffmpeg_go.Stream{videoStream, audioStream}, streamName,
			ffmpeg_go.KwArgs{
				"c:v":                   "copy",
				"c:a":                   "copy",
				"f":                     "hls",
				"hls_time":              "6",
				"hls_list_size":         "0",
				"hls_segment_filename":  segmentPattern,
				"hls_base_url":          fmt.Sprintf("segments/%s/%s/", job.ID, job.ID),
				"start_number":          "0",
			}).Run()

		// Change back to original directory
		os.Chdir(oldDir)

		if err != nil {
			logrus.WithError(err).Error("ffmpeg HLS conversion failed")
			broadcastToSubscribers(job.ID, WSMessage{
				Type:       "error",
				DownloadID: job.ID,
				Message:    fmt.Sprintf("Erreur conversion HLS: %v", err),
			})
			continue
		}

		// Streaming URL
		streamURL := fmt.Sprintf("/videos/%s", streamName)

		logrus.WithFields(logrus.Fields{
			"downloadId": job.ID,
			"streamURL":  streamURL,
		}).Info("HLS stream created successfully")

		// Notify completion with stream URL
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "done",
			DownloadID: job.ID,
			File:       streamURL,
			Message:    "Streaming prêt",
		})

		// Auto-play if requested
		if job.AutoPlay && job.VLCUrl != "" && job.BackendUrl != "" {
			go autoPlayVideo(streamURL, job.VLCUrl, job.BackendUrl)
		}
	}
}


// sanitizeFilename cleans a string to make it safe for use as a filename
func sanitizeFilename(filename string) string {
	// Replace characters that are problematic in filenames
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\n", "_",
		"\r", "_",
		"\t", "_",
	)

	sanitized := replacer.Replace(filename)

	// Trim spaces and limit length
	sanitized = strings.TrimSpace(sanitized)
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	// Ensure it's not empty
	if sanitized == "" {
		return ""
	}

	return sanitized
}

// Generate unique download ID
func generateDownloadID() string {
	return fmt.Sprintf("dl_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// vlcStatusHandler retourne l'état d'authentification pour une URL VLC
func vlcStatusHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	config := VLCConfig{
		URL:           vlcUrl,
		Authenticated: exists && session.Authenticated,
	}

	if exists {
		config.LastActivity = session.LastActivity.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// vlcConfigHandler gère la configuration VLC (GET pour récupérer, POST pour définir)
func vlcConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Récupérer toutes les sessions VLC actives
		vlcSessionsMutex.RLock()
		configs := make([]VLCConfig, 0, len(vlcSessions))
		for url, session := range vlcSessions {
			configs = append(configs, VLCConfig{
				URL:           url,
				Authenticated: session.Authenticated,
				LastActivity:  session.LastActivity.Format(time.RFC3339),
			})
		}
		vlcSessionsMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs)

	case http.MethodPost:
		// Définir une nouvelle URL VLC (sans authentification)
		var config VLCConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			sendError(w, "JSON invalide", http.StatusBadRequest)
			return
		}

		if config.URL == "" {
			sendError(w, "URL VLC manquante", http.StatusBadRequest)
			return
		}

		// Créer une nouvelle session non authentifiée
		vlcSessionsMutex.Lock()
		vlcSessions[config.URL] = &VLCSession{
			URL:           config.URL,
			Authenticated: false,
			LastActivity:  time.Now(),
		}
		vlcSessionsMutex.Unlock()

		logrus.WithField("vlc_url", config.URL).Info("VLC CONFIG - Nouvelle URL VLC définie")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Message: "URL VLC définie",
		})

	default:
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
	}
}

func vlcPlayHandler(w http.ResponseWriter, r *http.Request) {
	vlcUrl := r.URL.Query().Get("vlc")
	if vlcUrl == "" {
		sendError(w, "Paramètre vlc manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("vlc_url", vlcUrl).Info("VLC PLAY - Début lecture vidéo")

	// Récupérer la session VLC stockée
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists {
		logrus.WithField("vlc_url", vlcUrl).Error("VLC PLAY - Session VLC introuvable")
		sendError(w, "Session VLC expirée, veuillez vous authentifier", http.StatusUnauthorized)
		return
	}

	// Construire l'URL avec tous les paramètres (sans le paramètre vlc)
	playUrl := vlcUrl + "/play?" + r.URL.RawQuery
	// Retirer le paramètre vlc de l'URL
	if queryParams := r.URL.Query(); len(queryParams) > 0 {
		queryParams.Del("vlc")
		playUrl = vlcUrl + "/play?" + queryParams.Encode()
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("VLC PLAY - URL construite")

	// Créer une nouvelle requête en utilisant le client de la session
	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		sendError(w, "Erreur création requête", http.StatusInternalServerError)
		return
	}

	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": vlcUrl,
			"error":   err,
		}).Error("VLC PLAY - Erreur connexion VLC")
		sendError(w, "Erreur connexion VLC: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logrus.WithFields(logrus.Fields{
		"vlc_url": vlcUrl,
		"status":  resp.StatusCode,
	}).Info("VLC PLAY - Status reçu de VLC")

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, "Erreur lecture réponse VLC", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":  vlcUrl,
		"response": string(respBody),
	}).Info("VLC PLAY - Réponse VLC")

	// Use sendSuccess/sendError instead of proxying raw response
	if resp.StatusCode == http.StatusOK {
		sendSuccess(w, "Commande VLC envoyée avec succès", "")
	} else {
		sendError(w, fmt.Sprintf("VLC response %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody))), http.StatusBadGateway)
	}
}
