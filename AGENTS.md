# AGENTS.md

## Build/Test Commands

**Testing (Docker required):**
```bash
# Start services for testing
docker compose up -d --build

# Run unit tests
go test ./internal/... ./pkg/... -v

# Run single test
go test ./internal/download -v -run TestSpecificFunction

# Run integration tests (requires running server)
go test ./test/... -v -run TestFrontendIntegration
```

**Build:**
```bash
# Local build
go build -o server cmd/coloc/main.go

# Docker build
docker compose build
```

**Lint/Format:**
```bash
go fmt ./...
go vet ./...
```

## Code Style Guidelines

**Go:**
- Package naming: lowercase, single word (e.g., `download`, `handlers`)
- Types: PascalCase (e.g., `DownloadJob`, `WSMessage`)
- Functions: PascalCase for exported, camelCase for private
- Variables: camelCase (e.g., `downloadID`, `jobStatus`)
- Use structured logging with logrus
- Error handling: return errors, use fmt.Errorf for wrapping

**JavaScript:**
- ES6 modules, 2-space indentation
- camelCase for variables/functions, UPPER_SNAKE_CASE for constants
- Single quotes for strings

**Project Structure:**
- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public packages
- `web/static/` - Frontend assets
- Test files: `*_test.go` next to source code

**Imports:**
- Group standard library, third-party, and internal imports
- Use absolute imports for internal packages