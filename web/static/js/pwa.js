// ===== PWA MANAGER =====
// Handles Progressive Web App installation, service worker, and mobile features

class PWAManager {
  constructor() {
    this.deferredPrompt = null;
    this.isInstallable = false;
    this.isInstalled = window.matchMedia('(display-mode: standalone)').matches || 
                      window.navigator.standalone === true;
    
    this.init();
  }

  // ===== INITIALIZATION =====
  async init() {
    console.log('[PWA] Initializing PWA Manager');
    
    // Register service worker
    await this.registerServiceWorker();
    
    // Setup install prompt detection
    this.setupInstallPrompt();
    
    // Check if should show install banner
    this.checkInstallBanner();
    
    // Handle shared URLs (Web Share Target)
    this.handleSharedURL();
    
    // Setup mobile-specific features
    this.setupMobileFeatures();
    
    console.log('[PWA] PWA Manager initialized');
  }

  // ===== SERVICE WORKER =====
  async registerServiceWorker() {
    if ('serviceWorker' in navigator) {
      try {
        const registration = await navigator.serviceWorker.register('/static/sw.js');
        
        console.log('[PWA] Service Worker registered:', registration.scope);
        
        // Listen for updates
        registration.addEventListener('updatefound', () => {
          const newWorker = registration.installing;
          console.log('[PWA] New service worker found');
          
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              // New version available
              this.showUpdateAvailable();
            }
          });
        });
        
        // Handle controller change - disabled auto refresh
        navigator.serviceWorker.addEventListener('controllerchange', () => {
          console.log('[PWA] Service worker controller changed - auto refresh disabled');
        });
        
      } catch (error) {
        console.error('[PWA] Service Worker registration failed:', error);
      }
    } else {
      console.warn('[PWA] Service Worker not supported');
    }
  }

  // ===== INSTALLATION =====
  setupInstallPrompt() {
    window.addEventListener('beforeinstallprompt', (event) => {
      console.log('[PWA] Install prompt detected');
      event.preventDefault();
      this.deferredPrompt = event;
      this.isInstallable = true;
      
      // Show install banner after a delay
      setTimeout(() => {
        this.showInstallBanner();
      }, 3000);
    });

    // Handle successful installation
    window.addEventListener('appinstalled', () => {
      console.log('[PWA] App installed successfully');
      this.isInstalled = true;
      this.isInstallable = false;
      this.hideInstallBanner();
      this.showInstallSuccess();
    });
  }

  checkInstallBanner() {
    // Don't show if already installed or not installable
    if (this.isInstalled) {
      return;
    }

    // Check if user previously dismissed
    const dismissed = localStorage.getItem('pwa-install-dismissed');
    if (dismissed) {
      const dismissedTime = parseInt(dismissed);
      const oneWeek = 7 * 24 * 60 * 60 * 1000;
      
      // Show again after one week
      if (Date.now() - dismissedTime < oneWeek) {
        return;
      }
    }

    // For iOS, show instructions after some interactions
    if (this.isIOS() && !this.isInstalled) {
      let interactionCount = parseInt(localStorage.getItem('user-interactions') || '0');
      interactionCount++;
      localStorage.setItem('user-interactions', interactionCount.toString());
      
      // Show iOS instructions after 3 interactions
      if (interactionCount >= 3) {
        setTimeout(() => {
          this.showIOSInstructions();
        }, 2000);
      }
    }
  }

  async installApp() {
    if (!this.deferredPrompt) {
      console.warn('[PWA] No install prompt available');
      return false;
    }

    try {
      this.deferredPrompt.prompt();
      const { outcome } = await this.deferredPrompt.userChoice;
      
      console.log('[PWA] Install prompt outcome:', outcome);
      
      if (outcome === 'accepted') {
        console.log('[PWA] User accepted install');
        this.deferredPrompt = null;
        return true;
      } else {
        console.log('[PWA] User dismissed install');
        return false;
      }
    } catch (error) {
      console.error('[PWA] Install prompt error:', error);
      return false;
    }
  }

  dismissInstallBanner() {
    localStorage.setItem('pwa-install-dismissed', Date.now().toString());
    this.hideInstallBanner();
  }

  // ===== UI METHODS =====
  showInstallBanner() {
    if (this.isInstalled || !this.isInstallable) return;
    
    const banner = document.getElementById('pwaInstallBanner');
    if (banner) {
      banner.style.display = 'block';
      setTimeout(() => {
        banner.classList.add('show');
      }, 100);
    }
  }

  hideInstallBanner() {
    const banner = document.getElementById('pwaInstallBanner');
    if (banner) {
      banner.classList.remove('show');
      setTimeout(() => {
        banner.style.display = 'none';
      }, 300);
    }
  }

  showIOSInstructions() {
    const instructions = document.getElementById('pwaInstallInstructions');
    if (instructions) {
      instructions.style.display = 'flex';
      setTimeout(() => {
        instructions.classList.add('show');
      }, 100);
    }
  }

  hideIOSInstructions() {
    const instructions = document.getElementById('pwaInstallInstructions');
    if (instructions) {
      instructions.classList.remove('show');
      setTimeout(() => {
        instructions.style.display = 'none';
      }, 300);
    }
  }

  showInstallSuccess() {
    if (window.toastManager) {
      window.toastManager.success('📱 Coloc installé avec succès!');
    }
  }

  showUpdateAvailable() {
    if (window.toastManager) {
      const toast = window.toastManager.info('🔄 Nouvelle version disponible!');
      
      // Add click handler to refresh
      toast.element.addEventListener('click', () => {
        window.location.reload();
      });
    }
  }

  // ===== WEB SHARE TARGET =====
  handleSharedURL() {
    const urlParams = new URLSearchParams(window.location.search);
    const sharedURL = urlParams.get('url');
    const sharedTitle = urlParams.get('title');
    const sharedText = urlParams.get('text');
    
    if (sharedURL) {
      console.log('[PWA] Shared URL detected:', sharedURL);
      console.log('[PWA] Document ready state:', document.readyState);
      
      // Check if DOM is already loaded
      if (document.readyState === 'loading') {
        // DOM is still loading, wait for DOMContentLoaded
        document.addEventListener('DOMContentLoaded', () => {
          console.log('[PWA] DOM loaded, processing shared URL');
          this.processSharedURL(sharedURL, sharedTitle, sharedText);
        });
      } else {
        // DOM is already loaded, process immediately with a small delay
        setTimeout(() => {
          console.log('[PWA] DOM already loaded, processing shared URL immediately');
          this.processSharedURL(sharedURL, sharedTitle, sharedText);
        }, 100);
      }
    } else {
      console.log('[PWA] No shared URL detected');
    }
  }

  async processSharedURL(url, title, text) {
    try {
      console.log('[PWA] Processing shared URL:', { url, title, text });
      
      // Find the URL input
      const urlInput = document.getElementById('videoUrl');
      if (!urlInput) {
        console.error('[PWA] URL input not found - DOM elements:', {
          videoUrl: !!document.getElementById('videoUrl'),
          body: !!document.body,
          readyState: document.readyState
        });
        return;
      }

      console.log('[PWA] URL input found, filling with:', url);
      
      // Fill the input
      urlInput.value = url;
      
      // Analyze the URL to suggest download mode
      const analysis = this.analyzeURL(url);
      console.log('[PWA] URL analysis:', analysis);
      
      // Show visual feedback
      this.showSharedURLFeedback(url, title, text, analysis);
      
      // Scroll to download section
      const downloadSection = document.querySelector('.section:has(#videoUrl)');
      if (downloadSection) {
        console.log('[PWA] Scrolling to download section');
        downloadSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
      } else {
        console.log('[PWA] Download section not found, trying alternative selector');
        const alternativeSection = document.querySelector('#videoUrl')?.closest('.section');
        if (alternativeSection) {
          alternativeSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      }
      
      // Highlight the input
      console.log('[PWA] Adding highlight to input');
      urlInput.classList.add('shared-url-highlight');
      setTimeout(() => {
        urlInput.classList.remove('shared-url-highlight');
        console.log('[PWA] Removed highlight from input');
      }, 2000);
      
      // Clean URL for better UX
      const cleanURL = new URL(window.location);
      cleanURL.searchParams.delete('url');
      cleanURL.searchParams.delete('title');
      cleanURL.searchParams.delete('text');
      window.history.replaceState({}, '', cleanURL.toString());
      
      console.log('[PWA] Shared URL processed successfully');
      
    } catch (error) {
      console.error('[PWA] Error processing shared URL:', error);
      console.error('[PWA] Error stack:', error.stack);
    }
  }

  analyzeURL(url) {
    if (url.includes('youtube.com') || url.includes('youtu.be')) {
      return { platform: 'youtube', mode: 'download', icon: '/static/icons/yt-dlp.png', iconAlt: 'yt-dlp' };
    } else if (url.includes('twitch.tv')) {
      return { platform: 'twitch', mode: 'stream', icon: '🎮' };
    } else if (url.match(/\.(mp4|avi|mkv|webm|mov)$/i)) {
      return { platform: 'direct', mode: 'play', icon: '🎬' };
    }
    return { platform: 'unknown', mode: 'download', icon: '/static/icons/websocket.png', iconAlt: 'WebSocket' };
  }

  showSharedURLFeedback(url, title, text, analysis) {
    console.log('[PWA] Showing shared URL feedback:', { url, title, text, analysis });
    
    if (!window.toastManager) {
      console.error('[PWA] ToastManager not available for feedback');
      // Fallback to console feedback
      console.log(`[PWA] ${analysis.icon} URL reçu: ${analysis.platform}`);
      if (title || text) {
        console.log(`[PWA] Details: ${title || text || 'Prêt à télécharger!'}`);
      }
      return;
    }
    
    const message = `${analysis.icon} URL reçu: ${analysis.platform}`;
    console.log('[PWA] Showing success toast:', message);
    window.toastManager.success(message);
    
    // Show more detailed feedback if available
    if (title || text) {
      setTimeout(() => {
        const details = title || text || 'Prêt à télécharger!';
        console.log('[PWA] Showing info toast:', details);
        window.toastManager.info(details);
      }, 1000);
    }
  }

  // ===== MOBILE FEATURES =====
  setupMobileFeatures() {
    // Setup mobile navigation
    this.setupMobileNavigation();
    
    // Setup touch interactions
    this.setupTouchInteractions();
    
    // Setup viewport handling
    this.setupViewportHandling();
    
    // Setup screen orientation
    this.setupScreenOrientation();
  }

  setupMobileNavigation() {
    // Add mobile menu toggle if needed
    const header = document.querySelector('.header-content');
    if (header && window.innerWidth <= 768) {
      // Could add hamburger menu here
      console.log('[PWA] Mobile navigation setup');
    }
  }

  setupTouchInteractions() {
    // Add touch feedback to buttons
    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(button => {
      button.addEventListener('touchstart', () => {
        button.classList.add('touch-active');
      });
      
      button.addEventListener('touchend', () => {
        setTimeout(() => {
          button.classList.remove('touch-active');
        }, 150);
      });
    });
  }

  setupViewportHandling() {
    // Handle viewport changes for mobile
    const handleViewportChange = () => {
      const isMobile = window.innerWidth <= 768;
      document.documentElement.classList.toggle('mobile', isMobile);
      document.documentElement.classList.toggle('desktop', !isMobile);
    };
    
    window.addEventListener('resize', handleViewportChange);
    handleViewportChange();
  }

  setupScreenOrientation() {
    // Handle screen orientation changes
    if ('screen' in window && 'orientation' in screen) {
      screen.orientation.addEventListener('change', () => {
        console.log('[PWA] Screen orientation changed:', screen.orientation.type);
        
        // Could adjust layout based on orientation
        const isLandscape = screen.orientation.type.includes('landscape');
        document.documentElement.classList.toggle('landscape', isLandscape);
        document.documentElement.classList.toggle('portrait', !isLandscape);
      });
    }
  }

  // ===== UTILITY METHODS =====
  isIOS() {
    return /iPad|iPhone|iPod/.test(navigator.userAgent) && !window.MSStream;
  }

  isAndroid() {
    return /Android/.test(navigator.userAgent);
  }

  isMobile() {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
  }

  // ===== NOTIFICATION PERMISSION =====
  async requestNotificationPermission() {
    if ('Notification' in navigator) {
      try {
        const permission = await Notification.requestPermission();
        console.log('[PWA] Notification permission:', permission);
        return permission === 'granted';
      } catch (error) {
        console.error('[PWA] Notification permission error:', error);
        return false;
      }
    }
    return false;
  }

  // ===== BACKGROUND SYNC =====
  async registerBackgroundSync() {
    if ('serviceWorker' in navigator && 'sync' in window.ServiceWorkerRegistration.prototype) {
      try {
        const registration = await navigator.serviceWorker.ready;
        await registration.sync.register('background-download');
        console.log('[PWA] Background sync registered');
        return true;
      } catch (error) {
        console.error('[PWA] Background sync registration failed:', error);
        return false;
      }
    }
    return false;
  }
}

// ===== EVENT LISTENERS =====
document.addEventListener('DOMContentLoaded', () => {
  // Initialize PWA Manager
  window.pwaManager = new PWAManager();
  
  // Add debug function for testing share target
  window.testShareTarget = (url, title, text) => {
    console.log('[PWA] Manual test of share target with:', { url, title, text });
    if (window.pwaManager) {
      window.pwaManager.processSharedURL(url, title, text);
    } else {
      console.error('[PWA] PWAManager not available');
    }
  };
  
  // Setup install button
  const installBtn = document.getElementById('pwaInstallBtn');
  if (installBtn) {
    installBtn.addEventListener('click', async () => {
      const installed = await window.pwaManager.installApp();
      if (installed) {
        window.pwaManager.hideInstallBanner();
      }
    });
  }
  
  // Setup dismiss button
  const dismissBtn = document.getElementById('pwaDismissBtn');
  if (dismissBtn) {
    dismissBtn.addEventListener('click', () => {
      window.pwaManager.dismissInstallBanner();
    });
  }
  
  // Setup iOS instructions close button
  const closeInstructionsBtn = document.getElementById('closeInstructions');
  if (closeInstructionsBtn) {
    closeInstructionsBtn.addEventListener('click', () => {
      window.pwaManager.hideIOSInstructions();
    });
  }
});

export { PWAManager };