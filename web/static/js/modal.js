import { CONFIG } from './config.js';
import { elements } from './utils.js';

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
  }
}

const modalManager = new ModalManager();

export { ModalManager, modalManager };
