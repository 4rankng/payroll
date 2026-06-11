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
 * Drop-in replacement for the old useIsMobile / use-mobile hooks.
 */
export const useIsMobile = (): boolean => useBreakpoint('md');

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