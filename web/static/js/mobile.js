/**
 * Mobile UI Components
 * Handles mobile-specific interactions and components
 */

export class MobileComponents {
  constructor() {
    this.isMobile = window.innerWidth <= 768;
    this.touchStartY = 0;
    this.touchStartX = 0;
    this.pullToRefreshThreshold = 80;
    this.isPulling = false;
    this.init();
  }

  init() {
    // Wait a bit for other modules to initialize
    setTimeout(() => {
      this.setupBottomSheets();
      this.setupTouchOptimizations();
      this.setupResizeHandler();
    }, 100);
  }

  createMobileNavigation() {
    const header = document.querySelector('.header-content');
    if (!header) return;

    // Create mobile nav toggle
    const toggle = document.createElement('button');
    toggle.className = 'mobile-nav-toggle';
    toggle.innerHTML = '☰';
    toggle.setAttribute('aria-label', 'Menu');

    // Create mobile menu
    const menu = document.createElement('div');
    menu.className = 'mobile-menu';
    menu.innerHTML = `
      <div class="mobile-menu-header">
        <div class="logo">
          <img src="/static/icons/favicon-32x32.svg" alt="Coloc" class="logo-icon">
          <span>Coloc</span>
        </div>
        <button class="mobile-menu-close" aria-label="Close menu">✕</button>
      </div>
      <div class="mobile-menu-content">
        <div class="mobile-menu-section">
          <div class="mobile-menu-title">Navigation</div>
          <div class="btn-group">
            <a href="#download" class="btn btn-primary">Download</a>
            <a href="#vlc" class="btn btn-purple">VLC Remote</a>
          </div>
        </div>
        <div class="mobile-menu-section">
          <div class="mobile-menu-title">Status</div>
          <div id="mobileStatusGroup" class="status-group">
            <!-- Status will be populated here -->
          </div>
        </div>
        <div class="mobile-menu-section">
          <div class="mobile-menu-title">Theme</div>
          <button id="mobileThemeToggle" class="btn btn-ghost">
            <span class="theme-icon">🌙</span>
            <span>Dark Mode</span>
          </button>
        </div>
      </div>
    `;

    header.appendChild(toggle);
    document.body.appendChild(menu);

    // Setup theme toggle in mobile menu
    const mobileThemeToggle = document.getElementById('mobileThemeToggle');
    if (mobileThemeToggle) {
      mobileThemeToggle.addEventListener('click', () => {
        this.toggleTheme();
        this.closeMobileMenu();
      });
    }
  }

  toggleMobileMenu() {
    const menu = document.querySelector('.mobile-menu');
    if (menu) {
      menu.classList.toggle('active');
      document.body.style.overflow = menu.classList.contains('active') ? 'hidden' : '';
    }
  }

  closeMobileMenu() {
    const menu = document.querySelector('.mobile-menu');
    if (menu) {
      menu.classList.remove('active');
      document.body.style.overflow = '';
    }
  }

  setupBottomSheets() {
    // Create bottom sheet functionality
    window.showBottomSheet = (content) => {
      let sheet = document.querySelector('.bottom-sheet');
      if (!sheet) {
        sheet = this.createBottomSheet();
      }

      const contentEl = sheet.querySelector('.bottom-sheet-content');
      if (contentEl) {
        contentEl.innerHTML = content;
      }

      sheet.classList.add('active');
      document.body.style.overflow = 'hidden';
    };

    window.hideBottomSheet = () => {
      const sheet = document.querySelector('.bottom-sheet');
      if (sheet) {
        sheet.classList.remove('active');
        document.body.style.overflow = '';
      }
    };
  }

  createBottomSheet() {
    const sheet = document.createElement('div');
    sheet.className = 'bottom-sheet';
    sheet.innerHTML = `
      <div class="bottom-sheet-handle"></div>
      <div class="bottom-sheet-header">
        <h3>Options</h3>
      </div>
      <div class="bottom-sheet-content">
        <!-- Content will be inserted here -->
      </div>
    `;

    // Close on handle click or backdrop click
    sheet.addEventListener('click', (e) => {
      if (e.target === sheet || e.target.classList.contains('bottom-sheet-handle')) {
        window.hideBottomSheet();
      }
    });

    document.body.appendChild(sheet);
    return sheet;
  }

  setupTouchOptimizations() {
    // Add touch feedback to buttons
    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(btn => {
      btn.addEventListener('touchstart', () => {
        btn.classList.add('touch-active');
      });

      btn.addEventListener('touchend', () => {
        setTimeout(() => {
          btn.classList.remove('touch-active');
        }, 150);
      });
    });

    // Optimize inputs for mobile
    const inputs = document.querySelectorAll('input[type="url"], input[type="text"]');
    inputs.forEach(input => {
      // Add mobile-specific attributes
      input.setAttribute('autocomplete', 'off');
      input.setAttribute('autocorrect', 'off');
      input.setAttribute('autocapitalize', 'off');
      input.setAttribute('spellcheck', 'false');

      // Focus handling for mobile
      input.addEventListener('focus', () => {
        if (this.isMobile) {
          input.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      });
    });

    // Prevent zoom on input focus (iOS)
    const meta = document.querySelector('meta[name="viewport"]');
    if (meta && this.isMobile) {
      inputs.forEach(input => {
        input.addEventListener('touchstart', () => {
          meta.setAttribute('content', 'width=device-width,initial-scale=1,maximum-scale=1');
        });

        input.addEventListener('blur', () => {
          meta.setAttribute('content', 'width=device-width,initial-scale=1');
        });
      });
    }
  }

  setupResizeHandler() {
    let resizeTimeout;
    window.addEventListener('resize', () => {
      clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(() => {
        const wasMobile = this.isMobile;
        this.isMobile = window.innerWidth <= 768;

        if (wasMobile !== this.isMobile) {
          this.handleMobileChange();
        }
      }, 250);
    });
  }

  handleMobileChange() {
    if (this.isMobile) {
      // Switched to mobile
      document.body.classList.add('mobile-view');
    } else {
      // Switched to desktop
      document.body.classList.remove('mobile-view');
      this.closeMobileMenu();
      window.hideBottomSheet();
    }
  }

  toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);

    // Update theme toggle buttons
    this.updateThemeButtons(newTheme);
  }

  updateThemeButtons(theme) {
    const themeIcons = document.querySelectorAll('.theme-icon');
    const themeTexts = document.querySelectorAll('#mobileThemeToggle span:last-child');

    themeIcons.forEach(icon => {
      icon.textContent = theme === 'dark' ? '☀️' : '🌙';
    });

    themeTexts.forEach(text => {
      text.textContent = theme === 'dark' ? 'Light Mode' : 'Dark Mode';
    });
  }

  // Utility method to show mobile-specific toasts
  showToast(message, type = 'info', duration = 3000) {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
      <span class="toast-icon">${this.getToastIcon(type)}</span>
      <span class="toast-content">${message}</span>
      <button class="toast-close">✕</button>
    `;

    let container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }

    container.appendChild(toast);

    // Auto remove
    setTimeout(() => {
      toast.remove();
    }, duration);

    // Manual close
    toast.querySelector('.toast-close').addEventListener('click', () => {
      toast.remove();
    });
  }

  getToastIcon(type) {
    const icons = {
      success: '✓',
      error: '✕',
      warning: '⚠',
      info: 'ℹ'
    };
    return icons[type] || icons.info;
  }

  // Update mobile status indicators
  updateMobileStatus(statuses) {
    const mobileStatusGroup = document.getElementById('mobileStatusGroup');
    if (mobileStatusGroup && statuses) {
      mobileStatusGroup.innerHTML = statuses;
    }
  }
}

// Initialize mobile components when DOM is ready and other modules are loaded
document.addEventListener('DOMContentLoaded', () => {
  // Wait for app.js to initialize first
  setTimeout(() => {
    if (!window.mobileComponents && window.innerWidth <= 768) {
      try {
        window.mobileComponents = new MobileComponents();
        console.log('[Mobile] Mobile components initialized');
      } catch (error) {
        console.error('[Mobile] Failed to initialize mobile components:', error);
      }
    }
  }, 1000);
});

// Export for use in other modules
export default MobileComponents;