package download

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/sirupsen/logrus"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"video-server/internal/state"
	"video-server/internal/types"
	"video-server/internal/vlc"
	"video-server/internal/websocket"
	"video-server/pkg/config"
)

// GenerateDownloadID generates a unique download ID
func GenerateDownloadID() string {
	return fmt.Sprintf("dl_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// CheckAndUpdateYtDlp checks if yt-dlp is up to date and updates it if necessary
func CheckAndUpdateYtDlp(ctx context.Context) error {
	logrus.Info("Starting yt-dlp update check...")

	// Update global state and notify frontend
	state.SetYtdlpStatus("checking", "Vérification de la version yt-dlp...")
	websocket.BroadcastToAll(types.WSMessage{
		Type:    "ytdlp_update",
		Message: "Vérification de la version yt-dlp...",
	})

	// Create yt-dlp command for update
	logrus.Info("Creating yt-dlp update command...")
	cmd := ytdlp.New()

	// Run update - this will check for updates and update if needed
	logrus.Info("Running yt-dlp update...")
	result, err := cmd.Update(ctx)
	if err != nil {
		logrus.WithError(err).Error("yt-dlp update failed")
		state.SetYtdlpStatus("error", fmt.Sprintf("Erreur lors de la mise à jour yt-dlp: %v", err))
		websocket.BroadcastToAll(types.WSMessage{
			Type:    "ytdlp_update",
			Message: fmt.Sprintf("Erreur lors de la mise à jour yt-dlp: %v", err),
		})
		return err
	}

	logrus.Info("yt-dlp update completed, checking result...")

	// Check the result to see if an update was performed
	if result != nil && result.ExitCode == 0 {
		stdout := strings.TrimSpace(result.Stdout)
		logrus.WithFields(logrus.Fields{
			"exit_code": result.ExitCode,
			"stdout":    stdout,
		}).Info("yt-dlp update result")
		if strings.Contains(stdout, "Updated yt-dlp to") || strings.Contains(stdout, "yt-dlp is up to date") {
			if strings.Contains(stdout, "Updated yt-dlp to") {
				logrus.Info("yt-dlp was updated successfully")
				state.SetYtdlpStatus("updated", "yt-dlp mis à jour avec succès")
				websocket.BroadcastToAll(types.WSMessage{
					Type:    "ytdlp_update",
					Message: "yt-dlp mis à jour avec succès",
				})
			} else {
				logrus.Info("yt-dlp is already up to date")
				state.SetYtdlpStatus("uptodate", "yt-dlp est déjà à jour")
				websocket.BroadcastToAll(types.WSMessage{
					Type:    "ytdlp_update",
					Message: "yt-dlp est déjà à jour",
				})
			}
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"result":    result,
			"exit_code": result.ExitCode,
		}).Warn("yt-dlp update result was unexpected")
	}

	logrus.Info("yt-dlp update check completed")
	return nil
}

// ProcessDownloadJob handles downloading videos as MP4 files
func ProcessDownloadJob(job *types.DownloadJob, updateStatus func(string, string, string), cleanup func(string)) {
	// Check if job was cancelled before starting
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Job cancelled before processing")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}
	logrus.WithField("downloadId", job.ID).Info("Processing download job")

	// Check and update yt-dlp before downloading
	logrus.WithField("downloadId", job.ID).Info("Checking and updating yt-dlp...")
	if err := CheckAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lors de la vérification/mise à jour de yt-dlp",
		})
		// Update job status to error
		updateStatus(job.ID, "error", "Erreur lors de la vérification/mise à jour de yt-dlp")
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}
	logrus.WithField("downloadId", job.ID).Info("yt-dlp check/update completed")

	// Notify subscribers that download is starting
	logrus.WithField("downloadId", job.ID).Info("Sending download start message...")
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Téléchargement en cours...",
	})

	// Create yt-dlp command with progress callback
	outputTemplate := filepath.Join(config.VideoDir, "%(title)s.%(ext)s")
	logrus.WithFields(logrus.Fields{
		"downloadId":     job.ID,
		"outputTemplate": outputTemplate,
		"url":            job.URL,
	}).Info("Creating yt-dlp command...")

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
		SponsorblockRemove("sponsor")

	logrus.WithField("downloadId", job.ID).Info("Setting up yt-dlp progress callback...")

	progressCallCount := 0
	dl.ProgressFunc(500*time.Millisecond, func(update ytdlp.ProgressUpdate) {
		progressCallCount++
		if progressCallCount%10 == 1 { // Log every 10th progress call to avoid spam
			logrus.WithFields(logrus.Fields{
				"downloadId": job.ID,
				"status":     update.Status,
				"percent":    update.PercentString(),
				"count":      progressCallCount,
			}).Info("yt-dlp progress update")
		}
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

		// Update job status progress
		updateStatus(job.ID, "downloading", progressMsg)
		if update.Status == ytdlp.ProgressStatusDownloading {
			updateStatus(job.ID, "downloading", progressMsg)
		}

		// Throttle broadcasts
		currentPercent := update.Percent()
		timeSinceLastBroadcast := time.Since(lastBroadcast)
		percentDiff := currentPercent - lastPercent

		shouldBroadcast := timeSinceLastBroadcast >= 1*time.Second ||
			percentDiff >= 1.0 ||
			update.Status != ytdlp.ProgressStatusDownloading

		if shouldBroadcast {
			websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
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
	logrus.WithField("downloadId", job.ID).Info("Starting yt-dlp execution...")

	// Create context with timeout (30 minutes for large downloads) and cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// If job has a cancel context, combine them
	if job.CancelContext != nil {
		ctx = job.CancelContext
	}

	_, err := dl.Run(ctx, job.URL)

	if err != nil {
		// Check if the error is due to cancellation
		if job.CancelContext != nil && job.CancelContext.Err() == context.Canceled {
			logrus.WithField("downloadId", job.ID).Info("Download cancelled by user")
			websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
				Type:       "progress",
				DownloadID: job.ID,
				Message:    "Téléchargement annulé",
			})
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		}

		logrus.WithError(err).WithField("downloadId", job.ID).Error("yt-dlp download failed")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur yt-dlp: %v", err),
		})
		// Update job status to error
		updateStatus(job.ID, "error", fmt.Sprintf("Erreur yt-dlp: %v", err))
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	logrus.WithField("downloadId", job.ID).Info("yt-dlp execution completed successfully")

	// Find the downloaded file
	var newFileName string
	entries, err := os.ReadDir(config.VideoDir)
	if err != nil {
		logrus.WithError(err).Error("Failed to read video directory")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lecture dossier videos après téléchargement",
		})
		return
	}

	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Filter for .mp4 and .m3u8 files only
		filename := entry.Name()
		if !strings.HasSuffix(strings.ToLower(filename), ".mp4") &&
			!strings.HasSuffix(strings.ToLower(filename), ".m3u8") {
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
		go AutoPlayVideo(newFileName, job.VLCUrl, job.BackendUrl)
	}

	// Update job status to completed
	updateStatus(job.ID, "completed", "Téléchargement terminé")
	websocket.BroadcastQueueStatus()

	// Notify completion
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
		Type:       "done",
		DownloadID: job.ID,
		File:       newFileName,
		Message:    "Téléchargement terminé",
	})

	PruneVideos()

	// Schedule cleanup of this job from queue after 5 seconds
	if cleanup != nil {
		cleanup(job.ID)
	}
}

// waitForM3U8AndPlay waits for the .m3u8 file to exist and contain valid HLS content before playing
func waitForM3U8AndPlay(streamPath, streamFilename, vlcUrl, backendUrl string) {
	maxWaitTime := 60 * time.Second
	checkInterval := 1 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// Check if file exists
		if _, err := os.Stat(streamPath); err != nil {
			logrus.WithFields(logrus.Fields{
				"stream_path": streamPath,
				"elapsed":     time.Since(startTime).Seconds(),
			}).Debug("AUTO-PLAY - Waiting for m3u8 file to be created")
			time.Sleep(checkInterval)
			continue
		}

		// Check if file contains valid HLS content
		if isValidM3U8(streamPath) {
			logrus.WithFields(logrus.Fields{
				"stream_path": streamPath,
				"elapsed":     time.Since(startTime).Seconds(),
			}).Info("AUTO-PLAY - m3u8 file is ready, attempting playback")
			AutoPlayVideo(streamFilename, vlcUrl, backendUrl)
			return
		}

		logrus.WithFields(logrus.Fields{
			"stream_path": streamPath,
			"elapsed":     time.Since(startTime).Seconds(),
		}).Debug("AUTO-PLAY - m3u8 file exists but not ready yet")
		time.Sleep(checkInterval)
	}

	logrus.WithFields(logrus.Fields{
		"stream_path": streamPath,
		"elapsed":     time.Since(startTime).Seconds(),
	}).Error("AUTO-PLAY - Timeout waiting for m3u8 file to be ready")
}

// isValidM3U8 checks if the m3u8 file contains valid HLS playlist content
func isValidM3U8(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasSegments := false
	hasExtM3U := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "#EXTM3U" {
			hasExtM3U = true
		}
		if strings.HasPrefix(line, "#EXTINF:") || strings.HasSuffix(line, ".ts") {
			hasSegments = true
		}
		// If we have both, it's a valid HLS playlist
		if hasExtM3U && hasSegments {
			return true
		}
	}

	return hasExtM3U && hasSegments
}

// AutoPlayVideo launches a video automatically on VLC if an authenticated session exists
func AutoPlayVideo(filename string, vlcUrl string, backendUrl string) {
	if filename == "" || vlcUrl == "" {
		return
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
	}).Info("AUTO-PLAY - Tentative de lecture automatique")

	// Check if an authenticated VLC session exists
	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcUrl]

	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcUrl).Warn("AUTO-PLAY - Pas de session VLC authentifiée")
		return
	}

	// Build video URL
	cleanFilename := strings.TrimPrefix(filename, "/")
	videoPath := backendUrl + "/videos/" + cleanFilename

	// Verify video is accessible via HTTP before contacting VLC
	if !verifyVideoAccessible(videoPath, 10) {
		logrus.WithFields(logrus.Fields{
			"filename":   filename,
			"video_path": videoPath,
		}).Error("AUTO-PLAY - Vidéo non accessible, annulation auto-play")
		return
	}

	// Determine VLC type based on file extension
	var vlcType string
	if strings.HasSuffix(strings.ToLower(filename), ".m3u8") {
		vlcType = "stream"
		lastSlash := strings.LastIndex(videoPath, "/")
		baseURL := videoPath[:lastSlash+1]
		videoFilename := videoPath[lastSlash+1:]
		encodedFilename := url.PathEscape(videoFilename)
		fullPath := baseURL + encodedFilename
		encodedPath := strings.ReplaceAll(fullPath, "%", "%25")
		encodedPath = strings.ReplaceAll(encodedPath, ":", "%3A")
		encodedPath = strings.ReplaceAll(encodedPath, "/", "%2F")
		videoPath = encodedPath
	} else {
		vlcType = "stream"
		lastSlash := strings.LastIndex(videoPath, "/")
		baseURL := videoPath[:lastSlash+1]
		videoFilename := videoPath[lastSlash+1:]
		encodedFilename := url.PathEscape(videoFilename)
		fullPath := baseURL + encodedFilename
		encodedPath := strings.ReplaceAll(fullPath, "%", "%25")
		encodedPath = strings.ReplaceAll(encodedPath, ":", "%3A")
		encodedPath = strings.ReplaceAll(encodedPath, "/", "%2F")
		videoPath = encodedPath
	}

	playUrl := fmt.Sprintf("%s/play?id=-1&path=%s&type=%s", vlcUrl, videoPath, vlcType)

	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"vlc_url":  vlcUrl,
		"play_url": playUrl,
	}).Info("AUTO-PLAY - Envoi commande lecture à VLC")

	req, err := http.NewRequest("GET", playUrl, nil)
	if err != nil {
		logrus.WithError(err).Error("AUTO-PLAY - Erreur création requête")
		return
	}

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

// PlayDirectURL sends a direct URL to VLC for playback
func PlayDirectURL(videoURL string, vlcUrl string) {
	sessions := vlc.GetVLCSessions()
	session, exists := sessions[vlcUrl]

	if !exists || !session.Authenticated {
		logrus.WithField("vlc_url", vlcUrl).Warn("No authenticated VLC session for direct play")
		return
	}

	// URL encode the direct URL as a query parameter value
	// Use QueryEscape instead of PathEscape to properly encode query parameters in the video URL
	encodedURL := url.QueryEscape(videoURL)

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

// PruneVideos removes old videos to keep only the 10 most recent
func PruneVideos() {
	entries, err := os.ReadDir(config.VideoDir)
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
		if entry.IsDir() {
			continue
		}
		// Filter for .mp4 and .m3u8 files only
		filename := entry.Name()
		if !strings.HasSuffix(strings.ToLower(filename), ".mp4") &&
			!strings.HasSuffix(strings.ToLower(filename), ".m3u8") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), modTime: info.ModTime()})
	}

	if len(files) <= 10 {
		return
	}

	// Sort by modification time (oldest first)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	for _, fi := range files[:len(files)-10] {
		os.Remove(filepath.Join(config.VideoDir, fi.name))
	}
}

// verifyVideoAccessible checks if a video is accessible via HTTP before launching VLC
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

		waitTime := time.Duration(500*(i+1)) * time.Millisecond
		time.Sleep(waitTime)
	}

	logrus.WithFields(logrus.Fields{
		"video_path":  videoPath,
		"max_retries": maxRetries,
	}).Error("AUTO-PLAY - Vidéo toujours pas accessible après tous les essais")
	return false
}

// ProcessStreamJob handles streaming jobs using ffmpeg to create HLS streams
func ProcessStreamJob(job *types.DownloadJob, updateStatus func(string, string, string), cleanup func(string)) {
	// Check if job was cancelled before starting
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled before processing")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}
	logrus.WithField("downloadId", job.ID).Info("Processing stream job with ffmpeg")

	// Check and update yt-dlp before streaming
	if err := CheckAndUpdateYtDlp(context.Background()); err != nil {
		logrus.WithError(err).Error("Failed to check/update yt-dlp")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Erreur lors de la vérification/mise à jour de yt-dlp",
		})
		updateStatus(job.ID, "error", "Erreur lors de la vérification/mise à jour de yt-dlp")
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	// Notify subscribers that streaming is starting
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
		Type:       "queued",
		DownloadID: job.ID,
		Message:    "Streaming en file d'attente",
	})

	// Get video and audio URLs using yt-dlp
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
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
		// Check for cancellation before each attempt
		if job.CancelContext != nil {
			select {
			case <-job.CancelContext.Done():
				logrus.WithField("downloadId", job.ID).Info("Stream job cancelled during format extraction")
				updateStatus(job.ID, "cancelled", "Annulé")
				websocket.BroadcastQueueStatus()
				if cleanup != nil {
					cleanup(job.ID)
				}
				return
			default:
			}
		}

		// Use job's cancel context if available, otherwise context.TODO()
		ctx := context.TODO()
		if job.CancelContext != nil {
			ctx = job.CancelContext
		}

		output, err = dl.Run(ctx, job.URL)
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
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur extraction URL vidéo: %v", err),
		})
		updateStatus(job.ID, "error", fmt.Sprintf("Erreur extraction URL vidéo: %v", err))
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	// Parse the video output: title, then video URL
	lines := strings.Split(strings.TrimSpace(output.Stdout), "\n")
	logrus.WithFields(logrus.Fields{
		"urlCount": len(lines),
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
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    "Nombre inattendu de lignes vidéo de yt-dlp",
		})
		updateStatus(job.ID, "error", "Nombre inattendu de lignes vidéo de yt-dlp")
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	// Get audio URL
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
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
		// Check if the error is due to cancellation
		if job.CancelContext != nil && job.CancelContext.Err() == context.Canceled {
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled during audio extraction")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		}

		logrus.WithError(err).Error("yt-dlp audio URL extraction failed")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur extraction URL audio: %v", err),
		})
		updateStatus(job.ID, "error", fmt.Sprintf("Erreur extraction URL audio: %v", err))
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	audioURL := strings.TrimSpace(outputAudio.Stdout)
	logrus.Info("yt-dlp audio URL extracted successfully")

	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
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
	}).Info("Title and URLs extracted successfully")

	// Check for cancellation before creating segments directory
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled before segments directory creation")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}

	// Create segments directory for this video
	segmentSubDir := filepath.Join(config.VideoDir, "segments", job.ID)
	if err := os.MkdirAll(segmentSubDir, 0755); err != nil {
		logrus.WithError(err).Error("Failed to create segments directory")
		websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
			Type:       "error",
			DownloadID: job.ID,
			Message:    fmt.Sprintf("Erreur création dossier segments: %v", err),
		})
		updateStatus(job.ID, "error", fmt.Sprintf("Erreur création dossier segments: %v", err))
		websocket.BroadcastQueueStatus()
		// Schedule cleanup after error
		if cleanup != nil {
			cleanup(job.ID)
		}
		return
	}

	// Generate HLS stream name with full path
	streamName := filepath.Join(config.VideoDir, fmt.Sprintf("%s.m3u8", sanitizedTitle))
	segmentPattern := filepath.Join(config.VideoDir, "segments", job.ID, "segment_%03d.ts")

	// Check for cancellation before starting HLS conversion
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled before HLS conversion")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}

	// Start HLS conversion with ffmpeg
	websocket.BroadcastToSubscribers(job.ID, types.WSMessage{
		Type:       "progress",
		DownloadID: job.ID,
		Message:    "Conversion HLS en cours...",
	})

	updateStatus(job.ID, "streaming", "Conversion HLS avec ffmpeg...")
	websocket.BroadcastQueueStatus()

	// Check for cancellation before starting auto-play setup
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled before ffmpeg start")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}

	// Auto-play after HLS stream is ready (for HLS streaming)
	if job.AutoPlay && job.VLCUrl != "" && job.BackendUrl != "" {
		streamFilename := filepath.Base(streamName)
		go func(streamFilename, vlcUrl, backendUrl string) {
			waitForM3U8AndPlay(streamName, streamFilename, vlcUrl, backendUrl)
		}(streamFilename, job.VLCUrl, job.BackendUrl)
	}

	// Final cancellation check before starting ffmpeg
	if job.CancelContext != nil {
		select {
		case <-job.CancelContext.Done():
			logrus.WithField("downloadId", job.ID).Info("Stream job cancelled immediately before ffmpeg")
			updateStatus(job.ID, "cancelled", "Annulé")
			websocket.BroadcastQueueStatus()
			if cleanup != nil {
				cleanup(job.ID)
			}
			return
		default:
		}
	}

	// Run ffmpeg conversion in a goroutine with progress monitoring
	go func(jobID, videoURL, audioURL, streamName, segmentPattern string, cancelCtx context.Context) {
		// Create a pipe to capture ffmpeg stderr (progress output)
		stderrPipe := &bytes.Buffer{}
		stderrReader, stderrWriter := io.Pipe()

		// Start goroutine to parse progress from stderr
		progressDone := make(chan bool)
		go parseFFmpegProgress(stderrReader, jobID, progressDone, cancelCtx)

		// Build ffmpeg stream with ffmpeg-go
		videoInput := ffmpeg_go.Input(videoURL)
		audioInput := ffmpeg_go.Input(audioURL)

		videoStream := videoInput.Video()
		audioStream := audioInput.Audio()

		outputStream := ffmpeg_go.OutputContext(cancelCtx, []*ffmpeg_go.Stream{videoStream, audioStream}, streamName,
			ffmpeg_go.KwArgs{
				"c:v":                  "copy",
				"c:a":                  "copy",
				"f":                    "hls",
				"hls_time":             "6",
				"hls_list_size":        "0",
				"hls_segment_filename": segmentPattern,
				"hls_base_url":         fmt.Sprintf("segments/%s/", jobID),
				"start_number":         "0",
				"hls_flags":            "independent_segments",
			}).
			WithErrorOutput(io.MultiWriter(stderrPipe, stderrWriter)).
			OverWriteOutput()

		err := outputStream.Run(ffmpeg_go.SeparateProcessGroup())

		// Close stderr writer to signal completion
		stderrWriter.Close()
		<-progressDone // Wait for progress parser to finish

		if err != nil {
			// Check if the error is due to cancellation
			if cancelCtx != nil && cancelCtx.Err() == context.Canceled {
				logrus.WithField("downloadId", jobID).Info("FFmpeg conversion cancelled")
				websocket.BroadcastToSubscribers(jobID, types.WSMessage{
					Type:       "progress",
					DownloadID: jobID,
					Message:    "Conversion annulée",
				})
				updateStatus(jobID, "cancelled", "Annulé")
				websocket.BroadcastQueueStatus()
				if cleanup != nil {
					cleanup(jobID)
				}
				return
			}

			logrus.WithError(err).WithField("downloadId", jobID).Error("ffmpeg HLS conversion failed")
			websocket.BroadcastToSubscribers(jobID, types.WSMessage{
				Type:       "error",
				DownloadID: jobID,
				Message:    fmt.Sprintf("Erreur ffmpeg: %v", err),
			})
			updateStatus(jobID, "error", fmt.Sprintf("Erreur ffmpeg: %v", err))
			websocket.BroadcastQueueStatus()
			// Schedule cleanup after error
			if cleanup != nil {
				cleanup(jobID)
			}
			return
		}

		logrus.WithField("downloadId", jobID).Info("ffmpeg HLS conversion completed successfully")

		// Update job status to completed
		updateStatus(jobID, "completed", "Streaming HLS terminé")
		websocket.BroadcastQueueStatus()

		// Notify completion
		websocket.BroadcastToSubscribers(jobID, types.WSMessage{
			Type:       "done",
			DownloadID: jobID,
			File:       filepath.Base(streamName),
			Message:    "Streaming HLS terminé",
		})

		PruneVideos()

		// Schedule cleanup of this job from queue after 5 seconds
		if cleanup != nil {
			cleanup(jobID)
		}
	}(job.ID, videoURL, audioURL, streamName, segmentPattern, job.CancelContext)

	// Continue to next job immediately - ffmpeg is running in background
}

// parseFFmpegProgress parses ffmpeg stderr output for progress information
func parseFFmpegProgress(reader io.Reader, jobID string, done chan bool, cancelCtx context.Context) {
	defer func() { done <- true }()

	scanner := bufio.NewScanner(reader)
	// Regex to extract time, speed, and other info from ffmpeg output
	// Example: frame= 1234 fps= 30 q=-1.0 size=   12345kB time=00:01:23.45 bitrate=1234.5kbits/s speed=1.23x
	timeRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)
	speedRegex := regexp.MustCompile(`speed=\s*(\d+\.?\d*)x`)

	lastBroadcast := time.Now()

	for scanner.Scan() {
		// Check for cancellation
		if cancelCtx != nil {
			select {
			case <-cancelCtx.Done():
				logrus.WithField("downloadId", jobID).Info("FFmpeg progress parsing cancelled")
				return
			default:
			}
		}

		line := scanner.Text()

		// Only broadcast every 2 seconds to avoid overwhelming the clients
		if time.Since(lastBroadcast) < 2*time.Second {
			continue
		}

		// Try to extract time and speed
		var progressMsg string

		if timeMatch := timeRegex.FindStringSubmatch(line); len(timeMatch) > 0 {
			hours := timeMatch[1]
			minutes := timeMatch[2]
			seconds := timeMatch[3]
			progressMsg = fmt.Sprintf("Progression: %s:%s:%s", hours, minutes, seconds)

			// Add speed if available
			if speedMatch := speedRegex.FindStringSubmatch(line); len(speedMatch) > 1 {
				speed := speedMatch[1]
				progressMsg += fmt.Sprintf(" (vitesse: %sx)", speed)
			}

			// Broadcast progress update
			websocket.BroadcastToSubscribers(jobID, types.WSMessage{
				Type:       "progress",
				DownloadID: jobID,
				Message:    progressMsg,
			})

			lastBroadcast = time.Now()

			logrus.WithFields(logrus.Fields{
				"downloadId": jobID,
				"progress":   progressMsg,
			}).Debug("ffmpeg progress update")
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Warn("Error reading ffmpeg progress")
	}
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}
