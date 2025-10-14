// ===== CONSTANTS & CONFIG =====
const CONFIG = {
  endpoints: {
    direct: '/url',
    youtube: '/urlyt',
    list: '/list',
    vlcCode: '/vlc/code',
    vlcVerify: '/vlc/verify-code',
    vlcPlay: '/vlc/play',
    vlcConfig: '/vlc/config'
  },
  selectors: {
    backendUrl: '#backendUrl',
    vlcUrl: '#vlcUrl',
    videoUrl: '#videoUrl',
    autoPlay: '#autoPlay',
    vlcStatus: '#vlcStatus',
    vlcAuthStatus: '#vlcAuthStatus',
    videosGrid: '#videosGrid',
    vlcModal: '#vlcModal',
    vlcCode: '#vlcCode'
  },
  classes: {
    loading: 'loading',
    disabled: 'disabled',
    active: 'active'
  }
};

// ===== STATE MANAGEMENT =====
const state = {
  vlcChallenge: null,
  vlcAuthenticated: false,
  isLoading: false
};

// ===== DOM ELEMENTS =====
const elements = {};

// ===== UTILITY FUNCTIONS =====
function initializeElements() {
  Object.keys(CONFIG.selectors).forEach(key => {
    elements[key] = document.querySelector(CONFIG.selectors[key]);
  });
}

function createElement(tag, className = '', textContent = '') {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (textContent) element.textContent = textContent;
  return element;
}

function setLoadingState(isLoading) {
  state.isLoading = isLoading;
  const buttons = document.querySelectorAll('.btn');
  buttons.forEach(btn => {
    btn.disabled = isLoading;
    if (isLoading && !btn.querySelector('.spinner')) {
      const spinner = createElement('div', 'spinner');
      btn.insertBefore(spinner, btn.firstChild);
    } else if (!isLoading) {
      const spinner = btn.querySelector('.spinner');
      if (spinner) spinner.remove();
    }
  });
}

// ===== TOAST SYSTEM =====
class ToastManager {
  constructor() {
    this.container = this.createContainer();
    this.toasts = new Map();
  }

  createContainer() {
    let container = document.querySelector('.toast-container');
    if (!container) {
      container = createElement('div', 'toast-container');
      document.body.appendChild(container);
    }
    return container;
  }

  show(message, type = 'info', duration = 5000) {
    const id = Date.now() + Math.random();
    const toast = this.createToast(message, type, id);
    
    this.container.appendChild(toast);
    this.toasts.set(id, toast);

    // Auto-remove after duration
    if (duration > 0) {
      setTimeout(() => this.remove(id), duration);
    }

    return id;
  }

  createToast(message, type, id) {
    const toast = createElement('div', `toast toast-${type}`);
    
    const icon = this.getIcon(type);
    const iconEl = createElement('div', 'toast-icon');
    iconEl.innerHTML = icon;

    const content = createElement('div', 'toast-content', message);
    
    const closeBtn = createElement('button', 'toast-close');
    closeBtn.innerHTML = '×';
    closeBtn.onclick = () => this.remove(id);

    toast.append(iconEl, content, closeBtn);
    return toast;
  }

  getIcon(type) {
    const icons = {
      success: '✓',
      error: '✗',
      warning: '⚠',
      info: 'ℹ'
    };
    return icons[type] || icons.info;
  }

  remove(id) {
    const toast = this.toasts.get(id);
    if (toast) {
      toast.style.animation = 'slideOut 0.3s ease forwards';
      setTimeout(() => {
        if (toast.parentNode) {
          toast.parentNode.removeChild(toast);
        }
        this.toasts.delete(id);
      }, 300);
    }
  }

  clear() {
    this.toasts.forEach((_, id) => this.remove(id));
  }
}

// ===== API UTILITIES =====
class ApiClient {
  static async request(url, options = {}) {
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers
        },
        ...options
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const contentType = response.headers.get('content-type');
      const responseText = await response.text();
      
      // Only try to parse as JSON if it's actually JSON content type and valid JSON
      if (contentType && contentType.includes('application/json')) {
        try {
          return JSON.parse(responseText);
        } catch (jsonError) {
          console.warn('Response claimed to be JSON but failed to parse:', responseText);
          return responseText;
        }
      }
      
      // For non-JSON responses, return the text directly
      return responseText;
    } catch (error) {
      console.error('API Request failed:', error);
      throw error;
    }
  }

  static async post(url, data) {
    return this.request(url, {
      method: 'POST',
      body: JSON.stringify(data)
    });
  }

  static async get(url) {
    return this.request(url, { method: 'GET' });
  }
}

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

  // Legacy methods for compatibility
  static updateVlcConnection(isConnected) {
    this.updateVlcStatus(isConnected ? 'connectable' : 'disconnected');
  }

  static updateVlcAuth(isAuthenticated, message = '') {
    this.updateVlcStatus(isAuthenticated ? 'authenticated' : 'connectable');
  }
}

// ===== VLC FUNCTIONALITY =====
class VlcManager {
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
      StatusManager.updateVlcConnection(isConnected);
      
      if (isConnected) {
        toast.show('VLC accessible', 'success');
      } else {
        toast.show(`VLC inaccessible: Status ${response.status}`, 'error');
      }
    } catch (error) {
      StatusManager.updateVlcConnection(false);
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
      const challenge = await ApiClient.get(`${CONFIG.endpoints.vlcCode}?vlc=${encodeURIComponent(vlcUrl)}`);
      state.vlcChallenge = challenge;
      ModalManager.show('vlcModal');
      elements.vlcCode?.focus();
    } catch (error) {
      toast.show(`Impossible de contacter VLC: ${error.message}`, 'error');
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
      await ApiClient.post(`${CONFIG.endpoints.vlcVerify}?vlc=${encodeURIComponent(vlcUrl)}`, {
        code: code
      });

      StatusManager.updateVlcAuth(true);
      toast.show('Authentification VLC réussie', 'success');
      ModalManager.hide('vlcModal');
    } catch (error) {
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
      
      await ApiClient.get(url);
      toast.show(`Lecture lancée: ${filename}`, 'success');
    } catch (error) {
      if (error.message.includes('401') || error.message.includes('403')) {
        StatusManager.updateVlcAuth(false, 'Session expirée');
        toast.show('Session VLC expirée, reconnectez-vous', 'error');
      } else {
        toast.show(`Erreur lecture VLC: ${error.message}`, 'error');
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
          StatusManager.updateVlcAuth(true, 'Authentifié (persistant)');
          toast.show('Configuration VLC restaurée', 'success');
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
}

// ===== MODAL MANAGEMENT =====
class ModalManager {
  static show(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.classList.add(CONFIG.classes.active);
      document.body.style.overflow = 'hidden';
    }
  }

  static hide(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.classList.remove(CONFIG.classes.active);
      document.body.style.overflow = '';
      
      // Clear code input if it exists
      if (modalId === 'vlcModal' && elements.vlcCode) {
        elements.vlcCode.value = '';
      }
    }
  }

  static setupEventListeners() {
    // Close modal on background click
    document.addEventListener('click', (e) => {
      if (e.target.classList.contains('modal')) {
        const modalId = e.target.id;
        if (modalId) {
          ModalManager.hide(modalId);
        }
      }
    });

    // Handle keyboard events for VLC modal
    document.addEventListener('keydown', (e) => {
      const vlcModal = document.getElementById('vlcModal');
      if (vlcModal && vlcModal.classList.contains(CONFIG.classes.active)) {
        if (e.key === 'Enter') {
          e.preventDefault();
          VlcManager.verifyCode();
        } else if (e.key === 'Escape') {
          e.preventDefault();
          ModalManager.hide('vlcModal');
        }
      }
    });
  }
}

// ===== VIDEO MANAGEMENT =====
class VideoManager {
  static async download(endpoint, type) {
    const url = elements.videoUrl?.value?.trim();
    if (!url) {
      toast.show('Veuillez entrer une URL', 'error');
      return;
    }

    setLoadingState(true);
    toast.clear();

    const loadingToast = toast.show('Téléchargement en cours...', 'info', 0);

    try {
      const requestData = {
        url: url,
        autoPlay: elements.autoPlay?.checked || false,
        vlcUrl: elements.vlcUrl?.value?.trim() || '',
        backendUrl: elements.backendUrl?.value?.trim() || ''
      };

      const data = await ApiClient.post(endpoint, requestData);
      
      if (data.success) {
        let message = `✓ ${data.message}`;
        if (data.file) {
          message += ` - ${data.file}`;
        }
        if (requestData.autoPlay && data.file) {
          message += ' 🎬 Auto-play en cours...';
        }
        
        toast.show(message, 'success');
        elements.videoUrl.value = '';
        VideoManager.listVideos();
      } else {
        toast.show(`✗ ${data.message}`, 'error');
      }
    } catch (error) {
      toast.show(`✗ Erreur: ${error.message}`, 'error');
    } finally {
      toast.remove(loadingToast);
      setLoadingState(false);
    }
  }

  static async listVideos() {
    const backendUrl = elements.backendUrl?.value?.trim();
    if (!backendUrl) return;

    try {
      const files = await ApiClient.get(`${backendUrl}${CONFIG.endpoints.list}`);
      VideoManager.renderVideoGrid(files);
    } catch (error) {
      console.error('Erreur listVideos:', error);
      VideoManager.renderVideoGrid(null, error.message);
    }
  }

  static renderVideoGrid(files, errorMessage = null) {
    const grid = elements.videosGrid;
    if (!grid) return;

    grid.innerHTML = '';

    if (errorMessage) {
      grid.innerHTML = `
        <div class="empty-state">
          <div class="empty-icon">⚠️</div>
          <div class="empty-title">Erreur de chargement</div>
          <div class="empty-description">${errorMessage}</div>
        </div>
      `;
      return;
    }

    if (!files || !Array.isArray(files) || files.length === 0) {
      grid.innerHTML = `
        <div class="empty-state">
          <div class="empty-icon">📂</div>
          <div class="empty-title">Aucune vidéo disponible</div>
          <div class="empty-description">Téléchargez votre première vidéo pour commencer</div>
        </div>
      `;
      return;
    }

    files.forEach(filename => {
      const card = VideoManager.createVideoCard(filename);
      grid.appendChild(card);
    });
  }

  static createVideoCard(filename) {
    const card = createElement('div', 'video-card');
    
    const title = createElement('div', 'video-title', filename);
    
    const actions = createElement('div', 'video-actions');
    const playBtn = createElement('button', 'btn btn-accent btn-sm');
    playBtn.innerHTML = '▶ Lancer sur VLC';
    playBtn.onclick = () => VlcManager.playVideo(filename);
    
    actions.appendChild(playBtn);
    card.append(title, actions);
    
    return card;
  }
}

// ===== EVENT HANDLERS =====
function setupEventListeners() {
  // Download buttons
  const directBtn = document.getElementById('downloadDirect');
  const youtubeBtn = document.getElementById('downloadYoutube');
  
  if (directBtn) {
    directBtn.onclick = () => VideoManager.download(CONFIG.endpoints.direct, 'direct');
  }
  
  if (youtubeBtn) {
    youtubeBtn.onclick = () => VideoManager.download(CONFIG.endpoints.youtube, 'youtube');
  }

  // VLC buttons
  const testVlcBtn = document.getElementById('testVlc');
  const loginVlcBtn = document.getElementById('loginVlc');
  
  if (testVlcBtn) {
    testVlcBtn.onclick = VlcManager.testConnection;
  }
  
  if (loginVlcBtn) {
    loginVlcBtn.onclick = VlcManager.login;
  }

  // Modal buttons
  const submitCodeBtn = document.getElementById('submitCode');
  const cancelCodeBtn = document.getElementById('cancelCode');
  
  if (submitCodeBtn) {
    submitCodeBtn.onclick = VlcManager.verifyCode;
  }
  
  if (cancelCodeBtn) {
    cancelCodeBtn.onclick = () => ModalManager.hide('vlcModal');
  }

  // Enter key for download
  if (elements.videoUrl) {
    elements.videoUrl.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        VideoManager.download(CONFIG.endpoints.youtube, 'youtube');
      }
    });
  }

  // Setup modal event listeners
  ModalManager.setupEventListeners();
}

// ===== INITIALIZATION =====
let toast;

async function initializeApp() {
  // Initialize DOM elements
  initializeElements();
  
  // Initialize toast manager
  toast = new ToastManager();
  
  // Set default backend URL
  if (elements.backendUrl) {
    elements.backendUrl.value = window.location.origin;
  }
  
  // Setup event listeners
  setupEventListeners();
  
  // Load initial data
  await Promise.all([
    VlcManager.loadConfig(),
    VideoManager.listVideos()
  ]);
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
  injectAdditionalStyles();
  initializeApp().catch(error => {
    console.error('Failed to initialize app:', error);
  });
});

// ===== GLOBAL EXPORTS FOR COMPATIBILITY =====
window.VlcManager = VlcManager;
window.VideoManager = VideoManager;
window.ModalManager = ModalManager;
