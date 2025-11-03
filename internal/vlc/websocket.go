package vlc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"video-server/internal/state"
	"video-server/internal/types"
	ws "video-server/internal/websocket"
)

// VLCWebSocketClient manages a WebSocket connection to VLC Android
type VLCWebSocketClient struct {
	vlcURL       string
	session      *types.VLCSession
	conn         *websocket.Conn
	mu           sync.Mutex
	connected    bool
	authTicket   string
	ticketIssued time.Time
	reconnectCh  chan struct{}
	stopCh       chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

// VLCMessage represents a message from VLC WebSocket
type VLCMessage struct {
	Type               string              `json:"type"`
	Title              string              `json:"title,omitempty"`
	Artist             string              `json:"artist,omitempty"`
	Playing            bool                `json:"playing,omitempty"`
	IsVideoPlaying     bool                `json:"isVideoPlaying,omitempty"`
	Progress           int64               `json:"progress,omitempty"`
	Duration           int64               `json:"duration,omitempty"`
	ID                 int64               `json:"id,omitempty"`
	ArtworkURL         string              `json:"artworkURL,omitempty"`
	URI                string              `json:"uri,omitempty"`
	Volume             int                 `json:"volume,omitempty"`
	Speed              float64             `json:"speed,omitempty"`
	SleepTimer         int64               `json:"sleepTimer,omitempty"`
	WaitForMediaEnd    bool                `json:"waitForMediaEnd,omitempty"`
	ResetOnInteraction bool                `json:"resetOnInteraction,omitempty"`
	Shuffle            bool                `json:"shuffle,omitempty"`
	Repeat             int                 `json:"repeat,omitempty"`
	ShouldShow         bool                `json:"shouldShow,omitempty"`
	Bookmarks          []types.VLCBookmark `json:"bookmarks,omitempty"`
	Chapters           []types.VLCChapter  `json:"chapters,omitempty"`
	Medias             []types.VLCMedia    `json:"medias,omitempty"`
	RefreshNeeded      bool                `json:"refreshNeeded,omitempty"`
	Status             string              `json:"status,omitempty"`
	InitialMessage     string              `json:"initialMessage,omitempty"`
	DialogOpened       bool                `json:"dialogOpened,omitempty"`
	MediaTitle         string              `json:"mediaTitle,omitempty"`
	Consumed           bool                `json:"consumed,omitempty"`
	Path               string              `json:"path,omitempty"`
	Description        string              `json:"description,omitempty"`
	Forbidden          bool                `json:"forbidden,omitempty"`
	Text               string              `json:"text,omitempty"`
	// Legacy fields for backward compatibility
	Message     string   `json:"message,omitempty"`
	State       string   `json:"state,omitempty"`
	Time        int64    `json:"time,omitempty"`
	Length      int64    `json:"length,omitempty"`
	FloatValue  *float64 `json:"floatValue,omitempty"`
	LongValue   *int64   `json:"longValue,omitempty"`
	StringValue *string  `json:"stringValue,omitempty"`
}

// VLCCommand represents a command to send to VLC
type VLCCommand struct {
	Message     string   `json:"message"`
	ID          *int     `json:"id,omitempty"`
	FloatValue  *float64 `json:"floatValue,omitempty"`
	LongValue   *int64   `json:"longValue,omitempty"`
	StringValue *string  `json:"stringValue,omitempty"`
	AuthTicket  string   `json:"authTicket,omitempty"`
}

// Global WebSocket clients map
var vlcWebSocketClients = make(map[string]*VLCWebSocketClient)
var vlcWebSocketClientsMutex sync.RWMutex

// NewVLCWebSocketClient creates a new WebSocket client for VLC
func NewVLCWebSocketClient(vlcURL string, session *types.VLCSession) *VLCWebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &VLCWebSocketClient{
		vlcURL:      vlcURL,
		session:     session,
		reconnectCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// isTicketExpired checks if the current ticket is expired or about to expire
func (c *VLCWebSocketClient) isTicketExpired() bool {
	// Tickets expire after 60 seconds, renew after 50 seconds to be safe
	return time.Since(c.ticketIssued) > 50*time.Second
}

// renewTicket renews the authentication ticket
func (c *VLCWebSocketClient) renewTicket() error {
	logrus.WithField("vlc_url", c.vlcURL).Info("Renewing WebSocket ticket")

	ticket, err := c.RequestTicket()
	if err != nil {
		return fmt.Errorf("failed to renew ticket: %w", err)
	}

	c.authTicket = ticket
	c.ticketIssued = time.Now()

	logrus.WithFields(logrus.Fields{
		"vlc_url":       c.vlcURL,
		"ticket_length": len(ticket),
	}).Info("Successfully renewed WebSocket ticket")

	return nil
}

// startTicketRenewal periodically renews the authentication ticket
func (c *VLCWebSocketClient) startTicketRenewal() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.connected && c.isTicketExpired() {
				logrus.WithField("vlc_url", c.vlcURL).Info("Periodic ticket renewal triggered")
				if err := c.renewTicket(); err != nil {
					logrus.WithFields(logrus.Fields{
						"vlc_url": c.vlcURL,
						"error":   err,
					}).Error("Failed to renew ticket during periodic renewal")
				}
			}
			c.mu.Unlock()
		}
	}
}

// RequestTicket requests an authentication ticket from VLC
func (c *VLCWebSocketClient) RequestTicket() (string, error) {
	logrus.WithField("vlc_url", c.vlcURL).Info("Requesting WebSocket ticket from VLC")

	resp, err := c.session.Client.Get(c.vlcURL + "/wsticket")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": c.vlcURL,
			"error":   err,
		}).Error("Failed to request ticket from VLC")
		return "", fmt.Errorf("failed to request ticket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"vlc_url":     c.vlcURL,
			"status":      resp.StatusCode,
			"status_text": resp.Status,
		}).Error("VLC wsticket endpoint returned non-200 status, cannot get valid ticket")

		return "", fmt.Errorf("VLC wsticket endpoint returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var ticket string
	if _, err := fmt.Fscan(resp.Body, &ticket); err != nil {
		return "", fmt.Errorf("failed to read ticket: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":       c.vlcURL,
		"ticket_length": len(ticket),
	}).Info("Received WebSocket ticket from VLC")

	return ticket, nil
}

// Connect establishes a WebSocket connection to VLC
func (c *VLCWebSocketClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Request authentication ticket
	ticket, err := c.RequestTicket()
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}
	c.authTicket = ticket
	c.ticketIssued = time.Now()

	// Parse WebSocket URL
	wsURL, err := url.Parse(c.vlcURL)
	if err != nil {
		return fmt.Errorf("failed to parse VLC URL: %w", err)
	}

	// Convert HTTP to WebSocket URL
	scheme := "ws"
	if wsURL.Scheme == "https" {
		scheme = "wss"
	}
	wsURL.Scheme = scheme
	wsURL.Path = "/echo"

	logrus.WithFields(logrus.Fields{
		"vlc_url":  c.vlcURL,
		"ws_url":   wsURL.String(),
		"origin":   c.vlcURL,
		"protocol": "player",
	}).Info("Attempting WebSocket connection to VLC")

	// Connect to WebSocket with authenticated session cookies
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDial:          (&websocket.Dialer{}).NetDial,
		Jar:              c.session.Client.Jar,
	}

	// Create request headers to include cookies and protocol
	reqHeader := http.Header{}
	reqHeader.Set("Origin", c.vlcURL)
	reqHeader.Set("Sec-WebSocket-Protocol", "player")

	conn, _, err := dialer.Dial(wsURL.String(), reqHeader)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": c.vlcURL,
			"ws_url":  wsURL.String(),
			"error":   err,
		}).Error("WebSocket connection failed")
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url": c.vlcURL,
		"ws_url":  wsURL.String(),
	}).Info("WebSocket connection established successfully")

	c.conn = conn
	c.connected = true

	// Send authentication ticket as first message
	authMsg := VLCCommand{
		Message:    "hello",
		AuthTicket: c.authTicket,
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":    c.vlcURL,
		"message":    "hello",
		"authTicket": c.authTicket,
	}).Info("Sending authentication message to VLC WebSocket")

	if err := c.conn.WriteJSON(authMsg); err != nil {
		c.conn.Close()
		c.connected = false
		return fmt.Errorf("failed to send authentication: %w", err)
	}

	logrus.WithField("vlc_url", c.vlcURL).Info("Successfully connected and authenticated to VLC WebSocket")

	// Start message listener
	go c.listenForMessages()

	// Start periodic ticket renewal
	go c.startTicketRenewal()

	return nil
}

// listenForMessages handles incoming messages from VLC
func (c *VLCWebSocketClient) listenForMessages() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.connected = false
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		var msg VLCMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"vlc_url": c.vlcURL,
				"error":   err.Error(),
			}).Error("Failed to read message from VLC WebSocket")

			// Try to reconnect
			go c.tryReconnect()
			return
		}

		logrus.WithField("vlc_url", c.vlcURL).Debug("Successfully read message from VLC WebSocket")

		c.handleMessage(msg)
	}
}

// handleMessage processes incoming messages from VLC
func (c *VLCWebSocketClient) handleMessage(msg VLCMessage) {
	// Log the full raw message for debugging
	rawMsg, _ := json.Marshal(msg)
	logrus.WithFields(logrus.Fields{
		"vlc_url":  c.vlcURL,
		"msg_type": msg.Type,
		"title":    msg.Title,
		"artist":   msg.Artist,
		"playing":  msg.Playing,
		"progress": msg.Progress,
		"duration": msg.Duration,
		"volume":   msg.Volume,
		"raw_msg":  string(rawMsg),
	}).Info("Received message from VLC WebSocket")

	// Broadcast VLC status to frontend clients
	switch msg.Type {
	case "now-playing":
		// Comprehensive now playing information
		vlcStatus := &types.VLCStatus{
			Title:              msg.Title,
			Artist:             msg.Artist,
			Playing:            msg.Playing,
			IsVideoPlaying:     msg.IsVideoPlaying,
			Progress:           msg.Progress,
			Duration:           msg.Duration,
			ID:                 msg.ID,
			ArtworkURL:         msg.ArtworkURL,
			URI:                msg.URI,
			Volume:             msg.Volume,
			Speed:              msg.Speed,
			SleepTimer:         msg.SleepTimer,
			WaitForMediaEnd:    msg.WaitForMediaEnd,
			ResetOnInteraction: msg.ResetOnInteraction,
			Shuffle:            msg.Shuffle,
			Repeat:             msg.Repeat,
			ShouldShow:         msg.ShouldShow,
			Bookmarks:          msg.Bookmarks,
			Chapters:           msg.Chapters,
		}

		wsMsg := types.WSMessage{
			Type:      "vlc_now_playing",
			VLCStatus: vlcStatus,
		}
		ws.BroadcastToAll(wsMsg)

		// Persist VLC state in global state
		state.SetVLCState(vlcStatus, nil, nil)

	case "player-status":
		// Simple playing/paused status
		vlcStatus := &types.VLCStatus{
			Playing: msg.Playing,
		}

		wsMsg := types.WSMessage{
			Type:      "vlc_player_status",
			VLCStatus: vlcStatus,
		}
		ws.BroadcastToAll(wsMsg)

	case "play-queue":
		// Play queue with media items
		vlcQueue := &types.VLCQueue{
			Medias: msg.Medias,
		}

		wsMsg := types.WSMessage{
			Type:     "vlc_play_queue",
			VLCQueue: vlcQueue,
		}
		ws.BroadcastToAll(wsMsg)

		// Persist VLC queue state (preserve existing status)
		currentStatus, _, currentVolume, _ := state.GetVLCState()
		state.SetVLCState(currentStatus, vlcQueue, currentVolume)

	case "ml-refresh-needed":
		// Media library refresh request
		wsMsg := types.WSMessage{
			Type:    "vlc_ml_refresh_needed",
			Message: "Media library refresh requested",
		}
		ws.BroadcastToAll(wsMsg)

	case "auth":
		// Authentication status
		vlcAuth := &types.VLCAuth{
			Status:         msg.Status,
			InitialMessage: msg.InitialMessage,
		}

		logrus.WithFields(logrus.Fields{
			"vlc_url":         c.vlcURL,
			"status":          msg.Status,
			"initial_message": msg.InitialMessage,
		}).Debug("VLC Authentication response received")

		// Handle different authentication statuses
		switch msg.Status {
		case "ok":
			logrus.WithField("vlc_url", c.vlcURL).Info("VLC WebSocket authentication successful")
		case "forbidden":
			// Check if this is a command response (not initial auth)
			if msg.InitialMessage != "" && msg.InitialMessage != "null" {
				logrus.WithFields(logrus.Fields{
					"vlc_url": c.vlcURL,
					"command": msg.InitialMessage,
				}).Debug("VLC command forbidden (likely no media playing)")

				// Don't reconnect for command forbidden responses
				// This is normal behavior when no media is playing
			} else {
				logrus.WithField("vlc_url", c.vlcURL).Warn("VLC WebSocket authentication forbidden, attempting re-authentication")
				go c.tryReconnect()
			}
		default:
			logrus.WithFields(logrus.Fields{
				"vlc_url": c.vlcURL,
				"status":  msg.Status,
			}).Warn("Unknown VLC authentication status")
		}

		// Only broadcast auth status changes, not command responses
		if msg.InitialMessage == "" || msg.InitialMessage == "null" {
			wsMsg := types.WSMessage{
				Type:    "vlc_auth",
				VLCAuth: vlcAuth,
				Message: fmt.Sprintf("VLC Auth: %s", msg.Status),
			}
			ws.BroadcastToAll(wsMsg)
		}

	case "volume":
		// Volume update
		vlcVolume := &types.VLCVolume{
			Volume: msg.Volume,
		}

		wsMsg := types.WSMessage{
			Type:      "vlc_volume_update",
			VLCVolume: vlcVolume,
			Message:   fmt.Sprintf("Volume: %d%%", msg.Volume),
		}
		ws.BroadcastToAll(wsMsg)

		// Persist VLC volume state (preserve existing status)
		currentStatus, currentQueue, _, _ := state.GetVLCState()
		state.SetVLCState(currentStatus, currentQueue, vlcVolume)

	case "login-needed":
		// Login required
		wsMsg := types.WSMessage{
			Type:    "vlc_login_needed",
			Message: "VLC login required",
		}
		ws.BroadcastToAll(wsMsg)

	case "resume-confirmation":
		// Resume playback confirmation
		wsMsg := types.WSMessage{
			Type:    "vlc_resume_confirmation",
			Message: fmt.Sprintf("Resume confirmation: %s", msg.MediaTitle),
		}
		ws.BroadcastToAll(wsMsg)

	case "browser-description":
		// Browser description
		wsMsg := types.WSMessage{
			Type:    "vlc_browser_description",
			Message: fmt.Sprintf("Browser: %s - %s", msg.Path, msg.Description),
		}
		ws.BroadcastToAll(wsMsg)

	case "playback-control-forbidden":
		// Playback control is forbidden
		wsMsg := types.WSMessage{
			Type:    "vlc_playback_control_forbidden",
			Message: "Playback control forbidden",
		}
		ws.BroadcastToAll(wsMsg)

	case "error":
		// Error message from VLC
		vlcError := &types.VLCError{
			Text: msg.Text,
		}

		wsMsg := types.WSMessage{
			Type:     "vlc_error",
			VLCError: vlcError,
			Message:  fmt.Sprintf("VLC Error: %s", msg.Text),
		}
		ws.BroadcastToAll(wsMsg)

	case "network-shares":
		// Network shares discovered
		wsMsg := types.WSMessage{
			Type:    "vlc_network_shares",
			Message: fmt.Sprintf("Network shares: %d items", len(msg.Medias)),
		}
		ws.BroadcastToAll(wsMsg)

	// Legacy message types for backward compatibility
	case "status", "state", "time", "title":
		// Legacy playback status updates
		vlcStatus := &types.VLCStatus{
			Title:    msg.Title,
			Playing:  msg.State == "playing" || msg.Playing,
			Progress: msg.Time,
			Duration: msg.Length,
			Volume:   msg.Volume,
		}

		wsMsg := types.WSMessage{
			Type:      "vlc_status_legacy",
			VLCStatus: vlcStatus,
			Message:   fmt.Sprintf("VLC %s", msg.State),
		}
		ws.BroadcastToAll(wsMsg)

	default:
		logrus.WithFields(logrus.Fields{
			"vlc_url":  c.vlcURL,
			"msg_type": msg.Type,
		}).Warn("Unhandled VLC message type")

		// Send unhandled message notification for debugging
		wsMsg := types.WSMessage{
			Type:    "vlc_unhandled_message",
			Message: fmt.Sprintf("Unhandled VLC message type: %s", msg.Type),
		}
		ws.BroadcastToAll(wsMsg)
	}
}

// SendCommand sends a command to VLC
func (c *VLCWebSocketClient) SendCommand(command string, id *int, floatValue *float64, longValue *int64, stringValue *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		logrus.WithField("vlc_url", c.vlcURL).Warn("Cannot send command - WebSocket not connected")
		return fmt.Errorf("not connected to VLC WebSocket")
	}

	// Validate auth ticket
	if c.authTicket == "" {
		logrus.WithField("vlc_url", c.vlcURL).Error("Cannot send command - no auth ticket")
		return fmt.Errorf("no authentication ticket")
	}

	// Check if ticket needs renewal
	if c.isTicketExpired() {
		logrus.WithField("vlc_url", c.vlcURL).Info("Ticket expired, renewing before sending command")
		if err := c.renewTicket(); err != nil {
			logrus.WithFields(logrus.Fields{
				"vlc_url": c.vlcURL,
				"error":   err,
			}).Error("Failed to renew ticket")
			return fmt.Errorf("failed to renew ticket: %w", err)
		}
	}

	cmd := VLCCommand{
		Message:     command,
		ID:          id,
		FloatValue:  floatValue,
		LongValue:   longValue,
		StringValue: stringValue,
		AuthTicket:  c.authTicket,
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url":     c.vlcURL,
		"command":     command,
		"id":          id,
		"floatValue":  floatValue,
		"longValue":   longValue,
		"stringValue": stringValue,
		"authTicket":  c.authTicket,
	}).Info("Sending command to VLC WebSocket")

	// Log the exact JSON being sent
	if cmdJSON, err := json.Marshal(cmd); err == nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url":   c.vlcURL,
			"command":   command,
			"json_sent": string(cmdJSON),
		}).Info("WebSocket command JSON")
	}

	if err := c.conn.WriteJSON(cmd); err != nil {
		logrus.WithFields(logrus.Fields{
			"vlc_url": c.vlcURL,
			"command": command,
			"error":   err,
		}).Error("Failed to write JSON to WebSocket")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"vlc_url": c.vlcURL,
		"command": command,
	}).Info("Command sent successfully to WebSocket")
	return nil
}

// Play sends play command to VLC
func (c *VLCWebSocketClient) Play() error {
	return c.SendCommand("play", nil, nil, nil, nil)
}

// Pause sends pause command to VLC
func (c *VLCWebSocketClient) Pause() error {
	return c.SendCommand("pause", nil, nil, nil, nil)
}

// Stop sends stop command to VLC (not supported by VLC Android, using pause instead)
func (c *VLCWebSocketClient) Stop() error {
	return c.SendCommand("pause", nil, nil, nil, nil)
}

// Seek sends seek command to VLC (in milliseconds)
func (c *VLCWebSocketClient) Seek(position int64) error {
	// VLC Android expects the time value in the 'id' field for SET_PROGRESS
	id := int(position)
	return c.SendCommand("set-progress", &id, nil, nil, nil)
}

// SetVolume sets volume (0-100)
func (c *VLCWebSocketClient) SetVolume(volume int) error {
	vol := volume
	return c.SendCommand("set-volume", &vol, nil, nil, nil)
}

// tryReconnect attempts to reconnect to VLC WebSocket
func (c *VLCWebSocketClient) tryReconnect() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(backoff):
		}

		logrus.WithField("vlc_url", c.vlcURL).Info("Attempting to reconnect to VLC WebSocket")

		if err := c.Connect(); err != nil {
			logrus.WithError(err).Error("Failed to reconnect to VLC WebSocket")

			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		logrus.WithField("vlc_url", c.vlcURL).Info("Successfully reconnected to VLC WebSocket")
		return
	}
}

// Disconnect closes the WebSocket connection
func (c *VLCWebSocketClient) Disconnect() {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false

	logrus.WithField("vlc_url", c.vlcURL).Info("Disconnected from VLC WebSocket")
}

// IsConnected returns whether the client is connected
func (c *VLCWebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// GetVLCWebSocketClient returns or creates a WebSocket client for the given VLC URL
func GetVLCWebSocketClient(vlcURL string, session *types.VLCSession) *VLCWebSocketClient {
	vlcWebSocketClientsMutex.Lock()
	defer vlcWebSocketClientsMutex.Unlock()

	client, exists := vlcWebSocketClients[vlcURL]
	if !exists || client == nil {
		client = NewVLCWebSocketClient(vlcURL, session)
		vlcWebSocketClients[vlcURL] = client
	}

	return client
}

// RemoveVLCWebSocketClient removes a WebSocket client
func RemoveVLCWebSocketClient(vlcURL string) {
	vlcWebSocketClientsMutex.Lock()
	defer vlcWebSocketClientsMutex.Unlock()

	if client, exists := vlcWebSocketClients[vlcURL]; exists {
		client.Disconnect()
		delete(vlcWebSocketClients, vlcURL)
	}
}

// GetVLCWebSocketClients returns all WebSocket clients
func GetVLCWebSocketClients() map[string]*VLCWebSocketClient {
	vlcWebSocketClientsMutex.RLock()
	defer vlcWebSocketClientsMutex.RUnlock()

	clients := make(map[string]*VLCWebSocketClient)
	for k, v := range vlcWebSocketClients {
		clients[k] = v
	}
	return clients
}
