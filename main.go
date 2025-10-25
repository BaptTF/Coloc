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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lrstanley/go-ytdlp"
	"github.com/sirupsen/logrus"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

//go:embed index.html
var indexHTML []byte

//go:embed styles.css
var stylesCSS []byte

//go:embed app.js
var appJS []byte

const videoDir = "/videos"
const cookieDir = "/videos/cookie"
const cookieFile = "/videos/cookie/cookie.json"

// Structure pour maintenir les sessions VLC
type VLCSession struct {
	Challenge     string
	Client        *http.Client
	URL           string
	Authenticated bool
	LastActivity  time.Time
	Cookies       []*http.Cookie
}

// Map pour stocker les sessions VLC par URL
var vlcSessions = make(map[string]*VLCSession)
var vlcSessionsMutex sync.RWMutex

// Configuration VLC persistante (pour sauvegarder dans cookie.json)
type VLCCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Path   string `json:"path"`
	Domain string `json:"domain"`
}

type VLCConfig struct {
	URL           string      `json:"url"`
	Authenticated bool        `json:"authenticated"`
	LastActivity  string      `json:"last_activity"`
	Cookies       []VLCCookie `json:"cookies"`
}

type URLRequest struct {
	URL        string `json:"url"`
	AutoPlay   bool   `json:"autoPlay,omitempty"`
	VLCUrl     string `json:"vlcUrl,omitempty"`
	BackendUrl string `json:"backendUrl,omitempty"`
	Mode       string `json:"mode,omitempty"` // "stream" or "download"
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
}

// WebSocket structures
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
	ID             string    `json:"id"`
	URL            string    `json:"url"`
	OutputTemplate string    `json:"outputTemplate"`
	AutoPlay       bool      `json:"autoPlay"`
	VLCUrl         string    `json:"vlcUrl"`
	BackendUrl     string    `json:"backendUrl"`
	Mode           string    `json:"mode"` // "stream" or "download"
	CreatedAt      time.Time `json:"createdAt"`
}

// Job status for tracking download progress
type JobStatus struct {
	Job         *DownloadJob `json:"job"`
	Status      string       `json:"status"`                // "queued", "processing", "completed", "error"
	Progress    string       `json:"progress"`              // Current progress message
	Error       string       `json:"error,omitempty"`       // Error message if any
	CompletedAt *time.Time   `json:"completedAt,omitempty"` // Completion timestamp
	StreamURL   string       `json:"streamUrl,omitempty"`   // Final stream URL
}

// Global variables for download system
var (
	downloadJobs     = make(chan *DownloadJob, 100)
	jobStatuses      = make(map[string]*JobStatus) // downloadId -> status
	jobStatusesMutex sync.RWMutex
	wsClients        = make(map[*WSClient]bool)
	wsClientsMutex   sync.RWMutex
	upgrader         = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

// saveCookieToFile saves VLC session cookies to persistent storage
func saveCookieToFile(vlcURL string, session *VLCSession) error {
	// Create cookie directory if it doesn't exist
	if err := os.MkdirAll(cookieDir, 0755); err != nil {
		return fmt.Errorf("failed to create cookie directory: %w", err)
	}

	// Convert http.Cookie to VLCCookie for JSON serialization
	cookies := make([]VLCCookie, 0, len(session.Cookies))
	for _, cookie := range session.Cookies {
		cookies = append(cookies, VLCCookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Path:   cookie.Path,
			Domain: cookie.Domain,
		})
	}

	config := VLCConfig{
		URL:           vlcURL,
		Authenticated: session.Authenticated,
		LastActivity:  session.LastActivity.Format(time.RFC3339),
		Cookies:       cookies,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookie config: %w", err)
	}

	if err := os.WriteFile(cookieFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cookie file: %w", err)
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC cookies saved to file")
	return nil
}

// loadCookieFromFile loads VLC session cookies from persistent storage
func loadCookieFromFile() (*VLCConfig, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No saved cookies
		}
		return nil, fmt.Errorf("failed to read cookie file: %w", err)
	}

	var config VLCConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cookie config: %w", err)
	}

	return &config, nil
}

// verifyVLCSession checks if a VLC session is still valid by checking /wsticket
func verifyVLCSession(vlcURL string) error {
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcURL]
	vlcSessionsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Try to access /wsticket endpoint to verify authentication
	resp, err := session.Client.Get(vlcURL + "/wsticket")
	if err != nil {
		return fmt.Errorf("failed to connect to VLC: %w", err)
	}
	defer resp.Body.Close()

	// 401 means the cookie is invalid/expired
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session is no longer authenticated")
	}

	// 200 means the cookie is valid
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// restoreVLCSession restores a VLC session from saved cookies
func restoreVLCSession(config *VLCConfig) {
	if config == nil || !config.Authenticated {
		return
	}

	// Convert VLCCookie back to http.Cookie
	cookies := make([]*http.Cookie, 0, len(config.Cookies))
	for _, c := range config.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Path:   c.Path,
			Domain: c.Domain,
		})
	}

	// Create cookie jar and add cookies
	jar, _ := cookiejar.New(nil)
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		logrus.WithError(err).Warn("Failed to parse saved VLC URL")
		return
	}
	jar.SetCookies(parsedURL, cookies)

	// Create HTTP client with cookie jar
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	lastActivity, _ := time.Parse(time.RFC3339, config.LastActivity)

	session := &VLCSession{
		Client:        client,
		URL:           config.URL,
		Authenticated: config.Authenticated,
		LastActivity:  lastActivity,
		Cookies:       cookies,
	}

	vlcSessionsMutex.Lock()
	vlcSessions[config.URL] = session
	vlcSessionsMutex.Unlock()

	logrus.WithField("vlc_url", config.URL).Info("VLC session restored from saved cookies")

	// Verify the session is still valid using /wsticket endpoint
	go func() {
		if err := verifyVLCSession(config.URL); err != nil {
			logrus.WithError(err).Warn("Saved VLC session is no longer valid")
			// Remove invalid session
			vlcSessionsMutex.Lock()
			delete(vlcSessions, config.URL)
			vlcSessionsMutex.Unlock()
			// Delete cookie file
			os.Remove(cookieFile)
		} else {
			logrus.WithField("vlc_url", config.URL).Info("Saved VLC session is still valid")
		}
	}()
}

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

	// Load saved VLC cookies and restore session
	if config, err := loadCookieFromFile(); err != nil {
		logrus.WithError(err).Warn("Failed to load saved VLC cookies")
	} else if config != nil {
		restoreVLCSession(config)
	}

	// Start download worker
	go downloadWorker()

	// Start file watcher for video directory synchronization
	go watchVideoDirectory()

	// Servir les vidéos en static
	fs := http.FileServer(http.Dir(videoDir))
	http.Handle("/videos/", http.StripPrefix("/videos/", fs))

	// API endpoints (must be before the catch-all "/" handler)
	http.HandleFunc("/queue", queueStatusHandler)
	http.HandleFunc("/queue/clear", clearQueueHandler)
	http.HandleFunc("/url", downloadURLHandler)
	http.HandleFunc("/urlyt", downloadYouTubeHandler)
	http.HandleFunc("/twitch", downloadTwitchHandler)
	http.HandleFunc("/playurl", playURLHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/vlc/code", vlcCodeHandler)
	http.HandleFunc("/vlc/verify-code", vlcVerifyHandler)
	http.HandleFunc("/vlc/play", vlcPlayHandler)
	http.HandleFunc("/vlc/status", vlcStatusHandler)
	http.HandleFunc("/vlc/config", vlcConfigHandler)

	// Static assets
	http.HandleFunc("/styles.css", stylesHandler)
	http.HandleFunc("/app.js", appHandler)

	// Catch-all handler for the main page (must be last)
	http.HandleFunc("/", homeHandler)

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

	// Default to "stream" if mode not specified
	if req.Mode == "" {
		req.Mode = "stream"
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
		Mode:           req.Mode,
		CreatedAt:      time.Now(),
	}

	// Initialize job status
	jobStatus := &JobStatus{
		Job:      job,
		Status:   "queued",
		Progress: "En attente de traitement",
	}

	// Store job status globally
	jobStatusesMutex.Lock()
	jobStatuses[downloadID] = jobStatus
	jobStatusesMutex.Unlock()

	// Add job to queue (non-blocking)
	select {
	case downloadJobs <- job:
		logrus.WithFields(logrus.Fields{
			"downloadId": downloadID,
			"url":        req.URL,
			"autoplay":   req.AutoPlay,
			"mode":       req.Mode,
		}).Info("Download job added to queue")

		// Broadcast queue update to all clients
		broadcastQueueStatus()

		// Return immediately with download ID
		sendSuccess(w, fmt.Sprintf("Téléchargement ajouté à la file d'attente (ID: %s)", downloadID), downloadID)
	default:
		// Remove from job statuses if queue is full
		jobStatusesMutex.Lock()
		delete(jobStatuses, downloadID)
		jobStatusesMutex.Unlock()

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

	// Construire l'URL de la vidéo
	// filename already contains the video filename, prepend /videos/ for the URL
	cleanFilename := strings.TrimPrefix(filename, "/")
	videoPath := backendUrl + "/videos/" + cleanFilename

	// Vérifier que la vidéo est accessible via HTTP avant de contacter VLC
	// Extended wait time for debugging timing issues
	if !verifyVideoAccessible(videoPath, 60) {
		logrus.WithFields(logrus.Fields{
			"filename":   filename,
			"video_path": videoPath,
		}).Error("AUTO-PLAY - Vidéo non accessible, annulation auto-play")
		return
	}

	// Determine the VLC type based on file extension
	var vlcType string
	if strings.HasSuffix(strings.ToLower(filename), ".m3u8") {
		// Streaming file - use complex encoding
		vlcType = "stream"

		// Split to encode only the filename
		lastSlash := strings.LastIndex(videoPath, "/")
		baseURL := videoPath[:lastSlash+1]
		videoFilename := videoPath[lastSlash+1:]

		// Encode the filename (spaces become %20, etc.)
		encodedFilename := url.PathEscape(videoFilename)

		// Reconstruct the path with encoded filename
		fullPath := baseURL + encodedFilename

		// Manually encode for query parameter to preserve %20 as %2520
		// Replace special characters but keep the % from %20 as %25
		encodedPath := strings.ReplaceAll(fullPath, "%", "%25")
		encodedPath = strings.ReplaceAll(encodedPath, ":", "%3A")
		encodedPath = strings.ReplaceAll(encodedPath, "/", "%2F")

		// Construct the final URL
		videoPath = encodedPath
	} else {
		// Direct file (MP4, etc.) - use same encoding as streaming files
		vlcType = "stream" // VLC can handle direct URLs as stream

		// Split to encode only the filename
		lastSlash := strings.LastIndex(videoPath, "/")
		baseURL := videoPath[:lastSlash+1]
		videoFilename := videoPath[lastSlash+1:]

		// Encode the filename (spaces become %20, etc.)
		encodedFilename := url.PathEscape(videoFilename)

		// Reconstruct the path with encoded filename
		fullPath := baseURL + encodedFilename

		// Manually encode for query parameter to preserve %20 as %2520
		// Replace special characters but keep the % from %20 as %25
		encodedPath := strings.ReplaceAll(fullPath, "%", "%25")
		encodedPath = strings.ReplaceAll(encodedPath, ":", "%3A")
		encodedPath = strings.ReplaceAll(encodedPath, "/", "%2F")

		// Construct the final URL
		videoPath = encodedPath
	}

	playUrl := fmt.Sprintf("%s/play?id=-1&path=%s&type=%s", vlcUrl, videoPath, vlcType)

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

func pruneSegments() {
	segmentsDir := filepath.Join(videoDir, "segments")
	entries, err := os.ReadDir(segmentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Segments directory doesn't exist yet, nothing to prune
			return
		}
		logrus.WithError(err).Error("Erreur pruneSegments")
		return
	}
	type dirInfo struct {
		name    string
		modTime time.Time
	}
	var dirs []dirInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files, only process directories
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		dirs = append(dirs, dirInfo{name: entry.Name(), modTime: info.ModTime()})
	}
	if len(dirs) <= 10 {
		return
	}
	// Sort by modification time (oldest first)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].modTime.Before(dirs[j].modTime)
	})
	// Remove oldest directories beyond the limit of 10
	for _, di := range dirs[:len(dirs)-10] {
		dirPath := filepath.Join(segmentsDir, di.name)
		logrus.WithField("segmentDir", di.name).Info("Removing old segment directory")
		if err := os.RemoveAll(dirPath); err != nil {
			logrus.WithError(err).WithField("segmentDir", di.name).Error("Failed to remove segment directory")
		}
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

		// Extract cookies from the HTTP client's cookie jar
		if jar, ok := session.Client.Jar.(*cookiejar.Jar); ok {
			parsedURL, _ := url.Parse(vlcUrl)
			session.Cookies = jar.Cookies(parsedURL)
		}
		vlcSessionsMutex.Unlock()

		// Save cookies to file for persistence
		if err := saveCookieToFile(vlcUrl, session); err != nil {
			logrus.WithError(err).Warn("Failed to save VLC cookies to file")
		}

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
		case "list":
			// Send current video list
			files := getVideoList()
			response := WSMessage{
				Type:   "list",
				Videos: files,
			}
			client.send(response)
		case "subscribeAll":
			// Subscribe to all future downloads and get current queue status
			logrus.Info("Client subscribed to all downloads")
			// Send current queue status to this specific client
			jobStatusesMutex.RLock()
			var statuses []JobStatus
			for _, status := range jobStatuses {
				statuses = append(statuses, *status)
			}
			jobStatusesMutex.RUnlock()

			response := WSMessage{
				Type:  "queueStatus",
				Queue: statuses, // Always include queue, even if empty
			}
			logrus.WithField("queueSize", len(statuses)).Info("Sending initial queue status to client")
			client.send(response)
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

// Broadcast message to subscribers of a specific download (now just broadcasts to all)
// Kept for backward compatibility but simplified - everyone should always be in sync
func broadcastToSubscribers(downloadId string, msg WSMessage) {
	broadcastToAll(msg)
}

// Broadcast message to all WebSocket clients
func broadcastToAll(msg WSMessage) {
	wsClientsMutex.RLock()
	for client := range wsClients {
		go client.send(msg)
	}
	wsClientsMutex.RUnlock()
}

// Broadcast queue status to all WebSocket clients
func broadcastQueueStatus() {
	jobStatusesMutex.RLock()
	defer jobStatusesMutex.RUnlock()

	// Convert map to slice for JSON serialization
	var statuses []JobStatus
	for _, status := range jobStatuses {
		statuses = append(statuses, *status)
	}

	msg := WSMessage{
		Type:  "queueStatus",
		Queue: statuses,
	}

	logrus.WithField("queueSize", len(statuses)).Info("Broadcasting queue status to all clients")

	broadcastToAll(msg)
}

// API endpoint to get current queue status
func queueStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobStatusesMutex.RLock()
	defer jobStatusesMutex.RUnlock()

	// Convert map to slice for JSON serialization
	var statuses []JobStatus
	for _, status := range jobStatuses {
		statuses = append(statuses, *status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue": statuses,
	})
}

// API endpoint to clear the queue (completed and error jobs)
func clearQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobStatusesMutex.Lock()
	// Remove completed and error jobs from the queue
	for id, status := range jobStatuses {
		if status.Status == "completed" || status.Status == "error" {
			delete(jobStatuses, id)
		}
	}
	jobStatusesMutex.Unlock()

	// Broadcast updated queue status to all clients
	broadcastQueueStatus()

	logrus.Info("Queue cleared of completed and error jobs")
	sendSuccess(w, "File d'attente nettoyée", "")
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
			"mode":       job.Mode,
		}).Info("Processing download job")

		// Update job status to processing
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "processing"
			status.Progress = "Traitement en cours"
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()

		// Handle different modes
		if job.Mode == "download" {
			processDownloadJob(job)
		} else {
			// Default to stream mode
			processStreamJob(job)
		}
	}
}

// processDownloadJob handles downloading videos as MP4 files
func processDownloadJob(job *DownloadJob) {
	logrus.WithField("downloadId", job.ID).Info("Processing download job")

	// Check and update yt-dlp before downloading
	if err := checkAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lors de la vérification/mise à jour de yt-dlp",
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = "Erreur lors de la vérification/mise à jour de yt-dlp"
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		return
	}

	// Notify subscribers that download is starting
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Téléchargement en cours...",
	})

	// Create yt-dlp command with progress callback using go-ytdlp library
	// Use %(title)s.%(ext)s format to name files with video title
	outputTemplate := filepath.Join(videoDir, "%(title)s.%(ext)s")

	// Track last broadcast time to throttle updates
	var lastBroadcast time.Time
	var lastPercent float64

	dl := ytdlp.New().
		FormatSort("res,ext:mp4:m4a").
		MergeOutputFormat("mp4").
		NoPlaylist().
		Output(outputTemplate).
		Progress().
		Newline().
		SponsorblockMark("all").
		SponsorblockRemove("sponsor").
		ProgressFunc(500*time.Millisecond, func(update ytdlp.ProgressUpdate) {
			// Log that callback was called
			logrus.WithFields(logrus.Fields{
				"status":          update.Status,
				"percent":         update.Percent(),
				"downloadedBytes": update.DownloadedBytes,
				"totalBytes":      update.TotalBytes,
			}).Info("ProgressFunc callback called")

			// Build detailed progress message
			var progressMsg string

			switch update.Status {
			case ytdlp.ProgressStatusDownloading:
				// Show download progress with speed and ETA
				speed := ""
				if !update.Started.IsZero() && update.DownloadedBytes > 0 {
					elapsed := time.Since(update.Started).Seconds()
					if elapsed > 0 {
						bytesPerSec := float64(update.DownloadedBytes) / elapsed
						speed = fmt.Sprintf(" @ %.2f MiB/s", bytesPerSec/1024/1024)
					}
				}

				eta := ""
				if update.ETA() > 0 {
					eta = fmt.Sprintf(" ETA %s", update.ETA().Round(time.Second))
				}

				sizeInfo := ""
				if update.TotalBytes > 0 {
					sizeInfo = fmt.Sprintf(" (%.2f/%.2f MiB)",
						float64(update.DownloadedBytes)/1024/1024,
						float64(update.TotalBytes)/1024/1024)
				} else if update.DownloadedBytes > 0 {
					sizeInfo = fmt.Sprintf(" (%.2f MiB)", float64(update.DownloadedBytes)/1024/1024)
				}

				fragmentInfo := ""
				if update.FragmentCount > 0 {
					fragmentInfo = fmt.Sprintf(" [fragment %d/%d]", update.FragmentIndex, update.FragmentCount)
				}

				progressMsg = fmt.Sprintf("Téléchargement: %s%s%s%s%s",
					update.PercentString(), sizeInfo, speed, eta, fragmentInfo)

			case ytdlp.ProgressStatusFinished:
				progressMsg = "Téléchargement terminé, post-traitement en cours..."

			case ytdlp.ProgressStatusError:
				progressMsg = "Erreur lors du téléchargement"

			case ytdlp.ProgressStatusStarting:
				progressMsg = "Démarrage du téléchargement..."

			default:
				progressMsg = fmt.Sprintf("Status: %s @ %s", update.Status, update.PercentString())
			}

			// Update job status progress (always update local state)
			jobStatusesMutex.Lock()
			if status, exists := jobStatuses[job.ID]; exists {
				status.Progress = progressMsg
				// Update status to "downloading" when we get download progress
				if update.Status == ytdlp.ProgressStatusDownloading && status.Status != "downloading" {
					status.Status = "downloading"
				}
			}
			jobStatusesMutex.Unlock()

			// Throttle broadcasts: only broadcast if 1 second has passed OR percent changed by 1% OR status changed
			currentPercent := update.Percent()
			timeSinceLastBroadcast := time.Since(lastBroadcast)
			percentDiff := currentPercent - lastPercent

			shouldBroadcast := timeSinceLastBroadcast >= 1*time.Second ||
				percentDiff >= 1.0 ||
				update.Status != ytdlp.ProgressStatusDownloading

			if shouldBroadcast {
				logrus.WithFields(logrus.Fields{
					"downloadId": job.ID,
					"message":    progressMsg,
					"percent":    currentPercent,
				}).Info("Broadcasting progress update")

				broadcastToSubscribers(job.ID, WSMessage{
					Type:       "progress",
					DownloadID: job.ID,
					Message:    progressMsg,
					Percent:    currentPercent,
				})
				lastBroadcast = time.Now()
				lastPercent = currentPercent
			}
		})

	// Execute download
	logrus.Info("Starting yt-dlp download with progress tracking...")
	result, err := dl.Run(context.TODO(), job.URL)

	// Log result details
	if result != nil {
		logrus.WithFields(logrus.Fields{
			"exitCode":   result.ExitCode,
			"stdoutLen":  len(result.Stdout),
			"stderrLen":  len(result.Stderr),
			"outputLogs": len(result.OutputLogs),
		}).Info("yt-dlp execution completed")

		// Log some output logs if available
		if len(result.OutputLogs) > 0 {
			logrus.WithField("logCount", len(result.OutputLogs)).Info("yt-dlp output logs available")
			for i, log := range result.OutputLogs {
				if i < 5 { // Log first 5 entries
					logrus.WithFields(logrus.Fields{
						"line": log.Line,
						"pipe": log.Pipe,
					}).Info("yt-dlp output log sample")
				}
			}
		}
	}

	if err != nil {
		logrus.WithError(err).Error("yt-dlp download failed")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur yt-dlp: %v", err),
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = fmt.Sprintf("Erreur yt-dlp: %v", err)
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		return
	}

	// Find the downloaded file
	var newFileName string
	entries, err := os.ReadDir(videoDir)
	if err != nil {
		logrus.WithError(err).Error("Failed to read video directory")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lecture dossier videos après téléchargement",
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = "Erreur lecture dossier videos après téléchargement"
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		return
	}

	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newFileName = entry.Name()
		}
	}

	logrus.WithFields(logrus.Fields{
		"url":      job.URL,
		"new_file": newFileName,
	}).Info("Video downloaded successfully")

	// Auto-play if requested
	if job.AutoPlay && job.VLCUrl != "" && job.BackendUrl != "" && newFileName != "" {
		go autoPlayVideo(newFileName, job.VLCUrl, job.BackendUrl)
	}

	// Update job status to completed
	jobStatusesMutex.Lock()
	if status, exists := jobStatuses[job.ID]; exists {
		status.Status = "completed"
		status.Progress = "Téléchargement terminé"
		now := time.Now()
		status.CompletedAt = &now
	}
	jobStatusesMutex.Unlock()
	broadcastQueueStatus()

	// Notify completion
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "done",
		DownloadID: job.ID,
		File:       newFileName,
		Message:    "Téléchargement terminé",
	})

	// Refresh video list
	// Note: We can't call VideoManager.listVideos() here as it's frontend code
	// The frontend will refresh when it receives the 'done' message

	pruneVideos()
}

// downloadTwitchHandler handles Twitch streams using yt-dlp -g to get m3u8 URL
func downloadTwitchHandler(w http.ResponseWriter, r *http.Request) {
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

	logrus.WithField("url", req.URL).Info("Twitch - Extraction URL m3u8")

	// Check and update yt-dlp before extracting URL
	if err := checkAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		sendError(w, "Erreur lors de la vérification/mise à jour de yt-dlp", http.StatusInternalServerError)
		return
	}

	// Use yt-dlp to get the direct m3u8 URL using go-ytdlp library
	dl := ytdlp.New().
		GetURL().
		Format("best").
		NoPlaylist()

	output, err := dl.Run(context.TODO(), req.URL)
	if err != nil {
		logrus.WithError(err).Error("yt-dlp URL extraction failed")
		sendError(w, fmt.Sprintf("Erreur yt-dlp: %v", err), http.StatusInternalServerError)
		return
	}

	m3u8URL := strings.TrimSpace(output.Stdout)
	if m3u8URL == "" {
		sendError(w, "URL m3u8 vide", http.StatusInternalServerError)
		return
	}

	logrus.WithField("m3u8_url", m3u8URL).Info("Twitch m3u8 URL extracted successfully")

	// Play directly with VLC if requested
	if req.AutoPlay && req.VLCUrl != "" {
		go playDirectURL(m3u8URL, req.VLCUrl)
	}

	sendSuccess(w, "URL Twitch extraite avec succès", m3u8URL)
}

// playURLHandler plays a direct URL on VLC
func playURLHandler(w http.ResponseWriter, r *http.Request) {
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

	if req.VLCUrl == "" {
		sendError(w, "URL VLC manquante", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"url":     req.URL,
		"vlc_url": req.VLCUrl,
	}).Info("Direct URL play request")

	// Play the URL directly on VLC
	go playDirectURL(req.URL, req.VLCUrl)

	sendSuccess(w, "Lecture lancée sur VLC", "")
}

// playDirectURL sends a direct URL to VLC for playback
func playDirectURL(videoURL string, vlcUrl string) {
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcUrl]
	vlcSessionsMutex.RUnlock()

	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcUrl).Warn("No authenticated VLC session for direct play")
		return
	}

	// URL encode the direct URL
	encodedURL := url.PathEscape(videoURL)

	// Construct the VLC play URL for direct streaming
	playUrl := fmt.Sprintf("%s/play?id=-1&path=%s&type=stream", vlcUrl, encodedURL)

	logrus.WithFields(logrus.Fields{
		"url":      videoURL,
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("Sending direct URL to VLC")

	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create VLC play request")
		return
	}

	resp, err := session.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to send play request to VLC")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logrus.Info("Direct URL playback started successfully")
	} else {
		logrus.WithField("status", resp.StatusCode).Warn("VLC play request failed")
	}
}

// processStreamJob handles streaming videos via HLS
func processStreamJob(job *DownloadJob) {
	logrus.WithField("downloadId", job.ID).Info("Processing stream job")

	// Check and update yt-dlp before streaming
	if err := checkAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lors de la vérification/mise à jour de yt-dlp",
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = "Erreur lors de la vérification/mise à jour de yt-dlp"
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		// Notify that this job is done (failed)
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
	}

	// Notify subscribers that streaming is starting
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "queued",
		DownloadID: job.ID,
		Message:    "Streaming en file d'attente",
	})

	// Get video and audio URLs using yt-dlp
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Extraction du titre et URL vidéo...",
	})

	// Use -S vcodec:h264 to prefer h264 codec
	dl := ytdlp.New().
		GetTitle().
		GetURL().
		Format("bestvideo[ext=mp4]").
		FormatSort("vcodec:h264").
		NoPlaylist()

	// Retry up to 3 times if format is not available
	var output *ytdlp.Result
	var err error
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		output, err = dl.Run(context.TODO(), job.URL)
		if err == nil {
			break
		}

		// Check if it's a "format not available" error
		if strings.Contains(err.Error(), "Requested format is not available") {
			logrus.WithFields(logrus.Fields{
				"attempt": attempt,
				"max":     maxRetries,
			}).Warn("yt-dlp format not available, retrying...")

			if attempt < maxRetries {
				time.Sleep(2 * time.Second) // Wait before retry
				continue
			}
		}
		break
	}

	if err != nil {
		logrus.WithError(err).Error("yt-dlp video URL extraction failed")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur extraction URL vidéo: %v", err),
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = fmt.Sprintf("Erreur extraction URL vidéo: %v", err)
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		// Notify that this job is done (failed)
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
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
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = "Nombre inattendu de lignes vidéo de yt-dlp"
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		// Notify that this job is done (failed)
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
	}

	// Get audio URL
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Extraction de l'URL audio...",
	})

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
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = fmt.Sprintf("Erreur extraction URL audio: %v", err)
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		// Notify that this job is done (failed)
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
	}

	audioURL := strings.TrimSpace(outputAudio.Stdout)
	logrus.WithFields(logrus.Fields{
		"audioURL": audioURL[:50] + "...",
	}).Info("yt-dlp audio URL extracted successfully")

	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    fmt.Sprintf("URLs extraites - Titre: %s", title),
	})

	// Sanitize title for filename
	sanitizedTitle := sanitizeFilename(title)
	if sanitizedTitle == "" {
		sanitizedTitle = job.ID // fallback to ID if title is empty
	}

	logrus.WithFields(logrus.Fields{
		"title":          title,
		"sanitizedTitle": sanitizedTitle,
		"videoURL":       videoURL[:50] + "...",
		"audioURL":       audioURL[:50] + "...",
	}).Info("Title and URLs extracted successfully")

	// Create segments directory for this video
	// Structure: /videos/segments/{jobID}/segment_*.ts
	segmentSubDir := filepath.Join(videoDir, "segments", job.ID)
	if err := os.MkdirAll(segmentSubDir, 0755); err != nil {
		logrus.WithError(err).Error("Failed to create segments directory")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur création dossier segments: %v", err),
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = fmt.Sprintf("Erreur création dossier segments: %v", err)
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		// Notify that this job is done (failed)
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
	}

	// Generate HLS stream name with full path
	streamName := filepath.Join(videoDir, fmt.Sprintf("%s.m3u8", sanitizedTitle))
	segmentPattern := filepath.Join(videoDir, "segments", job.ID, "segment_%03d.ts")

	// Start HLS conversion with ffmpeg
	// Note: This can take several minutes for long videos as ffmpeg downloads and converts
	broadcastToSubscribers(job.ID, WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Conversion HLS en cours (peut prendre plusieurs minutes pour les longues vidéos)...",
	})

	// Update job status with more detailed progress
	jobStatusesMutex.Lock()
	if status, exists := jobStatuses[job.ID]; exists {
		status.Progress = "Conversion HLS en cours (téléchargement + conversion)..."
	}
	jobStatusesMutex.Unlock()
	broadcastQueueStatus()

	videoInput := ffmpeg_go.Input(videoURL)
	audioInput := ffmpeg_go.Input(audioURL)

	// Explicitly map video from first input and audio from second input
	videoStream := videoInput.Video()
	audioStream := audioInput.Audio()

	// Start ffmpeg asynchronously so it doesn't block
	cmd := ffmpeg_go.Output([]*ffmpeg_go.Stream{videoStream, audioStream}, streamName,
		ffmpeg_go.KwArgs{
			"c:v":                  "copy",
			"c:a":                  "copy",
			"f":                    "hls",
			"hls_time":             "6",
			"hls_list_size":        "0",
			"hls_segment_filename": segmentPattern,
			"hls_base_url":         fmt.Sprintf("segments/%s/", job.ID),
			"start_number":         "0",
			"hls_flags":            "independent_segments",
		}).Compile()

	// Start ffmpeg process
	err = cmd.Start()
	if err != nil {
		logrus.WithError(err).Error("ffmpeg failed to start")
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur démarrage ffmpeg: %v", err),
		})
		// Update job status to error
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[job.ID]; exists {
			status.Status = "error"
			status.Error = fmt.Sprintf("Erreur démarrage ffmpeg: %v", err)
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
		broadcastToSubscribers(job.ID, WSMessage{
			Type:       "completed",
			DownloadID: job.ID,
			Message:    "Job terminé avec erreur",
		})
		return
	}

	// Auto-play immediately after ffmpeg starts (for HLS streaming)
	// The m3u8 file will be created within seconds and segments will be available
	if job.AutoPlay && job.VLCUrl != "" && job.BackendUrl != "" {
		// streamName is full path, extract just the filename for URL
		streamURL := fmt.Sprintf("/videos/%s", filepath.Base(streamName))
		// Wait a few seconds for the first segments to be created
		go func(url, vlcUrl, backendUrl string) {
			time.Sleep(5 * time.Second)
			autoPlayVideo(url, vlcUrl, backendUrl)
		}(streamURL, job.VLCUrl, job.BackendUrl)
	}

	// Wait for ffmpeg to finish in a goroutine
	go func(jobID string, streamName string, cmd *exec.Cmd) {
		err := cmd.Wait()

		if err != nil {
			logrus.WithError(err).WithField("downloadId", jobID).Error("ffmpeg HLS conversion failed")
			broadcastToSubscribers(jobID, WSMessage{
				Type:       "error",
				DownloadID: jobID,
				Message:    fmt.Sprintf("Erreur conversion HLS: %v", err),
			})
			// Update job status to error
			jobStatusesMutex.Lock()
			if status, exists := jobStatuses[jobID]; exists {
				status.Status = "error"
				status.Error = fmt.Sprintf("Erreur conversion HLS: %v", err)
				now := time.Now()
				status.CompletedAt = &now
			}
			jobStatusesMutex.Unlock()
			broadcastQueueStatus()
			broadcastToSubscribers(jobID, WSMessage{
				Type:       "completed",
				DownloadID: jobID,
				Message:    "Job terminé avec erreur",
			})

			// Remove from job statuses after delay
			time.Sleep(30 * time.Second)
			jobStatusesMutex.Lock()
			delete(jobStatuses, jobID)
			jobStatusesMutex.Unlock()
			broadcastQueueStatus()
			return
		}

		// Streaming URL - streamName is full path, extract just the filename
		streamURL := fmt.Sprintf("/videos/%s", filepath.Base(streamName))

		logrus.WithFields(logrus.Fields{
			"downloadId": jobID,
			"streamURL":  streamURL,
		}).Info("HLS stream created successfully")

		// Prune old segments to keep only the 10 most recent
		pruneSegments()

		// Update job status to completed
		jobStatusesMutex.Lock()
		if status, exists := jobStatuses[jobID]; exists {
			status.Status = "completed"
			status.Progress = "Streaming prêt"
			status.StreamURL = streamURL
			now := time.Now()
			status.CompletedAt = &now
		}
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()

		// Notify completion with stream URL
		broadcastToSubscribers(jobID, WSMessage{
			Type:       "done",
			DownloadID: jobID,
			File:       streamURL,
			Message:    "Streaming prêt",
		})

		// Notify that this job is done (success)
		broadcastToSubscribers(jobID, WSMessage{
			Type:       "completed",
			DownloadID: jobID,
			Message:    "Job terminé avec succès",
		})

		// Remove from job statuses after 30 seconds
		time.Sleep(30 * time.Second)
		jobStatusesMutex.Lock()
		delete(jobStatuses, jobID)
		jobStatusesMutex.Unlock()
		broadcastQueueStatus()
	}(job.ID, streamName, cmd)

	// Continue to next job immediately - ffmpeg is running in background
}

// checkAndUpdateYtDlp checks if yt-dlp is up to date and updates it if necessary
// It sends WebSocket notifications to inform the frontend about the update process
func checkAndUpdateYtDlp(ctx context.Context) error {
	logrus.Info("Checking yt-dlp version and updating if necessary...")

	// Notify frontend that we're checking/updating yt-dlp
	broadcastToAll(WSMessage{
		Type:    "ytdlp_update",
		Message: "Vérification de la version yt-dlp...",
	})

	// Create yt-dlp command for update
	cmd := ytdlp.New()

	// Run update - this will check for updates and update if needed
	result, err := cmd.Update(ctx)
	if err != nil {
		logrus.WithError(err).Error("yt-dlp update failed")
		broadcastToAll(WSMessage{
			Type:    "ytdlp_update",
			Message: fmt.Sprintf("Erreur lors de la mise à jour yt-dlp: %v", err),
		})
		return err
	}

	// Check the result to see if an update was performed
	if result != nil && result.ExitCode == 0 {
		stdout := strings.TrimSpace(result.Stdout)
		if strings.Contains(stdout, "Updated yt-dlp to") || strings.Contains(stdout, "yt-dlp is up to date") {
			if strings.Contains(stdout, "Updated yt-dlp to") {
				logrus.Info("yt-dlp was updated successfully")
				broadcastToAll(WSMessage{
					Type:    "ytdlp_update",
					Message: "yt-dlp mis à jour avec succès",
				})
			} else {
				logrus.Info("yt-dlp is already up to date")
				broadcastToAll(WSMessage{
					Type:    "ytdlp_update",
					Message: "yt-dlp est déjà à jour",
				})
			}
		}
	}

	logrus.Info("yt-dlp update check completed")
	return nil
}

// watchVideoDirectory periodically monitors the video directory for changes
func watchVideoDirectory() {
	logrus.Info("Video directory watcher started")

	var lastFiles []string

	for {
		// Get current file list
		currentFiles := getVideoList()

		// Check if the list has changed
		if !slicesEqual(lastFiles, currentFiles) {
			logrus.WithFields(logrus.Fields{
				"previousCount": len(lastFiles),
				"currentCount":  len(currentFiles),
			}).Info("Video directory changed, broadcasting update")

			// Broadcast update to all clients
			msg := WSMessage{
				Type:   "list",
				Videos: currentFiles,
			}
			broadcastToAll(msg)

			// Update last known state
			lastFiles = make([]string, len(currentFiles))
			copy(lastFiles, currentFiles)
		}

		// Wait 2 seconds before checking again
		time.Sleep(2 * time.Second)
	}
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
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
