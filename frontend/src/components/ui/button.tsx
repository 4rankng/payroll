import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cx } from "@/utils/cx"

/**
 * Untitled UI PRO button vocabulary, ported from the vendored style source
 * `components/base/buttons/button.tsx` (its `styles` export). The class
 * strings are restated here instead of imported so the compat surface stays a
 * native <button>/<Slot> element: 255 call sites rely on the React
 * ButtonHTMLAttributes contract (implicit type="submit" inside forms, native
 * `disabled`, DOM refs), and importing the vendored module would pull
 * react-aria-components into the entry bundle. Mirror vendored changes into
 * the color entries below when the source is re-vendored.
 *
 * UU surface tokens (`bg-primary`, `bg-primary_hover`) collide with this
 * app's shadcn roles (green brand), so neutral surfaces are expressed with
 * the W1 bridge's unambiguous families instead: utility-gray ladder,
 * fg-*, disabled tokens, and the solid color families.
 */
const buttonVariants = cva(
  [
    // Vendored styles.common.root (group/before hooks, UU outline focus ring,
    // h-max lets size entries win the compact height) + this app's legacy
    // gap and child-svg contract.
    "group relative inline-flex h-max cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap rounded-lg text-sm font-semibold select-none outline-brand transition duration-100 ease-linear before:absolute focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2",
    // Disabled styles (UU: explicit disabled tokens replace opacity).
    "disabled:cursor-not-allowed disabled:text-fg-disabled",
    // Icon styles: vendored data-icon rules + the legacy [&_svg] contract.
    "disabled:[&>[data-icon]]:text-fg-disabled_subtle",
    "[&>[data-icon]]:pointer-events-none [&>[data-icon]]:size-5 [&>[data-icon]]:shrink-0 [&>[data-icon]]:transition-inherit-all",
    "[&_svg]:pointer-events-none [&_svg]:shrink-0",
  ].join(" "),
  {
    variants: {
      variant: {
        // UU color: primary
        default: [
          "bg-brand-solid text-white shadow-xs-skeumorphic ring-1 ring-transparent ring-inset hover:bg-brand-solid_hover active:bg-brand-solid_hover",
          // Inner border gradient
          "before:absolute before:inset-px before:border before:border-white/12",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
          // Icon styles
          "[&>[data-icon]]:text-button-primary-icon hover:[&>[data-icon]]:text-button-primary-icon_hover",
        ].join(" "),

        // UU color: primary-destructive
        destructive: [
          "bg-error-solid text-white shadow-xs-skeumorphic ring-1 ring-transparent outline-error ring-inset hover:bg-error-solid_hover active:bg-error-solid_hover",
          // Inner border gradient
          "before:absolute before:inset-px before:border before:border-white/12",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
          // Icon styles
          "[&>[data-icon]]:text-button-destructive-primary-icon hover:[&>[data-icon]]:text-button-destructive-primary-icon_hover",
        ].join(" "),

        secondary: [
          "border border-utility-gray-300 bg-utility-gray-100 text-utility-gray-700 font-semibold hover:bg-utility-gray-200 active:bg-utility-gray-300",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
        ].join(" "),

        // White surface + gray border, UU "secondary gray" look expressed
        // with the border idiom because many call sites tint border-*.
        outline:
          "border border-border bg-card text-foreground hover:bg-muted active:bg-muted/80",

        // UU color: tertiary (hover surface = UU gray-200 via secondary_hover)
        ghost:
          "text-tertiary hover:bg-secondary_hover hover:text-tertiary_hover",

        // App-specific solid, patterned on UU primary with the success family
        // (no UU equivalent color).
        success: [
          "bg-success-solid text-white shadow-xs-skeumorphic ring-1 ring-transparent ring-inset hover:bg-success-solid_hover active:bg-success-solid_hover",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
        ].join(" "),

        // App-specific; the W1 bridge has no blue family, so the Tailwind
        // palette stays until a utility-info ladder is added.
        info: "bg-sky-600 text-white font-semibold hover:bg-sky-500 active:bg-sky-700 disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",

        // App-specific solid, patterned on UU primary with the warning family
        // (no UU equivalent color).
        warning: [
          "bg-warning-solid text-white shadow-xs-skeumorphic ring-1 ring-transparent ring-inset hover:bg-warning-solid_hover active:bg-warning-solid_hover",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
        ].join(" "),

        // App-specific on the utility-gray ladder (no UU equivalent color).
        monochrome: [
          "bg-utility-gray-700 text-white shadow-xs-skeumorphic ring-1 ring-transparent ring-inset hover:bg-utility-gray-600 active:bg-utility-gray-600",
          // Disabled styles
          "disabled:bg-disabled disabled:shadow-xs disabled:ring-disabled_subtle",
        ].join(" "),

        // App-specific on the utility-gray ladder (no UU equivalent color).
        "monochrome-outline":
          "border border-utility-gray-700 bg-transparent text-utility-gray-700 hover:bg-utility-gray-100 active:bg-utility-gray-200",

        // UU link-color vocabulary adapted to direct children (no data-text
        // wrapper in this compat surface).
        link: "justify-normal rounded-lg !p-0 h-auto font-normal text-brand-secondary underline-offset-2 hover:underline hover:text-brand-secondary_hover",
      },
      // Compact scale is contractual (control-density tests pin the class
      // heights); only radius/weight/surface migrate to UU.
      size: {
        default: "h-11 px-4 py-2 sm:h-9 sm:px-3 sm:py-1.5",
        sm: "h-11 px-3 text-xs sm:h-8 sm:px-2.5",
        lg: "h-11 px-6 sm:h-10",
        xl: "h-12 px-8 sm:h-11",
        icon: "h-11 w-11 sm:h-9 sm:w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={cx(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
