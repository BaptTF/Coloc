import { CONFIG } from './config.js';
import { elements, setLoadingState, createElement } from './utils.js';
import { toast } from './toast.js';
import { ApiClient } from './api.js';
import { VlcManager } from './vlc.js';

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
    
    // Add event delegation to handle clicks on VLC buttons
    grid.addEventListener('click', (event) => {
      const button = event.target.closest('button');
      if (button && button.textContent.includes('Lancer sur VLC')) {
        event.preventDefault();
        event.stopPropagation();
        
        const filename = button.getAttribute('data-filename');
        VlcManager.playVideo(filename);
      }
    });
  }

  static createVideoCard(filename) {
    const card = createElement('div', 'video-card');

    const title = createElement('div', 'video-title', filename);

    const actions = createElement('div', 'video-actions');
    const playBtn = createElement('button', 'btn btn-accent btn-sm');
    playBtn.innerHTML = '▶ Lancer sur VLC';
    playBtn.setAttribute('data-filename', filename);

    actions.appendChild(playBtn);
    card.append(title, actions);

    return card;
  }
}

const videoManager = new VideoManager();

export { VideoManager, videoManager };
