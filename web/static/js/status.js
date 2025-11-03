import { elements } from './utils.js';
import { state } from './state.js';

// ===== STATUS MANAGEMENT =====
class StatusManager {
  static updateVlcStatus(status) {
    const statusEl = elements.vlcStatus;
    if (!statusEl) return;

    let className, text;

    switch(status) {
      case 'authenticated':
        className = 'status status-connected';
        text = 'Authentifié';
        state.vlcAuthenticated = true;
        break;
      case 'connectable':
        className = 'status status-connectable';
        text = 'Connectable';
        state.vlcAuthenticated = false;
        break;
      case 'disconnected':
      default:
        className = 'status status-disconnected';
        text = 'Déconnecté';
        state.vlcAuthenticated = false;
        break;
    }

    statusEl.className = className;
    statusEl.innerHTML = `
      <div class="status-dot"></div>
      ${text}
    `;

    // Update login button text
    const loginBtn = document.getElementById('loginVlc');
    if (loginBtn) {
      loginBtn.textContent = state.vlcAuthenticated ? 'Reconnexion' : 'Se connecter';
    }

    // Show/hide VLC remote control section
    const vlcRemoteSection = document.getElementById('vlcRemoteSection');
    if (vlcRemoteSection) {
      if (status === 'authenticated') {
        vlcRemoteSection.style.display = 'block';
      } else {
        vlcRemoteSection.style.display = 'none';
      }
    }
  }

  static updateWebSocketStatus(connected, text = '') {
    const statusDot = elements.wsStatusDot;
    const statusText = elements.wsStatusText;
    
    if (statusDot) {
      statusDot.className = 'status-dot ' + (connected ? 'connected' : '');
    }
    
    if (statusText) {
      statusText.textContent = text || (connected ? 'Connecté' : 'Non connecté');
    }
    
    state.wsConnected = connected;
  }

  static updateVlcWebSocketStatus(connected, text = '') {
    const statusDot = document.getElementById('vlcWsStatusDot');
    const statusText = document.getElementById('vlcWsStatusText');
    
    if (statusDot) {
      statusDot.className = 'status-dot ' + (connected ? 'connected' : 'disconnected');
    }
    
    if (statusText) {
      if (!text) {
        text = connected ? 'VLC WebSocket connecté' : 'VLC WebSocket déconnecté';
      }
      statusText.textContent = text;
    }
    
    state.vlcWebSocketConnected = connected;
  }

  static updateVlcPlaybackStatus(status) {
    const titleEl = elements.currentTitle;
    const durationEl = elements.currentDuration;
    const stateEl = elements.playbackState;
    
    if (titleEl && status.title) {
      titleEl.textContent = status.title;
    }
    
    if (durationEl && status.length) {
      const duration = Math.floor(status.length / 1000);
      const minutes = Math.floor(duration / 60);
      const seconds = duration % 60;
      durationEl.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }
    
    if (stateEl && status.state) {
      const stateMap = {
        'playing': 'Lecture',
        'paused': 'Pause',
        'stopped': 'Arrêté'
      };
      stateEl.textContent = stateMap[status.state] || status.state;
    }
    
    // Update seek slider if we have time info
    if (elements.seekSlider && status.time && status.length) {
      const position = (status.time / status.length) * 100;
      elements.seekSlider.value = position;
      
      const currentTime = Math.floor(status.time / 1000);
      const currentMinutes = Math.floor(currentTime / 60);
      const currentSeconds = currentTime % 60;
      elements.seekValue.textContent = `${currentMinutes}:${currentSeconds.toString().padStart(2, '0')}`;
    }
    
    // Update volume slider
    if (elements.volumeSlider && status.volume !== undefined) {
      elements.volumeSlider.value = status.volume;
      elements.volumeValue.textContent = status.volume + '%';
    }
  }

  static updateYtdlpStatus(status, message = '') {
    const statusEl = elements.ytdlpStatus;
    if (!statusEl) return;

    let className, text;

    switch(status) {
      case 'checking':
        className = 'status status-updating';
        text = 'Vérification...';
        break;
      case 'updating':
        className = 'status status-updating';
        text = 'Mise à jour...';
        break;
      case 'updated':
        className = 'status status-updated';
        text = 'À jour';
        break;
      case 'uptodate':
        className = 'status status-updated';
        text = 'À jour';
        break;
      case 'error':
        className = 'status status-update-error';
        text = 'Erreur';
        break;
      case 'unknown':
        className = 'status status-connectable';
        text = message || 'En attente...';
        break;
      default:
        className = 'status status-updating';
        text = message || 'yt-dlp...';
        break;
    }

    statusEl.className = className;
    statusEl.innerHTML = `
      <div class="status-dot"></div>
      ${text}
    `;
  }
}

export { StatusManager };
