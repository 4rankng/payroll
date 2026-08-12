/**
 * Recovery helpers for the classic Vite SPA "stale chunk" failure.
 *
 * After a deploy, the browser (or our service worker) may still hold a
 * reference to a hashed JS chunk that no longer exists on the server. When
 * React.lazy fires `import()` for that chunk, it fails with one of:
 *   - Chrome/Edge: "Failed to fetch dynamically imported module: <url>"
 *   - Firefox:     "Error loading dynamically imported module: <url>"
 *   - Safari:      "Importing a module script failed."
 *
 * The only reliable recovery is a hard reload so the browser fetches a fresh
 * `index.html` (served network-first by the SW) carrying the current chunk
 * hashes. We do it at most once per tab within a short window, guarded by
 * sessionStorage, so a persistently broken deploy surfaces an error UI
 * instead of trapping the user in a reload loop.
 */

const RELOAD_KEY = 'tt:chunk-reload-at';
const RELOAD_WINDOW_MS = 30_000;

/** Substrings browsers use when a dynamic import() 404s after a deploy. */
const CHUNK_ERROR_SIGNATURES = [
  'Failed to fetch dynamically imported module',
  'Error loading dynamically imported module',
  'Importing a module script failed',
] as const;

/** True when the error looks like a stale/missing hashed JS chunk. */
export function isChunkLoadError(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === 'string'
        ? error
        : (error as { message?: unknown })?.message instanceof Error
          ? ((error as { message: Error }).message.message as string)
          : String(error ?? '');
  if (!message) return false;
  return CHUNK_ERROR_SIGNATURES.some((sig) => message.includes(sig));
}

/**
 * Trigger a single hard reload to pick up fresh chunk hashes. Returns false
 * (without reloading) if we already attempted recently, so a persistently
 * broken deploy surfaces an error UI instead of looping forever. User-driven
 * reloads (the ErrorBoundary button) bypass this guard by calling
 * `window.location.reload()` directly.
 */
export function reloadForFreshChunk(): boolean {
  try {
    const last = sessionStorage.getItem(RELOAD_KEY);
    const now = Date.now();
    if (last && now - Number(last) < RELOAD_WINDOW_MS) {
      return false;
    }
    sessionStorage.setItem(RELOAD_KEY, String(now));
  } catch {
    // sessionStorage unavailable (e.g. private mode) — reload anyway.
  }
  // The SW uses network-first for navigations, so a normal reload fetches a
  // fresh index.html from the network with the current chunk references.
  window.location.reload();
  return true;
}
