# Coloc Video Downloader

> A web-based video downloader with VLC integration for seamless video streaming to your media player.

## 🎯 Project Goal

This application solves a common problem: downloading videos from various sources (YouTube, Twitch, direct URLs) and automatically playing them on VLC media player. Instead of manually downloading videos and opening them in VLC, this application:

1. **Downloads videos** from multiple sources using yt-dlp
2. **Manages a queue** of downloads with real-time progress tracking
3. **Automatically plays** completed videos on your VLC media player
4. **Provides a web interface** for easy control from any device on your network

**Perfect for**: Shared living spaces (coloc = French for "roommate"), home media servers, or anyone who wants to queue up videos for playback on a central media player.

## ✨ Key Features

- 🎬 **Multi-source Downloads**: YouTube, Twitch, and direct video URLs
- 📋 **Queue Management**: Download multiple videos with automatic processing
- 📡 **Real-time Progress**: WebSocket-based live updates (download progress, queue status)
- 🎮 **VLC Integration**: Secure 4-digit code authentication and automatic playback control
- 🔄 **Retry Failed Downloads**: One-click retry for failed jobs
- ⚡ **Smart Error Handling**: User-friendly error messages for unsupported URLs
- 🎯 **Dual Mode**: Download & save or direct streaming to VLC
- 🌐 **Modern Web UI**: Responsive interface accessible from any device
- 📱 **PWA Support**: Install as a native app on mobile devices
- 🔐 **Trusted HTTPS**: ngrok integration for secure PWA installation
- 🐳 **Docker Ready**: One-command deployment with Docker Compose
- 🔍 **yt-dlp Auto-update**: Automatic updates and version checking

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.23+
- VLC with libvlc (for media processing)

### Installation
```bash
# Clone the repository
git clone https://github.com/yourusername/coloc.git
cd coloc

# Build and run with Docker
docker compose up -d

# Access the web interface
open http://localhost:8080
```

### PWA Installation (Mobile App)
For mobile app experience with trusted HTTPS:

**Option 1: Cloudflare Tunnel (Recommended - Free, No Registration)**
```bash
# Start with automated script
./scripts/start-cloudflare.sh

# Or manually:
docker compose up -d
cloudflared tunnel --url http://localhost:8080
```

**Option 2: ngrok (Requires Registration)**
```bash
docker compose up -d
ngrok http 8080
```

Use the provided HTTPS URL on your mobile device - PWA install prompt will appear automatically.

That's it! The application is now running and ready to download videos.

### 📱 PWA Testing (Recommended)

For mobile testing and PWA installation, use ngrok for trusted HTTPS:

```bash
# Install ngrok (NixOS)
nix-env -iA nixpkgs.ngrok

# Start ngrok tunnel
ngrok http 8080

# Or use the automated script
./scripts/ngrok-start.sh
```

Use the ngrok URL (e.g., `https://abc123.ngrok.io`) to install the PWA on your mobile device.

### VLC Setup

To enable VLC's web interface:

**Option 1: Command Line**
```bash
vlc --intf http --http-password your-secure-password
```

**Option 2: VLC GUI**
1. Open VLC → Tools → Preferences
2. Show settings: "All"
3. Interface → Main interfaces → Check "Web"
4. Interface → Main interfaces → Lua → Set password
5. Restart VLC

**Option 3: Configuration File**

Edit VLC's config file (`~/.config/vlc/vlcrc` on Linux):
```ini
http-password=your-secure-password
```

## 📖 Documentation

For detailed technical information:

- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design, backend, frontend, and VLC integration
- **[API Reference](docs/API.md)** - Complete HTTP and WebSocket API documentation
- **[Development Guide](docs/DEVELOPMENT.md)** - Setup, testing, and contribution guidelines

## 🏗️ Project Structure

```
Coloc/
├── cmd/coloc/              # Application entry point
│   └── main.go            # Server initialization
├── internal/               # Private application code
│   ├── types/             # Data structures and models
│   ├── vlc/               # VLC player integration
│   ├── download/          # Video download processing
│   ├── websocket/         # Real-time communication
│   └── handlers/          # HTTP request handlers
├── pkg/config/            # Global configuration and state
├── web/                   # Web assets
│   ├── static/           # HTML, CSS, JavaScript files
│   └── embed.go          # Embedded file declarations
├── js/                    # Frontend JavaScript modules (ES6)
├── test/                  # Integration tests
├── docs/                  # Documentation
├── videos/                # Downloaded videos storage
├── docker-compose.yml     # Docker deployment config
├── Dockerfile            # Container build instructions
└── README.md             # This file
```

## 🎮 Usage

### Initial Setup
1. **Open the web interface** at `http://localhost:8080`
2. **Configure VLC** by clicking the VLC icon (top-right)
3. **Enter VLC server URL** (e.g., `http://192.168.1.100:8080`)
4. **Request authentication code** - A 4-digit code appears in VLC
5. **Enter the code** to complete authentication

### Downloading Videos
1. **Paste a video URL** in the input field
   - YouTube: `https://www.youtube.com/watch?v=...`
   - Twitch: `https://www.twitch.tv/videos/...`
   - Direct URLs: Any direct video link
2. **Choose download mode**:
   - **📥 Download**: Downloads to server, then plays on VLC (supports seeking)
   - **📺 Stream**: Direct streaming to VLC (faster, no server storage)
3. **Optional**: Enable "AutoPlay" to start playback immediately
4. **Click Submit** to add to queue

### Managing Downloads
- **View Progress**: Real-time updates show download percentage and speed
- **Retry Failed**: Click "🔄 Réessayer" on failed downloads to retry
- **Clear Queue**: Remove completed and failed downloads
- **Multiple Clients**: All connected browsers see the same queue in real-time

## 🧪 Testing

```bash
# Run unit tests
go test ./internal/... ./pkg/... -v

# Run integration tests (requires running server)
docker compose up -d
go test ./test/... -v -run TestFrontendIntegration
go test ./test/... -v -run TestAPIEndpoints
go test ./test/... -v -run TestWebSocketProgressUpdates
```

## 🐳 Docker Commands

```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f video-server

# Restart
docker compose restart

# Stop and remove
docker compose down

# Rebuild from scratch
docker compose build --no-cache
docker compose up -d
```

## 🔧 Configuration

### Environment Variables

Edit `docker-compose.yml` to customize:

```yaml
environment:
  - LOG_LEVEL=info    # debug, info, warn, error
  - PORT=8080         # HTTP server port
```

### Storage

Videos are stored in `./videos` directory, which is mounted as a Docker volume.

## 🐛 Troubleshooting

### VLC Connection Issues
- Ensure VLC web interface is enabled and password is set
- Check that VLC is accessible at the configured URL
- Verify firewall isn't blocking the connection

### Download Failures
- Check Docker logs: `docker compose logs video-server`
- Verify the video URL is accessible
- Ensure sufficient disk space in `./videos`

### WebSocket Not Connecting
- Check browser console for errors (F12)
- Verify the server is running: `docker compose ps`
- Ensure no proxy is blocking WebSocket connections

## 🗺️ Roadmap

### 🚧 In Progress / Planned Features

#### 1. VLC WebSocket Integration (High Priority)
**Goal**: Real-time bidirectional communication with VLC for better control and error feedback.

**Technical Details**:
- VLC provides a WebSocket endpoint at `/echo`
- Authentication flow:
  1. GET `/wsticket` → Receive challenge token (string)
  2. Connect to WebSocket at `/echo`
  3. Send challenge token for authentication
- **Reference**: [VLC Android Remote Access Server](https://code.videolan.org/videolan/vlc-android/-/tree/master/application/remote-access-server/src/main/java/org/videolan/vlc/remoteaccessserver?ref_type=heads)

**Benefits**:
- ✅ Real-time playback status (playing, paused, stopped)
- ✅ Live position tracking (current time, duration)
- ✅ Error feedback (codec errors, file not found, network issues)
- ✅ Advanced controls (play, pause, stop, seek, volume)
- ✅ Playlist management
- ✅ Detect when video finishes playing

**Implementation Steps**:
1. Add VLC WebSocket client in `internal/vlc/websocket.go`
2. Implement `/wsticket` challenge-response flow
3. Create WebSocket connection manager
4. Parse VLC WebSocket messages (JSON format)
5. Broadcast VLC events to frontend via our WebSocket
6. Add playback controls to UI (play/pause/stop buttons)
7. Show real-time playback progress bar

#### 2. Mobile Application (Medium Priority)
**Goal**: Native mobile app to share links directly to the video downloader.

**Features**:
- 📱 Android app (later iOS)
- 🔗 Register as "Share" target for video URLs
- ⚡ One-tap sharing from YouTube, Twitch, browsers
- 🎯 Pre-fills URL in download form
- 📡 Uses same backend API
- 🔔 Push notifications for download completion

**Technical Stack**:
- Flutter or React Native for cross-platform
- Deep linking for URL interception
- WebSocket connection for real-time updates
- Local storage for server configuration

**User Flow**:
1. User sees video in YouTube app
2. Tap "Share" → Select "Coloc Downloader"
3. App opens with URL pre-filled
4. Choose download mode and submit
5. Notification when download completes

#### 3. Additional Future Features
- 🎵 Audio-only download option (extract MP3)
- 📝 Download history with search
- 🏷️ Custom video naming and metadata
- 🗂️ Categories and playlists
- 👥 Multi-user support with authentication
- 📊 Download statistics and analytics
- 🌙 Dark mode theme
- 🔐 HTTPS support with Let's Encrypt
- 🎨 Custom quality selection (1080p, 720p, etc.)
- 📺 Support for more video sources (Vimeo, Dailymotion, etc.)

### ✅ Recently Completed
- **Retry Failed Downloads**: One-click retry button for failed jobs
- **Error Handling**: User-friendly error messages (no more HTTP 500)
- **Jobs Persistence**: Error jobs stay in queue until manually cleared
- **FFmpeg Progress**: Real-time encoding progress for downloaded videos
- **Queue Synchronization**: All clients see same queue state
- **Clear Queue**: Remove completed/failed jobs
- **yt-dlp Auto-update**: Automatic version checking and updates
- **VLC 4-digit Authentication**: Secure code-based auth flow

## 📝 License

MIT License - See LICENSE file for details

## 🤝 Contributing

Contributions are welcome! Please read [DEVELOPMENT.md](docs/DEVELOPMENT.md) for guidelines.

### Priority Development Areas
1. **VLC WebSocket Integration** - Most impactful feature
2. **Mobile App** - Improves user experience significantly
3. **Error Handling** - Always room for improvement
4. **UI/UX Polish** - Make it more intuitive
