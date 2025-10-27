# 🗺️ Coloc Video Downloader - Roadmap

## 📋 Table of Contents
- [High Priority Features](#high-priority-features)
- [Medium Priority Features](#medium-priority-features)
- [Future Enhancements](#future-enhancements)
- [Technical Debt](#technical-debt)
- [Recently Completed](#recently-completed)

---

## 🚀 High Priority Features

### 1. VLC WebSocket Integration
**Status**: 🔴 Not Started  
**Estimated Effort**: 2-3 days  
**Priority**: Critical

#### Problem Statement
Currently, the application uses VLC's HTTP API for playback control, which has limitations:
- No real-time feedback on playback status
- Cannot detect when videos finish playing
- No error reporting from VLC (codec issues, file not found, etc.)
- Limited control capabilities

#### Solution: WebSocket Connection to VLC
VLC provides a WebSocket interface at `/echo` for bidirectional real-time communication.

#### Technical Details

**VLC WebSocket Endpoints:**
- `GET /wsticket` - Request authentication challenge
  - Returns: Plain text challenge string
  - Example: `"abc123def456"`
  
- `WebSocket /echo` - WebSocket connection endpoint
  - Authentication: Send challenge string after connection
  - Protocol: JSON messages

**Reference Implementation:**
- [VLC Android Remote Access Server](https://code.videolan.org/videolan/vlc-android/-/tree/master/application/remote-access-server/src/main/java/org/videolan/vlc/remoteaccessserver?ref_type=heads)
- Key files:
  - `WebSocketHandler.java` - WebSocket message handling
  - `RemoteAccessService.java` - HTTP endpoint for ticket generation
  - `Protocol.kt` - Message format definitions

#### Authentication Flow
```
1. Client → GET http://vlc-host:8080/wsticket
2. VLC → Response: "challenge_token_string"
3. Client → WebSocket connection to ws://vlc-host:8080/echo
4. Client → Send: challenge_token_string (as first message)
5. VLC → Connection authenticated
6. Client ↔ VLC → Bidirectional JSON messages
```

#### Message Format (from VLC code analysis)
```json
// Player Status Update (from VLC)
{
  "type": "status",
  "state": "playing",  // playing, paused, stopped
  "time": 123456,      // milliseconds
  "length": 654321,    // milliseconds
  "volume": 100,       // 0-200
  "title": "Video Title"
}

// Player Error (from VLC)
{
  "type": "error",
  "message": "Cannot open file",
  "code": 404
}

// Control Command (to VLC)
{
  "type": "command",
  "action": "play"     // play, pause, stop, seek
}

// Seek Command (to VLC)
{
  "type": "seek",
  "position": 60000    // milliseconds
}
```

#### Implementation Tasks

**Backend (`internal/vlc/websocket.go`):**
- [ ] Create `VLCWebSocketClient` struct
- [ ] Implement `RequestTicket()` - GET /wsticket
- [ ] Implement `Connect(ticket string)` - WebSocket connection
- [ ] Implement `SendAuthentication(challenge string)`
- [ ] Create message parser for VLC JSON messages
- [ ] Handle connection lifecycle (connect, disconnect, reconnect)
- [ ] Broadcast VLC events to frontend via existing WebSocket

**Backend (`internal/handlers/vlc_handlers.go`):**
- [ ] Add `VLCWebSocketStatusHandler` - Get current VLC status
- [ ] Add `VLCWebSocketControlHandler` - Send control commands
- [ ] Integrate WebSocket status into existing VLC session management

**Frontend (`web/static/js/vlc.js`):**
- [ ] Add WebSocket status subscription
- [ ] Display real-time playback status
- [ ] Show playback progress bar
- [ ] Add play/pause/stop controls
- [ ] Show error notifications from VLC

**Testing:**
- [ ] Unit tests for WebSocket client
- [ ] Integration test with real VLC instance
- [ ] Test reconnection on connection loss
- [ ] Test error handling

#### Benefits
✅ Real-time playback monitoring  
✅ Detect video completion  
✅ Error feedback (codec, network, file issues)  
✅ Advanced playback controls  
✅ Better user experience  

#### Risks & Challenges
- VLC WebSocket API might differ between versions
- Need to handle connection drops gracefully
- Challenge token expiration handling
- May need to maintain both HTTP and WebSocket for compatibility

---

### 2. API Endpoint Documentation
**Status**: 🟡 Partially Complete  
**Estimated Effort**: 1 day  
**Priority**: High

#### Tasks
- [ ] Create OpenAPI/Swagger specification
- [ ] Document all REST endpoints with examples
- [ ] Document WebSocket message types
- [ ] Add Postman collection
- [ ] Generate API documentation site

---

## 🎯 Medium Priority Features

### 3. Mobile Application
**Status**: 🔴 Not Started  
**Estimated Effort**: 2-3 weeks  
**Priority**: Medium

#### Problem Statement
Users want to share video URLs from mobile apps (YouTube, Twitch, browsers) directly to the downloader without:
1. Opening a browser
2. Navigating to the web interface
3. Copy-pasting the URL manually

#### Solution: Native Mobile App

#### Features
**Core Functionality:**
- Register as "Share" target for video URLs
- Pre-fill URL in download form
- Submit directly from share sheet
- Real-time download progress notifications
- Configure server URL once (persistent)

**Additional Features:**
- View download queue
- Browse download history
- Playback controls for VLC
- Dark mode support
- Multi-server support (home, work, etc.)

#### Technology Stack Options

**Option A: Flutter (Recommended)**
- ✅ Single codebase for Android & iOS
- ✅ Native performance
- ✅ Great for forms and lists
- ✅ Built-in deep linking support
- ❌ Larger app size

**Option B: React Native**
- ✅ JavaScript familiarity
- ✅ Hot reload
- ✅ Large ecosystem
- ❌ More complex native integration

**Option C: Native (Kotlin + Swift)**
- ✅ Best performance
- ✅ Full platform control
- ❌ Maintain two codebases
- ❌ Longer development time

#### Architecture

**App Structure:**
```
mobile/
├── lib/
│   ├── main.dart
│   ├── models/
│   │   └── download_job.dart
│   ├── services/
│   │   ├── api_client.dart
│   │   └── websocket_service.dart
│   ├── screens/
│   │   ├── share_handler_screen.dart
│   │   ├── queue_screen.dart
│   │   └── settings_screen.dart
│   └── widgets/
│       ├── download_item.dart
│       └── progress_indicator.dart
└── android/
    └── app/src/main/AndroidManifest.xml
```

**Android Manifest (Share Target):**
```xml
<activity android:name=".ShareActivity">
    <intent-filter>
        <action android:name="android.intent.action.SEND" />
        <category android:name="android.intent.category.DEFAULT" />
        <data android:mimeType="text/plain" />
    </intent-filter>
</activity>
```

#### User Flow
```
1. User sees video in YouTube app
   ↓
2. Tap Share button
   ↓
3. Select "Coloc Downloader" from share sheet
   ↓
4. App opens with:
   - URL pre-filled
   - Mode selection (Download/Stream)
   - AutoPlay checkbox
   ↓
5. Tap "Submit"
   ↓
6. Show toast "Added to queue"
   ↓
7. Background notification shows progress
   ↓
8. Completion notification: "Video ready on VLC"
```

#### API Integration
- Use existing REST API (`/urlyt`, `/queue`)
- WebSocket connection for real-time updates
- Local storage for server configuration

#### Implementation Phases

**Phase 1: MVP (Week 1)**
- [ ] Share URL handler
- [ ] Basic form (URL, mode, autoplay)
- [ ] Submit to server
- [ ] Settings page (server URL)

**Phase 2: Queue Management (Week 2)**
- [ ] View download queue
- [ ] Real-time progress updates
- [ ] Retry failed downloads
- [ ] Clear queue

**Phase 3: Polish (Week 3)**
- [ ] Push notifications
- [ ] Error handling
- [ ] Dark mode
- [ ] App icon & branding

**Phase 4: iOS Support**
- [ ] Port to iOS
- [ ] iOS share extension
- [ ] App Store submission

---

### 4. Download History & Search
**Status**: 🔴 Not Started  
**Estimated Effort**: 3-4 days  
**Priority**: Medium

#### Features
- [ ] Persist download history in SQLite
- [ ] Search by title, URL, date
- [ ] Filter by source (YouTube, Twitch, etc.)
- [ ] Re-download from history
- [ ] Export history as JSON/CSV

#### Schema
```sql
CREATE TABLE downloads (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    title TEXT,
    source TEXT,
    mode TEXT,
    status TEXT,
    created_at TIMESTAMP,
    completed_at TIMESTAMP,
    file_path TEXT,
    error TEXT
);

CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);
CREATE INDEX idx_downloads_title ON downloads(title);
```

---

## 🌟 Future Enhancements

### 5. Audio-Only Downloads
**Status**: 🔴 Not Started  
**Estimated Effort**: 1-2 days

#### Features
- Extract MP3/M4A from videos
- yt-dlp format: `bestaudio[ext=m4a]`
- Toggle in UI: "Audio Only"
- Smaller file sizes
- Faster downloads

---

### 6. Custom Quality Selection
**Status**: 🔴 Not Started  
**Estimated Effort**: 2-3 days

#### Features
- Dropdown: 4K, 1080p, 720p, 480p, Auto
- yt-dlp format strings per quality
- Show available qualities before download
- Default quality in settings

---

### 7. Playlist Support
**Status**: 🔴 Not Started  
**Estimated Effort**: 3-5 days

#### Features
- Detect YouTube/Twitch playlists
- Download all videos in playlist
- Sequential or parallel downloads
- Progress tracking per video
- VLC playlist integration

---

### 8. Multi-User Support
**Status**: 🔴 Not Started  
**Estimated Effort**: 1 week

#### Features
- User authentication (JWT)
- Per-user download queues
- User preferences (VLC URL, download mode)
- Admin panel
- User quotas (disk space, downloads)

---

### 9. Dark Mode
**Status**: 🔴 Not Started  
**Estimated Effort**: 1 day

#### Implementation
- [ ] CSS variables for colors
- [ ] Dark theme stylesheet
- [ ] Toggle in UI
- [ ] Persist preference in localStorage

---

### 10. HTTPS Support
**Status**: 🔴 Not Started  
**Estimated Effort**: 1 day

#### Features
- Let's Encrypt integration
- Automatic certificate renewal
- HTTP → HTTPS redirect
- Secure WebSocket (WSS)

---

## 🔧 Technical Debt

### Code Quality
- [ ] Add more unit tests (target: 80% coverage)
- [ ] Add integration tests for all API endpoints
- [ ] Linting with golangci-lint
- [ ] Frontend linting with ESLint
- [ ] Type safety with TypeScript

### Performance
- [ ] Database connection pooling
- [ ] Implement caching for yt-dlp results
- [ ] Optimize Docker image size
- [ ] Add rate limiting to API

### Documentation
- [ ] Add inline code comments
- [ ] Create architecture diagrams
- [ ] Document deployment strategies
- [ ] Add troubleshooting guide

### Security
- [ ] Add CSRF protection
- [ ] Implement rate limiting
- [ ] Input validation improvements
- [ ] Security headers (CSP, HSTS, etc.)

---

## ✅ Recently Completed (October 2025)

### Retry Failed Downloads (Oct 27, 2025)
- ✅ Added "🔄 Réessayer" button for failed jobs
- ✅ Backend endpoint `/retry/:jobID`
- ✅ Re-insert failed jobs into queue
- ✅ Jobs in error status persist (not auto-removed)
- ✅ Real-time queue updates via WebSocket

### Smart Error Handling (Oct 27, 2025)
- ✅ HTTP 200 with `success: false` for business errors
- ✅ HTTP 500 only for real server errors
- ✅ Frontend displays specific error messages
- ✅ No more generic "HTTP 500" errors
- ✅ Fixed "Clear Queue" success message

### FFmpeg Progress Tracking (Oct 26, 2025)
- ✅ Parse FFmpeg stderr for progress
- ✅ Broadcast progress via WebSocket
- ✅ Show time elapsed and speed (e.g., "00:05:23 (1.2x)")
- ✅ Throttle updates to every 2 seconds

### Queue Synchronization (Oct 26, 2025)
- ✅ Centralized job status in `config` package
- ✅ All clients see same queue state
- ✅ New WebSocket clients receive queue on connect
- ✅ Broadcast updates to all connected clients

### VLC Authentication (Oct 25, 2025)
- ✅ 4-digit code verification flow
- ✅ Session management with HTTP client
- ✅ Persistent authentication across requests
- ✅ VLC status indicator in UI

### yt-dlp Auto-Update (Oct 25, 2025)
- ✅ Automatic update check on startup
- ✅ Update check before each download
- ✅ Status broadcast via WebSocket
- ✅ Fallback to existing binary on failure

---

## 📅 Development Priorities

### Next Session (Immediate)
1. **VLC WebSocket Integration** (Highest impact)
   - Research VLC WebSocket protocol
   - Implement `/wsticket` endpoint
   - Create WebSocket client
   - Add playback controls to UI

### Following Sessions
2. **Mobile App** (High user value)
   - Set up Flutter project
   - Implement share handler
   - Create basic UI

3. **Download History** (Nice to have)
   - Add SQLite database
   - Create history UI

---

## 🤝 Contributing

Interested in working on any of these features? Check out:
1. [DEVELOPMENT.md](docs/DEVELOPMENT.md) - Development setup
2. [ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design
3. Create an issue to discuss your approach
4. Submit a PR with tests

**Good First Issues:**
- Dark mode implementation
- Add more unit tests
- Improve error messages
- UI/UX improvements

**Advanced Issues:**
- VLC WebSocket integration
- Mobile app development
- Multi-user authentication
