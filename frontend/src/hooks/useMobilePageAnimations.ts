/**
 * Mobile page animation utilities using anime.js v4.
 *
 * Provides:
 * - `useMobilePageAnimations(rootRef)` — staggered reveal for page sections
 * - `animateWalletReveal(element)` — premium wallet hero entrance
 * - `animateCardsReveal(elements)` — staggered card list entrance
 *
 * All animations respect `prefers-reduced-motion`.
 */
import { useEffect, useRef, useCallback } from 'react';
import { animate, stagger, createScope, utils } from 'animejs';

/** Check once at load — avoids per-frame media queries. */
const prefersReducedMotion =
  typeof window !== 'undefined'
    ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
    : false;

// ---------------------------------------------------------------------------
// Wallet hero entrance — premium gradient glow + scale + fade
// ---------------------------------------------------------------------------

export function animateWalletReveal(el: HTMLElement) {
  if (prefersReducedMotion || !el) return;

  animate(el, {
    opacity: [0, 1],
    translateY: [12, 0],
    scale: [0.97, 1],
    duration: 600,
    ease: 'out(3)',
  });

  // Glow pulse on the balance number
  const balance = el.querySelector('[data-balance]');
  if (balance) {
    animate(balance, {
      opacity: [0, 1],
      translateY: [8, 0],
      duration: 500,
      delay: 200,
      ease: 'out(4)',
    });
  }

  // Meta grid items stagger
  const metaItems = el.querySelectorAll('[data-meta]');
  if (metaItems.length) {
    animate(metaItems, {
      opacity: [0, 1],
      translateY: [6, 0],
      delay: stagger(80, { start: 350 }),
      duration: 400,
      ease: 'out(3)',
    });
  }
}

// ---------------------------------------------------------------------------
// Staggered card list entrance
// ---------------------------------------------------------------------------

export function animateCardsReveal(elements: HTMLElement[], baseDelay = 0) {
  if (prefersReducedMotion || !elements.length) return;

  animate(elements, {
    opacity: [0, 1],
    translateY: [16, 0],
    scale: [0.98, 1],
    delay: stagger(60, { start: baseDelay }),
    duration: 450,
    ease: 'out(3)',
  });
}

// ---------------------------------------------------------------------------
// Stats strip entrance
// ---------------------------------------------------------------------------

export function animateStatsReveal(el: HTMLElement) {
  if (prefersReducedMotion || !el) return;

  const items = el.querySelectorAll('[data-stat]');
  if (items.length) {
    animate(items, {
      opacity: [0, 1],
      translateY: [10, 0],
      delay: stagger(50, { start: 400 }),
      duration: 400,
      ease: 'out(3)',
    });
  }
}

// ---------------------------------------------------------------------------
// Section reveal (generic)
// ---------------------------------------------------------------------------

export function animateSectionReveal(el: HTMLElement, delayMs = 0) {
  if (prefersReducedMotion || !el) return;

  animate(el, {
    opacity: [0, 1],
    translateY: [20, 0],
    duration: 500,
    delay: delayMs,
    ease: 'out(3)',
  });
}

// ---------------------------------------------------------------------------
// Hook: full-page mobile reveal orchestration
// ---------------------------------------------------------------------------

export function useMobilePageAnimations() {
  const rootRef = useRef<HTMLDivElement>(null);
  const scopeRef = useRef<ReturnType<typeof createScope> | null>(null);

  useEffect(() => {
    if (!rootRef.current || prefersReducedMotion) return;

    scopeRef.current = createScope({ root: rootRef.current }).add(() => {
      // 1. Wallet hero
      const wallet = rootRef.current!.querySelector<HTMLElement>('[data-mobile-wallet]');
      if (wallet) animateWalletReveal(wallet);

      // 2. Page header
      const header = rootRef.current!.querySelector<HTMLElement>('[data-mobile-header]');
      if (header) {
        animate(header, {
          opacity: [0, 1],
          translateY: [-8, 0],
          duration: 400,
          delay: 100,
          ease: 'out(3)',
        });
      }

      // 3. Stats strips
      const stats = rootRef.current!.querySelectorAll<HTMLElement>('[data-mobile-stats]');
      stats.forEach((s) => animateStatsReveal(s));

      // 4. Main content section
      const content = rootRef.current!.querySelector<HTMLElement>('[data-mobile-content]');
      if (content) {
        animateSectionReveal(content, 500);
      }

      // 5. Tab bar
      const tabBar = rootRef.current!.querySelector<HTMLElement>('[data-mobile-tabs]');
      if (tabBar) {
        animate(tabBar, {
          opacity: [0, 1],
          duration: 400,
          delay: 300,
          ease: 'out(3)',
        });
      }
    });

    return () => {
      scopeRef.current?.revert();
      scopeRef.current = null;
    };
  }, []);

  return rootRef;
}

// ---------------------------------------------------------------------------
// Number counter animation (for balance display)
// ---------------------------------------------------------------------------

export function animateNumberCounter(
  el: HTMLElement,
  from: number,
  to: number,
  duration = 800,
) {
  if (prefersReducedMotion || !el) return;

  const obj = { value: from };
  animate(obj, {
    value: to,
    round: 1,
    duration,
    ease: 'outExpo',
    onUpdate: () => {
      el.textContent = obj.value.toLocaleString('vi-VN');
    },
  });
}
