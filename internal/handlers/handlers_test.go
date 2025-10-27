package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"video-server/internal/types"
)

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	sendError(w, "Test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp types.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}

	if resp.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", resp.Message)
	}
}

func TestSendSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	sendSuccess(w, "Test success", "test.mp4")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp types.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	if resp.Message != "Test success" {
		t.Errorf("Expected message 'Test success', got '%s'", resp.Message)
	}

	if resp.File != "test.mp4" {
		t.Errorf("Expected file 'test.mp4', got '%s'", resp.File)
	}
}

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	HomeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
	}
}

func TestStylesHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/styles.css", nil)
	w := httptest.NewRecorder()

	StylesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/css; charset=utf-8', got '%s'", contentType)
	}
}

// TestAppHandler removed - app.js is now served via embedded filesystem at /static/js/app.js

func TestDownloadURLHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/url", nil)
	w := httptest.NewRecorder()

	DownloadURLHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestDownloadURLHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/url", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	DownloadURLHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDownloadURLHandler_MissingURL(t *testing.T) {
	reqBody := types.URLRequest{
		URL: "",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/url", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	DownloadURLHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
