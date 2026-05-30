// Lightweight logging utilities for development-only diagnostics.
// Avoids React DevTools component stack injection by deferring logs
// outside of React's commit phase and deduplicates by pathname.

const loggedNotFound = new Set<string>();

function diagnosticsEnabled(): boolean {
  try {
    if (typeof window === 'undefined') return false;
    // Prefer explicit env flag
    // Vite exposes import.meta.env.VITE_*; compare as string "true"
    const envFlag = import.meta.env?.VITE_DEBUG_DIAGNOSTICS as string | undefined;
    if (envFlag === 'true') return true;
    // Allow runtime toggle via localStorage (no reactivity; read at call time)
    const ls = window.localStorage?.getItem('debug:diagnostics');
    return ls === 'true';
  } catch {
    return false;
  }
}

export function logNotFoundPath(pathname: string): void {
  if (typeof window === 'undefined') return;
  if (!import.meta.env.DEV) return;
  if (!diagnosticsEnabled()) return;
  if (loggedNotFound.has(pathname)) return;
  loggedNotFound.add(pathname);

  // Defer to escape React commit stack so DevTools doesn't append component stack
  setTimeout(() => {
    // Vietnamese message to align with UI copy
    // Use info to avoid implying an actual error
    console.info('404: Người dùng truy cập đường dẫn không tồn tại:', pathname);
  }, 0);
}
