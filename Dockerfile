# Stage 1: Build Go application
FROM golang:1.24.6 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o video-server main.go

# Install yt-dlp and ffmpeg tools
RUN ./video-server install-tools

# Stage 2: Final minimal image
FROM denoland/deno:debian

# Install ca-certificates for HTTPS downloads
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy installed tools from builder
COPY --from=builder /root/.cache/go-ytdlp /root/.cache/go-ytdlp

# Copy the Go binary
COPY --from=builder /app/video-server /video-server

# Expose port
EXPOSE 8080

# Run the application
CMD ["/video-server"]
