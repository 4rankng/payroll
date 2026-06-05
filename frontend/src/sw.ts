/// <reference lib="webworker" />

// Custom Service Worker for TingTing PWA
// Combines Workbox precaching with push notification handling

declare const self: ServiceWorkerGlobalScope & {
  __WB_MANIFEST: (string | { url: string; revision: string })[];
};

// Precache manifest injected by VitePWA injectManifest
const PRECACHE_MANIFEST = self.__WB_MANIFEST;

// Cache name — bump version to force SW update when push handler changes
const CACHE_NAME = 'tingting-cache-v2';

// Install event — precache assets
self.addEventListener('install', (event: ExtendableEvent) => {
  const precacheUrls = PRECACHE_MANIFEST.map((entry) =>
    typeof entry === 'string' ? entry : entry.url
  );
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(precacheUrls))
  );
  self.skipWaiting();
});

// Activate event — clean old caches
self.addEventListener('activate', (event: ExtendableEvent) => {
  event.waitUntil(
    caches.keys().then((names) =>
      Promise.all(
        names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name))
      )
    )
  );
  self.clients.claim();
});

// Fetch event — network first for API, cache first for assets
self.addEventListener('fetch', (event: FetchEvent) => {
  const url = new URL(event.request.url);

  // Only handle same-origin requests; let cross-origin (dev server HMR, etc.) pass through
  if (url.origin !== self.location.origin) return;

  // Skip non-GET requests
  if (event.request.method !== 'GET') return;

  // Static assets — cache first, then network
  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) return cached;
      return fetch(event.request).catch(() => new Response('Offline', { status: 503, statusText: 'Service Unavailable' }));
    })
  );
});

// =====================
// Push Notification Handler
// =====================

/**
 * Strip HTML tags from a string so OS push notifications show plain text.
 * The backend should already send plain text, but this is a safety net.
 */
function stripHtml(str: string): string {
  return str
    .replace(/<[^>]+>/g, '')          // remove tags
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&nbsp;/g, ' ')
    .trim();
}

self.addEventListener('push', (event: PushEvent) => {
  if (!event.data) return;

  let data;
  try {
    data = event.data.json();
  } catch {
    data = { body: event.data.text() };
  }

  const title = data.title || 'TingTing';
  const url = data.url || '/';
  const rawBody: string = data.body || '';
  const plainBody = stripHtml(rawBody);
  const options = {
    body: plainBody,
    icon: data.icon || '/favicon.png',
    badge: '/favicon.png',
    tag: 'tingting-notification',
    data: { url },
    vibrate: [100, 50, 100],
    requireInteraction: false,
    renotify: true,
  };

  // Also notify all open clients so the app can react (badge count, toast)
  const notifyClients = self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
    for (const client of clients) {
      client.postMessage({
        type: 'PUSH_NOTIFICATION',
        payload: { title, body: plainBody, url },
      });
    }
  });

  event.waitUntil(
    Promise.all([
      self.registration.showNotification(title, options),
      notifyClients,
    ])
  );
});

// Handle notification click — open/focus the app
self.addEventListener('notificationclick', (event: NotificationEvent) => {
  event.notification.close();

  const urlToOpen = event.notification.data?.url || '/';

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // If app is already open, navigate to the target URL and focus
      for (const client of clientList) {
        if (client.url.includes(self.location.origin) && 'focus' in client) {
          // Post message so app can handle navigation internally
          client.postMessage({
            type: 'NOTIFICATION_CLICK',
            payload: { url: urlToOpen },
          });
          return client.focus();
        }
      }
      // No existing window — open new one
      return self.clients.openWindow(urlToOpen);
    })
  );
});

// Push subscription change — app will re-subscribe on next load
self.addEventListener('pushsubscriptionchange', () => {
  console.log('Push subscription changed');
});
