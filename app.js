// ===== CONSTANTS & CONFIG =====
const CONFIG = {
  endpoints: {
    direct: '/url',
    youtube: '/urlyt',
    twitch: '/twitch',
    playurl: '/playurl',
    list: '/list',
    vlcCode: '/vlc/code',
    vlcVerify: '/vlc/verify-code',
    vlcPlay: '/vlc/play',
    vlcConfig: '/vlc/config',
    websocket: '/ws',
    queueClear: '/queue/clear'
  },
  selectors: {
    backendUrl: '#backendUrl',
    vlcUrl: '#vlcUrl',
    videoUrl: '#videoUrl',
    autoPlay: '#autoPlay',
    vlcStatus: '#vlcStatus',
    videosGrid: '#videosGrid',
    vlcModal: '#vlcModal',
    vlcCode: '#vlcCode',
    downloadProgressSection: '#downloadProgressSection',
    downloadsList: '#downloadsList'
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
  isLoading: false,
  websocket: null,
  wsConnected: false,
  downloads: new Map(), // downloadId -> download info
  reconnectAttempts: 0,
  maxReconnectAttempts: 5
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
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }
      
      return await response.text();
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
        DownloadManager.updateQueueStatus(message.queue || []);
        break;
      case 'list':
        VideoManager.renderVideoGrid(message.videos);
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

// ===== DOWNLOAD MANAGEMENT =====
class DownloadManager {
  static createDownload(downloadId, url) {
    const download = {
      id: downloadId,
      url: url,
      status: 'queued',
      progress: 'En attente de traitement',
      percent: 0,
      createdAt: new Date()
    };
    
    state.downloads.set(downloadId, download);
    DownloadManager.renderDownload(download);
    DownloadManager.showProgressSection();
    
    return download;
  }

  static updateDownload(downloadId, updates) {
    const download = state.downloads.get(downloadId);
    if (!download) return;
    
    // Update download object
    Object.assign(download, updates);
    
    // Add line to log if provided (for backward compatibility)
    if (updates.line) {
      if (!download.log) download.log = [];
      download.log.push({
        timestamp: new Date(),
        line: updates.line
      });
    }
    
    // Re-render the download item
    DownloadManager.renderDownload(download);
  }

  static updateQueueStatus(queueData) {
    try {
      // Check if queueData is valid
      if (!queueData || !Array.isArray(queueData)) {
        console.warn('Invalid queue data received:', queueData);
        return;
      }

      // Get current job IDs in the queue
      const currentJobIds = new Set(queueData
        .filter(jobStatus => jobStatus && jobStatus.job && jobStatus.job.id)
        .map(jobStatus => jobStatus.job.id));
      
      // Remove jobs that are no longer in the queue (finished/removed)
      for (const [jobId, download] of state.downloads.entries()) {
        if (!currentJobIds.has(jobId) && (download.status === 'completed' || download.status === 'error')) {
          // Job finished and is no longer in queue - remove from display after a delay
          setTimeout(() => {
            const downloadEl = document.getElementById(`download-${jobId}`);
            if (downloadEl) downloadEl.remove();
            state.downloads.delete(jobId);
            
            // Hide progress section if no active downloads
            if (state.downloads.size === 0) {
              DownloadManager.hideProgressSection();
            }
          }, 3000); // Keep completed/error jobs visible for 3 seconds
        }
      }
      
      // Update or add jobs from queue status
      queueData.forEach(jobStatus => {
        // Validate jobStatus structure
        if (!jobStatus || !jobStatus.job || !jobStatus.job.id || jobStatus.job.id.trim() === '') {
          console.warn('Invalid jobStatus received:', jobStatus);
          return;
        }

        const download = {
          id: jobStatus.job.id,
          url: jobStatus.job.url || '',
          status: jobStatus.status || 'queued',
          progress: jobStatus.progress || 'Traitement en cours',
          percent: jobStatus.status === 'completed' ? 100 : 
                  jobStatus.status === 'error' ? 0 : 0, // Don't show fake progress
          error: jobStatus.error,
          streamUrl: jobStatus.streamUrl,
          completedAt: jobStatus.completedAt ? new Date(jobStatus.completedAt) : null,
          createdAt: jobStatus.job.createdAt ? new Date(jobStatus.job.createdAt) : new Date()
        };
        
        state.downloads.set(jobStatus.job.id, download);
        DownloadManager.renderDownload(download);
      });
      
      // Show progress section if there are any downloads (active or completed/error)
      if (state.downloads.size > 0) {
        DownloadManager.showProgressSection();
      } else {
        DownloadManager.hideProgressSection();
      }
    } catch (error) {
      console.error('Error in updateQueueStatus:', error);
    }
  }

  static renderDownload(download) {
    let downloadEl = document.getElementById(`download-${download.id}`);
    
    if (!downloadEl) {
      downloadEl = createElement('div', 'download-item');
      downloadEl.id = `download-${download.id}`;
      elements.downloadsList.appendChild(downloadEl);
    }
    
    // Update classes based on status
    downloadEl.className = `download-item ${download.status}`;
    
    // Use progress message instead of old message field
    const displayMessage = download.progress || download.message || 'Traitement en cours';
    
    downloadEl.innerHTML = `
      <div class="download-header">
        <div class="download-url" title="${download.url}">${download.url}</div>
        <div class="download-status ${download.status}">
          ${DownloadManager.getStatusText(download.status)}
        </div>
      </div>
      
      <div class="download-progress">
        ${download.status === 'completed' || download.status === 'error' ? `
          <div class="progress-bar">
            <div class="progress-fill ${download.status === 'completed' ? 'completed' : ''}" 
                 style="width: ${download.percent}%"></div>
          </div>
        ` : ''}
        <div class="progress-text">
          <span>${displayMessage}</span>
          ${(download.status === 'completed' || download.status === 'error') ? 
            `<span>${download.percent.toFixed(1)}%</span>` : ''}
        </div>
      </div>
      
      ${download.error ? `
        <div class="download-error">
          <div class="error-icon">⚠️</div>
          <div class="error-message">${download.error}</div>
        </div>
      ` : ''}
      
      ${download.log && download.log.length > 0 ? `
        <div class="download-log">${download.log.map(entry => entry.line).join('\n')}</div>
      ` : ''}
      
      ${(download.status === 'completed' && download.streamUrl) ? `
        <div class="download-actions">
          <button class="btn btn-accent btn-sm" onclick="VlcManager.playVideo('${download.streamUrl}')">
            ▶ Lancer sur VLC
          </button>
        </div>
      ` : ''}
    `;
  }

  static getStatusText(status) {
    const statusTexts = {
      queued: 'En file',
      processing: 'Traitement',
      completed: 'Terminé',
      error: 'Erreur'
    };
    return statusTexts[status] || status;
  }

  static showProgressSection() {
    if (elements.downloadProgressSection) {
      elements.downloadProgressSection.style.display = 'block';
    }
  }

  static hideProgressSection() {
    if (elements.downloadProgressSection && state.downloads.size === 0) {
      elements.downloadProgressSection.style.display = 'none';
    }
  }

  static async clearCompleted() {
    try {
      // Call backend to clear the queue for everyone
      const response = await ApiClient.post(CONFIG.endpoints.queueClear, {});
      
      if (response.success) {
        toast.show('File d\'attente nettoyée', 'success');
        // The backend will broadcast the updated queue status to all clients
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
      }
    } catch (error) {
      console.error('Error clearing queue:', error);
      toast.show(`Erreur lors du nettoyage: ${error.message}`, 'error');
    }
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
      const response = await ApiClient.post(`${CONFIG.endpoints.vlcVerify}?vlc=${encodeURIComponent(vlcUrl)}`, {
        code: code
      });

      if (response.success) {
        StatusManager.updateVlcStatus('authenticated');
        toast.show('Authentification VLC réussie', 'success');
        ModalManager.hide('vlcModal');
      } else {
        toast.show(`Erreur authentification: ${response.message}`, 'error');
      }
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
      
      const response = await ApiClient.get(url);
      
      if (response.success) {
        toast.show(`Lecture lancée: ${filename}`, 'success');
      } else {
        toast.show(`Erreur VLC: ${response.message}`, 'error');
      }
    } catch (error) {
      if (error.message.includes('401') || error.message.includes('403')) {
        StatusManager.updateVlcStatus('connectable');
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
          StatusManager.updateVlcStatus('authenticated');
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
  static async download(endpoint, type, mode = 'stream') {
    const url = elements.videoUrl?.value?.trim();
    if (!url) {
      toast.show('Veuillez entrer une URL', 'error');
      return;
    }

    setLoadingState(true);
    toast.clear();

    try {
      const requestData = {
        url: url,
        autoPlay: elements.autoPlay?.checked || false,
        vlcUrl: elements.vlcUrl?.value?.trim() || '',
        backendUrl: elements.backendUrl?.value?.trim() || ''
      };

      // Add mode for YouTube downloads
      if (endpoint === CONFIG.endpoints.youtube) {
        requestData.mode = mode;
      }

      const response = await ApiClient.post(endpoint, requestData);
      
      if (response.success) {
        toast.show(`${response.message}`, 'success');
        elements.videoUrl.value = '';
        
        // For YouTube downloads, the backend now handles queue management
        // We don't need to create download tracking here anymore
        if (endpoint === CONFIG.endpoints.youtube) {
          // The WebSocket will send us queue updates automatically
          toast.show('Téléchargement ajouté à la file d\'attente', 'info');
        } else {
          // For direct downloads, refresh video list immediately
          VideoManager.listVideos();
        }
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
      }
    } catch (error) {
      toast.show(`Erreur: ${error.message}`, 'error');
    } finally {
      setLoadingState(false);
    }
  }

  static async downloadTwitch() {
    const url = elements.videoUrl?.value?.trim();
    if (!url) {
      toast.show('Veuillez entrer une URL Twitch', 'error');
      return;
    }

    setLoadingState(true);
    toast.clear();

    try {
      const requestData = {
        url: url,
        autoPlay: elements.autoPlay?.checked || false,
        vlcUrl: elements.vlcUrl?.value?.trim() || '',
        backendUrl: elements.backendUrl?.value?.trim() || ''
      };

      const response = await ApiClient.post(CONFIG.endpoints.twitch, requestData);
      
      if (response.success) {
        toast.show(`${response.message}`, 'success');
        elements.videoUrl.value = '';
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
      }
    } catch (error) {
      toast.show(`Erreur: ${error.message}`, 'error');
    } finally {
      setLoadingState(false);
    }
  }

  static async playDirectUrl() {
    const url = elements.videoUrl?.value?.trim();
    if (!url) {
      toast.show('Veuillez entrer une URL', 'error');
      return;
    }

    const vlcUrl = elements.vlcUrl?.value?.trim();
    if (!vlcUrl) {
      toast.show('URL VLC requise', 'error');
      return;
    }

    setLoadingState(true);
    toast.clear();

    try {
      const requestData = {
        url: url,
        vlcUrl: vlcUrl
      };

      const response = await ApiClient.post(CONFIG.endpoints.playurl, requestData);
      
      if (response.success) {
        toast.show(`${response.message}`, 'success');
        elements.videoUrl.value = '';
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
      }
    } catch (error) {
      toast.show(`Erreur: ${error.message}`, 'error');
    } finally {
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
  const youtubeStreamBtn = document.getElementById('downloadYoutubeStream');
  const youtubeDownloadBtn = document.getElementById('downloadYoutubeDownload');
  const twitchBtn = document.getElementById('downloadTwitch');
  const playDirectBtn = document.getElementById('playDirect');
  
  if (directBtn) {
    directBtn.onclick = () => VideoManager.download(CONFIG.endpoints.direct, 'direct');
  }
  
  if (youtubeStreamBtn) {
    youtubeStreamBtn.onclick = () => VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'stream');
  }
  
  if (youtubeDownloadBtn) {
    youtubeDownloadBtn.onclick = () => VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'download');
  }
  
  if (twitchBtn) {
    twitchBtn.onclick = VideoManager.downloadTwitch;
  }
  
  if (playDirectBtn) {
    playDirectBtn.onclick = VideoManager.playDirectUrl;
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
        VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'stream');
      }
    });
  }

  // Setup modal event listeners
  ModalManager.setupEventListeners();
  
  // Add clear completed downloads button to header if section exists
  const progressSection = elements.downloadProgressSection;
  if (progressSection) {
    const header = progressSection.querySelector('.card-header');
    if (header) {
      const clearBtn = createElement('button', 'btn btn-ghost btn-sm');
      clearBtn.textContent = 'Effacer terminés';
      clearBtn.onclick = DownloadManager.clearCompleted;
      
      const description = header.querySelector('.card-description');
      if (description) {
        description.appendChild(clearBtn);
      }
    }
  }
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
  
  // Connect WebSocket
  WebSocketManager.connect();
  
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

// ===== CLEANUP ON PAGE UNLOAD =====
window.addEventListener('beforeunload', () => {
  WebSocketManager.disconnect();
});

// ===== GLOBAL EXPORTS FOR COMPATIBILITY =====
window.VlcManager = VlcManager;
window.VideoManager = VideoManager;
window.ModalManager = ModalManager;
window.DownloadManager = DownloadManager;
window.WebSocketManager = WebSocketManager;
