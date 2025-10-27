package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/sirupsen/logrus"
	"video-server/internal/download"
	"video-server/internal/types"
	"video-server/internal/websocket"
	"video-server/pkg/config"
)

// downloadYouTubeHandler handles YouTube downloads via queue system
func DownloadYouTubeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req types.URLRequest
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
	downloadID := download.GenerateDownloadID()

	// Create download job
	job := &types.DownloadJob{
		ID:             downloadID,
		URL:            req.URL,
		OutputTemplate: filepath.Join(config.VideoDir, "%(title)s.%(ext)s"),
		AutoPlay:       req.AutoPlay,
		VLCUrl:         req.VLCUrl,
		BackendUrl:     req.BackendUrl,
		Mode:           req.Mode,
		CreatedAt:      time.Now(),
	}

	// Initialize job status
	jobStatus := &types.JobStatus{
		Job:      job,
		Status:   "queued",
		Progress: "En attente de traitement",
	}

	// Store job status globally
	config.SetJobStatus(downloadID, jobStatus)

	logrus.WithField("downloadId", downloadID).Info("About to add job to queue channel")

	// Add job to queue (non-blocking)
	select {
	case config.GetDownloadJobs() <- job:
		logrus.WithFields(logrus.Fields{
			"downloadId": downloadID,
			"url":        req.URL,
			"autoplay":   req.AutoPlay,
			"mode":       req.Mode,
		}).Info("Download job added to queue")

		// Broadcast queue update to all clients
		websocket.BroadcastQueueStatus()

		// Return immediately with download ID
		sendSuccess(w, fmt.Sprintf("Téléchargement ajouté à la file d'attente (ID: %s)", downloadID), downloadID)
	default:
		// Remove from job statuses if queue is full
		logrus.WithField("downloadId", downloadID).Warn("Queue channel is full, cannot add job")
		config.DeleteJobStatus(downloadID)

		sendError(w, "File d'attente des téléchargements pleine, veuillez réessayer plus tard", http.StatusServiceUnavailable)
	}
}

// downloadTwitchHandler handles Twitch streams
func DownloadTwitchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req types.URLRequest
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
	if err := download.CheckAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		sendError(w, "Erreur lors de la vérification/mise à jour de yt-dlp", http.StatusInternalServerError)
		return
	}

	// Use yt-dlp to get the direct m3u8 URL
	dl := ytdlp.New().
		GetURL().
		Format("best").
		NoPlaylist()

	output, err := dl.Run(context.TODO(), req.URL)
	if err != nil {
		logrus.WithError(err).Error("yt-dlp URL extraction failed")
		// This is a business error (unsupported URL, etc), not a server error
		// Return 200 with success: false instead of 500
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: fmt.Sprintf("Erreur yt-dlp: %v", err),
		})
		return
	}

	m3u8URL := strings.TrimSpace(output.Stdout)
	if m3u8URL == "" {
		// Business error, not server error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "URL m3u8 vide",
		})
		return
	}

	logrus.WithField("m3u8_url", m3u8URL).Info("Twitch m3u8 URL extracted successfully")

	// Play directly with VLC if requested
	if req.AutoPlay && req.VLCUrl != "" {
		go download.PlayDirectURL(m3u8URL, req.VLCUrl)
	}

	sendSuccess(w, "URL Twitch extraite avec succès", m3u8URL)
}

// playURLHandler plays a direct URL on VLC
func PlayURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req types.URLRequest
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
	go download.PlayDirectURL(req.URL, req.VLCUrl)

	sendSuccess(w, "Lecture lancée sur VLC", "")
}

// listHandler lists available videos
func ListHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(config.VideoDir)
	if err != nil {
		sendError(w, "Impossible de lister les vidéos", http.StatusInternalServerError)
		return
	}

	// Structure for sorting by modification time
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

	// Sort by modification time (newest first)
	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.After(filesWithTime[j].modTime)
	})

	// Extract just the filenames
	var files []string
	for _, f := range filesWithTime {
		files = append(files, f.name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// QueueStatusHandler returns the current download queue status
func QueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	statuses := config.GetJobStatuses()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// ClearQueueHandler clears completed and error jobs from the queue
func ClearQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Clear completed and error jobs
	jobStatuses := config.GetJobStatuses()
	config.GetJobStatusesMutex().Lock()
	for id, status := range jobStatuses {
		if status.Status == "completed" || status.Status == "error" {
			config.DeleteJobStatus(id)
		}
	}
	config.GetJobStatusesMutex().Unlock()

	// Broadcast updated queue status
	websocket.BroadcastQueueStatus()

	sendSuccess(w, "File d'attente nettoyée", "")
}
