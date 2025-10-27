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
