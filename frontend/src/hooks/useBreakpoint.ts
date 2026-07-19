import { useState, useEffect } from 'react';

const breakpoints = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
} as const;

type BreakpointKey = keyof typeof breakpoints;

export const useBreakpoint = (breakpoint: BreakpointKey): boolean => {
  const [isBelow, setIsBelow] = useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia(`(max-width: ${breakpoints[breakpoint] - 1}px)`);

    const handleChange = (e: MediaQueryListEvent) => {
      setIsBelow(e.matches);
    };

    setIsBelow(mediaQuery.matches);
    mediaQuery.addEventListener('change', handleChange);

    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [breakpoint]);

  return isBelow;
};

/**
 * Convenience hook: returns true when viewport < 1024px (Tailwind lg breakpoint).
 *
 * Why `lg` instead of `md`?
 * - The desktop admin/partner layout renders a 256px sidebar. Below lg (1024px),
 *   the remaining content area is too narrow for data tables — they overflow
 *   horizontally.
 * - The mobile-optimised pages (under `pages/mobile/`) are already polished for
 *   vertical scrolling on any width < 1024px, including tablets (768x1024).
 * - Treating tablet as mobile here gives us a clean rule: <lg uses mobile page,
 *   ≥lg uses desktop page. The single threshold keeps routing simple and avoids
 *   maintaining a third tablet layout.
 *
 * Drop-in replacement for the old useIsMobile / use-mobile hooks.
 */
export const useIsMobile = (): boolean => useBreakpoint('lg');

/**
 * Convenience hook: returns true when viewport is in the tablet range
 * (768px ≤ width ≤ 1023px). Useful for opting into tablet-specific layout
 * branching inside mobile-rendered pages (since useIsMobile treats tablet
 * as mobile). Does NOT change routing — purely a presentation hint.
 */
export const useIsTablet = (): boolean =>
  useMediaQuery('(min-width: 768px) and (max-width: 1023px)');

/**
 * Convenience hook: returns true when viewport ≥ 768px (Tailwind md breakpoint).
 * Useful for "tablet or desktop" branches (e.g., show a two-column grid).
 */
export const useIsTabletOrAbove = (): boolean => !useBreakpoint('md');

/**
 * Generic media-query hook — matches any CSS media query string.
 * Drop-in replacement for the old useMediaQuery / use-media-query hook.
 */
export const useMediaQuery = (query: string): boolean => {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    const media = window.matchMedia(query);
    setMatches(media.matches);

    const listener = (event: MediaQueryListEvent) => {
      setMatches(event.matches);
    };

    media.addEventListener('change', listener);
    return () => media.removeEventListener('change', listener);
  }, [query]);

  return matches;
};