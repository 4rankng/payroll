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
  writable: true,
  value: ResizeObserverStub,
});
