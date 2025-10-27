# Development Guide

> Setup, testing, and contribution guidelines for the Coloc Video Downloader.

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Building](#building)
4. [Testing](#testing)
5. [Code Style](#code-style)
6. [Contributing](#contributing)
7. [Troubleshooting](#troubleshooting)

---

## Development Setup

### Prerequisites

**Required:**
- Go 1.24.6 or later
- Docker and Docker Compose
- Git

**Optional:**
- VLC media player (for testing VLC integration)
- Make (for build automation)

---

### Local Development Setup

#### 1. Clone the Repository

```bash
git clone <repository-url>
cd Coloc
```

#### 2. Install Go Dependencies

```bash
go mod download
```

#### 3. Run Locally (Without Docker)

```bash
# Build the application
go build -o server cmd/coloc/main.go

# Run the server
./server
```

The server will start on `http://localhost:8080`.

**Note:** yt-dlp and ffmpeg are automatically installed by the `go-ytdlp` library on first run.

---

#### 4. Run with Docker (Recommended)

```bash
# Build and start
docker compose up -d --build

# View logs
docker compose logs -f video-server

# Stop
docker compose down
```

---

### IDE Setup

#### VS Code

Recommended extensions:
- **Go** (golang.go)
- **Docker** (ms-azuretools.vscode-docker)
- **ESLint** (dbaeumer.vscode-eslint) - for JavaScript

**Settings** (`.vscode/settings.json`):
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  }
}
```

---

#### GoLand / IntelliJ IDEA

1. Open project
2. Enable Go modules support
3. Configure Go SDK (1.24.6+)
4. Enable "Format on save"

---

## Project Structure

```
Coloc/
├── cmd/
│   └── coloc/
│       └── main.go              # Application entry point
│
├── internal/                    # Private application code
│   ├── types/
│   │   ├── types.go            # Data structures
│   │   └── types_test.go       # Unit tests
│   │
│   ├── vlc/
│   │   └── vlc.go              # VLC integration
│   │
│   ├── download/
│   │   ├── download.go         # Download processing
│   │   └── download_test.go    # Unit tests
│   │
│   ├── websocket/
│   │   └── websocket.go        # WebSocket handling
│   │
│   └── handlers/
│       ├── handlers.go         # HTTP handlers
│       ├── download_handlers.go
│       ├── vlc_handlers.go
│       └── handlers_test.go    # Unit tests
│
├── pkg/
│   └── config/
│       ├── config.go           # Global configuration
│       └── config_test.go      # Unit tests
│
├── web/
│   ├── static/
│   │   ├── index.html          # Main HTML page
│   │   ├── styles.css          # CSS styles
│   │   └── app.js              # Legacy JS (compatibility)
│   └── embed.go                # Embedded file declarations
│
├── js/                          # Frontend JavaScript modules
│   ├── config.js
│   ├── state.js
│   ├── utils.js
│   ├── toast.js
│   ├── api.js
│   ├── websocket.js
│   ├── download.js
│   ├── status.js
│   ├── vlc.js
│   ├── modal.js
│   ├── video.js
│   ├── events.js
│   └── app.js
│
├── test/                        # Integration tests
│   ├── integration_test.go
│   └── websocket_progress_test.go
│
├── docs/                        # Documentation
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── DEVELOPMENT.md          # This file
│
├── videos/                      # Downloaded videos (gitignored)
│   └── .gitkeep
│
├── docker-compose.yml           # Docker Compose configuration
├── Dockerfile                   # Container build instructions
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── .gitignore                   # Git ignore patterns
└── README.md                    # Project overview
```

---

## Building

### Local Build

```bash
# Build for current platform
go build -o server cmd/coloc/main.go

# Build for Linux (from macOS/Windows)
GOOS=linux GOARCH=amd64 go build -o server cmd/coloc/main.go

# Build with optimizations
go build -ldflags="-s -w" -o server cmd/coloc/main.go
```

**Flags:**
- `-ldflags="-s -w"` - Strip debug information (smaller binary)
- `-o server` - Output filename

---

### Docker Build

```bash
# Build Docker image
docker compose build

# Build without cache
docker compose build --no-cache

# Build specific service
docker build -t coloc-video-server .
```

---

### Multi-stage Dockerfile

The Dockerfile uses multi-stage builds for optimization:

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o video-server cmd/coloc/main.go

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tini
COPY --from=builder /app/video-server /usr/local/bin/
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["video-server"]
```

**Benefits:**
- Smaller final image (~15MB vs ~500MB)
- No build tools in production image
- Faster deployment

---

## Testing

### Unit Tests

Unit tests are located next to the code they test (`*_test.go`).

```bash
# Run all unit tests
go test ./internal/... ./pkg/...

# Run with verbose output
go test -v ./internal/... ./pkg/...

# Run with coverage
go test -cover ./internal/... ./pkg/...

# Generate coverage report
go test -coverprofile=coverage.out ./internal/... ./pkg/...
go tool cover -html=coverage.out
```

---

### Integration Tests

Integration tests require a running server.

```bash
# Start the server
docker compose up -d

# Wait for server to be ready
sleep 5

# Run integration tests
go test ./test/... -v

# Run specific test
go test ./test/... -v -run TestFrontendIntegration
go test ./test/... -v -run TestAPIEndpoints
go test ./test/... -v -run TestWebSocketProgressUpdates
```

**Test Suites:**

1. **TestFrontendIntegration** - Tests static file serving
2. **TestAPIEndpoints** - Tests HTTP API endpoints
3. **TestWebSocketProgressUpdates** - Tests WebSocket communication
4. **TestVLCAuthIntegration** - Tests VLC authentication (requires VLC)

---

### Writing Tests

#### Unit Test Example

```go
package download

import (
    "testing"
)

func TestGenerateDownloadID(t *testing.T) {
    id := GenerateDownloadID()
    
    if id == "" {
        t.Error("Expected non-empty download ID")
    }
    
    if len(id) < 10 {
        t.Errorf("Expected download ID length >= 10, got %d", len(id))
    }
}
```

#### Integration Test Example

```go
package integration

import (
    "net/http"
    "testing"
)

func TestAPIEndpoint(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    resp, err := http.Get("http://localhost:8080/list")
    if err != nil {
        t.Fatalf("Failed to call endpoint: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
```

---

### Test Coverage Goals

- **Unit Tests**: > 70% coverage
- **Integration Tests**: All critical paths covered
- **E2E Tests**: Main user flows verified

---

## Code Style

### Go Code Style

Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines.

#### Formatting

```bash
# Format all Go files
go fmt ./...

# Check formatting
gofmt -l .
```

#### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

#### Naming Conventions

- **Packages**: lowercase, single word (e.g., `download`, `handlers`)
- **Files**: lowercase with underscores (e.g., `download_handlers.go`)
- **Types**: PascalCase (e.g., `DownloadJob`, `WSMessage`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase (e.g., `downloadID`, `jobStatus`)
- **Constants**: PascalCase or UPPER_CASE (e.g., `VideoDir`, `MAX_RETRIES`)

#### Comments

```go
// Package download handles video download processing.
package download

// DownloadJob represents a video download task.
type DownloadJob struct {
    ID  string // Unique identifier
    URL string // Source video URL
}

// ProcessDownloadJob processes a download job from start to finish.
// It updates the job status and broadcasts progress via WebSocket.
func ProcessDownloadJob(job *DownloadJob) {
    // Implementation
}
```

---

### JavaScript Code Style

#### Formatting

- **Indentation**: 2 spaces
- **Quotes**: Single quotes for strings
- **Semicolons**: Required
- **Line length**: Max 100 characters

#### Naming Conventions

- **Variables**: camelCase (e.g., `downloadId`, `wsManager`)
- **Constants**: UPPER_SNAKE_CASE (e.g., `WS_RECONNECT_DELAY`)
- **Classes**: PascalCase (e.g., `WebSocketManager`, `ApiClient`)
- **Functions**: camelCase (e.g., `connectWebSocket`, `updateProgress`)

#### Module Structure

```javascript
// config.js
export const CONFIG = {
  WS_RECONNECT_DELAY: 3000,
};

// api.js
class ApiClient {
  async request(url, options) {
    // Implementation
  }
}

export const api = new ApiClient();

// app.js
import { CONFIG } from './config.js';
import { api } from './api.js';

document.addEventListener('DOMContentLoaded', () => {
  // Initialization
});
```

---

### CSS Code Style

- **Indentation**: 2 spaces
- **Naming**: kebab-case (e.g., `.download-btn`, `.queue-item`)
- **Organization**: Group related styles
- **Variables**: Use CSS custom properties

```css
:root {
  --primary-color: #4a90e2;
  --secondary-color: #50c878;
  --error-color: #e74c3c;
}

.download-btn {
  background-color: var(--primary-color);
  color: white;
  padding: 10px 20px;
  border-radius: 5px;
}
```

---

## Contributing

### Workflow

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```
3. **Make your changes**
4. **Write tests**
5. **Run tests**
   ```bash
   go test ./...
   ```
6. **Format code**
   ```bash
   go fmt ./...
   ```
7. **Commit changes**
   ```bash
   git commit -m "Add feature: description"
   ```
8. **Push to your fork**
   ```bash
   git push origin feature/my-feature
   ```
9. **Create a Pull Request**

---

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(download): add support for Twitch streams

Add Twitch stream extraction using yt-dlp.
Streams are sent directly to VLC without downloading.

Closes #123
```

```
fix(websocket): handle connection closure gracefully

Prevent panic when WebSocket connection is closed unexpectedly.
Add proper error handling and reconnection logic.
```

---

### Pull Request Guidelines

**Before submitting:**
- [ ] Code follows style guidelines
- [ ] Tests pass (`go test ./...`)
- [ ] New features have tests
- [ ] Documentation updated
- [ ] Commit messages follow convention
- [ ] No merge conflicts

**PR Description should include:**
- What changes were made
- Why the changes were made
- How to test the changes
- Screenshots (if UI changes)
- Related issues

---

### Code Review Process

1. **Automated checks** run on PR
2. **Reviewer assigned** by maintainer
3. **Review feedback** addressed
4. **Approval** from maintainer
5. **Merge** to main branch

---

## Troubleshooting

### Common Issues

#### "yt-dlp not found"

**Solution:** yt-dlp is auto-installed on first run. If it fails:
```bash
# Manually install yt-dlp
pip install yt-dlp

# Or use the system package manager
brew install yt-dlp  # macOS
apt install yt-dlp   # Ubuntu/Debian
```

---

#### "Port 8080 already in use"

**Solution:** Change the port in `docker-compose.yml`:
```yaml
ports:
  - "8081:8080"  # Host:Container
```

Or stop the conflicting service:
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

---

#### "Cannot connect to VLC"

**Checklist:**
- [ ] VLC web interface enabled
- [ ] VLC password set
- [ ] VLC URL correct (e.g., `http://192.168.1.100:8080`)
- [ ] Firewall not blocking connection
- [ ] VLC running on the specified machine

**Test VLC connection:**
```bash
curl http://192.168.1.100:8080/requests/status.json
```

---

#### "WebSocket connection failed"

**Checklist:**
- [ ] Server running (`docker compose ps`)
- [ ] No proxy blocking WebSocket
- [ ] Browser console shows no errors
- [ ] Correct WebSocket URL (`ws://localhost:8080/ws`)

**Debug:**
```javascript
// Browser console
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => console.log('Connected');
ws.onerror = (e) => console.error('Error:', e);
ws.onclose = (e) => console.log('Closed:', e.code, e.reason);
```

---

#### "Tests fail with timeout"

**Solution:** Increase test timeout:
```bash
go test ./test/... -v -timeout 60s
```

Or run tests individually:
```bash
go test ./test/... -v -run TestFrontendIntegration
```

---

#### "Docker build fails"

**Solution:** Clear Docker cache and rebuild:
```bash
docker compose down
docker system prune -a
docker compose build --no-cache
docker compose up -d
```

---

### Debug Mode

Enable debug logging:

```bash
# Environment variable
export LOG_LEVEL=debug

# Or in docker-compose.yml
environment:
  - LOG_LEVEL=debug
```

**Log levels:**
- `debug` - Verbose logging
- `info` - Normal logging (default)
- `warn` - Warnings only
- `error` - Errors only

---

### Performance Profiling

#### CPU Profiling

```go
import _ "net/http/pprof"

// In main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiles at `http://localhost:6060/debug/pprof/`

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

---

### Useful Commands

```bash
# Check Go version
go version

# List dependencies
go list -m all

# Update dependencies
go get -u ./...
go mod tidy

# Check for vulnerabilities
go list -json -m all | nancy sleuth

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o server-linux cmd/coloc/main.go
GOOS=darwin GOARCH=amd64 go build -o server-macos cmd/coloc/main.go
GOOS=windows GOARCH=amd64 go build -o server-windows.exe cmd/coloc/main.go

# Docker cleanup
docker compose down -v  # Remove volumes
docker system prune -a  # Remove all unused images
```

---

## Resources

### Documentation
- [Go Documentation](https://golang.org/doc/)
- [Docker Documentation](https://docs.docker.com/)
- [WebSocket RFC](https://tools.ietf.org/html/rfc6455)
- [yt-dlp Documentation](https://github.com/yt-dlp/yt-dlp)

### Tools
- [Go Playground](https://play.golang.org/)
- [JSON Formatter](https://jsonformatter.org/)
- [WebSocket Test Client](https://www.websocket.org/echo.html)

### Community
- [Go Forum](https://forum.golangbridge.org/)
- [Stack Overflow - Go](https://stackoverflow.com/questions/tagged/go)
- [Reddit - r/golang](https://www.reddit.com/r/golang/)

---

Happy coding! 🚀
