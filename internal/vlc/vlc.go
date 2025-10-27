package vlc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"video-server/internal/types"
)

// Constants for VLC management
const (
	cookieDir  = "/videos/cookie"
	cookieFile = "/videos/cookie/cookie.json"
)

// Global VLC sessions map
var vlcSessions = make(map[string]*types.VLCSession)
var vlcSessionsMutex sync.RWMutex

// saveCookieToFile saves VLC session cookies to persistent storage
func saveCookieToFile(vlcURL string, session *types.VLCSession) error {
	// Create cookie directory if it doesn't exist
	if err := os.MkdirAll(cookieDir, 0755); err != nil {
		return fmt.Errorf("failed to create cookie directory: %w", err)
	}

	// Convert http.Cookie to VLCCookie for JSON serialization
	cookies := make([]types.VLCCookie, 0, len(session.Cookies))
	for _, cookie := range session.Cookies {
		cookies = append(cookies, types.VLCCookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Path:   cookie.Path,
			Domain: cookie.Domain,
		})
	}

	config := types.VLCConfig{
		URL:           vlcURL,
		Authenticated: session.Authenticated,
		LastActivity:  session.LastActivity.Format(time.RFC3339),
		Cookies:       cookies,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookie config: %w", err)
	}

	if err := os.WriteFile(cookieFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cookie file: %w", err)
	}

	logrus.WithField("vlc_url", vlcURL).Info("VLC cookies saved to file")
	return nil
}

// loadCookieFromFile loads VLC session cookies from persistent storage
func loadCookieFromFile() (*types.VLCConfig, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No saved cookies
		}
		return nil, fmt.Errorf("failed to read cookie file: %w", err)
	}

	var config types.VLCConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cookie config: %w", err)
	}

	return &config, nil
}

// verifyVLCSession checks if a VLC session is still valid by checking /wsticket
func verifyVLCSession(vlcURL string) error {
	vlcSessionsMutex.RLock()
	session, exists := vlcSessions[vlcURL]
	vlcSessionsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Try to access /wsticket endpoint to verify authentication
	resp, err := session.Client.Get(vlcURL + "/wsticket")
	if err != nil {
		return fmt.Errorf("failed to connect to VLC: %w", err)
	}
	defer resp.Body.Close()

	// 401 means the cookie is invalid/expired
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session is no longer authenticated")
	}

	// 200 means the cookie is valid
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// restoreVLCSession restores a VLC session from saved cookies
func restoreVLCSession(config *types.VLCConfig) {
	if config == nil || !config.Authenticated {
		return
	}

	// Convert VLCCookie back to http.Cookie
	cookies := make([]*http.Cookie, 0, len(config.Cookies))
	for _, c := range config.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Path:   c.Path,
			Domain: c.Domain,
		})
	}

	// Create cookie jar and add cookies
	jar, _ := cookiejar.New(nil)
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		logrus.WithError(err).Warn("Failed to parse saved VLC URL")
		return
	}
	jar.SetCookies(parsedURL, cookies)

	// Create HTTP client with cookie jar
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	lastActivity, _ := time.Parse(time.RFC3339, config.LastActivity)

	session := &types.VLCSession{
		Client:        client,
		URL:           config.URL,
		Authenticated: config.Authenticated,
		LastActivity:  lastActivity,
		Cookies:       cookies,
	}

	vlcSessionsMutex.Lock()
	vlcSessions[config.URL] = session
	vlcSessionsMutex.Unlock()

	logrus.WithField("vlc_url", config.URL).Info("VLC session restored from saved cookies")

	// Verify the session is still valid using /wsticket endpoint
	go func() {
		if err := verifyVLCSession(config.URL); err != nil {
			logrus.WithError(err).Warn("Saved VLC session is no longer valid")
			// Remove invalid session
			vlcSessionsMutex.Lock()
			delete(vlcSessions, config.URL)
			vlcSessionsMutex.Unlock()
			// Delete cookie file
			os.Remove(cookieFile)
		} else {
			logrus.WithField("vlc_url", config.URL).Info("Saved VLC session is still valid")
		}
	}()
}

// GetVLCSessions returns the global VLC sessions map (thread-safe)
func GetVLCSessions() map[string]*types.VLCSession {
	vlcSessionsMutex.RLock()
	defer vlcSessionsMutex.RUnlock()

	// Return a copy to prevent external modifications
	sessions := make(map[string]*types.VLCSession)
	for k, v := range vlcSessions {
		sessions[k] = v
	}
	return sessions
}

// SetVLCSession sets a VLC session in the global map (thread-safe)
func SetVLCSession(url string, session *types.VLCSession) {
	vlcSessionsMutex.Lock()
	vlcSessions[url] = session
	vlcSessionsMutex.Unlock()
}

// DeleteVLCSession removes a VLC session from the global map (thread-safe)
func DeleteVLCSession(url string) {
	vlcSessionsMutex.Lock()
	delete(vlcSessions, url)
	vlcSessionsMutex.Unlock()
}

// InitializeVLCSessions loads saved VLC sessions on startup
func InitializeVLCSessions() {
	if config, err := loadCookieFromFile(); err != nil {
		logrus.WithError(err).Warn("Failed to load saved VLC cookies")
	} else if config != nil {
		restoreVLCSession(config)
	}
}

// SaveVLCSession saves a VLC session's cookies to disk
func SaveVLCSession(vlcURL string, session *types.VLCSession) error {
	return saveCookieToFile(vlcURL, session)
}
