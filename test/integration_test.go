package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

// TestFrontendIntegration tests the frontend is properly served
func TestFrontendIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	t.Run("MainPage", func(t *testing.T) {
		resp, err := http.Get(baseURL)
		if err != nil {
			t.Fatalf("Failed to get main page: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		content := string(body)

		if !strings.Contains(content, "Téléchargeur de Vidéos") {
			t.Error("Main page missing title")
		}

		if !strings.Contains(content, "styles.css") {
			t.Error("Main page missing CSS reference")
		}

		if !strings.Contains(content, "app.js") && !strings.Contains(content, "js/app.js") {
			t.Error("Main page missing JS reference")
		}
	})

	t.Run("CSSFile", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/styles.css")
		if err != nil {
			t.Fatalf("Failed to get CSS: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/css") {
			t.Errorf("Expected CSS content type, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		content := string(body)

		if !strings.Contains(content, "--primary-color") {
			t.Error("CSS missing CSS variables")
		}

		if !strings.Contains(content, ".btn") {
			t.Error("CSS missing button styles")
		}
	})

	t.Run("JavaScriptFile", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/app.js")
		if err != nil {
			t.Fatalf("Failed to get JS: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "javascript") {
			t.Errorf("Expected JavaScript content type, got %s", contentType)
		}
	})

	t.Run("ListEndpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/list")
		if err != nil {
			t.Fatalf("Failed to get list: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var files []string
		if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}
	})

	t.Run("VideoServer", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/videos/")
		if err != nil {
			t.Fatalf("Failed to get videos: %v", err)
		}
		defer resp.Body.Close()

		// 200, 403, or 404 are all acceptable (depends on directory listing settings)
		if resp.StatusCode != http.StatusOK && 
		   resp.StatusCode != http.StatusForbidden && 
		   resp.StatusCode != http.StatusNotFound {
			t.Errorf("Unexpected status for videos endpoint: %d", resp.StatusCode)
		}
	})
}

// TestVLCAuthIntegration tests VLC authentication flow
func TestVLCAuthIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a running VLC server
	vlcURL := "http://192.168.4.29:8080" // Change to your VLC URL
	
	t.Run("GetVLCCode", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/vlc/code?vlc=" + vlcURL)
		if err != nil {
			t.Skipf("VLC server not available: %v", err)
			return
		}
		defer resp.Body.Close()

		// If VLC is not available, skip the test
		if resp.StatusCode == http.StatusInternalServerError || 
		   resp.StatusCode == http.StatusBadGateway {
			t.Skip("VLC server not available for testing")
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("VLCStatus", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/vlc/status?vlc=" + vlcURL)
		if err != nil {
			t.Fatalf("Failed to get VLC status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var status map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Errorf("Failed to decode status response: %v", err)
		}

		if _, ok := status["authenticated"]; !ok {
			t.Error("Status response missing 'authenticated' field")
		}
	})
}

// TestAPIEndpoints tests various API endpoints
func TestAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("QueueStatus", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/queue")
		if err != nil {
			t.Fatalf("Failed to get queue status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var queue map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&queue); err != nil {
			t.Errorf("Failed to decode queue response: %v", err)
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/url")
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}
