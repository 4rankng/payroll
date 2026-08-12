import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { isChunkLoadError, reloadForFreshChunk } from './chunk-reload';

describe('isChunkLoadError', () => {
  it('matches Chrome/Edge "Failed to fetch dynamically imported module" signature', () => {
    expect(
      isChunkLoadError(
        new Error('Failed to fetch dynamically imported module: https://tingting.vip/assets/index.dQkRnl5W.js'),
      ),
    ).toBe(true);
  });

  it('matches Firefox "Error loading dynamically imported module" signature', () => {
    expect(
      isChunkLoadError(new Error('Error loading dynamically imported module: https://tingting.vip/assets/index.abc.js')),
    ).toBe(true);
  });

  it('matches Safari "Importing a module script failed" signature', () => {
    expect(isChunkLoadError(new Error('Importing a module script failed.'))).toBe(true);
  });

  it('returns false for unrelated errors', () => {
    expect(isChunkLoadError(new Error('Network Error'))).toBe(false);
    expect(isChunkLoadError(new Error('undefined is not a function'))).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError(undefined)).toBe(false);
    expect(isChunkLoadError('')).toBe(false);
  });

  it('accepts a bare string', () => {
    expect(isChunkLoadError('Failed to fetch dynamically imported module: /assets/x.js')).toBe(true);
  });
});

describe('reloadForFreshChunk', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('reloads the page on the first call within a tab', () => {
    const reload = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { reload, href: 'https://tingting.vip/admin' },
      writable: true,
    });

    expect(reloadForFreshChunk()).toBe(true);
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it('refuses to reload again inside the guard window to prevent loops', () => {
    const reload = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { reload, href: 'https://tingting.vip/admin' },
      writable: true,
    });

    reloadForFreshChunk();
    expect(reloadForFreshChunk()).toBe(false);
    expect(reload).toHaveBeenCalledTimes(1); // still only the first call
  });
});
