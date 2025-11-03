# Stage 1: Tools Builder
FROM golang:1.24.6 AS tools-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Copy and build tools installer
COPY cmd/tools/ ./cmd/tools/
RUN CGO_ENABLED=1 GOOS=linux go build -o tools-installer cmd/tools/main.go

# Install yt-dlp and ffmpeg tools
RUN ./tools-installer

# Stage 2: Server Builder
FROM golang:1.24.6 AS server-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build server
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o video-server cmd/coloc/main.go

# Stage 3: Final minimal image
FROM denoland/deno:debian

# Install ca-certificates for HTTPS downloads
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy installed tools from tools-builder
COPY --from=tools-builder /root/.cache/go-ytdlp /root/.cache/go-ytdlp

# Copy server binary from server-builder
COPY --from=server-builder /app/video-server /video-server

# Expose port
EXPOSE 8080

# Run the application
CMD ["/video-server"]
