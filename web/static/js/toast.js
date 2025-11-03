import { createElement } from "./utils.js";

// ===== TOAST SYSTEM =====
class ToastManager {
  constructor() {
    this.container = this.createContainer();
    this.toasts = new Map();
  }

  createContainer() {
    let container = document.querySelector(".toast-container");
    if (!container) {
      container = createElement("div", "toast-container");
      document.body.appendChild(container);
    }
    return container;
  }

  show(message, type = "info", duration = 5000) {
    const id = Date.now() + Math.random();
    const toast = this.createToast(message, type, id);

    this.container.appendChild(toast);
    this.toasts.set(id, toast);

    // Auto-remove after duration
    if (duration > 0) {
      setTimeout(() => this.remove(id), duration);
    }

    return id;
  }

  createToast(message, type, id) {
    const toast = createElement("div", `toast toast-${type}`);

    const icon = this.getIcon(type);
    const iconEl = createElement("div", "toast-icon");
    iconEl.innerHTML = icon;

    const content = createElement("div", "toast-content", message);

    const closeBtn = createElement("button", "toast-close");
    closeBtn.innerHTML = "×";
    closeBtn.onclick = () => this.remove(id);

    toast.append(iconEl, content, closeBtn);
    return toast;
  }

  getIcon(type) {
    const icons = {
      success: "✓",
      error: "✗",
      warning: "⚠",
      info: "ℹ",
    };
    return icons[type] || icons.info;
  }

  remove(id) {
    const toast = this.toasts.get(id);
    if (toast) {
      toast.style.animation = "slideOut 0.3s ease forwards";
      setTimeout(() => {
        if (toast.parentNode) {
          toast.parentNode.removeChild(toast);
        }
        this.toasts.delete(id);
      }, 300);
    }
  }

  clear() {
    this.toasts.forEach((_, id) => this.remove(id));
  }

  // Convenience methods
  success(message, duration = 5000) {
    return this.show(message, "success", duration);
  }

  error(message, duration = 5000) {
    return this.show(message, "error", duration);
  }

  warning(message, duration = 5000) {
    return this.show(message, "warning", duration);
  }

  info(message, duration = 5000) {
    return this.show(message, "info", duration);
  }
}

const toast = new ToastManager();

export { ToastManager, toast };
