# Stage 1: Build Go application
FROM golang:1.24.6 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o video-server main.go

# Stage 2: Download yt-dlp static binary
FROM alpine:latest AS downloader

RUN apk add --no-cache curl ca-certificates \
    && curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /yt-dlp \
    && chmod +x /yt-dlp

# Stage 3: Final scratch image
FROM python:latest

# Copy the Go binary
COPY --from=builder /app/video-server /video-server

# Copy yt-dlp static binary
COPY --from=downloader /yt-dlp /yt-dlp

# Expose port
EXPOSE 8080

# Run the application
CMD ["/video-server"]
