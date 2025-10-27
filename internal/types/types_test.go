package types

import (
	"testing"
	"time"
)

func TestURLRequest(t *testing.T) {
	req := URLRequest{
		URL:        "https://youtube.com/watch?v=test",
		AutoPlay:   true,
		VLCUrl:     "http://localhost:8080",
		BackendUrl: "http://localhost:8080",
		Mode:       "stream",
	}

	if req.URL == "" {
		t.Error("URL should not be empty")
	}

	if req.Mode != "stream" {
		t.Errorf("Expected mode 'stream', got '%s'", req.Mode)
	}
}

func TestResponse(t *testing.T) {
	resp := Response{
		Success: true,
		Message: "Test message",
		File:    "test.mp4",
	}

	if !resp.Success {
		t.Error("Success should be true")
	}

	if resp.File != "test.mp4" {
		t.Errorf("Expected file 'test.mp4', got '%s'", resp.File)
	}
}

func TestJobStatus(t *testing.T) {
	now := time.Now()
	job := &DownloadJob{
		ID:             "test-123",
		URL:            "https://youtube.com/watch?v=test",
		OutputTemplate: "/videos/%(title)s.%(ext)s",
		AutoPlay:       false,
		Mode:           "stream",
		CreatedAt:      now,
	}

	status := &JobStatus{
		Job:      job,
		Status:   "queued",
		Progress: "En attente",
	}

	if status.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", status.Status)
	}

	if status.Job.ID != "test-123" {
		t.Errorf("Expected job ID 'test-123', got '%s'", status.Job.ID)
	}
}

func TestWSMessage(t *testing.T) {
	msg := WSMessage{
		Type:       "progress",
		DownloadID: "test-123",
		Message:    "Downloading...",
		Percent:    50.5,
	}

	if msg.Type != "progress" {
		t.Errorf("Expected type 'progress', got '%s'", msg.Type)
	}

	if msg.Percent != 50.5 {
		t.Errorf("Expected percent 50.5, got %f", msg.Percent)
	}
}

func TestVLCSession(t *testing.T) {
	session := &VLCSession{
		URL:           "http://vlc:8080",
		Authenticated: true,
		LastActivity:  time.Now(),
	}

	if !session.Authenticated {
		t.Error("Session should be authenticated")
	}

	if session.URL != "http://vlc:8080" {
		t.Errorf("Expected URL 'http://vlc:8080', got '%s'", session.URL)
	}
}
