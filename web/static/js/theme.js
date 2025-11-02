// ===== THEME MANAGER =====
class ThemeManager {
  constructor() {
    this.storageKey = 'coloc-theme';
    this.darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)');
    this.currentTheme = this.getStoredTheme() || this.getSystemTheme();
    this.init();
  }

  // Initialize theme system
  init() {
    this.applyTheme(this.currentTheme);
    this.setupMediaQueryListener();
  }

  // Get stored theme from localStorage
  getStoredTheme() {
    return localStorage.getItem(this.storageKey);
  }

  // Get system theme preference
  getSystemTheme() {
    return this.darkModeQuery.matches ? 'dark' : 'light';
  }

  // Apply theme to document
  applyTheme(theme) {
    if (theme === 'dark') {
      document.documentElement.setAttribute('data-theme', 'dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
    this.currentTheme = theme;
    this.updateThemeToggle();
  }

  // Toggle between themes
  toggleTheme() {
    const newTheme = this.currentTheme === 'dark' ? 'light' : 'dark';
    this.setTheme(newTheme);
  }

  // Set specific theme
  setTheme(theme) {
    this.applyTheme(theme);
    localStorage.setItem(this.storageKey, theme);
  }

  // Reset to system theme
  resetToSystemTheme() {
    localStorage.removeItem(this.storageKey);
    const systemTheme = this.getSystemTheme();
    this.applyTheme(systemTheme);
  }

  // Listen for system theme changes
  setupMediaQueryListener() {
    this.darkModeQuery.addEventListener('change', (e) => {
      // Only auto-switch if user hasn't manually set a preference
      if (!this.getStoredTheme()) {
        this.applyTheme(e.matches ? 'dark' : 'light');
      }
    });
  }

  // Update theme toggle button appearance
  updateThemeToggle() {
    const themeToggle = document.getElementById('themeToggle');
    if (themeToggle) {
      const icon = themeToggle.querySelector('.theme-icon');
      if (icon) {
        icon.textContent = this.currentTheme === 'dark' ? '☀️' : '🌙';
      }
      themeToggle.title = this.currentTheme === 'dark' ? 'Passer au mode clair' : 'Passer au mode sombre';
    }
  }

  // Get current theme
  getCurrentTheme() {
    return this.currentTheme;
  }

  // Check if dark mode is active
  isDarkMode() {
    return this.currentTheme === 'dark';
  }
}

// Export singleton instance
export const themeManager = new ThemeManager();

// Export for global access
window.themeManager = themeManager;