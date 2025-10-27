import { CONFIG } from './config.js';
import { elements, setLoadingState } from './utils.js';
import { state } from './state.js';
import { toast } from './toast.js';
import { ApiClient } from './api.js';
import { ModalManager } from './modal.js';
import { StatusManager } from './status.js';

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

const vlcManager = new VlcManager();

export { VlcManager, vlcManager };
