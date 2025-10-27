# Architecture Guide

> Comprehensive technical documentation for the Coloc Video Downloader application.

## Table of Contents

1. [System Overview](#system-overview)
2. [Backend Architecture](#backend-architecture)
3. [Frontend Architecture](#frontend-architecture)
4. [VLC Integration](#vlc-integration)
5. [Data Flow](#data-flow)
6. [Technology Stack](#technology-stack)

---

## System Overview

### High-Level Architecture

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────┐
│   Web Browser   │◄───────►│   Go Backend     │◄───────►│ VLC Player  │
│   (Frontend)    │  HTTP/  │  (Web Server)    │  HTTP   │  (External) │
│                 │  WS     │                  │         │             │
└─────────────────┘         └──────────────────┘         └─────────────┘
                                     │
                                     │
                                     ▼
                            ┌──────────────────┐
                            │   yt-dlp         │
                            │   (Download)     │
                            └──────────────────┘
                                     │
                                     ▼
                            ┌──────────────────┐
                            │  File System     │
                            │  (/videos)       │
                            └──────────────────┘
```

### Core Components

1. **Web Frontend**: Browser-based UI for user interaction
2. **Go Backend**: HTTP server handling requests, downloads, and WebSocket communication
3. **VLC Integration**: Secure authentication and playback control
4. **Download Engine**: yt-dlp wrapper for video downloading
5. **Queue System**: Asynchronous job processing with progress tracking

---

## Backend Architecture

### Package Structure

The backend follows Go's standard project layout:

```
internal/
├── types/          # Data structures and models
├── vlc/            # VLC player integration
├── download/       # Video download processing
├── websocket/      # Real-time WebSocket communication
└── handlers/       # HTTP request handlers

pkg/
└── config/         # Global configuration and state management

cmd/
└── coloc/
    └── main.go     # Application entry point
```

### Core Packages

#### 1. `internal/types` - Data Structures

Defines all data models used throughout the application:

**Key Types:**

```go
// URLRequest - Client request for video download
type URLRequest struct {
    URL        string `json:"url"`         // Video URL
    AutoPlay   bool   `json:"autoPlay"`    // Auto-play on VLC
    VLCUrl     string `json:"vlcUrl"`      // VLC server URL
    BackendURL string `json:"backendUrl"`  // This server's URL
    Mode       string `json:"mode"`        // "download" or "stream"
}

// DownloadJob - Represents a download task
type DownloadJob struct {
    ID         string    // Unique job identifier
    URL        string    // Source video URL
    Mode       string    // Download mode
    AutoPlay   bool      // Auto-play flag
    VLCUrl     string    // VLC target
    BackendURL string    // Server URL
    CreatedAt  time.Time // Job creation time
}

// JobStatus - Current state of a download job
type JobStatus struct {
    Job         *DownloadJob // Associated job
    Status      string       // "queued", "processing", "downloading", "completed", "error"
    Progress    string       // Human-readable progress
    Error       string       // Error message if failed
    CompletedAt time.Time    // Completion timestamp
    StreamURL   string       // Direct stream URL (for stream mode)
}

// WSMessage - WebSocket message format
type WSMessage struct {
    Type       string       // Message type
    DownloadID string       // Associated download ID
    Line       string       // Progress line from yt-dlp
    Percent    float64      // Download percentage
    File       string       // Downloaded file path
    Message    string       // Status message
    Videos     []string     // List of available videos
    Queue      []JobStatus  // Current queue state
}

// VLCSession - VLC authentication session
type VLCSession struct {
    Code       string    // 4-digit authentication code
    Cookie     string    // Session cookie
    ExpiresAt  time.Time // Session expiration
    VLCUrl     string    // VLC server URL
}
```

**Purpose**: Centralized type definitions ensure consistency across the application and make the codebase self-documenting.

---

#### 2. `internal/vlc` - VLC Integration

Handles all VLC media player interactions:

**Key Functions:**

```go
// RequestAuthCode - Initiates VLC authentication
// Returns a 4-digit code that user must enter in VLC
func RequestAuthCode(vlcURL string) (string, error)

// VerifyAuthCode - Verifies the authentication code
// Returns session cookie for authenticated requests
func VerifyAuthCode(vlcURL, code string) (string, error)

// PlayVideo - Sends play command to VLC
// Adds video to VLC playlist and starts playback
func PlayVideo(vlcURL, cookie, videoURL string) error

// GetVLCStatus - Checks VLC player status
// Returns current playback state
func GetVLCStatus(vlcURL, cookie string) (map[string]interface{}, error)
```

**Authentication Flow:**

1. Client requests authentication code
2. Backend calls VLC's `/requests/auth.json` endpoint
3. VLC displays 4-digit code to user
4. User enters code in web interface
5. Backend verifies code with VLC
6. VLC returns session cookie
7. Cookie stored for subsequent requests

**Security:**
- Cookies never exposed to client
- All VLC communication server-side
- Session expiration handled automatically

---

#### 3. `internal/download` - Download Engine

Manages video downloads using yt-dlp:

**Key Functions:**

```go
// ProcessDownloadJob - Main download processor
// Handles the entire download lifecycle
func ProcessDownloadJob(job *DownloadJob)

// ProcessStreamJob - Handles streaming mode
// Extracts direct video URL without downloading
func ProcessStreamJob(job *DownloadJob)

// GenerateDownloadID - Creates unique job identifier
func GenerateDownloadID() string

// AutoPlayVideo - Automatically plays video on VLC after download
func AutoPlayVideo(vlcURL, cookie, videoPath, backendURL string) error

// PruneVideos - Cleanup old videos (keeps last 10)
func PruneVideos()
```

**Download Process:**

1. Job added to queue channel
2. Worker goroutine picks up job
3. Update job status to "processing"
4. Execute yt-dlp with progress tracking
5. Parse yt-dlp output for progress updates
6. Broadcast progress via WebSocket
7. On completion:
   - Update status to "completed"
   - Trigger auto-play if enabled
   - Broadcast completion message
8. On error:
   - Update status to "error"
   - Store error message
   - Broadcast error

**Progress Tracking:**

yt-dlp output is parsed in real-time:
```
[download]  45.2% of 123.45MiB at 2.34MiB/s ETA 00:25
```

Parsed into structured progress updates sent via WebSocket.

---

#### 4. `internal/websocket` - Real-time Communication

Manages WebSocket connections for live updates:

**Key Functions:**

```go
// WSHandler - Handles WebSocket connection upgrades
func WSHandler(w http.ResponseWriter, r *http.Request)

// BroadcastProgress - Sends progress update to subscribed clients
func BroadcastProgress(downloadID, line string, percent float64)

// BroadcastCompletion - Notifies clients of download completion
func BroadcastCompletion(downloadID, file string)

// BroadcastError - Notifies clients of download errors
func BroadcastError(downloadID, errorMsg string)

// BroadcastVideoList - Sends updated video list to all clients
func BroadcastVideoList(videos []string)

// BroadcastQueueStatus - Sends current queue state to all clients
func BroadcastQueueStatus()
```

**WebSocket Protocol:**

Client → Server messages:
```json
{
  "action": "subscribe",
  "downloadId": "dl_123456789"
}
```

Server → Client messages:
```json
{
  "type": "progress",
  "downloadId": "dl_123456789",
  "percent": 45.2,
  "line": "[download] 45.2% of 123.45MiB at 2.34MiB/s ETA 00:25"
}
```

**Message Types:**
- `progress` - Download progress update
- `done` - Download completed
- `error` - Download failed
- `videoList` - Available videos updated
- `queueStatus` - Queue state changed

---

#### 5. `internal/handlers` - HTTP Handlers

Handles all HTTP endpoints:

**Static File Handlers:**
```go
func HomeHandler(w http.ResponseWriter, r *http.Request)    // Serves index.html
func StylesHandler(w http.ResponseWriter, r *http.Request)  // Serves CSS
func AppHandler(w http.ResponseWriter, r *http.Request)     // Serves JavaScript
```

**Download Handlers:**
```go
func DownloadURLHandler(w http.ResponseWriter, r *http.Request)      // Direct URL download
func DownloadYouTubeHandler(w http.ResponseWriter, r *http.Request)  // YouTube download
func TwitchHandler(w http.ResponseWriter, r *http.Request)           // Twitch stream
func PlayURLHandler(w http.ResponseWriter, r *http.Request)          // Play direct URL
```

**Queue Handlers:**
```go
func QueueStatusHandler(w http.ResponseWriter, r *http.Request)  // Get queue status
func ClearQueueHandler(w http.ResponseWriter, r *http.Request)   // Clear completed jobs
```

**VLC Handlers:**
```go
func VLCCodeHandler(w http.ResponseWriter, r *http.Request)      // Request auth code
func VLCVerifyHandler(w http.ResponseWriter, r *http.Request)    // Verify auth code
func VLCPlayHandler(w http.ResponseWriter, r *http.Request)      // Send play command
func VLCStatusHandler(w http.ResponseWriter, r *http.Request)    // Get VLC status
func VLCConfigHandler(w http.ResponseWriter, r *http.Request)    // VLC configuration
```

**Video Handlers:**
```go
func ListVideosHandler(w http.ResponseWriter, r *http.Request)  // List downloaded videos
```

---

#### 6. `pkg/config` - Global State

Manages application-wide state and configuration:

**Global Variables:**

```go
// Download queue management
var downloadJobs = make(chan *DownloadJob, 100)  // Buffered channel for jobs
var jobStatuses = make(map[string]*JobStatus)    // Job status tracking
var jobStatusesMutex sync.RWMutex                // Thread-safe access

// WebSocket client management
var wsClients = make(map[*WSClient]bool)         // Active WebSocket connections
var wsClientsMutex sync.RWMutex                  // Thread-safe access

// VLC session management
var vlcSessions = make(map[string]*VLCSession)   // Active VLC sessions
var vlcSessionsMutex sync.RWMutex                // Thread-safe access
```

**Key Functions:**

```go
// Job Status Management
func AddJobStatus(downloadID string, job *DownloadJob)
func GetJobStatus(downloadID string) *JobStatus
func UpdateJobStatus(downloadID, status, progress string)
func GetAllJobStatuses() []JobStatus

// WebSocket Client Management
func AddWSClient(client *WSClient)
func RemoveWSClient(client *WSClient)
func GetWSClients() []*WSClient

// VLC Session Management
func StoreVLCSession(code string, session *VLCSession)
func GetVLCSession(code string) *VLCSession
func CleanupExpiredSessions()
```

**Concurrency Safety:**

All global state access is protected by mutexes to ensure thread-safety in the concurrent environment.

---

### Application Lifecycle

**Startup Sequence:**

1. **Initialize yt-dlp and ffmpeg**
   ```go
   ytdlp.MustInstall(context.Background(), nil)
   ```

2. **Start download worker**
   ```go
   go download.StartDownloadWorker()
   ```
   - Listens on `downloadJobs` channel
   - Processes jobs sequentially
   - Runs in background goroutine

3. **Start video directory watcher**
   ```go
   go download.WatchVideoDirectory()
   ```
   - Monitors `/videos` directory for changes
   - Broadcasts video list updates
   - Runs in background goroutine

4. **Register HTTP routes**
   ```go
   http.HandleFunc("/", handlers.HomeHandler)
   http.HandleFunc("/ws", websocket.WSHandler)
   // ... all other routes
   ```

5. **Start HTTP server**
   ```go
   http.ListenAndServe(":8080", nil)
   ```

**Runtime Behavior:**

- **Concurrent job processing**: Single worker processes jobs sequentially from queue
- **Real-time updates**: WebSocket broadcasts to all connected clients
- **Automatic cleanup**: Periodic pruning of old videos
- **Session management**: Automatic cleanup of expired VLC sessions

---

## Frontend Architecture

### Module Structure

The frontend is built with vanilla JavaScript using ES6 modules:

```
js/
├── config.js       # Configuration and constants
├── state.js        # Global state management
├── utils.js        # Utility functions
├── toast.js        # Notification system
├── api.js          # HTTP API client
├── websocket.js    # WebSocket client
├── download.js     # Download queue UI
├── status.js       # Status indicators
├── vlc.js          # VLC integration UI
├── modal.js        # Modal dialogs
├── video.js        # Video list UI
├── events.js       # Event listeners
└── app.js          # Application initialization
```

### Core Modules

#### 1. `config.js` - Configuration

Defines application constants and DOM selectors:

```javascript
export const CONFIG = {
  WS_RECONNECT_DELAY: 3000,
  TOAST_DURATION: 3000,
  VLC_STATUS_INTERVAL: 5000,
};

export const SELECTORS = {
  urlInput: '#urlInput',
  downloadBtn: '#downloadBtn',
  queueContainer: '#queueContainer',
  // ... all DOM selectors
};
```

**Purpose**: Centralized configuration makes the application easy to customize and maintain.

---

#### 2. `state.js` - State Management

Manages global application state:

```javascript
export const state = {
  // VLC authentication
  vlcAuthenticated: false,
  vlcUrl: localStorage.getItem('vlcUrl') || '',
  vlcCookie: null,
  
  // WebSocket connection
  ws: null,
  wsConnected: false,
  
  // Download tracking
  activeDownloads: new Map(),
  downloadQueue: [],
  
  // UI state
  isLoading: false,
  currentModal: null,
};
```

**State Persistence:**
- VLC URL stored in `localStorage`
- Survives page refreshes
- Cleared on logout

---

#### 3. `api.js` - HTTP Client

Handles all HTTP requests to the backend:

```javascript
class ApiClient {
  async request(url, options = {}) {
    // Base request method with error handling
  }
  
  async post(url, data) {
    // POST request with JSON body
  }
  
  async get(url) {
    // GET request
  }
}

export const api = new ApiClient();
```

**Usage Example:**
```javascript
const response = await api.post('/urlyt', {
  url: videoUrl,
  mode: 'download',
  autoPlay: true,
});
```

---

#### 4. `websocket.js` - WebSocket Client

Manages WebSocket connection and message handling:

```javascript
class WebSocketManager {
  connect() {
    // Establish WebSocket connection
    // Handle reconnection on disconnect
  }
  
  send(message) {
    // Send message to server
  }
  
  subscribe(downloadId) {
    // Subscribe to specific download updates
  }
  
  subscribeAll() {
    // Subscribe to all downloads
  }
  
  onMessage(handler) {
    // Register message handler
  }
}
```

**Message Handling:**
```javascript
wsManager.onMessage((message) => {
  switch (message.type) {
    case 'progress':
      updateProgressBar(message.downloadId, message.percent);
      break;
    case 'done':
      showCompletionNotification(message.downloadId);
      break;
    case 'error':
      showErrorNotification(message.downloadId, message.message);
      break;
  }
});
```

**Auto-Reconnection:**
- Detects connection loss
- Attempts reconnection every 3 seconds
- Resubscribes to downloads on reconnect

---

#### 5. `download.js` - Download Queue UI

Manages the download queue display:

```javascript
class DownloadManager {
  renderQueue(queueData) {
    // Render queue items with progress bars
  }
  
  updateProgress(downloadId, percent, message) {
    // Update specific download progress
  }
  
  removeCompleted(downloadId) {
    // Remove completed download from UI
  }
}
```

**Queue Item Structure:**
```html
<div class="queue-item" data-download-id="dl_123">
  <div class="queue-item-header">
    <span class="queue-item-url">https://youtube.com/...</span>
    <span class="queue-item-status">Downloading</span>
  </div>
  <div class="progress-bar">
    <div class="progress-fill" style="width: 45%"></div>
  </div>
  <div class="queue-item-progress">45.2% - 2.34MiB/s - ETA 00:25</div>
</div>
```

---

#### 6. `vlc.js` - VLC Integration UI

Handles VLC authentication and control:

```javascript
class VlcManager {
  async testConnection(vlcUrl) {
    // Test if VLC is accessible
  }
  
  async requestAuthCode(vlcUrl) {
    // Request authentication code from VLC
  }
  
  async verifyCode(vlcUrl, code) {
    // Verify authentication code
  }
  
  async getStatus() {
    // Get current VLC status
  }
  
  saveConfig(vlcUrl) {
    // Save VLC URL to localStorage
  }
  
  clearConfig() {
    // Clear saved VLC configuration
  }
}
```

**Authentication Flow:**

1. User clicks VLC icon
2. Modal opens with VLC URL input
3. User enters VLC URL and clicks "Connect"
4. Backend requests auth code from VLC
5. VLC displays 4-digit code
6. User enters code in modal
7. Backend verifies code
8. On success: Store session, close modal, show success toast
9. On failure: Show error message

---

#### 7. `status.js` - Status Indicators

Manages status indicators for VLC and yt-dlp:

```javascript
class StatusManager {
  updateVLCStatus(authenticated) {
    // Update VLC connection indicator
    // Green = connected, Red = disconnected
  }
  
  updateYtDlpStatus(available) {
    // Update yt-dlp availability indicator
  }
  
  startStatusPolling() {
    // Poll VLC status every 5 seconds
  }
  
  stopStatusPolling() {
    // Stop status polling
  }
}
```

**Visual Indicators:**
- 🟢 Green: Connected/Available
- 🔴 Red: Disconnected/Unavailable
- 🟡 Yellow: Connecting/Loading

---

#### 8. `toast.js` - Notification System

Displays temporary notifications:

```javascript
class ToastManager {
  show(message, type = 'info') {
    // Show toast notification
    // Types: 'success', 'error', 'info', 'warning'
  }
  
  success(message) {
    // Show success toast (green)
  }
  
  error(message) {
    // Show error toast (red)
  }
  
  info(message) {
    // Show info toast (blue)
  }
}
```

**Toast Appearance:**
- Slides in from top
- Auto-dismisses after 3 seconds
- Click to dismiss immediately
- Stacks multiple toasts

---

### Application Flow

**Initialization (`app.js`):**

```javascript
document.addEventListener('DOMContentLoaded', () => {
  // 1. Initialize all managers
  const wsManager = new WebSocketManager();
  const downloadManager = new DownloadManager();
  const vlcManager = new VlcManager();
  const statusManager = new StatusManager();
  
  // 2. Connect WebSocket
  wsManager.connect();
  
  // 3. Setup event listeners
  setupEventListeners();
  
  // 4. Load initial data
  loadVideoList();
  loadQueueStatus();
  
  // 5. Start status polling
  statusManager.startStatusPolling();
});
```

**User Interaction Flow:**

1. **User enters video URL**
   - Input validation
   - Enable/disable download button

2. **User clicks download**
   - Show loading state
   - Send POST request to `/urlyt`
   - Receive download ID
   - Subscribe to WebSocket updates
   - Add to queue UI

3. **Download progresses**
   - Receive WebSocket progress messages
   - Update progress bar
   - Update status text
   - Show speed and ETA

4. **Download completes**
   - Receive completion message
   - Show success toast
   - Update video list
   - Remove from queue (or mark complete)
   - Auto-play if enabled

5. **Error occurs**
   - Receive error message
   - Show error toast
   - Mark job as failed
   - Display error details

---

## VLC Integration

### Authentication Protocol

VLC uses a challenge-response authentication system:

#### Step 1: Request Authentication Code

**Request:**
```http
GET http://vlc-server:8080/requests/auth.json
```

**Response:**
```json
{
  "code": "1234",
  "expiry": 1698765432
}
```

VLC displays this 4-digit code in its interface.

#### Step 2: Verify Code

**Request:**
```http
GET http://vlc-server:8080/requests/auth.json?code=1234
```

**Response:**
```http
Set-Cookie: vlc_session=abc123xyz; Path=/; HttpOnly
```

The cookie is used for all subsequent requests.

---

### Playback Control

#### Add to Playlist and Play

**Request:**
```http
GET http://vlc-server:8080/requests/status.json?command=in_play&input=http://server/video.mp4
Cookie: vlc_session=abc123xyz
```

**Response:**
```json
{
  "state": "playing",
  "position": 0.0,
  "length": 1234,
  "volume": 256
}
```

#### Get Status

**Request:**
```http
GET http://vlc-server:8080/requests/status.json
Cookie: vlc_session=abc123xyz
```

**Response:**
```json
{
  "state": "playing",
  "position": 0.45,
  "length": 1234,
  "volume": 256,
  "currentplid": 1,
  "information": {
    "category": {
      "meta": {
        "filename": "video.mp4"
      }
    }
  }
}
```

---

### Security Considerations

**Server-Side Cookie Management:**
- Cookies never sent to client browser
- All VLC requests proxied through backend
- Session cookies stored server-side only

**Session Expiration:**
- Sessions expire after 24 hours
- Automatic cleanup of expired sessions
- User must re-authenticate after expiry

**Network Security:**
- VLC typically runs on local network
- No external VLC exposure required
- Backend acts as secure proxy

---

## Data Flow

### Download Flow

```
User Browser                Backend                 yt-dlp              VLC
     │                         │                      │                  │
     │  POST /urlyt            │                      │                  │
     ├────────────────────────>│                      │                  │
     │                         │                      │                  │
     │  200 OK (downloadId)    │                      │                  │
     │<────────────────────────┤                      │                  │
     │                         │                      │                  │
     │  WS: subscribe          │                      │                  │
     ├────────────────────────>│                      │                  │
     │                         │                      │                  │
     │                         │  Execute yt-dlp      │                  │
     │                         ├─────────────────────>│                  │
     │                         │                      │                  │
     │                         │  Progress output     │                  │
     │                         │<─────────────────────┤                  │
     │                         │                      │                  │
     │  WS: progress update    │                      │                  │
     │<────────────────────────┤                      │                  │
     │                         │                      │                  │
     │  WS: progress update    │                      │                  │
     │<────────────────────────┤                      │                  │
     │                         │                      │                  │
     │                         │  Download complete   │                  │
     │                         │<─────────────────────┤                  │
     │                         │                      │                  │
     │                         │  Play video          │                  │
     │                         ├──────────────────────────────────────>│
     │                         │                      │                  │
     │                         │  200 OK              │                  │
     │                         │<──────────────────────────────────────┤
     │                         │                      │                  │
     │  WS: done               │                      │                  │
     │<────────────────────────┤                      │                  │
```

### VLC Authentication Flow

```
User Browser                Backend                 VLC Player
     │                         │                      │
     │  Click "Connect VLC"    │                      │
     │                         │                      │
     │  POST /vlc/code         │                      │
     ├────────────────────────>│                      │
     │                         │                      │
     │                         │  GET /requests/auth.json
     │                         ├─────────────────────>│
     │                         │                      │
     │                         │  { "code": "1234" }  │
     │                         │<─────────────────────┤
     │                         │                      │
     │  { "code": "1234" }     │                      │
     │<────────────────────────┤                      │
     │                         │                      │
     │  [User sees code in VLC]│                      │
     │                         │                      │
     │  POST /vlc/verify-code  │                      │
     │  { "code": "1234" }     │                      │
     ├────────────────────────>│                      │
     │                         │                      │
     │                         │  GET /requests/auth.json?code=1234
     │                         ├─────────────────────>│
     │                         │                      │
     │                         │  Set-Cookie: session │
     │                         │<─────────────────────┤
     │                         │                      │
     │  { "success": true }    │                      │
     │<────────────────────────┤                      │
     │                         │                      │
     │  [Authenticated]        │                      │
```

---

## Technology Stack

### Backend

| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Primary language | 1.24.6 |
| **gorilla/websocket** | WebSocket support | 1.5.3 |
| **go-ytdlp** | YouTube downloading | Latest |
| **u2takey/ffmpeg-go** | Video processing | Latest |
| **logrus** | Structured logging | 1.9.3 |

### Frontend

| Technology | Purpose |
|------------|---------|
| **ES6 Modules** | Code organization |
| **Vanilla JavaScript** | No framework dependencies |
| **WebSocket API** | Real-time communication |
| **Fetch API** | HTTP requests |
| **CSS3** | Modern styling |
| **LocalStorage** | Client-side persistence |

### Infrastructure

| Technology | Purpose |
|------------|---------|
| **Docker** | Containerization |
| **Docker Compose** | Multi-container orchestration |
| **Alpine Linux** | Minimal base image |
| **Multi-stage builds** | Optimized image size |

### External Dependencies

| Dependency | Purpose | Installation |
|------------|---------|--------------|
| **yt-dlp** | Video downloading | Auto-installed by go-ytdlp |
| **ffmpeg** | Video processing | Auto-installed by go-ytdlp |
| **VLC** | Media playback | User-installed |

---

## Performance Considerations

### Backend Optimizations

1. **Buffered Channels**: Download queue uses buffered channel (100 jobs)
2. **Goroutines**: Concurrent processing for downloads and WebSocket broadcasts
3. **Mutex Protection**: Fine-grained locking for shared state
4. **Embedded Assets**: Static files embedded in binary (no disk I/O)

### Frontend Optimizations

1. **ES6 Modules**: Code splitting and lazy loading
2. **WebSocket**: Efficient real-time updates (no polling)
3. **LocalStorage**: Reduce server requests for configuration
4. **CSS Caching**: Long cache times for static assets

### Docker Optimizations

1. **Multi-stage Build**: Separate build and runtime images
2. **Minimal Base**: Alpine Linux for small image size
3. **Layer Caching**: Optimized Dockerfile for fast rebuilds
4. **Volume Mounts**: Persistent storage without container bloat

---

## Scalability Considerations

### Current Limitations

- **Single Worker**: One download at a time (sequential processing)
- **In-Memory State**: No persistence across restarts
- **Single Instance**: No horizontal scaling support

### Future Improvements

1. **Multiple Workers**: Parallel download processing
2. **Database**: Persistent job storage (PostgreSQL/Redis)
3. **Load Balancer**: Multiple backend instances
4. **Object Storage**: Distributed video storage (S3/MinIO)
5. **Message Queue**: RabbitMQ/Redis for job distribution

---

## Security Best Practices

### Implemented

- ✅ Server-side VLC authentication
- ✅ No cookie exposure to client
- ✅ Input validation on all endpoints
- ✅ CORS configuration
- ✅ Structured logging for audit trails

### Recommended

- 🔒 HTTPS/TLS for production
- 🔒 Rate limiting on API endpoints
- 🔒 Authentication for web interface
- 🔒 Video access control
- 🔒 Reverse proxy (nginx/Traefik)

---

This architecture provides a solid foundation for a reliable, maintainable video downloading application with seamless VLC integration.
