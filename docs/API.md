# API Reference

> Complete HTTP and WebSocket API documentation for the Coloc Video Downloader.

## Table of Contents

1. [HTTP API](#http-api)
   - [Static Assets](#static-assets)
   - [Download Endpoints](#download-endpoints)
   - [Queue Management](#queue-management)
   - [VLC Integration](#vlc-integration)
   - [Video Management](#video-management)
2. [WebSocket API](#websocket-api)
3. [Data Models](#data-models)
4. [Error Handling](#error-handling)

---

## HTTP API

Base URL: `http://localhost:8080`

### Static Assets

#### GET `/`
Serves the main HTML page.

**Response:**
- Content-Type: `text/html; charset=utf-8`
- Body: HTML content

---

#### GET `/styles.css`
Serves the CSS stylesheet.

**Response:**
- Content-Type: `text/css; charset=utf-8`
- Cache-Control: `public, max-age=3600`
- Body: CSS content

---

#### GET `/app.js`
Serves the JavaScript application.

**Response:**
- Content-Type: `application/javascript; charset=utf-8`
- Cache-Control: `public, max-age=3600`
- Body: JavaScript content

---

### Download Endpoints

#### POST `/url`
Download a direct video URL.

**Request Body:**
```json
{
  "url": "https://example.com/video.mp4",
  "autoPlay": true,
  "vlcUrl": "http://192.168.1.100:8080",
  "backendUrl": "http://localhost:8080"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Téléchargement démarré",
  "file": "/videos/video.mp4"
}
```

**Status Codes:**
- `200 OK` - Download started successfully
- `400 Bad Request` - Invalid request body
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

#### POST `/urlyt`
Download or stream a YouTube video.

**Request Body:**
```json
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "mode": "download",
  "autoPlay": true,
  "vlcUrl": "http://192.168.1.100:8080",
  "backendUrl": "http://localhost:8080"
}
```

**Parameters:**
- `url` (string, required): YouTube video URL
- `mode` (string, optional): "download" or "stream" (default: "download")
- `autoPlay` (boolean, optional): Auto-play on VLC after download (default: false)
- `vlcUrl` (string, optional): VLC server URL for auto-play
- `backendUrl` (string, optional): This server's URL for video access

**Response:**
```json
{
  "success": true,
  "message": "Téléchargement ajouté à la file d'attente (ID: dl_1234567890_123456789)",
  "downloadId": "dl_1234567890_123456789"
}
```

**Status Codes:**
- `200 OK` - Job added to queue
- `400 Bad Request` - Invalid URL or parameters
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

#### POST `/twitch`
Extract and stream a Twitch live stream.

**Request Body:**
```json
{
  "url": "https://www.twitch.tv/username",
  "autoPlay": true,
  "vlcUrl": "http://192.168.1.100:8080"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Stream Twitch extrait et envoyé à VLC",
  "streamUrl": "https://video-weaver.example.com/..."
}
```

**Status Codes:**
- `200 OK` - Stream extracted and sent to VLC
- `400 Bad Request` - Invalid URL or VLC not configured
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Extraction failed

---

#### POST `/playurl`
Play a direct URL on VLC without downloading.

**Request Body:**
```json
{
  "url": "https://example.com/video.mp4",
  "vlcUrl": "http://192.168.1.100:8080"
}
```

**Response:**
```json
{
  "success": true,
  "message": "URL envoyée à VLC pour lecture"
}
```

**Status Codes:**
- `200 OK` - URL sent to VLC
- `400 Bad Request` - Invalid URL or VLC not configured
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - VLC communication failed

---

### Queue Management

#### GET `/queue`
Get current download queue status.

**Response:**
```json
{
  "dl_1234567890_123456789": {
    "job": {
      "id": "dl_1234567890_123456789",
      "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
      "mode": "download",
      "autoPlay": true,
      "vlcUrl": "http://192.168.1.100:8080",
      "backendUrl": "http://localhost:8080",
      "createdAt": "2025-10-26T21:00:00Z"
    },
    "status": "downloading",
    "progress": "[download] 45.2% of 123.45MiB at 2.34MiB/s ETA 00:25",
    "error": "",
    "completedAt": "0001-01-01T00:00:00Z",
    "streamUrl": ""
  }
}
```

**Status Values:**
- `queued` - Waiting to be processed
- `processing` - Being processed (extracting info)
- `downloading` - Actively downloading
- `completed` - Download finished
- `error` - Download failed

**Status Codes:**
- `200 OK` - Queue status returned
- `500 Internal Server Error` - Server error

---

#### POST `/queue/clear`
Clear completed and error jobs from the queue.

**Response:**
```json
{
  "success": true,
  "message": "File d'attente nettoyée"
}
```

**Status Codes:**
- `200 OK` - Queue cleared
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

### VLC Integration

#### GET `/vlc/code`
Request a VLC authentication code.

**Query Parameters:**
- `vlcUrl` (string, required): VLC server URL

**Example:**
```
GET /vlc/code?vlcUrl=http://192.168.1.100:8080
```

**Response:**
```json
{
  "success": true,
  "code": "1234",
  "message": "Code d'authentification généré. Entrez ce code dans VLC."
}
```

**Status Codes:**
- `200 OK` - Code generated successfully
- `400 Bad Request` - Missing or invalid VLC URL
- `500 Internal Server Error` - VLC communication failed

**Notes:**
- The 4-digit code is displayed in VLC's interface
- User must enter this code to complete authentication
- Code expires after a short period (VLC-defined)

---

#### POST `/vlc/verify-code`
Verify a VLC authentication code.

**Request Body:**
```json
{
  "vlcUrl": "http://192.168.1.100:8080",
  "code": "1234"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Authentification VLC réussie"
}
```

**Status Codes:**
- `200 OK` - Authentication successful
- `400 Bad Request` - Invalid code or VLC URL
- `401 Unauthorized` - Code verification failed
- `500 Internal Server Error` - VLC communication failed

**Notes:**
- Session cookie is stored server-side
- Cookie is associated with the VLC URL
- Cookie never exposed to client

---

#### GET `/vlc/play`
Send a play command to VLC.

**Query Parameters:**
- `vlcUrl` (string, required): VLC server URL
- `videoUrl` (string, required): Video URL to play

**Example:**
```
GET /vlc/play?vlcUrl=http://192.168.1.100:8080&videoUrl=http://localhost:8080/videos/video.mp4
```

**Response:**
```json
{
  "success": true,
  "message": "Vidéo envoyée à VLC"
}
```

**Status Codes:**
- `200 OK` - Play command sent successfully
- `400 Bad Request` - Missing parameters
- `401 Unauthorized` - Not authenticated with VLC
- `500 Internal Server Error` - VLC communication failed

---

#### GET `/vlc/status`
Get VLC authentication status.

**Response:**
```json
{
  "authenticated": true,
  "vlcUrl": "http://192.168.1.100:8080"
}
```

**Status Codes:**
- `200 OK` - Status returned

---

#### GET `/vlc/config`
Get VLC configuration.

**Response:**
```json
{
  "vlcUrl": "http://192.168.1.100:8080"
}
```

**Status Codes:**
- `200 OK` - Configuration returned

---

#### POST `/vlc/config`
Save VLC configuration.

**Request Body:**
```json
{
  "vlcUrl": "http://192.168.1.100:8080"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration VLC sauvegardée"
}
```

**Status Codes:**
- `200 OK` - Configuration saved
- `400 Bad Request` - Invalid VLC URL
- `500 Internal Server Error` - Server error

---

### Video Management

#### GET `/list`
List all downloaded videos.

**Response:**
```json
[
  "video1.mp4",
  "video2.webm",
  "video3.mkv"
]
```

**Status Codes:**
- `200 OK` - Video list returned
- `500 Internal Server Error` - Server error

**Notes:**
- Returns only filenames, not full paths
- Videos are in `/videos` directory
- Sorted alphabetically

---

#### GET `/videos/{filename}`
Serve a downloaded video file.

**Example:**
```
GET /videos/video1.mp4
```

**Response:**
- Content-Type: Detected from file extension
- Body: Video file content

**Status Codes:**
- `200 OK` - Video file served
- `404 Not Found` - Video not found
- `500 Internal Server Error` - Server error

**Notes:**
- Supports range requests for video seeking
- Proper MIME type detection
- Direct file serving (no buffering)

---

## WebSocket API

### Connection

**Endpoint:** `ws://localhost:8080/ws`

**Protocol:** WebSocket (RFC 6455)

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

---

### Client → Server Messages

#### Subscribe to Specific Download

```json
{
  "action": "subscribe",
  "downloadId": "dl_1234567890_123456789"
}
```

**Purpose:** Receive updates only for a specific download.

---

#### Subscribe to All Downloads

```json
{
  "action": "subscribeAll"
}
```

**Purpose:** Receive updates for all downloads and queue changes.

---

### Server → Client Messages

#### Progress Update

```json
{
  "type": "progress",
  "downloadId": "dl_1234567890_123456789",
  "line": "[download] 45.2% of 123.45MiB at 2.34MiB/s ETA 00:25",
  "percent": 45.2
}
```

**When:** During active download, sent for each progress line from yt-dlp.

**Fields:**
- `type`: Always "progress"
- `downloadId`: Unique download identifier
- `line`: Raw progress line from yt-dlp
- `percent`: Download percentage (0-100)

---

#### Download Complete

```json
{
  "type": "done",
  "downloadId": "dl_1234567890_123456789",
  "file": "video.mp4",
  "message": "Téléchargement terminé"
}
```

**When:** Download successfully completed.

**Fields:**
- `type`: Always "done"
- `downloadId`: Unique download identifier
- `file`: Downloaded filename
- `message`: Success message

---

#### Download Error

```json
{
  "type": "error",
  "downloadId": "dl_1234567890_123456789",
  "message": "ERROR: Video unavailable"
}
```

**When:** Download failed.

**Fields:**
- `type`: Always "error"
- `downloadId`: Unique download identifier
- `message`: Error description

---

#### Video List Update

```json
{
  "type": "videoList",
  "videos": [
    "video1.mp4",
    "video2.webm",
    "video3.mkv"
  ]
}
```

**When:** Video directory changes (new download, file deleted).

**Fields:**
- `type`: Always "videoList"
- `videos`: Array of video filenames

---

#### Queue Status Update

```json
{
  "type": "queueStatus",
  "queue": [
    {
      "job": {
        "id": "dl_1234567890_123456789",
        "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
        "mode": "download",
        "autoPlay": true,
        "vlcUrl": "http://192.168.1.100:8080",
        "backendUrl": "http://localhost:8080",
        "createdAt": "2025-10-26T21:00:00Z"
      },
      "status": "downloading",
      "progress": "[download] 45.2% of 123.45MiB at 2.34MiB/s ETA 00:25",
      "error": "",
      "completedAt": "0001-01-01T00:00:00Z",
      "streamUrl": ""
    }
  ]
}
```

**When:** Queue state changes (job added, status updated, job completed).

**Fields:**
- `type`: Always "queueStatus"
- `queue`: Array of job status objects

---

### WebSocket Lifecycle

```javascript
// Connection established
ws.onopen = () => {
  console.log('WebSocket connected');
  
  // Subscribe to all downloads
  ws.send(JSON.stringify({
    action: 'subscribeAll'
  }));
};

// Message received
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch (message.type) {
    case 'progress':
      updateProgress(message.downloadId, message.percent);
      break;
    case 'done':
      showSuccess(message.downloadId, message.file);
      break;
    case 'error':
      showError(message.downloadId, message.message);
      break;
    case 'videoList':
      updateVideoList(message.videos);
      break;
    case 'queueStatus':
      updateQueue(message.queue);
      break;
  }
};

// Connection closed
ws.onclose = () => {
  console.log('WebSocket disconnected');
  // Attempt reconnection
  setTimeout(connectWebSocket, 3000);
};

// Error occurred
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

---

## Data Models

### URLRequest

```typescript
interface URLRequest {
  url: string;         // Video URL
  autoPlay?: boolean;  // Auto-play on VLC (default: false)
  vlcUrl?: string;     // VLC server URL
  backendUrl?: string; // This server's URL
  mode?: string;       // "download" or "stream" (default: "download")
}
```

---

### DownloadJob

```typescript
interface DownloadJob {
  id: string;          // Unique identifier (e.g., "dl_1234567890_123456789")
  url: string;         // Source video URL
  mode: string;        // "download" or "stream"
  autoPlay: boolean;   // Auto-play flag
  vlcUrl: string;      // VLC server URL
  backendUrl: string;  // This server's URL
  createdAt: string;   // ISO 8601 timestamp
}
```

---

### JobStatus

```typescript
interface JobStatus {
  job: DownloadJob;    // Associated job
  status: string;      // "queued" | "processing" | "downloading" | "completed" | "error"
  progress: string;    // Human-readable progress
  error: string;       // Error message (if failed)
  completedAt: string; // ISO 8601 timestamp (if completed)
  streamUrl: string;   // Direct stream URL (for stream mode)
}
```

---

### Response

```typescript
interface Response {
  success: boolean;    // Operation success
  message: string;     // Human-readable message
  file?: string;       // Downloaded filename (if applicable)
  downloadId?: string; // Download ID (if applicable)
  code?: string;       // VLC auth code (if applicable)
}
```

---

### WSMessage

```typescript
interface WSMessage {
  type: string;        // Message type
  downloadId?: string; // Download ID
  line?: string;       // Progress line
  percent?: number;    // Download percentage
  file?: string;       // Filename
  message?: string;    // Status message
  videos?: string[];   // Video list
  queue?: JobStatus[]; // Queue status
}
```

---

## Error Handling

### HTTP Error Responses

All error responses follow this format:

```json
{
  "success": false,
  "message": "Error description"
}
```

### Common HTTP Status Codes

| Code | Meaning | When |
|------|---------|------|
| 200 | OK | Request successful |
| 400 | Bad Request | Invalid parameters or request body |
| 401 | Unauthorized | VLC authentication required |
| 404 | Not Found | Resource not found |
| 405 | Method Not Allowed | Wrong HTTP method |
| 500 | Internal Server Error | Server-side error |

---

### Error Messages

#### Download Errors

- `"URL invalide"` - Invalid or malformed URL
- `"Mode invalide"` - Invalid mode (must be "download" or "stream")
- `"ERROR: Video unavailable"` - Video not accessible
- `"ERROR: This video is private"` - Video is private
- `"ERROR: Unable to download webpage"` - Network error

#### VLC Errors

- `"URL VLC manquante"` - VLC URL not provided
- `"Code d'authentification invalide"` - Invalid auth code
- `"VLC non authentifié"` - VLC authentication required
- `"Échec de l'envoi à VLC"` - Failed to communicate with VLC

#### Queue Errors

- `"Échec de la récupération du statut de la file"` - Failed to get queue status
- `"Échec du nettoyage de la file"` - Failed to clear queue

---

### WebSocket Error Handling

```javascript
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
  // Connection will close, triggering onclose
};

ws.onclose = (event) => {
  if (event.code === 1000) {
    // Normal closure
    console.log('WebSocket closed normally');
  } else {
    // Abnormal closure
    console.error('WebSocket closed abnormally:', event.code, event.reason);
    // Attempt reconnection
    setTimeout(connectWebSocket, 3000);
  }
};
```

**WebSocket Close Codes:**
- `1000` - Normal closure
- `1001` - Going away (server shutdown)
- `1006` - Abnormal closure (connection lost)
- `1011` - Internal server error

---

## Rate Limiting

Currently, there is **no rate limiting** implemented. For production use, consider:

- Limiting download requests per IP
- Limiting WebSocket connections per IP
- Throttling VLC authentication attempts

---

## CORS Configuration

The server allows WebSocket upgrades from any origin. For production:

```go
upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // Implement origin checking
        return r.Header.Get("Origin") == "https://yourdomain.com"
    },
}
```

---

## API Versioning

Currently, the API is **unversioned**. All endpoints are at the root path. For future versions, consider:

- `/api/v1/download`
- `/api/v1/queue`
- `/api/v1/vlc`

---

This API provides a complete interface for video downloading, queue management, and VLC integration.
