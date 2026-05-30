import { cn } from "@/lib/utils";
import { type ClassValue } from "clsx";

/**
 * Typography utility for consistent text styling across the application
 * Based on Material Design 3 typography system
 */

export type TypographyVariant = 
  | "display-large"
  | "display-medium" 
  | "display-small"
  | "headline-large"
  | "headline-medium"
  | "headline-small"
  | "title-large"
  | "title-medium"
  | "title-small"
  | "body-large"
  | "body-medium"
  | "body-small"
  | "label-large"
  | "label-medium"
  | "label-small";

export type TypographyColor =
  | "primary"
  | "secondary"
  | "muted"
  | "accent"
  | "destructive"
  | "success"
  | "warning"
  | "info"
  | "high-contrast"
  | "medium-contrast"
  | "monochrome"
  | "monochrome-secondary";

interface TypographyProps {
  variant?: TypographyVariant;
  color?: TypographyColor;
  className?: ClassValue;
  responsive?: boolean;
}

/**
 * Get typography classes for a specific variant
 */
export function getTypographyClasses({
  variant = "body-medium",
  color = "primary",
  className,
  responsive = true,
}: TypographyProps): string {
  const baseClasses = `typography-${variant}`;
  
  const colorClasses = {
    primary: "text-foreground",
    secondary: "text-secondary-foreground",
    muted: "text-muted-foreground",
    accent: "text-accent-foreground",
    destructive: "text-destructive",
    success: "text-emerald-600 dark:text-emerald-400",
    warning: "text-amber-600 dark:text-amber-400",
    info: "text-blue-600 dark:text-blue-400",
    "high-contrast": "text-high-contrast",
    "medium-contrast": "text-medium-contrast",
    monochrome: "text-black dark:text-white",
    "monochrome-secondary": "text-gray-800 dark:text-gray-200",
  };

  const responsiveClasses = responsive ? "transition-smooth" : "";

  return cn(
    baseClasses,
    colorClasses[color],
    responsiveClasses,
    className
  );
}

/**
 * Typography component props for React components
 */
export interface TypographyComponentProps extends TypographyProps {
  children: React.ReactNode;
  as?: keyof JSX.IntrinsicElements;
}

/**
 * Semantic heading variants mapping
 */
export const headingVariants = {
  h1: "display-large",
  h2: "display-medium", 
  h3: "headline-large",
  h4: "headline-medium",
  h5: "headline-small",
  h6: "title-large",
} as const;

/**
 * Common typography presets for different use cases
 */
export const typographyPresets = {
  // Page headers
  pageTitle: {
    variant: "display-medium" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  pageSubtitle: {
    variant: "headline-small" as TypographyVariant,
    color: "muted" as TypographyColor,
  },
  
  // Card content
  cardTitle: {
    variant: "title-large" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  cardDescription: {
    variant: "body-medium" as TypographyVariant,
    color: "muted" as TypographyColor,
  },
  
  // Form elements
  formLabel: {
    variant: "label-medium" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  formHelper: {
    variant: "body-small" as TypographyVariant,
    color: "muted" as TypographyColor,
  },
  formError: {
    variant: "body-small" as TypographyVariant,
    color: "destructive" as TypographyColor,
  },
  
  // Table content
  tableHeader: {
    variant: "label-large" as TypographyVariant,
    color: "medium-contrast" as TypographyColor,
  },
  tableCell: {
    variant: "body-medium" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  
  // Navigation
  navLabel: {
    variant: "label-large" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  
  // Status and badges
  statusText: {
    variant: "label-small" as TypographyVariant,
    color: "medium-contrast" as TypographyColor,
  },
  
  // Modal content
  modalTitle: {
    variant: "headline-small" as TypographyVariant,
    color: "primary" as TypographyColor,
  },
  modalDescription: {
    variant: "body-medium" as TypographyVariant,
    color: "muted" as TypographyColor,
  },

  // Monochrome variants
  monochromeTitle: {
    variant: "display-medium" as TypographyVariant,
    color: "monochrome" as TypographyColor,
  },
  monochromeHeadline: {
    variant: "headline-large" as TypographyVariant,
    color: "monochrome" as TypographyColor,
  },
  monochromeSubtitle: {
    variant: "title-large" as TypographyVariant,
    color: "monochrome-secondary" as TypographyColor,
  },
} as const;

/**
 * Helper function to get responsive font sizes
 */
export function getResponsiveFontSize(
  base: string,
  sm?: string,
  md?: string,
  lg?: string
): string {
  let classes = `text-${base}`;
  if (sm) classes += ` sm:text-${sm}`;
  if (md) classes += ` md:text-${md}`;
  if (lg) classes += ` lg:text-${lg}`;
  return classes;
}

/**
 * Helper to ensure text readability and contrast
 */
export function getReadableTextClasses(
  background: "light" | "dark" | "colored" = "light"
): string {
  const contrastClasses = {
    light: "text-slate-900 dark:text-slate-100",
    dark: "text-slate-100 dark:text-slate-900", 
    colored: "text-white dark:text-slate-900",
  };
  
  return cn(
    contrastClasses[background],
    "text-readable" // Custom class from index.css
  );
}

/**
 * Typography scale utilities for consistent spacing
 */
export const typographyScale = {
  xs: "0.6875rem",    // 11px
  sm: "0.75rem",      // 12px
  base: "0.8125rem",  // 13px
  lg: "0.875rem",     // 14px
  xl: "1rem",         // 16px
  "2xl": "1.125rem",  // 18px
  "3xl": "1.25rem",   // 20px
  "4xl": "1.5rem",    // 24px
  "5xl": "1.75rem",   // 28px
} as const;

/**
 * Line height utilities
 */
export const lineHeights = {
  none: "1",
  tight: "1.15",
  snug: "1.3",
  normal: "1.4",
  relaxed: "1.5",
  loose: "1.65",
} as const;

/**
 * Letter spacing utilities
 */
export const letterSpacing = {
  tighter: "-0.05em",
  tight: "-0.025em",
  normal: "0em",
  wide: "0.025em",
  wider: "0.05em",
  widest: "0.1em",
} as const;