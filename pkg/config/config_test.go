package config

import (
	"testing"

	"video-server/internal/types"
)

func TestGetJobStatuses(t *testing.T) {
	// Clear any existing statuses
	jobStatusesMutex.Lock()
	jobStatuses = make(map[string]*types.JobStatus)
	jobStatusesMutex.Unlock()

	// Test empty statuses
	statuses := GetJobStatuses()
	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses, got %d", len(statuses))
	}

	// Add a status
	testStatus := &types.JobStatus{
		Status:   "queued",
		Progress: "Test",
	}
	SetJobStatus("test-1", testStatus)

	// Verify it was added
	statuses = GetJobStatuses()
	if len(statuses) != 1 {
		t.Errorf("Expected 1 status, got %d", len(statuses))
	}

	if statuses["test-1"].Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", statuses["test-1"].Status)
	}
}

func TestSetJobStatus(t *testing.T) {
	// Clear any existing statuses
	jobStatusesMutex.Lock()
	jobStatuses = make(map[string]*types.JobStatus)
	jobStatusesMutex.Unlock()

	testStatus := &types.JobStatus{
		Status:   "downloading",
		Progress: "50%",
	}

	SetJobStatus("test-2", testStatus)

	// Verify it was set
	statuses := GetJobStatuses()
	if statuses["test-2"].Status != "downloading" {
		t.Errorf("Expected status 'downloading', got '%s'", statuses["test-2"].Status)
	}
}

func TestDeleteJobStatus(t *testing.T) {
	// Clear and add a status
	jobStatusesMutex.Lock()
	jobStatuses = make(map[string]*types.JobStatus)
	jobStatuses["test-3"] = &types.JobStatus{Status: "completed"}
	jobStatusesMutex.Unlock()

	// Delete it
	DeleteJobStatus("test-3")

	// Verify it was deleted
	statuses := GetJobStatuses()
	if _, exists := statuses["test-3"]; exists {
		t.Error("Status should have been deleted")
	}
}

func TestWSClientManagement(t *testing.T) {
	// Clear any existing clients
	wsClientsMutex.Lock()
	wsClients = make(map[*types.WSClient]bool)
	wsClientsMutex.Unlock()

	// Create a test client
	client := &types.WSClient{}

	// Add client
	AddWSClient(client)

	// Verify it was added
	clients := GetWSClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}

	// Remove client
	RemoveWSClient(client)

	// Verify it was removed
	clients = GetWSClients()
	if len(clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(clients))
	}
}

func TestGetJobStatusesMutex(t *testing.T) {
	mutex := GetJobStatusesMutex()
	if mutex == nil {
		t.Error("Mutex should not be nil")
	}
}

func TestGetWSMutex(t *testing.T) {
	mutex := GetWSMutex()
	if mutex == nil {
		t.Error("Mutex should not be nil")
	}
}

func TestConstants(t *testing.T) {
	if VideoDir != "/videos" {
		t.Errorf("Expected VideoDir '/videos', got '%s'", VideoDir)
	}

	if CookieDir != "/videos/cookie" {
		t.Errorf("Expected CookieDir '/videos/cookie', got '%s'", CookieDir)
	}

	if CookieFile != "/videos/cookie/cookie.json" {
		t.Errorf("Expected CookieFile '/videos/cookie/cookie.json', got '%s'", CookieFile)
	}
}
