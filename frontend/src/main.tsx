import { createRoot } from 'react-dom/client'
import { registerSW } from 'virtual:pwa-register'
import App from './App.tsx'
import './index.css'
import { installConsoleFilters } from './lib/console-filter';

// Dev-only: filter noisy extension console errors without masking real issues
if (import.meta.env.DEV) {
  installConsoleFilters();
}

// Register the service worker (drives push notifications + offline cache).
// VitePWA's `injectManifest` strategy does NOT auto-register — we have to
// call this ourselves. Safe in both dev and prod once `devOptions.enabled`
// is set in vite.config.ts.
if ('serviceWorker' in navigator) {
  registerSW({ immediate: true });
}

createRoot(document.getElementById("root")!).render(<App />);
