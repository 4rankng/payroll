import '@testing-library/jest-dom';

// jsdom does not implement `window.matchMedia`. Several modules (e.g.
// `AnimatedCurrency`, `useMobilePageAnimations`) read it at module-eval time,
// so without this mock any test that imports them fails before running.
// `matches: true` for `(prefers-reduced-motion: reduce)` keeps animated
// amounts static so tests can assert their rendered text immediately.
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string): MediaQueryList => ({
    matches: query.includes('prefers-reduced-motion'),
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

// jsdom does not implement `ResizeObserver`. Radix UI primitives (e.g. the
// Slider in SettingCard) measure elements through it at mount time, so any
// test rendering them crashes without a no-op stub.
class ResizeObserverStub {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
Object.defineProperty(window, 'ResizeObserver', {
  configurable: true,
  writable: true,
  value: ResizeObserverStub,
});

// jsdom ships no working `window.localStorage` (it is a bare object with no
// getItem/setItem), so any code touching storage throws or silently no-ops
// under test. Back it with an in-memory Storage so persistence-dependent
// behaviour is exercised instead of accidentally taking a failure path.
// A Map (not an object) because Storage keys are dynamic and `key()` must
// answer by insertion order.
const memoryStorage = new Map<string, string>();
const storageStub: Storage = {
  get length() {
    return memoryStorage.size;
  },
  clear: () => memoryStorage.clear(),
  getItem: (key) => memoryStorage.get(key) ?? null,
  key: (index) => Array.from(memoryStorage.keys())[index] ?? null,
  removeItem: (key) => void memoryStorage.delete(key),
  setItem: (key, value) => void memoryStorage.set(key, String(value)),
};
Object.defineProperty(window, 'localStorage', {
  configurable: true,
  writable: true,
  value: storageStub,
});