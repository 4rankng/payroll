import { createRoot } from 'react-dom/client'
import { registerSW } from 'virtual:pwa-register'
import App from './App.tsx'
import './index.css'
import { installConsoleFilters } from './lib/console-filter';
import { isChunkLoadError, reloadForFreshChunk } from './lib/chunk-reload';

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
