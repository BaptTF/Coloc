import { CONFIG } from './config.js';
import { elements, createElement } from './utils.js';
import { VideoManager } from './video.js';
import { VlcManager } from './vlc.js';
import { DownloadManager } from './download.js';
import { ModalManager } from './modal.js';

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

export { setupEventListeners };
