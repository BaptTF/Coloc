// ===== SERVICE WORKER FOR COLOC VIDEO DOWNLOADER =====
// Version: 1.0.0
// Features: Offline caching, background sync, push notifications

const CACHE_NAME = 'coloc-downloader-v1.0.0';
const STATIC_CACHE = 'coloc-static-v1.0.0';
const RUNTIME_CACHE = 'coloc-runtime-v1.0.0';

// Assets to cache for offline functionality
const STATIC_ASSETS = [
  '/',
  '/static/styles.css',
  '/static/js/app.js',
  '/static/js/config.js',
  '/static/js/state.js',
  '/static/js/utils.js',
  '/static/js/toast.js',
  '/static/js/api.js',
  '/static/js/websocket.js',
  '/static/js/download.js',
  '/static/js/status.js',
  '/static/js/vlc.js',
  '/static/js/modal.js',
  '/static/js/video.js',
  '/static/js/theme.js',
  '/static/js/events.js',
  '/static/js/pwa.js',
  '/static/js/mobile.js',
  '/static/js/mobile-performance.js',
  '/static/manifest.json',
  '/static/icons/icon-192x192.png',
  '/static/icons/icon-512x512.png'
];

// ===== INSTALLATION =====
self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker v1.0.0');
  
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => {
        console.log('[SW] Caching static assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => {
        console.log('[SW] Static assets cached successfully');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[SW] Failed to cache static assets:', error);
      })
  );
});

// ===== ACTIVATION =====
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker v1.0.0');
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== STATIC_CACHE && cacheName !== RUNTIME_CACHE) {
              console.log('[SW] Deleting old cache:', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        console.log('[SW] Service worker activated');
        return self.clients.claim();
      })
  );
});

// ===== FETCH STRATEGY =====
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Skip non-GET requests
  if (request.method !== 'GET') {
    return;
  }
  
  // Handle different request types
  if (url.origin === self.location.origin) {
    // Same origin requests
    if (url.pathname === '/') {
      event.respondWith(handleRootRequest(request));
    } else if (isStaticAsset(request)) {
      event.respondWith(handleStaticAsset(request));
    } else if (isAPIRequest(request)) {
      event.respondWith(handleAPIRequest(request));
    } else {
      event.respondWith(handleNavigationRequest(request));
    }
  } else {
    // Cross-origin requests (API calls, external resources)
    event.respondWith(handleExternalRequest(request));
  }
});

// ===== REQUEST HANDLERS =====

// Handle root navigation
async function handleRootRequest(request) {
  try {
    // Try network first for fresh content
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      // Cache the fresh response
      const cache = await caches.open(STATIC_CACHE);
      cache.put(request, networkResponse.clone());
      return networkResponse;
    }
  } catch (error) {
    console.log('[SW] Network failed, serving from cache');
  }
  
  // Fallback to cache
  return caches.match(request) || caches.match('/');
}

// Handle static assets (CSS, JS, images)
async function handleStaticAsset(request) {
  const cache = await caches.open(STATIC_CACHE);
  const cachedResponse = await cache.match(request);
  
  if (cachedResponse) {
    // Return cached version immediately
    return cachedResponse;
  }
  
  try {
    // Fetch from network and cache
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    console.error('[SW] Failed to fetch static asset:', request.url);
    return new Response('Asset not available offline', { status: 503 });
  }
}

// Handle API requests
async function handleAPIRequest(request) {
  try {
    // Always try network first for API calls
    const networkResponse = await fetch(request);
    
    // Cache successful GET responses
    if (networkResponse.ok && request.method === 'GET') {
      const cache = await caches.open(RUNTIME_CACHE);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('[SW] API request failed, checking cache');
    
    // Try to serve from cache for GET requests
    if (request.method === 'GET') {
      const cachedResponse = await caches.match(request);
      if (cachedResponse) {
        return cachedResponse;
      }
    }
    
    // Return offline error
    return new Response(
      JSON.stringify({ 
        error: 'Network unavailable', 
        offline: true,
        message: 'This request requires an internet connection'
      }),
      {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
      }
    );
  }
}

// Handle navigation requests (SPA routing)
async function handleNavigationRequest(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      return networkResponse;
    }
  } catch (error) {
    console.log('[SW] Navigation failed, serving app shell');
  }
  
  // Always serve the app shell for navigation
  return caches.match('/');
}

// Handle external requests
async function handleExternalRequest(request) {
  try {
    return await fetch(request);
  } catch (error) {
    console.error('[SW] External request failed:', request.url);
    return new Response('Network unavailable', { status: 503 });
  }
}

// ===== BACKGROUND SYNC =====
self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync event:', event.tag);
  
  if (event.tag === 'background-download') {
    event.waitUntil(handleBackgroundDownload());
  } else if (event.tag === 'retry-failed-downloads') {
    event.waitUntil(handleRetryFailedDownloads());
  }
});

// Handle background downloads
async function handleBackgroundDownload() {
  try {
    // Get pending downloads from IndexedDB
    const pendingDownloads = await getPendingDownloads();
    
    for (const download of pendingDownloads) {
      try {
        // Retry the download
        const response = await fetch('/api/download', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(download)
        });
        
        if (response.ok) {
          // Remove from pending queue
          await removePendingDownload(download.id);
          console.log('[SW] Background download successful:', download.id);
        }
      } catch (error) {
        console.error('[SW] Background download failed:', download.id, error);
      }
    }
  } catch (error) {
    console.error('[SW] Background sync failed:', error);
  }
}

// ===== PUSH NOTIFICATIONS =====
self.addEventListener('push', (event) => {
  console.log('[SW] Push notification received');
  
  if (!event.data) {
    return;
  }
  
  const options = {
    body: event.data.text(),
    icon: '/static/icons/icon-192x192.png',
    badge: '/static/icons/badge-72x72.png',
    vibrate: [200, 100, 200],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: 1
    },
    actions: [
      {
        action: 'explore',
        title: 'View Download',
        icon: '/static/icons/checkmark.png'
      },
      {
        action: 'close',
        title: 'Close',
        icon: '/static/icons/xmark.png'
      }
    ]
  };
  
  event.waitUntil(
    self.registration.showNotification('Coloc Downloader', options)
  );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
  console.log('[SW] Notification click received');
  
  event.notification.close();
  
  if (event.action === 'explore') {
    // Open the app to the downloads section
    event.waitUntil(
      clients.openWindow('/?section=downloads')
    );
  } else if (event.action === 'close') {
    // Just close the notification
    return;
  } else {
    // Default action: open the app
    event.waitUntil(
      clients.matchAll().then((clientList) => {
        for (const client of clientList) {
          if (client.url === '/' && 'focus' in client) {
            return client.focus();
          }
        }
        if (clients.openWindow) {
          return clients.openWindow('/');
        }
      })
    );
  }
});

// ===== UTILITY FUNCTIONS =====

function isStaticAsset(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/static/') || 
         url.pathname.endsWith('.css') || 
         url.pathname.endsWith('.js') || 
         url.pathname.endsWith('.png') || 
         url.pathname.endsWith('.jpg') || 
         url.pathname.endsWith('.svg');
}

function isAPIRequest(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/api/') || 
         url.pathname.startsWith('/ws') ||
         url.pathname.startsWith('/share-target');
}

// IndexedDB helpers for background sync
async function getPendingDownloads() {
  // This would integrate with IndexedDB
  // For now, return empty array
  return [];
}

async function removePendingDownload(id) {
  // This would remove from IndexedDB
  console.log('[SW] Would remove pending download:', id);
}

// ===== MESSAGE HANDLING =====
self.addEventListener('message', (event) => {
  console.log('[SW] Message received:', event.data);
  
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  } else if (event.data && event.data.type === 'CACHE_UPDATE') {
    event.waitUntil(updateCache());
  }
});

// Update cache manually
async function updateCache() {
  try {
    const cache = await caches.open(STATIC_CACHE);
    await cache.addAll(STATIC_ASSETS);
    console.log('[SW] Cache updated successfully');
  } catch (error) {
    console.error('[SW] Cache update failed:', error);
  }
}

console.log('[SW] Service worker loaded successfully');