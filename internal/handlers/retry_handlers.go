package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"video-server/internal/types"
	"video-server/pkg/config"
)

// RetryDownloadHandler retries a failed download by re-adding it to the queue
func RetryDownloadHandler(w http.ResponseWriter, r *http.Request, downloadJobs chan<- *types.DownloadJob, broadcastQueue func()) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Extract jobID from URL path (/retry/:jobID)
	path := strings.TrimPrefix(r.URL.Path, "/retry/")
	jobID := strings.TrimSpace(path)

	if jobID == "" {
		sendError(w, "Job ID manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("jobID", jobID).Info("Retry download request received")

	// Get the job status from the queue
	jobStatuses := config.GetJobStatuses()
	jobStatus, exists := jobStatuses[jobID]

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Job introuvable",
		})
		return
	}

	// Only allow retry for error or completed jobs
	if jobStatus.Status != "error" && jobStatus.Status != "completed" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Seuls les jobs en erreur ou terminés peuvent être relancés",
		})
		return
	}

	// Create a new job with the same parameters
	newJob := &types.DownloadJob{
		ID:             jobID, // Keep the same ID so UI can track it
		URL:            jobStatus.Job.URL,
		OutputTemplate: jobStatus.Job.OutputTemplate,
		AutoPlay:       jobStatus.Job.AutoPlay,
		VLCUrl:         jobStatus.Job.VLCUrl,
		BackendUrl:     jobStatus.Job.BackendUrl,
		Mode:           jobStatus.Job.Mode,
		CreatedAt:      jobStatus.Job.CreatedAt, // Keep original creation time
	}

	// Update job status to queued
	jobStatus.Status = "queued"
	jobStatus.Progress = "Réessai en cours..."
	jobStatus.Error = ""

	// Try to add to queue (non-blocking)
	select {
	case downloadJobs <- newJob:
		logrus.WithField("jobID", jobID).Info("Job re-added to queue for retry")

		// Broadcast queue status update to all connected clients
		if broadcastQueue != nil {
			broadcastQueue()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: true,
			Message: "Téléchargement relancé",
			File:    jobID,
		})
	default:
		logrus.WithField("jobID", jobID).Warn("Queue full, cannot retry")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "File d'attente pleine, veuillez réessayer plus tard",
		})
	}
}
