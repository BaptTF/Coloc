import { CONFIG } from './config.js';
import { elements, setLoadingState } from './utils.js';
import { state } from './state.js';
import { toast } from './toast.js';
import { ApiClient } from './api.js';
import { ModalManager } from './modal.js';
import { StatusManager } from './status.js';

// ===== VLC FUNCTIONALITY =====
class VlcManager {
  static websocket = null;
  static reconnectAttempts = 0;
  static maxReconnectAttempts = 5;
  static reconnectDelay = 2000; // 2 seconds
  static heartbeatInterval = null;

  static async testConnection() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('Veuillez entrer l\'URL du serveur VLC', 'error');
      return;
    }

    try {
      const response = await fetch(vlcUrl + '/', {
        method: 'GET',
        mode: 'cors'
      });

      const isConnected = response.ok || response.status === 401;
      StatusManager.updateVlcStatus(isConnected ? 'connectable' : 'disconnected');

      if (isConnected) {
        toast.show('VLC accessible', 'success');
      } else {
        toast.show(`VLC inaccessible: Status ${response.status}`, 'error');
      }
    } catch (error) {
      StatusManager.updateVlcStatus('disconnected');
      toast.show(`VLC inaccessible: ${error.message}`, 'error');
    }
  }

  static async login() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('Veuillez entrer l\'URL du serveur VLC', 'error');
      return;
    }

    try {
      await VlcManager.saveConfig();
      const response = await ApiClient.get(`${CONFIG.endpoints.vlcCode}?vlc=${encodeURIComponent(vlcUrl)}`);

      if (response.success) {
        state.vlcChallenge = response.file; // The challenge is in the 'file' field
        ModalManager.show('vlcModal');
        elements.vlcCode?.focus();
      } else {
        toast.show(`Erreur VLC: ${response.message}`, 'error');
      }
    } catch (error) {
      console.error('VLC login error:', error);
      toast.show(`Impossible de contacter VLC: ${String(error)}`, 'error');
    }
  }

  static async verifyCode() {
    const code = elements.vlcCode?.value?.trim();
    if (!code || code.length !== 4) {
      toast.show('Veuillez entrer un code à 4 chiffres', 'error');
      return;
    }

    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('URL VLC manquante', 'error');
      return;
    }

    try {
      const response = await ApiClient.post(`${CONFIG.endpoints.vlcVerify}?vlc=${encodeURIComponent(vlcUrl)}`, {
        code: code
      });

      if (response.success) {
        StatusManager.updateVlcStatus('authenticated');
        toast.show('Authentification VLC réussie', 'success');
        ModalManager.hide('vlcModal');
        
        // Automatically connect to WebSocket after successful authentication
        setTimeout(async () => {
          const connected = await VlcManager.connectWebSocket();
          if (connected) {
            toast.show('Connexion WebSocket VLC automatique établie', 'success');
          }
        }, 1000);
      } else {
        toast.show(`Erreur authentification: ${response.message}`, 'error');
      }
    } catch (error) {
      console.error('VLC verify error:', error);
      toast.show('Code incorrect', 'error');
    }
  }

  static async playVideo(filename) {
    if (!state.vlcAuthenticated) {
      toast.show('Veuillez d\'abord vous authentifier avec VLC', 'error');
      return;
    }

    const vlcUrl = elements.vlcUrl?.value?.trim();
    const backendUrl = elements.backendUrl?.value?.trim();

    if (!vlcUrl || !backendUrl) {
      toast.show('URLs VLC et backend requises', 'error');
      return;
    }

    try {
      const videoPath = `${backendUrl}/videos/${encodeURIComponent(filename)}`;
      const url = `${CONFIG.endpoints.vlcPlay}?vlc=${encodeURIComponent(vlcUrl)}&id=-1&path=${encodeURIComponent(videoPath)}&type=stream`;

      const response = await ApiClient.get(url);

      if (response.success) {
        toast.show(`Lecture lancée: ${filename}`, 'success');
      } else {
        toast.show(`Erreur VLC: ${response.message}`, 'error');
      }
    } catch (error) {
      console.error('VLC play error:', error);
      if (String(error).includes('401') || String(error).includes('403')) {
        StatusManager.updateVlcStatus('connectable');
        toast.show('Session VLC expirée, reconnectez-vous', 'error');
      } else {
        toast.show(`Erreur lecture VLC: ${String(error)}`, 'error');
      }
    }
  }

  static async loadConfig() {
    try {
      const configs = await ApiClient.get(CONFIG.endpoints.vlcConfig);
      if (configs.length > 0) {
        const lastConfig = configs[configs.length - 1];
        if (elements.vlcUrl) {
          elements.vlcUrl.value = lastConfig.url;
        }

        if (lastConfig.authenticated) {
          StatusManager.updateVlcStatus('authenticated');
          toast.show('Configuration VLC restaurée', 'success');
          
          // Automatically connect to WebSocket when authenticated config is loaded
          setTimeout(async () => {
            const connected = await VlcManager.connectWebSocket();
            if (connected) {
              toast.show('Connexion WebSocket VLC automatique établie', 'success');
            }
          }, 1000);
        }
      }
    } catch (error) {
      console.log('Aucune configuration VLC sauvegardée');
    }
  }

  static async saveConfig() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) return;

    try {
      await ApiClient.post(CONFIG.endpoints.vlcConfig, { url: vlcUrl });
    } catch (error) {
      console.error('Erreur sauvegarde config VLC:', error);
    }
  }

  // ===== WEBSOCKET FUNCTIONALITY =====
  static async connectWebSocket() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('URL VLC requise pour la connexion WebSocket', 'error');
      return;
    }

    try {
      // Update status to connecting
      StatusManager.updateVlcWebSocketStatus(false, 'Connexion en cours...');
      
      // First connect via HTTP API
      const response = await ApiClient.post(`${CONFIG.endpoints.vlcWsConnect}?vlc=${encodeURIComponent(vlcUrl)}`, {});
      
      if (response.success) {
        toast.show('Connexion WebSocket VLC établie', 'success');
        StatusManager.updateVlcWebSocketStatus(true);
        VlcManager.startHeartbeat();
        VlcManager.reconnectAttempts = 0;
        return true;
      } else {
        toast.show(`Erreur connexion WebSocket: ${response.message}`, 'error');
        StatusManager.updateVlcWebSocketStatus(false);
        return false;
      }
    } catch (error) {
      console.error('WebSocket connection error:', error);
      toast.show(`Erreur connexion WebSocket: ${String(error)}`, 'error');
      StatusManager.updateVlcWebSocketStatus(false);
      return false;
    }
  }

  static async disconnectWebSocket() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) return;

    try {
      await ApiClient.post(`${CONFIG.endpoints.vlcWsDisconnect}?vlc=${encodeURIComponent(vlcUrl)}`, {});
      this.stopHeartbeat();
      StatusManager.updateVlcWebSocketStatus(false);
      toast.show('Connexion WebSocket VLC fermée', 'info');
    } catch (error) {
      console.error('WebSocket disconnect error:', error);
    }
  }

  static async getWebSocketStatus() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) return null;

    try {
      const response = await ApiClient.get(`${CONFIG.endpoints.vlcWsStatus}?vlc=${encodeURIComponent(vlcUrl)}`);
      return response;
    } catch (error) {
      console.error('WebSocket status error:', error);
      return null;
    }
  }

  static async sendWebSocketCommand(command, params = {}) {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('URL VLC requise pour les commandes WebSocket', 'error');
      return;
    }

    try {
      // Build payload according to VLC WebSocket command structure
      const payload = { command };
      
      if (params.id !== undefined) {
        payload.id = params.id;
      }
      if (params.floatValue !== undefined) {
        payload.floatValue = params.floatValue;
      }
      if (params.longValue !== undefined) {
        payload.longValue = params.longValue;
      }
      if (params.stringValue !== undefined) {
        payload.stringValue = params.stringValue;
      }

      console.log('Sending VLC command:', payload);
      const response = await ApiClient.post(`${CONFIG.endpoints.vlcWsControl}?vlc=${encodeURIComponent(vlcUrl)}`, payload);
      
      if (response.success) {
        console.log(`VLC command ${command} sent successfully`);
        // Only show toast for user-initiated actions, not automatic updates
        if (!params.silent) {
          toast.show(`Commande VLC: ${command}`, 'success');
        }
      } else {
        toast.show(`Erreur commande VLC: ${response.message}`, 'error');
      }
      return response;
    } catch (error) {
      console.error('VLC command error:', error);
      toast.show(`Erreur commande VLC: ${String(error)}`, 'error');
      return null;
    }
  }

  static startHeartbeat() {
    VlcManager.stopHeartbeat(); // Clear any existing interval
    VlcManager.heartbeatInterval = setInterval(async () => {
      const status = await VlcManager.getWebSocketStatus();
      if (status) {
        StatusManager.updateVlcWebSocketStatus(status.connected);
        if (!status.connected) {
          console.warn('WebSocket connection lost, attempting to reconnect...');
          VlcManager.stopHeartbeat();
          await VlcManager.attemptReconnect();
        }
      }
    }, 5000); // Check every 5 seconds for more responsive updates
  }

  static stopHeartbeat() {
    if (VlcManager.heartbeatInterval) {
      clearInterval(VlcManager.heartbeatInterval);
      VlcManager.heartbeatInterval = null;
    }
  }

  static async attemptReconnect() {
    if (VlcManager.reconnectAttempts >= VlcManager.maxReconnectAttempts) {
      StatusManager.updateVlcWebSocketStatus(false, 'Échec de reconnexion');
      toast.show('Impossible de reconnecter WebSocket après plusieurs tentatives', 'error');
      return;
    }

    VlcManager.reconnectAttempts++;
    StatusManager.updateVlcWebSocketStatus(false, `Reconnexion... (${VlcManager.reconnectAttempts}/${VlcManager.maxReconnectAttempts})`);
    console.log(`Attempting WebSocket reconnection ${VlcManager.reconnectAttempts}/${VlcManager.maxReconnectAttempts}`);
    
    await new Promise(resolve => setTimeout(resolve, VlcManager.reconnectDelay));
    
    const connected = await VlcManager.connectWebSocket();
    if (!connected) {
      await VlcManager.attemptReconnect();
    }
  }

  // ===== STATE MANAGEMENT =====
  static async fetchInitialState() {
    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      console.warn('No VLC URL available for state fetch');
      return false;
    }

    try {
      console.log('Fetching VLC state from proxy...');
      const response = await ApiClient.get(`${CONFIG.endpoints.vlcState}?vlc=${encodeURIComponent(vlcUrl)}`);
      
      console.log('Raw VLC state response:', response);
      
      if (response) {
        // Update global state with fetched data (response is direct, not wrapped in data)
        state.vlcState = {
          status: response.vlc_status || null,
          queue: response.vlc_queue || null,
          volume: response.vlc_volume || null,
          lastUpdate: response.last_update ? new Date(response.last_update) : null,
          hasState: response.has_state || false,
          wsConnected: response.ws_connected || false
        };

        console.log('VLC state fetched:', state.vlcState);

        // Restore UI state if we have valid data
        if (state.vlcState.hasState) {
          VlcManager.restoreUIFromState();
          return true;
        }
      }
    } catch (error) {
      console.warn('Failed to fetch VLC state:', error);
    }
    
    return false;
  }

  static restoreUIFromState() {
    const { status, volume } = state.vlcState;
    
    // Restore playback status
    if (status) {
      VlcManager.updateNowPlaying(status);
    }
    
    // Restore volume
    if (volume && volume.volume >= 0) {
      VlcManager.updateVolume(volume);
    }
    
    console.log('UI restored from persisted VLC state');
  }

  // ===== REMOTE CONTROL COMMANDS =====
  static async play() {
    return await VlcManager.sendWebSocketCommand('play');
  }

  static async pause() {
    return await VlcManager.sendWebSocketCommand('pause');
  }

  

  static async seek(position) {
    // Convert percentage to milliseconds (assuming position is 0-100)
    const duration = state.vlcDuration || 0;
    const milliseconds = Math.floor((position / 100) * duration);
    // VLC Android expects the time value in the 'id' field for SET_PROGRESS
    return await VlcManager.sendWebSocketCommand('set-progress', { id: milliseconds });
  }

  static async setVolume(volume) {
    // Volume uses ID parameter (0-100)
    return await VlcManager.sendWebSocketCommand('set-volume', { id: volume });
  }

  static async next() {
    return await VlcManager.sendWebSocketCommand('next');
  }

  static async previous() {
    return await VlcManager.sendWebSocketCommand('previous');
  }

  // ===== VLC STATUS UPDATE METHODS =====
  static updateNowPlaying(vlcStatus) {
    console.log('VLC Now Playing:', vlcStatus);
    
    // Update UI elements
    const titleEl = document.getElementById('currentTitle');
    const stateEl = document.getElementById('playbackState');
    const durationEl = document.getElementById('currentDuration');
    const volumeEl = document.getElementById('volumeValue');
    const volumeSlider = document.getElementById('volumeSlider');
    const seekSlider = document.getElementById('seekSlider');

    if (titleEl && vlcStatus.title) {
      titleEl.textContent = vlcStatus.title;
    }

    if (stateEl) {
      stateEl.textContent = vlcStatus.playing ? 'Lecture' : 'Pause';
    }

    if (durationEl && vlcStatus.duration > 0) {
      durationEl.textContent = VlcManager.formatTime(vlcStatus.duration);
    }

    if (volumeEl && vlcStatus.volume >= 0) {
      volumeEl.textContent = `${vlcStatus.volume}%`;
    }

    if (volumeSlider && vlcStatus.volume >= 0) {
      volumeSlider.value = vlcStatus.volume;
    }

    if (seekSlider && vlcStatus.duration > 0) {
      const progress = (vlcStatus.progress / vlcStatus.duration) * 100;
      seekSlider.value = progress;
      document.getElementById('seekValue').textContent = VlcManager.formatTime(vlcStatus.progress);
    }

    // Update playback state
    state.vlcPlaying = vlcStatus.playing;
    state.vlcCurrentTitle = vlcStatus.title;
    state.vlcVolume = vlcStatus.volume;
    state.vlcProgress = vlcStatus.progress;
    state.vlcDuration = vlcStatus.duration;
  }

  static updatePlayerStatus(vlcStatus) {
    console.log('VLC Player Status:', vlcStatus);
    
    const stateEl = document.getElementById('playbackState');
    if (stateEl) {
      stateEl.textContent = vlcStatus.playing ? 'Lecture' : 'Pause';
    }

    state.vlcPlaying = vlcStatus.playing;
  }

  static updatePlayQueue(vlcQueue) {
    console.log('VLC Play Queue:', vlcQueue);
    // TODO: Update queue UI if we add one
    state.vlcQueue = vlcQueue.medias;
  }

  static updateVolume(volume) {
    console.log('VLC Volume Update:', volume);
    
    const volumeEl = document.getElementById('volumeValue');
    const volumeSlider = document.getElementById('volumeSlider');

    if (volumeEl) {
      volumeEl.textContent = `${volume}%`;
    }

    if (volumeSlider) {
      volumeSlider.value = volume;
    }

    state.vlcVolume = volume;
  }

  static updateLegacyStatus(vlcStatus) {
    console.log('VLC Legacy Status:', vlcStatus);
    // Handle legacy status messages
    VlcManager.updateNowPlaying(vlcStatus);
  }

  static formatTime(milliseconds) {
    if (!milliseconds || milliseconds < 0) return '0:00';
    
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  }
}

const vlcManager = new VlcManager();

export { VlcManager, vlcManager };
