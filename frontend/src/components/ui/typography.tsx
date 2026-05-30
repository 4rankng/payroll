import React from "react";
import { cn } from "@/lib/utils";
import { 
  getTypographyClasses, 
  type TypographyComponentProps,
  type TypographyVariant,
  type TypographyColor,
  headingVariants,
  typographyPresets 
} from "@/lib/typography";

/**
 * Typography component for consistent text styling
 * Supports all Material Design 3 typography variants
 */
export const Typography = React.forwardRef<
  HTMLElement,
  TypographyComponentProps
>(({ 
  children, 
  variant = "body-medium", 
  color = "primary", 
  className,
  responsive = true,
  as: Component = "p",
  ...props 
}, ref) => {
  const typographyClasses = getTypographyClasses({
    variant,
    color,
    className,
    responsive,
  });

  return (
    <Component
      ref={ref as React.Ref<HTMLElement>}
      className={typographyClasses}
      {...props}
    >
      {children}
    </Component>
  );
});

Typography.displayName = "Typography";

/**
 * Pre-configured heading components
 */
export const H1 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h1"
    variant="display-large"
    className={className}
    {...props}
  />
));

export const H2 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h2"
    variant="display-medium"
    className={className}
    {...props}
  />
));

export const H3 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h3"
    variant="headline-large"
    className={className}
    {...props}
  />
));

export const H4 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h4"
    variant="headline-medium"
    className={className}
    {...props}
  />
));

export const H5 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h5"
    variant="headline-small"
    className={className}
    {...props}
  />
));

export const H6 = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h6"
    variant="title-large"
    className={className}
    {...props}
  />
));

/**
 * Pre-configured text components for common use cases
 */
export const Body = React.forwardRef<
  HTMLParagraphElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="p"
    variant="body-medium"
    className={className}
    {...props}
  />
));

export const BodyLarge = React.forwardRef<
  HTMLParagraphElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="p"
    variant="body-large"
    className={className}
    {...props}
  />
));

export const BodySmall = React.forwardRef<
  HTMLParagraphElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="p"
    variant="body-small"
    className={className}
    {...props}
  />
));

export const Label = React.forwardRef<
  HTMLLabelElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="label"
    variant="label-medium"
    className={className}
    {...props}
  />
));

export const Caption = React.forwardRef<
  HTMLSpanElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="span"
    variant="body-small"
    color="muted"
    className={className}
    {...props}
  />
));

/**
 * Hook for using typography presets
 */
export function useTypographyPreset(preset: keyof typeof typographyPresets) {
  return typographyPresets[preset];
}

/**
 * Helper function to get semantic heading classes
 */
export function getHeadingClasses(level: 1 | 2 | 3 | 4 | 5 | 6): string {
  const variants: Record<number, TypographyVariant> = {
    1: "display-large",
    2: "display-medium", 
    3: "headline-large",
    4: "headline-medium",
    5: "headline-small",
    6: "title-large",
  };
  
  return getTypographyClasses({
    variant: variants[level],
    color: "primary",
  });
}

/**
 * Page title component with consistent styling
 */
export const PageTitle = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h1"
    variant="display-medium"
    className={cn("mb-2", className)}
    {...props}
  />
));

/**
 * Page subtitle component
 */
export const PageSubtitle = React.forwardRef<
  HTMLParagraphElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="p"
    variant="body-large"
    color="muted"
    className={className}
    {...props}
  />
));

/**
 * Card title component
 */
export const CardTitle = React.forwardRef<
  HTMLHeadingElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="h3"
    variant="title-large"
    className={cn("mb-2", className)}
    {...props}
  />
));

/**
 * Card description component
 */
export const CardDescription = React.forwardRef<
  HTMLParagraphElement,
  Omit<TypographyComponentProps, "as" | "variant">
>(({ className, ...props }, ref) => (
  <Typography
    ref={ref}
    as="p"
    variant="body-medium"
    color="muted"
    className={className}
    {...props}
  />
));

H1.displayName = "H1";
H2.displayName = "H2";
H3.displayName = "H3";
H4.displayName = "H4";
H5.displayName = "H5";
H6.displayName = "H6";
Body.displayName = "Body";
BodyLarge.displayName = "BodyLarge";
BodySmall.displayName = "BodySmall";
Label.displayName = "Label";
Caption.displayName = "Caption";
PageTitle.displayName = "PageTitle";
PageSubtitle.displayName = "PageSubtitle";
CardTitle.displayName = "CardTitle";
CardDescription.displayName = "CardDescription";
