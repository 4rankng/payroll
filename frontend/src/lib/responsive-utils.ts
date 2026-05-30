import { type ClassValue } from "clsx";

export const responsiveClasses = {
  padding: {
    sm: "p-3 sm:p-4 md:p-6",
    md: "p-4 sm:p-6 md:p-8",
    lg: "p-6 sm:p-8 md:p-10",
  },
  paddingX: {
    sm: "px-3 sm:px-4 md:px-6",
    md: "px-4 sm:px-6 md:px-8",
    lg: "px-6 sm:px-8 md:px-10",
  },
  paddingY: {
    sm: "py-3 sm:py-4 md:py-6",
    md: "py-4 sm:py-6 md:py-8",
    lg: "py-6 sm:py-8 md:py-10",
  },
  gap: {
    sm: "gap-2 sm:gap-3 md:gap-4",
    md: "gap-3 sm:gap-4 md:gap-6",
    lg: "gap-4 sm:gap-6 md:gap-8",
  },
  text: {
    xs: "typography-body-small sm:typography-body-medium",
    sm: "typography-body-medium sm:typography-body-large",
    base: "typography-body-large sm:typography-title-large",
    lg: "typography-title-large sm:typography-headline-medium md:typography-headline-large",
    xl: "typography-headline-medium sm:typography-headline-large md:typography-display-small",
    "2xl": "typography-headline-large sm:typography-display-small md:typography-display-medium",
    "3xl": "typography-display-small sm:typography-display-medium md:typography-display-large",
    heading: "typography-headline-large sm:typography-display-small md:typography-display-medium",
    subheading: "typography-title-large sm:typography-headline-medium md:typography-headline-large",
    body: "typography-body-medium sm:typography-body-large leading-relaxed",
  },
  container: {
    sm: "max-w-sm mx-auto",
    md: "max-w-3xl mx-auto",
    lg: "max-w-5xl mx-auto",
    xl: "max-w-7xl mx-auto",
    full: "w-full",
  },
  grid: {
    cols1: "grid-cols-1",
    cols2: "grid-cols-1 sm:grid-cols-2",
    cols3: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3",
    cols4: "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4",
    cols5: "grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5",
    cols6: "grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6",
  },
  flex: {
    stackedToRow: "flex flex-col sm:flex-row",
    stackedToRowReverse: "flex flex-col-reverse sm:flex-row",
    wrapToNoWrap: "flex flex-wrap sm:flex-nowrap",
    centerStacked: "flex flex-col items-center sm:flex-row sm:items-start",
  },
  spacing: {
    section: "py-6 sm:py-12 md:py-16", // Reduced mobile section padding
    sectionX: "px-3 sm:px-6 md:px-8", // Tighter mobile horizontal padding
    card: "p-3 sm:p-6", // Reduced mobile card padding
    modal: "p-4 sm:p-6 md:p-8",
    list: "py-2 px-3 sm:py-3 sm:px-4", // List item spacing
    tight: "py-1 px-2 sm:py-2 sm:px-3", // Tight spacing for mobile
  },
  minHeight: {
    touch: "min-h-[48px]", // iOS recommended touch target - increased for better accessibility
    button: "min-h-[48px] sm:min-h-[40px]", // Larger touch target on mobile
    input: "min-h-[48px] sm:min-h-[44px]", // Improved input accessibility
    row: "min-h-[52px] sm:min-h-[44px]", // Better row spacing on mobile
    card: "min-h-[60px]", // Minimum card height for touch interaction
  },
  width: {
    modalSm: "w-full sm:max-w-md",
    modalMd: "w-full sm:max-w-lg",
    modalLg: "w-full sm:max-w-2xl",
    modalXl: "w-full sm:max-w-4xl",
    modalFull: "w-full sm:max-w-6xl",
  },
  visibility: {
    mobileOnly: "sm:hidden",
    desktopOnly: "hidden sm:block",
    tabletUp: "hidden md:block",
    tabletOnly: "hidden sm:block lg:hidden",
  },
} as const;

/**
 * Media query breakpoints matching Tailwind CSS defaults
 */
export const breakpoints = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  "2xl": 1536,
} as const;

/**
 * Check if current viewport matches a breakpoint
 */
export function isBreakpoint(breakpoint: keyof typeof breakpoints): boolean {
  if (typeof window === "undefined") return false;
  return window.innerWidth >= breakpoints[breakpoint];
}

/**
 * Get current breakpoint
 */
export function getCurrentBreakpoint(): keyof typeof breakpoints | "xs" {
  if (typeof window === "undefined") return "xs";
  
  const width = window.innerWidth;
  
  if (width >= breakpoints["2xl"]) return "2xl";
  if (width >= breakpoints.xl) return "xl";
  if (width >= breakpoints.lg) return "lg";
  if (width >= breakpoints.md) return "md";
  if (width >= breakpoints.sm) return "sm";
  
  return "xs";
}

/**
 * Check if device is touch-enabled
 */
export function isTouchDevice(): boolean {
  if (typeof window === "undefined") return false;
  
  return (
    "ontouchstart" in window ||
    navigator.maxTouchPoints > 0 ||
    // @ts-ignore
    navigator.msMaxTouchPoints > 0
  );
}

/**
 * Check if device is mobile (based on viewport width)
 */
export function isMobile(): boolean {
  if (typeof window === "undefined") return false;
  return window.innerWidth < breakpoints.sm;
}

/**
 * Check if device is tablet (based on viewport width)
 */
export function isTablet(): boolean {
  if (typeof window === "undefined") return false;
  const width = window.innerWidth;
  return width >= breakpoints.sm && width < breakpoints.lg;
}

/**
 * Check if device is desktop (based on viewport width)
 */
export function isDesktop(): boolean {
  if (typeof window === "undefined") return false;
  return window.innerWidth >= breakpoints.lg;
}

/**
 * Get responsive value based on current breakpoint
 */
export function getResponsiveValue<T>(
  values: {
    xs?: T;
    sm?: T;
    md?: T;
    lg?: T;
    xl?: T;
    "2xl"?: T;
  },
  defaultValue: T
): T {
  const breakpoint = getCurrentBreakpoint();
  
  // Find the value for current breakpoint or fall back to smaller ones
  const breakpointOrder: Array<keyof typeof values> = ["2xl", "xl", "lg", "md", "sm", "xs"];
  const currentIndex = breakpointOrder.indexOf(breakpoint);
  
  for (let i = currentIndex; i < breakpointOrder.length; i++) {
    const bp = breakpointOrder[i];
    if (values[bp] !== undefined) {
      return values[bp];
    }
  }
  
  return defaultValue;
}

/**
 * Format responsive heading classes
 */
export function getResponsiveHeading(level: 1 | 2 | 3 | 4 | 5 | 6): string {
  const headingClasses = {
    1: "typography-display-small sm:typography-display-medium md:typography-display-large leading-tight",
    2: "typography-headline-large sm:typography-display-small md:typography-display-medium leading-tight",
    3: "typography-headline-medium sm:typography-headline-large md:typography-display-small leading-snug",
    4: "typography-title-large sm:typography-headline-medium md:typography-headline-large leading-snug",
    5: "typography-body-large sm:typography-title-large md:typography-headline-medium",
    6: "typography-body-medium sm:typography-body-large md:typography-title-large",
  };
  
  return headingClasses[level];
}

/**
 * Ensure minimum touch target size for interactive elements
 */
export function ensureTouchTarget(className?: ClassValue): string {
  return `${className || ""} ${responsiveClasses.minHeight.touch} min-w-[48px]`;
}

/**
 * Mobile-optimized button classes for better touch interaction
 */
export function getMobileButtonClasses(size: 'sm' | 'md' | 'lg' = 'md'): string {
  const sizeClasses = {
    sm: "h-10 px-3 text-sm",
    md: "h-12 px-4 text-base", 
    lg: "h-14 px-6 text-lg"
  };
  
  return `${sizeClasses[size]} sm:h-auto sm:px-auto sm:text-auto touch-manipulation active:scale-95 transition-transform`;
}

/**
 * Get responsive table classes (hide on mobile, show mobile-friendly alternative)
 */
export function getResponsiveTableClasses(): {
  desktop: string;
  mobile: string;
} {
  return {
    desktop: "hidden md:table",
    mobile: "md:hidden",
  };
}

/**
 * Safe area insets for mobile devices (notch, home indicator, etc.)
 */
export function getSafeAreaClasses(): string {
  return "pb-safe pt-safe px-safe";
}

/**
 * Mobile-optimized navigation classes
 */
export function getMobileNavClasses(): string {
  return "sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60";
}

/**
 * Mobile-optimized content container
 */
export function getMobileContentClasses(): string {
  return "px-3 sm:px-6 md:px-8 pb-20 sm:pb-8"; // Extra bottom padding for mobile nav
}

/**
 * Mobile-optimized modal classes
 */
export function getMobileModalClasses(): string {
  return "max-h-[90vh] overflow-y-auto rounded-t-lg sm:rounded-lg";
}

/**
 * Export utility for creating responsive font sizes
 */
export function responsiveFontSize(
  base: number,
  scale: { sm?: number; md?: number; lg?: number } = {}
): string {
  const sizes = {
    base: `${base}px`,
    sm: scale.sm ? `${base * scale.sm}px` : undefined,
    md: scale.md ? `${base * scale.md}px` : undefined,
    lg: scale.lg ? `${base * scale.lg}px` : undefined,
  };
  
  let className = `text-[${sizes.base}]`;
  if (sizes.sm) className += ` sm:text-[${sizes.sm}]`;
  if (sizes.md) className += ` md:text-[${sizes.md}]`;
  if (sizes.lg) className += ` lg:text-[${sizes.lg}]`;
  
  return className;
}