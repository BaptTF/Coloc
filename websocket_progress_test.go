package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const (
	testBackendURL = "http://localhost:8080"
	testWSURL      = "ws://localhost:8080/ws"
	testVideoURL   = "https://youtu.be/xR-40NwDI7U?si=EiBsbE15pOIyOuKf"
)

func TestWebSocketProgressUpdates(t *testing.T) {
	t.Log("Starting WebSocket progress test...")

	// Connect to WebSocket
	u, err := url.Parse(testWSURL)
	if err != nil {
		t.Fatalf("Failed to parse WebSocket URL: %v", err)
	}

	t.Logf("Connecting to WebSocket: %s", testWSURL)
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	t.Log("✅ WebSocket connected!")

	// Subscribe to all downloads
	subscribeMsg := map[string]string{
		"action": "subscribeAll",
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		t.Fatalf("Failed to send subscribe message: %v", err)
	}
	t.Log("📡 Subscribed to all downloads")

	// Trigger download
	downloadID := triggerDownload(t)
	if downloadID == "" {
		t.Fatal("Failed to trigger download")
	}
	t.Logf("✅ Download triggered: %s", downloadID)

	// Monitor WebSocket messages
	progressUpdates := []WSMessage{}
	messageCount := 0
	downloadComplete := false
	timeout := time.After(60 * time.Second) // 60 second timeout

	t.Log("⏳ Monitoring WebSocket for progress updates...")

	for !downloadComplete {
		select {
		case <-timeout:
			t.Fatalf("Test timeout after 60 seconds. Received %d messages, %d progress updates",
				messageCount, len(progressUpdates))

		default:
			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			var msg WSMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Check if it's a timeout
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					continue
				}
				t.Logf("⚠️  Error reading message: %v", err)
				continue
			}

			messageCount++
			t.Logf("\n[Message #%d] Type: %s", messageCount, msg.Type)

			switch msg.Type {
			case "queueStatus":
				if msg.Queue != nil {
					t.Logf("  Queue size: %d", len(msg.Queue))
					for _, job := range msg.Queue {
						if job.Job != nil {
							t.Logf("  - Job %s: %s", job.Job.ID, job.Status)
							t.Logf("    Progress: %s", job.Progress)
						}
					}
				}

			case "progress":
				progressUpdates = append(progressUpdates, msg)
				t.Logf("  📥 Download ID: %s", msg.DownloadID)
				t.Logf("  📊 Progress: %.2f%%", msg.Percent)
				t.Logf("  💬 Message: %s", msg.Message)

			case "done":
				t.Logf("  ✅ Download ID: %s", msg.DownloadID)
				t.Logf("  📁 File: %s", msg.File)
				t.Logf("  💬 Message: %s", msg.Message)
				downloadComplete = true

			case "error":
				t.Logf("  ❌ Download ID: %s", msg.DownloadID)
				t.Logf("  💬 Error: %s", msg.Message)
				t.Fatalf("Download failed with error: %s", msg.Message)

			case "list":
				if msg.Videos != nil {
					t.Logf("  📂 Videos available: %d", len(msg.Videos))
				}

			default:
				data, _ := json.MarshalIndent(msg, "  ", "  ")
				t.Logf("  Raw data: %s", string(data))
			}
		}
	}

	// Print summary
	t.Log("\n" + strings.Repeat("=", 60))
	t.Log("DOWNLOAD COMPLETE!")
	t.Logf("Total messages received: %d", messageCount)
	t.Logf("Progress updates: %d", len(progressUpdates))

	if len(progressUpdates) > 0 {
		t.Logf("First update: %.2f%% - %s", progressUpdates[0].Percent, progressUpdates[0].Message)
		t.Logf("Last update: %.2f%% - %s", progressUpdates[len(progressUpdates)-1].Percent, progressUpdates[len(progressUpdates)-1].Message)
	}
	t.Log(strings.Repeat("=", 60))

	// Verify we got progress updates
	if len(progressUpdates) == 0 {
		t.Error("❌ No progress updates received!")
	} else {
		t.Logf("✅ Received %d progress updates", len(progressUpdates))
	}

	// Verify progress messages contain expected information
	hasDetailedProgress := false
	for _, update := range progressUpdates {
		if strings.Contains(update.Message, "MiB") || strings.Contains(update.Message, "ETA") {
			hasDetailedProgress = true
			break
		}
	}

	if !hasDetailedProgress {
		t.Error("❌ Progress updates don't contain detailed information (speed, size, ETA)")
	} else {
		t.Log("✅ Progress updates contain detailed information")
	}
}

func triggerDownload(t *testing.T) string {
	t.Log("\n" + strings.Repeat("=", 60))
	t.Logf("Triggering download: %s", testVideoURL)
	t.Log(strings.Repeat("=", 60))

	requestBody := map[string]interface{}{
		"url":      testVideoURL,
		"mode":     "download",
		"autoPlay": false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
		return ""
	}

	resp, err := http.Post(
		testBackendURL+"/urlyt",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		t.Fatalf("Failed to trigger download: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Download request failed with status: %d", resp.StatusCode)
		return ""
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
		return ""
	}

	if !result.Success {
		t.Fatalf("Download request failed: %s", result.Message)
		return ""
	}

	t.Logf("✅ Download triggered successfully!")
	t.Logf("   Download ID: %s", result.File)
	t.Logf("   Message: %s", result.Message)

	return result.File
}

func TestWebSocketConnection(t *testing.T) {
	t.Log("Testing basic WebSocket connection...")

	u, err := url.Parse(testWSURL)
	if err != nil {
		t.Fatalf("Failed to parse WebSocket URL: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	t.Log("✅ WebSocket connection successful!")

	// Subscribe to all downloads
	subscribeMsg := map[string]string{
		"action": "subscribeAll",
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		t.Fatalf("Failed to send subscribe message: %v", err)
	}

	// Wait for initial queue status
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var msg WSMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read initial message: %v", err)
	}

	t.Logf("✅ Received initial message: type=%s", msg.Type)

	if msg.Type == "queueStatus" {
		t.Log("✅ Initial queue status received")
	}
}

func TestBackendAPI(t *testing.T) {
	t.Log("Testing backend API availability...")

	resp, err := http.Get(testBackendURL + "/list")
	if err != nil {
		t.Fatalf("Failed to connect to backend: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Backend returned status: %d", resp.StatusCode)
	}

	t.Log("✅ Backend API is available!")
}
