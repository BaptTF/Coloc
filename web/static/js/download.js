import { CONFIG } from './config.js';
import { state } from './state.js';
import { elements, createElement } from './utils.js';
import { toast } from './toast.js';
import { ApiClient } from './api.js';

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
      console.log('DownloadManager.updateQueueStatus called with:', queueData);
      // Check if queueData is valid
      if (!queueData || !Array.isArray(queueData)) {
        console.warn('Invalid queue data received:', queueData);
        return;
      }

      console.log('Processing', queueData.length, 'queue items');

      // Get current job IDs in the queue
      const currentJobIds = new Set(queueData
        .filter(jobStatus => jobStatus && jobStatus.job && jobStatus.job.id)
        .map(jobStatus => jobStatus.job.id));

      // Remove jobs that are no longer in the queue (finished/removed)
      for (const [jobId, download] of state.downloads.entries()) {
        if (!currentJobIds.has(jobId) && (download.status === 'completed' || download.status === 'error' || download.status === 'cancelled')) {
          // Job finished and is no longer in queue - remove from display after a delay
          setTimeout(() => {
            const downloadEl = document.getElementById(`download-${jobId}`);
            if (downloadEl) downloadEl.remove();
            state.downloads.delete(jobId);

            // Hide progress section if no active downloads
            if (state.downloads.size === 0) {
              DownloadManager.hideProgressSection();
            }
          }, 3000); // Keep completed/error/cancelled jobs visible for 3 seconds
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
                  jobStatus.status === 'error' ? 0 : 
                  jobStatus.status === 'cancelled' ? 0 : 0, // Don't show fake progress
          error: jobStatus.error,
          streamUrl: jobStatus.streamUrl,
          completedAt: jobStatus.completedAt ? new Date(jobStatus.completedAt) : null,
          createdAt: jobStatus.job.createdAt ? new Date(jobStatus.job.createdAt) : new Date(),
          cancelled: jobStatus.cancelled || false
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
    
    // Add cancelled class for styling
    if (download.cancelled || download.status === 'cancelled') {
      downloadEl.classList.add('cancelled');
    }

    // Use progress message instead of old message field
    const displayMessage = download.progress || download.message || 'Traitement en cours';
    const showProgressBar = download.status === 'downloading' || download.status === 'completed' || download.status === 'error';
    const percent = download.percent || 0;

    downloadEl.innerHTML = `
      <div class="download-header">
        <div class="download-url" title="${download.url}">${download.url}</div>
        <div class="download-status ${download.status}">
          ${DownloadManager.getStatusText(download.status)}
        </div>
      </div>

      <div class="download-progress">
        ${showProgressBar ? `
          <div class="progress-bar">
            <div class="progress-fill ${download.status === 'completed' ? 'completed' : download.status === 'downloading' ? 'active' : ''}"
                 style="width: ${percent}%"></div>
          </div>
        ` : ''}
        <div class="progress-text">
          <span class="progress-message">${displayMessage}</span>
          ${showProgressBar && percent > 0 ?
            `<span class="progress-percent">${percent.toFixed(1)}%</span>` : ''}
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

      ${download.status === 'completed' && download.streamUrl ? `
        <div class="download-actions">
          <button class="btn btn-accent btn-sm" onclick="VlcManager.playVideo('${download.streamUrl}')">
            ▶ Lancer sur VLC
          </button>
        </div>
      ` : ''}
      
      ${download.status === 'error' ? `
        <div class="download-actions">
          <button class="btn btn-primary btn-sm" onclick="DownloadManager.retryDownload('${download.id}')">
            🔄 Réessayer
          </button>
        </div>
      ` : ''}
      
      ${(download.status === 'queued' || download.status === 'processing' || download.status === 'downloading' || download.status === 'streaming') ? `
        <div class="download-actions">
          <button class="btn btn-danger btn-sm" onclick="DownloadManager.cancelDownload('${download.id}')">
            ❌ Annuler
          </button>
        </div>
      ` : ''}
    `;
  }

  static getStatusText(status) {
    const statusTexts = {
      queued: 'En file',
      processing: 'Traitement',
      downloading: 'Téléchargement',
      streaming: 'Conversion HLS',
      completed: 'Terminé',
      error: 'Erreur',
      cancelled: 'Annulé'
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

  static async retryDownload(downloadId) {
    try {
      console.log('Retrying download:', downloadId);
      
      // Update UI to show retrying status
      DownloadManager.updateDownload(downloadId, {
        status: 'queued',
        progress: 'Réessai en cours...',
        error: null
      });

      // Call backend retry endpoint
      const response = await ApiClient.post(`/retry/${downloadId}`, {});

      if (response.success) {
        toast.show('Téléchargement relancé', 'success');
        // The backend will broadcast the updated queue status
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
        DownloadManager.updateDownload(downloadId, {
          status: 'error',
          message: response.message
        });
      }
    } catch (error) {
      console.error('Error retrying download:', error);
      toast.show(`Erreur lors du réessai: ${error.message}`, 'error');
      DownloadManager.updateDownload(downloadId, {
        status: 'error',
        message: error.message
      });
    }
  }

  static async cancelDownload(downloadId) {
    try {
      console.log('Cancelling download:', downloadId);
      
      // Update UI to show cancelling status
      DownloadManager.updateDownload(downloadId, {
        status: 'cancelled',
        progress: 'Annulation en cours...'
      });

      // Call backend cancel endpoint
      const response = await ApiClient.post(`/cancel/${downloadId}`, {});

      if (response.success) {
        toast.show('Téléchargement annulé', 'success');
        // The backend will broadcast the updated queue status
      } else {
        toast.show(`Erreur: ${response.message}`, 'error');
        // If backend fails, we might need to revert the UI change
        // but for now, let the backend broadcast handle it
      }
    } catch (error) {
      console.error('Error cancelling download:', error);
      toast.show(`Erreur lors de l'annulation: ${error.message}`, 'error');
      // Revert UI change on error
      DownloadManager.updateDownload(downloadId, {
        status: 'queued',
        progress: 'Erreur lors de l\'annulation'
      });
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

const downloadManager = new DownloadManager();

export { DownloadManager, downloadManager };
