import { CONFIG } from './config.js';
import { state } from './state.js';
import { toast } from './toast.js';
import { DownloadManager } from './download.js';
import { VideoManager } from './video.js';
import { StatusManager } from './status.js';

// ===== WEBSOCKET MANAGEMENT =====
class WebSocketManager {
  static vlcMessageTimeout = null;
  static lastVlcMessageTime = null;

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

        // Always attempt to reconnect with exponential backoff
        state.reconnectAttempts++;
        const delay = Math.min(1000 * Math.pow(2, state.reconnectAttempts), 30000); // Max 30 seconds
        console.log(`Reconnecting in ${delay/1000}s... (attempt ${state.reconnectAttempts})`);
        setTimeout(() => WebSocketManager.connect(), delay);
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

    // Check if this is a VLC-related message and update last message time
    if (message.type && message.type.startsWith('vlc_')) {
      WebSocketManager.lastVlcMessageTime = Date.now();
      WebSocketManager.resetVlcMessageTimeout();
    }

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
      case 'vlc_now_playing':
        // Comprehensive now playing information
        if (message.vlcStatus) {
          VlcManager.updateNowPlaying(message.vlcStatus);
          // Update global state
          state.vlcState.status = message.vlcStatus;
          state.vlcState.lastUpdate = new Date();
          state.vlcState.hasState = true;
          // Update WebSocket connection status - receiving messages means connection is active
          StatusManager.updateVlcWebSocketStatus(true);
        }
        break;
      case 'vlc_player_status':
        // Simple playing/paused status
        if (message.vlcStatus) {
          VlcManager.updatePlayerStatus(message.vlcStatus);
          // Update WebSocket connection status - receiving messages means connection is active
          StatusManager.updateVlcWebSocketStatus(true);
        }
        break;
      case 'vlc_play_queue':
        // Play queue update
        if (message.vlcQueue) {
          VlcManager.updatePlayQueue(message.vlcQueue);
          // Update global state
          state.vlcState.queue = message.vlcQueue;
          state.vlcState.lastUpdate = new Date();
        }
        break;
      case 'vlc_volume_update':
        // Volume update
        if (message.vlcVolume) {
          VlcManager.updateVolume(message.vlcVolume.volume);
          // Update global state
          state.vlcState.volume = message.vlcVolume;
          state.vlcState.lastUpdate = new Date();
          // Update WebSocket connection status - receiving messages means connection is active
          StatusManager.updateVlcWebSocketStatus(true);
        }
        break;
      case 'vlc_error':
        // VLC error
        toast.show(message.message || 'VLC Error', 'error');
        console.error('VLC Error:', message.vlcError);
        break;
      case 'vlc_auth':
        // VLC authentication status
        console.log('VLC Auth:', message.vlcAuth);
        if (message.vlcAuth && message.vlcAuth.status === 'ok') {
          StatusManager.updateVlcStatus('authenticated');
          // Update WebSocket connection status - receiving auth messages means connection is active
          StatusManager.updateVlcWebSocketStatus(true);
        } else if (message.vlcAuth && message.vlcAuth.status === 'forbidden') {
          // Only show forbidden as error if it's initial auth, not command response
          if (!message.vlcAuth.initialMessage || message.vlcAuth.initialMessage === 'null') {
            console.warn('VLC authentication forbidden');
            StatusManager.updateVlcStatus('connectable');
            // Update WebSocket connection status - receiving auth messages means connection is active
            StatusManager.updateVlcWebSocketStatus(true);
          } else {
            // Command forbidden - this is normal when no media is playing
            console.debug('VLC command forbidden (no media playing):', message.vlcAuth.initialMessage);
            // Update WebSocket connection status - receiving messages means connection is active
            StatusManager.updateVlcWebSocketStatus(true);
          }
        }
        break;
      case 'vlc_login_needed':
        // VLC login required
        StatusManager.updateVlcStatus('connectable');
        toast.show('VLC login required', 'warning');
        break;
      case 'vlc_resume_confirmation':
        // Resume playback confirmation
        if (message.message) {
          toast.show(message.message, 'info');
        }
        break;
      case 'vlc_browser_description':
        // Browser description
        console.log('VLC Browser:', message.message);
        break;
      case 'vlc_playback_control_forbidden':
        // Playback control forbidden
        toast.show('Playback control forbidden', 'error');
        break;
      case 'vlc_ml_refresh_needed':
        // Media library refresh needed
        console.log('VLC ML refresh needed');
        break;
      case 'vlc_network_shares':
        // Network shares discovered
        console.log('VLC Network shares:', message.message);
        break;
      case 'vlc_status_legacy':
        // Legacy VLC status
        if (message.vlcStatus) {
          VlcManager.updateLegacyStatus(message.vlcStatus);
          // Update WebSocket connection status - receiving messages means connection is active
          StatusManager.updateVlcWebSocketStatus(true);
        }
        break;
      case 'vlc_unhandled_message':
        // Unhandled VLC message type (for debugging)
        console.warn('Unhandled VLC message:', message.message);
        // Update WebSocket connection status - receiving any VLC message means connection is active
        StatusManager.updateVlcWebSocketStatus(true);
        break;
    }
  }

  static resetVlcMessageTimeout() {
    // Clear existing timeout
    if (WebSocketManager.vlcMessageTimeout) {
      clearTimeout(WebSocketManager.vlcMessageTimeout);
    }

    // Set new timeout to detect connection loss (30 seconds without messages)
    WebSocketManager.vlcMessageTimeout = setTimeout(() => {
      console.warn('No VLC WebSocket messages received for 30 seconds, connection may be lost');
      StatusManager.updateVlcWebSocketStatus(false);
    }, 30000);
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
    // Note: We don't actually disconnect anymore - WebSocket should always stay connected
    console.log('WebSocket disconnect requested, but keeping connection alive for auto-reconnection');
    // Only clear the reference, don't actually close the connection
    if (state.websocket) {
      state.websocket = null;
    }
    state.wsConnected = false;
  }
}

const wsManager = new WebSocketManager();

export { WebSocketManager, wsManager };
