import { CONFIG } from './config.js';
import { state } from './state.js';

// ===== DOM ELEMENTS =====
const elements = {};

// ===== UTILITY FUNCTIONS =====
function initializeElements() {
  Object.keys(CONFIG.selectors).forEach(key => {
    elements[key] = document.querySelector(CONFIG.selectors[key]);
  });
}

function createElement(tag, className = '', textContent = '') {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (textContent) element.textContent = textContent;
  return element;
}

function setLoadingState(isLoading) {
  state.isLoading = isLoading;
  const buttons = document.querySelectorAll('.btn');
  buttons.forEach(btn => {
    btn.disabled = isLoading;
    if (isLoading && !btn.querySelector('.spinner')) {
      const spinner = createElement('div', 'spinner');
      btn.insertBefore(spinner, btn.firstChild);
    } else if (!isLoading) {
      const spinner = btn.querySelector('.spinner');
      if (spinner) spinner.remove();
    }
  });
}

export { elements, initializeElements, createElement, setLoadingState };
