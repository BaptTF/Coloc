/**
 * Mobile Performance Optimizations
 * Optimizes JavaScript performance for mobile devices
 */

export class MobilePerformance {
  constructor() {
    this.isMobile = this.detectMobile();
    this.isLowEnd = this.detectLowEndDevice();
    this.passiveSupported = this.detectPassiveEvents();
    this.intersectionSupported = 'IntersectionObserver' in window;
    this.resizeObserverSupported = 'ResizeObserver' in window;
    
    this.init();
  }

  init() {
    // Wait for other modules to initialize first
    setTimeout(() => {
      this.setupPerformanceMonitoring();
      this.optimizeEventListeners();
      this.setupLazyLoading();
      this.optimizeAnimations();
      this.setupThrottling();
      this.optimizeScrollPerformance();
      this.setupMemoryManagement();
    }, 200);
  }

  detectMobile() {
    return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ||
           window.innerWidth <= 768;
  }

  detectLowEndDevice() {
    // Simple heuristics to detect low-end devices
    const navigator = window.navigator;
    const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
    
    // Check for slow network
    if (connection && (connection.effectiveType === 'slow-2g' || connection.effectiveType === '2g')) {
      return true;
    }

    // Check for low memory (if available)
    if (navigator.deviceMemory && navigator.deviceMemory < 4) {
      return true;
    }

    // Check for low CPU cores (if available)
    if (navigator.hardwareConcurrency && navigator.hardwareConcurrency < 4) {
      return true;
    }

    // Check screen resolution as a proxy
    if (window.screen && window.screen.width * window.screen.height < 1000000) {
      return true;
    }

    return false;
  }

  detectPassiveEvents() {
    let passiveSupported = false;
    try {
      const options = Object.defineProperty({}, 'passive', {
        get: () => {
          passiveSupported = true;
          return false;
        }
      });
      window.addEventListener('test', null, options);
    } catch (err) {
      passiveSupported = false;
    }
    return passiveSupported;
  }

  setupPerformanceMonitoring() {
    if (!this.isMobile) return;

    // Monitor frame rate
    let lastTime = performance.now();
    let frames = 0;
    
    const measureFPS = () => {
      frames++;
      const currentTime = performance.now();
      
      if (currentTime >= lastTime + 1000) {
        const fps = Math.round((frames * 1000) / (currentTime - lastTime));
        
        if (fps < 30) {
          this.enableLowPerformanceMode();
        }
        
        frames = 0;
        lastTime = currentTime;
      }
      
      requestAnimationFrame(measureFPS);
    };

    requestAnimationFrame(measureFPS);

    // Monitor memory usage (if available)
    if (performance.memory) {
      setInterval(() => {
        const memoryUsage = performance.memory.usedJSHeapSize / performance.memory.jsHeapSizeLimit;
        if (memoryUsage > 0.8) {
          this.cleanupMemory();
        }
      }, 30000); // Check every 30 seconds
    }
  }

  enableLowPerformanceMode() {
    document.body.classList.add('low-performance-mode');
    
    // Reduce animation complexity
    document.documentElement.style.setProperty('--animation-duration', '0.1s');
    
    // Disable non-essential animations
    const animatedElements = document.querySelectorAll('[data-animate]');
    animatedElements.forEach(el => {
      el.style.animation = 'none';
    });

    // Reduce update frequency for real-time data
    if (window.websocketManager) {
      window.websocketManager.reduceUpdateFrequency();
    }
  }

  optimizeEventListeners() {
    if (!this.isMobile) return;

    // Use passive event listeners for better scroll performance
    const scrollOptions = this.passiveSupported ? { passive: true } : false;
    const touchOptions = this.passiveSupported ? { passive: true } : false;

    // Optimize touch events
    document.addEventListener('touchstart', this.handleTouchStart.bind(this), touchOptions);
    document.addEventListener('touchmove', this.handleTouchMove.bind(this), touchOptions);
    document.addEventListener('touchend', this.handleTouchEnd.bind(this), touchOptions);

    // Optimize scroll events
    window.addEventListener('scroll', this.throttle(this.handleScroll.bind(this), 16), scrollOptions);
    window.addEventListener('resize', this.throttle(this.handleResize.bind(this), 100));
  }

  setupLazyLoading() {
    if (!this.intersectionSupported) return;

    const lazyImages = document.querySelectorAll('img[data-src]');
    const lazySections = document.querySelectorAll('[data-lazy-section]');

    const imageObserver = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const img = entry.target;
          img.src = img.dataset.src;
          img.classList.remove('lazy');
          imageObserver.unobserve(img);
        }
      });
    }, {
      rootMargin: '50px 0px',
      threshold: 0.01
    });

    const sectionObserver = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const section = entry.target;
          this.loadSectionContent(section);
          sectionObserver.unobserve(section);
        }
      });
    }, {
      rootMargin: '100px 0px',
      threshold: 0.01
    });

    lazyImages.forEach(img => imageObserver.observe(img));
    lazySections.forEach(section => sectionObserver.observe(section));
  }

  loadSectionContent(section) {
    // Load content for sections that are now visible
    if (section.dataset.lazySection === 'videos') {
      // Trigger video loading
      if (window.videoManager) {
        window.videoManager.loadVideos();
      }
    }
  }

  optimizeAnimations() {
    if (!this.isMobile) return;

    // Use CSS transforms instead of position changes for better performance
    const style = document.createElement('style');
    style.textContent = `
      .mobile-view * {
        will-change: auto;
      }
      
      .mobile-view .animating {
        will-change: transform, opacity;
      }
      
      .mobile-view .btn {
        transform: translateZ(0);
        backface-visibility: hidden;
      }
      
      .low-performance-mode * {
        animation-duration: 0.1s !important;
        transition-duration: 0.1s !important;
      }
    `;
    document.head.appendChild(style);

    // Optimize animation frames
    this.optimizeAnimationFrames();
  }

  optimizeAnimationFrames() {
    let rafId = null;
    const callbacks = new Set();

    window.optimizedRequestAnimationFrame = (callback) => {
      callbacks.add(callback);
      
      if (!rafId) {
        rafId = requestAnimationFrame(() => {
          callbacks.forEach(cb => cb());
          callbacks.clear();
          rafId = null;
        });
      }
    };
  }

  setupThrottling() {
    // Throttle expensive operations
    this.throttledUpdates = new Map();
    
    // Throttle WebSocket updates
    if (window.websocketManager) {
      const originalUpdate = window.websocketManager.updateDownloads;
      window.websocketManager.updateDownloads = this.throttle(originalUpdate.bind(window.websocketManager), 100);
    }
  }

  throttle(func, delay) {
    let timeoutId;
    let lastExecTime = 0;
    
    return function (...args) {
      const currentTime = Date.now();
      
      if (currentTime - lastExecTime > delay) {
        func.apply(this, args);
        lastExecTime = currentTime;
      } else {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
          func.apply(this, args);
          lastExecTime = Date.now();
        }, delay - (currentTime - lastExecTime));
      }
    };
  }

  optimizeScrollPerformance() {
    if (!this.isMobile) return;

    // Use requestAnimationFrame for scroll-related updates
    let ticking = false;

    const updateOnScroll = () => {
      // Update scroll-based animations
      this.updateScrollAnimations();
      ticking = false;
    };

    window.addEventListener('scroll', () => {
      if (!ticking) {
        requestAnimationFrame(updateOnScroll);
        ticking = true;
      }
    }, this.passiveSupported ? { passive: true } : false);
  }

  updateScrollAnimations() {
    const scrollTop = window.pageYOffset;
    
    // Parallax effects (disabled on low-end devices)
    if (!this.isLowEnd) {
      const parallaxElements = document.querySelectorAll('[data-parallax]');
      parallaxElements.forEach(el => {
        const speed = parseFloat(el.dataset.parallax) || 0.5;
        el.style.transform = `translateY(${scrollTop * speed}px)`;
      });
    }

    // Show/hide elements based on scroll
    const scrollElements = document.querySelectorAll('[data-scroll-show]');
    scrollElements.forEach(el => {
      const threshold = parseFloat(el.dataset.scrollShow) || 100;
      if (scrollTop > threshold) {
        el.classList.add('visible');
      } else {
        el.classList.remove('visible');
      }
    });
  }

  setupMemoryManagement() {
    // Cleanup event listeners on page unload
    window.addEventListener('beforeunload', () => {
      this.cleanup();
    });

    // Cleanup on visibility change (app backgrounded)
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) {
        this.pauseBackgroundTasks();
      } else {
        this.resumeBackgroundTasks();
      }
    });
  }

  cleanupMemory() {
    // Force garbage collection if available
    if (window.gc) {
      window.gc();
    }

    // Clear caches
    if (window.caches) {
      window.caches.keys().then(cacheNames => {
        return Promise.all(
          cacheNames.map(cacheName => {
            if (cacheName.includes('temp-')) {
              return window.caches.delete(cacheName);
            }
          })
        );
      });
    }

    // Remove unused DOM elements
    const unusedElements = document.querySelectorAll('.toast:not(.show), .modal:not(.active)');
    unusedElements.forEach(el => el.remove());
  }

  cleanup() {
    // Remove event listeners
    this.eventListeners?.forEach(({ element, event, handler }) => {
      element.removeEventListener(event, handler);
    });

    // Clear timeouts and intervals
    this.timeouts?.forEach(timeout => clearTimeout(timeout));
    this.intervals?.forEach(interval => clearInterval(interval));
  }

  pauseBackgroundTasks() {
    // Pause WebSocket updates
    if (window.websocketManager) {
      window.websocketManager.pause();
    }

    // Pause video polling
    if (window.videoManager) {
      window.videoManager.pause();
    }
  }

  resumeBackgroundTasks() {
    // Resume WebSocket updates
    if (window.websocketManager) {
      window.websocketManager.resume();
    }

    // Resume video polling
    if (window.videoManager) {
      window.videoManager.resume();
    }
  }

  // Touch event handlers
  handleTouchStart(e) {
    // Add touch feedback
    const target = e.target.closest('.btn, .video-card, .download-item');
    if (target) {
      target.classList.add('touch-active');
    }
  }

  handleTouchMove(e) {
    // Handle swipe gestures
    const touch = e.touches[0];
    const deltaX = touch.clientX - this.touchStartX;
    const deltaY = touch.clientY - this.touchStartY;

    // Detect horizontal swipe
    if (Math.abs(deltaX) > Math.abs(deltaY) && Math.abs(deltaX) > 50) {
      this.handleSwipe(deltaX > 0 ? 'right' : 'left');
    }
  }

  handleTouchEnd(e) {
    // Remove touch feedback
    const activeElements = document.querySelectorAll('.touch-active');
    activeElements.forEach(el => el.classList.remove('touch-active'));
  }

  handleSwipe(direction) {
    // Handle swipe gestures for navigation
    if (direction === 'right' && window.mobileComponents) {
      // Open mobile menu on right swipe
      window.mobileComponents.toggleMobileMenu();
    }
  }

  handleScroll() {
    // Optimized scroll handler
    const scrollTop = window.pageYOffset;
    
    // Update sticky elements
    this.updateStickyElements(scrollTop);
    
    // Lazy load content
    this.checkLazyLoad(scrollTop);
  }

  handleResize() {
    // Optimized resize handler
    const wasMobile = this.isMobile;
    this.isMobile = this.detectMobile();
    
    if (wasMobile !== this.isMobile) {
      // Handle mobile/desktop transition
      this.handleViewportChange();
    }
  }

  updateStickyElements(scrollTop) {
    const header = document.querySelector('.header');
    if (header) {
      if (scrollTop > 100) {
        header.classList.add('scrolled');
      } else {
        header.classList.remove('scrolled');
      }
    }
  }

  checkLazyLoad(scrollTop) {
    // Simple lazy load check for elements without IntersectionObserver
    const lazyElements = document.querySelectorAll('[data-lazy]');
    lazyElements.forEach(el => {
      const rect = el.getBoundingClientRect();
      if (rect.top < window.innerHeight + 200) {
        this.loadElement(el);
      }
    });
  }

  loadElement(element) {
    // Load element content
    if (element.dataset.lazy === 'image' && element.dataset.src) {
      element.src = element.dataset.src;
      element.removeAttribute('data-lazy');
    }
  }

  handleViewportChange() {
    if (this.isMobile) {
      document.body.classList.add('mobile-view');
      this.enableMobileOptimizations();
    } else {
      document.body.classList.remove('mobile-view');
      this.disableMobileOptimizations();
    }
  }

  enableMobileOptimizations() {
    // Enable mobile-specific optimizations
    document.documentElement.style.setProperty('--font-size-base', '16px');
    
    // Reduce animation complexity on low-end devices
    if (this.isLowEnd) {
      this.enableLowPerformanceMode();
    }
  }

  disableMobileOptimizations() {
    // Disable mobile-specific optimizations
    document.documentElement.style.setProperty('--font-size-base', '14px');
    document.body.classList.remove('low-performance-mode');
  }

  // Utility method to check if device is in portrait mode
  isPortrait() {
    return window.innerHeight > window.innerWidth;
  }

  // Utility method to get device pixel ratio
  getPixelRatio() {
    return window.devicePixelRatio || 1;
  }

  // Utility method to check for touch support
  hasTouchSupport() {
    return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
  }
}

// Initialize mobile performance optimizations
document.addEventListener('DOMContentLoaded', () => {
  // Wait for app.js to initialize first
  setTimeout(() => {
    if (!window.mobilePerformance) {
      try {
        window.mobilePerformance = new MobilePerformance();
        console.log('[Mobile Performance] Performance optimizations initialized');
      } catch (error) {
        console.error('[Mobile Performance] Failed to initialize performance optimizations:', error);
      }
    }
  }, 1200);
});

// Export for use in other modules
export default MobilePerformance;