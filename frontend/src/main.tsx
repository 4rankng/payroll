import { createRoot } from 'react-dom/client'
import { registerSW } from 'virtual:pwa-register'
import App from './App.tsx'
import './index.css'
import { installConsoleFilters } from './lib/console-filter';
import { isChunkLoadError, reloadForFreshChunk } from './lib/chunk-reload';

// Remote font CSS must not hold up the initial interface. Apply it once loaded,
// including cached sheets that finished before this module ran. Keep handlers
// in this module because production CSP disallows inline event handlers.
document.querySelectorAll<HTMLLinkElement>('link[data-font-stylesheet]').forEach((link) => {
  const applyFontStyles = () => { link.media = 'all'; };
  if (link.sheet) applyFontStyles();
  else link.addEventListener('load', applyFontStyles, { once: true });
});

// Dev-only: filter noisy extension console errors without masking real issues
if (import.meta.env.DEV) {
  installConsoleFilters();
}

// Recover from stale-chunk failures (hashed assets deleted by a newer deploy)
// that happen OUTSIDE React.lazy — e.g. a shared chunk referenced by another
// chunk, or an inline dynamic import() in a dependency. Each React.lazy call
// already self-heals via lib/lazy; this catches the rest. The reload is
// guarded by sessionStorage (see chunk-reload.ts) so we never loop.
if (typeof window !== 'undefined') {
  const handleChunkFailure = (event: Event) => {
    const detail =
      (event as PromiseRejectionEvent).reason ??
      (event as ErrorEvent).error ??
      (event as ErrorEvent).message;
    if (isChunkLoadError(detail)) {
      reloadForFreshChunk();
    }
  };
  window.addEventListener('error', handleChunkFailure);
  window.addEventListener('unhandledrejection', handleChunkFailure);
}

// Register the service worker (drives push notifications + offline cache).
// VitePWA's `injectManifest` strategy does NOT auto-register — we have to
// call this ourselves. Safe in both dev and prod once `devOptions.enabled`
// is set in vite.config.ts.
if ('serviceWorker' in navigator) {
  registerSW({ immediate: true });
}

createRoot(document.getElementById("root")!).render(<App />);
