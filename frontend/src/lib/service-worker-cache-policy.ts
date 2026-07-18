export interface AppCacheRequestContext {
  isDevelopment: boolean;
  isSameOrigin: boolean;
  method: string;
  pathname: string;
}

/**
 * Development runs through Vite's module graph and HMR, so its source files
 * must never be served from the PWA cache. The worker remains registered in
 * development for push-notification testing, but caching is production-only.
 */
export const shouldHandleAppCacheRequest = ({
  isDevelopment,
  isSameOrigin,
  method,
  pathname,
}: AppCacheRequestContext): boolean =>
  !isDevelopment &&
  isSameOrigin &&
  method === 'GET' &&
  !pathname.startsWith('/api/');
