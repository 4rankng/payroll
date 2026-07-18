import { describe, expect, it } from 'vitest';
import { shouldHandleAppCacheRequest } from './service-worker-cache-policy';

const productionRequest = {
  isDevelopment: false,
  isSameOrigin: true,
  method: 'GET',
  pathname: '/assets/index.abc123.js',
};

describe('service worker application cache policy', () => {
  it('never handles Vite application requests in development', () => {
    expect(
      shouldHandleAppCacheRequest({
        ...productionRequest,
        isDevelopment: true,
        pathname: '/src/components/AdminSidebar.tsx',
      }),
    ).toBe(false);
  });

  it('keeps production hashed assets eligible for offline caching', () => {
    expect(shouldHandleAppCacheRequest(productionRequest)).toBe(true);
  });

  it.each([
    ['cross-origin request', { isSameOrigin: false }],
    ['mutation request', { method: 'POST' }],
    ['API request', { pathname: '/api/v1/users' }],
  ])('does not handle a %s', (_label, override) => {
    expect(
      shouldHandleAppCacheRequest({ ...productionRequest, ...override }),
    ).toBe(false);
  });
});
