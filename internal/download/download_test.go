package download

import (
	"testing"
	"time"
)

func TestGenerateDownloadID(t *testing.T) {
	id1 := GenerateDownloadID()
	time.Sleep(1 * time.Millisecond)
	id2 := GenerateDownloadID()

	if id1 == "" {
		t.Error("Download ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Download IDs should be unique")
	}

	// Check format
	if len(id1) < 10 {
		t.Errorf("Download ID seems too short: %s", id1)
	}
}

func TestPruneVideos(t *testing.T) {
	// This is a basic test that just ensures the function doesn't panic
	// In a real test, you would create a temporary directory with test files
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PruneVideos panicked: %v", r)
		}
	}()

	PruneVideos()
}

func TestVerifyVideoAccessible(t *testing.T) {
	// Test with an invalid URL - should return false
	result := verifyVideoAccessible("http://invalid-url-that-does-not-exist.local/video.mp4", 1)
	if result {
		t.Error("Expected verifyVideoAccessible to return false for invalid URL")
	}
}

func TestAutoPlayVideo(t *testing.T) {
	// Test with empty parameters - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AutoPlayVideo panicked: %v", r)
		}
	}()

	AutoPlayVideo("", "", "")
	AutoPlayVideo("test.mp4", "", "")
}

func TestPlayDirectURL(t *testing.T) {
	// Test with empty VLC URL - should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlayDirectURL panicked: %v", r)
		}
	}()

	PlayDirectURL("http://example.com/video.mp4", "")
}
