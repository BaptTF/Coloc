// ===== TAB MANAGER =====
class TabManager {
  constructor() {
    this.activeTab = 'config';
    this.tabButtons = null;
    this.tabPanels = null;
    this.init();
  }

  init() {
    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => this.setupTabs());
    } else {
      this.setupTabs();
    }
  }

  setupTabs() {
    this.tabButtons = document.querySelectorAll('.tab-btn');
    this.tabPanels = document.querySelectorAll('.tab-panel');

    if (this.tabButtons.length === 0 || this.tabPanels.length === 0) {
      console.warn('[TabManager] No tabs found');
      return;
    }

    // Add click handlers to tab buttons
    this.tabButtons.forEach(button => {
      button.addEventListener('click', (e) => {
        const tabId = button.getAttribute('data-tab');
        if (tabId) {
          this.switchToTab(tabId);
        }
      });
    });

    // Add keyboard navigation
    document.addEventListener('keydown', (e) => {
      if (e.altKey) {
        switch (e.key) {
          case '1':
            this.switchToTab('config');
            e.preventDefault();
            break;
          case '2':
            this.switchToTab('remote');
            e.preventDefault();
            break;
          case '3':
            this.switchToTab('download');
            e.preventDefault();
            break;
          case '4':
            this.switchToTab('videos');
            e.preventDefault();
            break;
        }
      }
    });

    console.log('[TabManager] Tabs initialized');
  }

  switchToTab(tabId) {
    if (!tabId || !this.isValidTab(tabId)) {
      console.warn(`[TabManager] Invalid tab ID: ${tabId}`);
      return;
    }

    // Update active states
    this.tabButtons.forEach(button => {
      const buttonTabId = button.getAttribute('data-tab');
      if (buttonTabId === tabId) {
        button.classList.add('active');
        button.setAttribute('aria-selected', 'true');
      } else {
        button.classList.remove('active');
        button.setAttribute('aria-selected', 'false');
      }
    });

    // Show/hide panels
    this.tabPanels.forEach(panel => {
      if (panel.id === `${tabId}-panel`) {
        panel.classList.add('active');
        panel.setAttribute('aria-hidden', 'false');
      } else {
        panel.classList.remove('active');
        panel.setAttribute('aria-hidden', 'true');
      }
    });

    this.activeTab = tabId;
    
    // Dispatch custom event for other components
    document.dispatchEvent(new CustomEvent('tabChanged', { 
      detail: { activeTab: tabId } 
    }));

    console.log(`[TabManager] Switched to tab: ${tabId}`);
  }

  isValidTab(tabId) {
    const validTabs = ['config', 'remote', 'download', 'videos'];
    return validTabs.includes(tabId);
  }

  getCurrentTab() {
    return this.activeTab;
  }

  // Auto-switch to videos tab (for download completion)
  switchToVideos() {
    this.switchToTab('videos');
  }

  // Show download progress section when switching to videos tab
  showDownloadProgress() {
    const progressSection = document.getElementById('downloadProgressSection');
    if (progressSection) {
      progressSection.style.display = 'block';
    }
  }
}

// Create global instance
const tabManager = new TabManager();

// Export for use in other modules
export { TabManager, tabManager };

// Also add to window for global access
window.TabManager = TabManager;
window.tabManager = tabManager;