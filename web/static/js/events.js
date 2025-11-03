import { CONFIG } from './config.js';
import { elements, createElement } from './utils.js';
import { state } from './state.js';
import { VideoManager } from './video.js';
import { VlcManager } from './vlc.js';
import { DownloadManager } from './download.js';
import { ModalManager } from './modal.js';
import { themeManager } from './theme.js';
import { tabManager } from './tabs.js';

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
    youtubeStreamBtn.onclick = async () => {
      await VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'stream');
      // Auto-switch to videos tab to show download progress
      tabManager.switchToVideos();
    };
  }

  if (youtubeDownloadBtn) {
    youtubeDownloadBtn.onclick = async () => {
      await VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'download');
      // Auto-switch to videos tab to show download progress
      tabManager.switchToVideos();
    };
  }

  if (twitchBtn) {
    twitchBtn.onclick = async () => {
      await VideoManager.downloadTwitch();
      // Auto-switch to videos tab to show download progress
      tabManager.switchToVideos();
    };
  }

  if (playDirectBtn) {
    playDirectBtn.onclick = async () => {
      await VideoManager.playDirectUrl();
      // Auto-switch to videos tab to see the video
      tabManager.switchToVideos();
    };
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

  // VLC WebSocket buttons (removed manual connect/disconnect - now automatic)
  const vlcPlayBtn = document.getElementById('vlcPlay');
  const vlcPauseBtn = document.getElementById('vlcPause');
  const vlcPreviousBtn = document.getElementById('vlcPrevious');
  const vlcNextBtn = document.getElementById('vlcNext');
  const volumeSlider = document.getElementById('volumeSlider');
  const seekSlider = document.getElementById('seekSlider');

  if (vlcPlayBtn) {
    vlcPlayBtn.onclick = () => VlcManager.play();
  }

  if (vlcPauseBtn) {
    vlcPauseBtn.onclick = () => VlcManager.pause();
  }

  

  if (vlcPreviousBtn) {
    vlcPreviousBtn.onclick = () => VlcManager.previous();
  }

  if (vlcNextBtn) {
    vlcNextBtn.onclick = () => VlcManager.next();
  }

  if (volumeSlider) {
    volumeSlider.oninput = (e) => {
      const volume = parseInt(e.target.value);
      document.getElementById('volumeValue').textContent = volume + '%';
    };
    
    volumeSlider.onchange = (e) => {
      const volume = parseInt(e.target.value);
      VlcManager.setVolume(volume);
    };
  }

  if (seekSlider) {
    seekSlider.oninput = (e) => {
      const position = parseInt(e.target.value);
      // Update seek display with current time based on position
      const duration = state.vlcDuration || 0;
      if (duration > 0) {
        const currentTime = Math.floor((position / 100) * duration);
        document.getElementById('seekValue').textContent = VlcManager.formatTime(currentTime);
      } else {
        document.getElementById('seekValue').textContent = position + '%';
      }
    };
    
    seekSlider.onchange = (e) => {
      const position = parseInt(e.target.value);
      VlcManager.seek(position);
    };
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
    elements.videoUrl.addEventListener('keypress', async (e) => {
      if (e.key === 'Enter') {
        await VideoManager.download(CONFIG.endpoints.youtube, 'youtube', 'stream');
        // Auto-switch to videos tab to show download progress
        tabManager.switchToVideos();
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

  // Theme toggle button
  const themeToggleBtn = document.getElementById('themeToggle');
  if (themeToggleBtn) {
    themeToggleBtn.onclick = () => themeManager.toggleTheme();
  }
}

export { setupEventListeners };
