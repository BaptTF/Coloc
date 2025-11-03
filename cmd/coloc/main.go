package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"video-server/internal/download"
	"video-server/internal/handlers"
	"video-server/internal/types"
	"video-server/internal/vlc"
	"video-server/internal/websocket"
	"video-server/pkg/config"
	"video-server/web"

	"github.com/sirupsen/logrus"
)

// Global job management (like original monolithic design)
var (
	downloadJobs = make(chan *types.DownloadJob, 100)
)

func updateJobStatus(jobID, newStatus, progress string) {
	statuses := config.GetJobStatuses()
	if status, exists := statuses[jobID]; exists {
		status.Status = newStatus
		if progress != "" {
			status.Progress = progress
		}
		config.SetJobStatus(jobID, status)
	}
}

func getJobStatuses() map[string]*types.JobStatus {
	return config.GetJobStatuses()
}

func addJobStatus(jobID string, status *types.JobStatus) {
	config.SetJobStatus(jobID, status)
}

func removeJobStatus(jobID string) {
	config.DeleteJobStatus(jobID)
}

// broadcastQueueStatus broadcasts the current queue status to all clients
func broadcastQueueStatus() {
	jobStatusesMap := config.GetJobStatuses()
	statuses := make([]types.JobStatus, 0, len(jobStatusesMap))
	for _, status := range jobStatusesMap {
		statuses = append(statuses, *status)
	}

	msg := types.WSMessage{
		Type:  "queueStatus",
		Queue: statuses,
	}

	logrus.Info("Broadcasting queue status to all clients")
	websocket.BroadcastToAll(msg)
}

// queueStatusHandler returns the current download queue status
func queueStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	statuses := getJobStatuses()
	if err := json.NewEncoder(w).Encode(statuses); err != nil {
		logrus.WithError(err).Error("Failed to encode queue status")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// clearQueueHandler clears completed and error jobs from the queue
func clearQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clear completed and error jobs
	statuses := getJobStatuses()
	for id, status := range statuses {
		if status.Status == "completed" || status.Status == "error" {
			removeJobStatus(id)
		}
	}

	// Broadcast updated queue status
	broadcastQueueStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "File d'attente nettoyée",
	})
}

// downloadYouTubeHandler handles YouTube downloads via queue system
func downloadYouTubeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.URLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL manquante", http.StatusBadRequest)
		return
	}

	// Default to "stream" if mode not specified
	if req.Mode == "" {
		req.Mode = "stream"
	}

	logrus.WithField("url", req.URL).Info("Ajout téléchargement YouTube à la file")

	// Generate unique download ID
	downloadID := download.GenerateDownloadID()

	// Create cancel context for the job
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

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
		CancelContext:  cancelCtx,
		CancelFunc:     cancelFunc,
	}

	// Initialize job status
	jobStatus := &types.JobStatus{
		Job:      job,
		Status:   "queued",
		Progress: "En attente de traitement",
	}

	// Add job status
	addJobStatus(downloadID, jobStatus)

	// Add job to queue (non-blocking)
	select {
	case downloadJobs <- job:
		logrus.WithField("downloadId", downloadID).Info("Job added to queue")
	default:
		logrus.WithField("downloadId", downloadID).Warn("Queue full, removing job")
		removeJobStatus(downloadID)
		http.Error(w, "File d'attente des téléchargements pleine, veuillez réessayer plus tard", http.StatusServiceUnavailable)
		return
	}

	// Broadcast queue update to all clients
	broadcastQueueStatus()

	// Return immediately with download ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Téléchargement ajouté à la file d'attente (ID: %s)", downloadID),
		"file":    downloadID,
	})
}

func main() {

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

	// yt-dlp and ffmpeg are installed during Docker build
	logrus.Info("yt-dlp and ffmpeg ready")

	// Create videos directory
	if err := os.MkdirAll(config.VideoDir, 0755); err != nil {
		logrus.Fatal(err)
	}

	// Create segments directory
	segmentsDir := filepath.Join(config.VideoDir, "segments")
	if err := os.MkdirAll(segmentsDir, 0755); err != nil {
		logrus.Fatal(err)
	}

	// Load saved VLC cookies and restore session
	vlc.InitializeVLCSessions()

	// Start download worker
	go downloadWorker()

	// Start file watcher for video directory synchronization
	go watchVideoDirectory()

	// Serve videos statically
	fs := http.FileServer(http.Dir(config.VideoDir))
	http.Handle("/videos/", http.StripPrefix("/videos/", fs))

	// Serve static files (CSS, JS, etc.) from embedded filesystem
	staticFS := http.FileServer(http.FS(web.Static))
	http.Handle("/static/", staticFS)

	// API endpoints (must be before the catch-all "/" handler)
	http.HandleFunc("/api/state", handlers.ServerStateHandler)
	http.HandleFunc("/queue", queueStatusHandler)
	http.HandleFunc("/queue/clear", clearQueueHandler)
	http.HandleFunc("/url", handlers.DownloadURLHandler)
	http.HandleFunc("/urlyt", downloadYouTubeHandler)
	http.HandleFunc("/twitch", handlers.DownloadTwitchHandler)
	http.HandleFunc("/playurl", handlers.PlayURLHandler)
	http.HandleFunc("/list", handlers.ListHandler)
	http.HandleFunc("/ws", websocket.WSHandler)
	http.HandleFunc("/vlc/code", handlers.VLCCodeHandler)
	http.HandleFunc("/vlc/verify-code", handlers.VLCVerifyHandler)
	http.HandleFunc("/vlc/play", handlers.VLCPlayHandler)
	http.HandleFunc("/vlc/status", handlers.VLCStatusHandler)
	http.HandleFunc("/vlc/state", handlers.VLCStateHandler)
	http.HandleFunc("/vlc/config", handlers.VLCConfigHandler)

	// VLC WebSocket endpoints
	http.HandleFunc("/vlc/ws/connect", handlers.VLCWebSocketConnectHandler)
	http.HandleFunc("/vlc/ws/status", handlers.VLCWebSocketStatusHandler)
	http.HandleFunc("/vlc/ws/control", handlers.VLCWebSocketControlHandler)
	http.HandleFunc("/vlc/ws/disconnect", handlers.VLCWebSocketDisconnectHandler)

	// Retry endpoint (needs access to downloadJobs channel and broadcast function)
	http.HandleFunc("/retry/", func(w http.ResponseWriter, r *http.Request) {
		handlers.RetryDownloadHandler(w, r, downloadJobs, broadcastQueueStatus)
	})

	// Cancel endpoint
	http.HandleFunc("/cancel/", handlers.CancelDownloadHandler)

	// Legacy static assets (for backward compatibility)
	http.HandleFunc("/styles.css", handlers.StylesHandler)

	// Catch-all handler for the main page (must be last)
	http.HandleFunc("/", handlers.HomeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logrus.Infof("Server started on http://localhost:%s", port)
	logrus.Fatal(http.ListenAndServe(":"+port, nil))
}

// downloadWorker processes the download queue
func downloadWorker() {
	logrus.Info("Download worker started")
	jobCount := 0

	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("panic", r).Error("Download worker panicked and died")
		}
	}()

	logrus.Info("Download worker entering job processing loop")

	for job := range downloadJobs {
		jobCount++
		logrus.WithFields(logrus.Fields{
			"downloadId": job.ID,
			"url":        job.URL,
			"mode":       job.Mode,
			"jobNumber":  jobCount,
		}).Info("Download worker received job")

		// Check if job was cancelled before processing
		statuses := config.GetJobStatuses()
		if jobStatus, exists := statuses[job.ID]; exists && jobStatus.Cancelled {
			logrus.WithField("downloadId", job.ID).Info("Job was cancelled, skipping processing")
			continue
		}

		// Update job status to processing
		updateJobStatus(job.ID, "processing", "Traitement en cours")
		broadcastQueueStatus()

		logrus.WithField("downloadId", job.ID).Info("Starting job processing...")

		// Handle different modes with error recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.WithFields(logrus.Fields{
						"downloadId": job.ID,
						"panic":      r,
					}).Error("Job processing panicked")

					// Update job status to error
					updateJobStatus(job.ID, "error", "Erreur lors du traitement")
					broadcastQueueStatus()
				}
			}()

			// Create a job status updater function to pass to download functions
			jobStatusUpdater := func(jobID, status, progress string) {
				updateJobStatus(jobID, status, progress)
			}

			// Create a cleanup function that removes completed jobs after 5 seconds
			// Error jobs are kept so users can retry them
			jobCleanup := func(jobID string) {
				go func() {
					time.Sleep(5 * time.Second)
					statuses := config.GetJobStatuses()
					if status, exists := statuses[jobID]; exists {
						// Only auto-remove completed jobs, keep error jobs for retry
						if status.Status == "completed" {
							config.DeleteJobStatus(jobID)
							logrus.WithField("downloadId", jobID).Info("Job cleaned up from queue")
							websocket.BroadcastQueueStatus()
							return
						}
					}
				}()
			}

			if job.Mode == "download" {
				logrus.WithField("downloadId", job.ID).Info("Calling ProcessDownloadJob...")
				download.ProcessDownloadJob(job, jobStatusUpdater, jobCleanup)
				logrus.WithField("downloadId", job.ID).Info("ProcessDownloadJob completed")
			} else {
				// Default to stream mode
				logrus.WithField("downloadId", job.ID).Info("Calling ProcessStreamJob...")
				download.ProcessStreamJob(job, jobStatusUpdater, jobCleanup)
				logrus.WithField("downloadId", job.ID).Info("ProcessStreamJob completed")
			}
		}()

		logrus.WithField("downloadId", job.ID).Info("Job processing finished, worker ready for next job")
	}
	logrus.Info("Download worker stopped (channel closed)")
}

// watchVideoDirectory monitors the video directory for changes
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
			msg := types.WSMessage{
				Type:   "list",
				Videos: currentFiles,
			}
			websocket.BroadcastToAll(msg)

			// Update last known state
			lastFiles = make([]string, len(currentFiles))
			copy(lastFiles, currentFiles)
		}

		// Wait 2 seconds before checking again
		time.Sleep(2 * time.Second)
	}
}

// getVideoList returns the current list of video files
func getVideoList() []string {
	entries, err := os.ReadDir(config.VideoDir)
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

// test comment
// another test comment
