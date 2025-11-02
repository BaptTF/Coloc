package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"video-server/internal/types"
	"video-server/internal/websocket"
	"video-server/pkg/config"
)

// CancelDownloadHandler cancels a download job
func CancelDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Extract jobID from URL path (/cancel/:jobID)
	path := strings.TrimPrefix(r.URL.Path, "/cancel/")
	jobID := strings.TrimSpace(path)

	if jobID == "" {
		sendError(w, "Job ID manquant", http.StatusBadRequest)
		return
	}

	logrus.WithField("jobID", jobID).Info("Cancel download request received")

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

	// Only allow cancellation for queued or processing jobs
	if jobStatus.Status != "queued" && jobStatus.Status != "processing" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.Response{
			Success: false,
			Message: "Seuls les jobs en file ou en cours peuvent être annulés",
		})
		return
	}

	// Mark job as cancelled
	jobStatus.Status = "cancelled"
	jobStatus.Progress = "Annulé"
	jobStatus.Cancelled = true
	now := time.Now()
	jobStatus.CompletedAt = &now

	// Call the cancel function if it exists
	if jobStatus.Job.CancelFunc != nil {
		logrus.WithField("jobID", jobID).Info("Calling cancel function for job")
		jobStatus.Job.CancelFunc()
	}

	// Broadcast queue status update to all connected clients
	websocket.BroadcastQueueStatus()

	// Schedule cleanup of cancelled job after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		config.DeleteJobStatus(jobID)
		logrus.WithField("jobID", jobID).Info("Cancelled job cleaned up from queue")
		websocket.BroadcastQueueStatus()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.Response{
		Success: true,
		Message: "Téléchargement annulé",
		File:    jobID,
	})
}
