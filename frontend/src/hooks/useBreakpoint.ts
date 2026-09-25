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
 * Convenience hook: returns true when viewport < 768px (Tailwind md breakpoint).
 *
 * Why `md` instead of `lg`?
 * - Below 768px (phones), the dedicated mobile pages (under `pages/mobile/`)
 *   give the best vertical-scroll experience.
 * - From 768px up — tablets — the desktop admin layout renders with the
 *   sidebar collapsed to an icon rail and the responsive data tables in dense
 *   mode, giving tablet users the full tabular data density they asked for.
 *   (Product decision 2026-09-25: tablets are data-first, no longer treated
 *   as phones.)
 *
 * Historical note: this hook used to switch at `lg` (1024px), which forced
 * iPads onto the phone card layouts. That was the regression the tablet
 * overhaul fixed.
 */
export const useIsMobile = (): boolean => useBreakpoint('md');

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