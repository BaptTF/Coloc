import { CONFIG } from './config.js';
import { state } from './state.js';
import { toast } from './toast.js';
import { DownloadManager } from './download.js';
import { VideoManager } from './video.js';
import { StatusManager } from './status.js';

// ===== WEBSOCKET MANAGEMENT =====
class WebSocketManager {
  static connect() {
    if (state.websocket && state.websocket.readyState === WebSocket.OPEN) {
      return; // Already connected
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${CONFIG.endpoints.websocket}`;

    console.log('Connecting to WebSocket:', wsUrl);

    try {
      state.websocket = new WebSocket(wsUrl);

      state.websocket.onopen = () => {
        console.log('WebSocket connected');
        state.wsConnected = true;
        state.reconnectAttempts = 0;
        WebSocketManager.updateStatus('connected');

        // Subscribe to all downloads and get current queue status
        WebSocketManager.send({ action: 'subscribeAll' });
      };

      state.websocket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          WebSocketManager.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
          console.error('Raw message data:', event.data);
        }
      };

      state.websocket.onclose = () => {
        console.log('WebSocket disconnected');
        state.wsConnected = false;
        WebSocketManager.updateStatus('disconnected');

        // Attempt to reconnect
        if (state.reconnectAttempts < state.maxReconnectAttempts) {
          state.reconnectAttempts++;
          console.log(`Reconnecting... (${state.reconnectAttempts}/${state.maxReconnectAttempts})`);
          setTimeout(() => WebSocketManager.connect(), 2000 * state.reconnectAttempts);
        }
      };

      state.websocket.onerror = (error) => {
        console.error('WebSocket error:', error);
        WebSocketManager.updateStatus('disconnected');
      };

    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      WebSocketManager.updateStatus('disconnected');
    }
  }

  static send(message) {
    if (state.websocket && state.websocket.readyState === WebSocket.OPEN) {
      state.websocket.send(JSON.stringify(message));
    }
  }

  static handleMessage(message) {
    // Defensive check for message structure
    if (!message || typeof message !== 'object') {
      console.warn('Invalid message received:', message);
      return;
    }

    console.log('WebSocket message received:', message);

    switch (message.type) {
      case 'queued':
        DownloadManager.updateDownload(message.downloadId, {
          status: 'queued',
          message: message.message
        });
        break;
      case 'progress':
        DownloadManager.updateDownload(message.downloadId, {
          status: 'downloading',
          progress: message.message, // Use the progress message from backend
          percent: message.percent || 0
        });
        break;
      case 'done':
        DownloadManager.updateDownload(message.downloadId, {
          status: 'completed',
          message: message.message,
          file: message.file,
          percent: 100
        });
        // Refresh video list
        VideoManager.listVideos();
        break;
      case 'error':
        DownloadManager.updateDownload(message.downloadId, {
          status: 'error',
          message: message.message
        });
        break;
      case 'completed':
        // Job completed (either success or error) - update the queue display
        DownloadManager.updateDownload(message.downloadId, {
          status: message.message.includes('succès') ? 'completed' : 'error'
        });
        break;
      case 'queueStatus':
        // Full queue status update - replace entire queue
        console.log('Received queueStatus message:', message);
        console.log('Queue data:', message.queue);
        DownloadManager.updateQueueStatus(message.queue || []);
        break;
      case 'list':
        VideoManager.renderVideoGrid(message.videos);
        break;
      case 'done':
        // Job successfully completed
        DownloadManager.updateDownload(message.downloadId, {
          status: 'completed',
          message: message.message || 'Téléchargement terminé',
          file: message.file
        });
        toast.show('Téléchargement terminé avec succès!', 'success');
        break;
      case 'ytdlp':
      case 'ytdlp_update':
        // Handle yt-dlp update status messages
        StatusManager.updateYtdlpStatus('updating', message.message);
        // Show toast notification for important update messages
        if (message.message.includes('mis à jour')) {
          toast.show(message.message, 'success');
          StatusManager.updateYtdlpStatus('updated');
        } else if (message.message.includes('Erreur')) {
          toast.show(message.message, 'error');
          StatusManager.updateYtdlpStatus('error');
        } else if (message.message.includes('à jour')) {
          StatusManager.updateYtdlpStatus('uptodate');
        } else {
          StatusManager.updateYtdlpStatus('checking', message.message);
        }
        break;
      case 'vlc_authenticated':
        // Update VLC status for all clients when someone authenticates
        StatusManager.updateVlcStatus('authenticated');
        toast.show(message.message, 'success');
        break;
    }
  }

  static updateStatus(status) {
    // Update WebSocket status in header if we add it
    const statusEl = document.querySelector('.ws-status');
    if (statusEl) {
      statusEl.className = `ws-status ${status}`;
      statusEl.textContent = status === 'connected' ? 'WebSocket connecté' : 'WebSocket déconnecté';
    }
  }

  static disconnect() {
    if (state.websocket) {
      state.websocket.close();
      state.websocket = null;
    }
  }
}

const wsManager = new WebSocketManager();

export { WebSocketManager, wsManager };
