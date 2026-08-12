import { lazy } from 'react';
import type { ComponentType } from 'react';
import { isChunkLoadError, reloadForFreshChunk } from './chunk-reload';

/**
 * Drop-in replacement for `React.lazy` that recovers from stale-chunk
 * failures (hashed assets deleted by a newer deploy) by triggering a single
 * hard reload instead of leaving the route blank or throwing into the
 * ErrorBoundary.
 *
 * Use this everywhere we previously used `lazy(() => import(...))` so the
 * router self-heals on the first navigation after a deploy.
 */
export function lazyWithReload<T extends ComponentType<unknown>>(
  factory: () => Promise<{ default: T }>,
) {
  const LazyComponent = lazy(async () => {
    try {
      return await factory();
    } catch (error) {
      if (isChunkLoadError(error)) {
        reloadForFreshChunk();
      }
      // Re-throw so Suspense / ErrorBoundary can render a fallback during the
      // brief window before the reload takes effect.
      throw error;
    }
  });
  return LazyComponent;
}
