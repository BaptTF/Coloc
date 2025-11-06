import { elements, initializeElements } from './utils.js';
import { setupEventListeners } from './events.js';
import { WebSocketManager } from './websocket.js';
import { VlcManager } from './vlc.js';
import { VideoManager } from './video.js';
import { ModalManager } from './modal.js';
import { DownloadManager } from './download.js';
import { StatusManager } from './status.js';
import { PWAManager } from './pwa.js';
import { toast } from './toast.js';

// ===== INITIALIZATION =====
async function initializeApp() {
  // Initialize DOM elements
  initializeElements();

  // Set default backend URL
  if (elements.backendUrl) {
    elements.backendUrl.value = window.location.origin;
  }

  // Setup event listeners
  setupEventListeners();

  // Initialize PWA features
  if (window.pwaManager) {
    console.log('[App] PWA Manager initialized');
  }

  // Connect WebSocket
  WebSocketManager.connect();

  // Load server state first to establish baseline
  await loadServerState();
  
  // Then load VLC config (which may override server state if more recent)
  await VlcManager.loadConfig();
  
  // Load videos list
  await VideoManager.listVideos();

  // Fetch VLC state after initial load
  await VlcManager.fetchInitialState();
}

// Load server state from backend
async function loadServerState() {
  try {
    const response = await fetch('/api/state');
    if (!response.ok) throw new Error('Failed to load server state');
    
    const state = await response.json();
    
    // Update yt-dlp status from server
    if (state.ytdlp) {
      StatusManager.updateYtdlpStatus(state.ytdlp.status, state.ytdlp.message);
    }
    
    // Update VLC status from server
    if (state.vlc && state.vlc.sessions) {
      if (state.vlc.sessions.length > 0) {
        const hasAuthenticatedSession = state.vlc.sessions.some(s => s.authenticated);
        if (hasAuthenticatedSession) {
          StatusManager.updateVlcStatus('authenticated');
        } else {
          // Has sessions but none authenticated - can connect
          StatusManager.updateVlcStatus('connectable');
        }
      } else {
        // No VLC sessions at all
        StatusManager.updateVlcStatus('disconnected');
      }
    } else {
      // No VLC data available
      StatusManager.updateVlcStatus('disconnected');
    }
    
    // Sync autoplay setting with server
    if (state.autoPlay !== undefined && elements.autoPlay) {
      elements.autoPlay.checked = state.autoPlay;
    }
  } catch (error) {
    console.error('Failed to load server state:', error);
  }
}

// ===== CSS INJECTION FOR ANIMATIONS =====
function injectAdditionalStyles() {
  const style = document.createElement('style');
  style.textContent = `
    @keyframes slideOut {
      from {
        transform: translateX(0);
        opacity: 1;
      }
      to {
        transform: translateX(100%);
        opacity: 0;
      }
    }
  `;
  document.head.appendChild(style);
}

// ===== APP STARTUP =====
document.addEventListener('DOMContentLoaded', () => {
  
  
  // Wait for CSS to load before initializing
  function initializeWhenReady() {
    if (document.body.classList.contains('styles-loaded')) {
      injectAdditionalStyles();
      initializeApp().catch(error => {
        console.error('Failed to initialize app:', error);
      });
    } else {
      // CSS not loaded yet, wait a bit and try again
      setTimeout(initializeWhenReady, 10);
    }
  }

  initializeWhenReady();
});

// ===== CLEANUP ON PAGE UNLOAD =====
window.addEventListener('beforeunload', () => {
  if (WebSocketManager && typeof WebSocketManager.disconnect === 'function') {
    WebSocketManager.disconnect();
  }
});

// ===== GLOBAL EXPORTS FOR COMPATIBILITY =====
window.VlcManager = VlcManager;
window.VideoManager = VideoManager;
window.ModalManager = ModalManager;
window.DownloadManager = DownloadManager;
window.WebSocketManager = WebSocketManager;
window.PWAManager = PWAManager;
window.toastManager = toast;
