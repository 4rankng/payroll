import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

/**
 * UU PRO badge system (vendored reference: `@/components/base/badges`):
 * soft-tinted utility-ladder chips (utility-{c}-50 surface, -700 copy,
 * -200 ring). App-specific role variants have no UU equivalent and stay
 * as solid cva entries layered on UU tokens where the W1 bridge provides
 * the family (`partner`, `role`).
 *
 * Bridge gap (owner call, not silently swapped): no utility-blue ladder
 * exists yet, so `info` and `admin` keep the legacy tailwind-default
 * palette.
 */
const badgeVariants = cva(
  "inline-flex items-center whitespace-nowrap rounded-md ring-1 ring-inset px-1.5 py-0.5 text-xs font-medium leading-none transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring",
  {
    variants: {
      variant: {
        default:
          "bg-utility-brand-50 text-utility-brand-700 ring-utility-brand-200",
        secondary:
          "bg-utility-gray-50 text-utility-gray-700 ring-utility-gray-200",
        destructive:
          "bg-utility-error-50 text-utility-error-700 ring-utility-error-200",
        outline: "bg-transparent text-fg-primary ring-utility-gray-300",
        success:
          "bg-utility-success-50 text-utility-success-700 ring-utility-success-200",
        warning:
          "bg-utility-warning-50 text-utility-warning-700 ring-utility-warning-200",
        // info/admin/manager: solid role-identity chips kept, darkened to the
        // 700 palette steps so white copy clears the repo's 4.5:1 WCAG gate
        // (sky-600 was 4.1:1, teal-600 3.8:1, blue-600 borderline).
        info: "bg-sky-700 text-white hover:bg-sky-800",
        admin: "bg-blue-700 text-white hover:bg-blue-800",
        partner:
          "bg-brand-solid text-fg-white hover:bg-brand-solid_hover",
        manager: "bg-teal-700 text-white hover:bg-teal-800",
        role: "bg-utility-gray-600 text-fg-white hover:bg-utility-gray-700",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant, ...props }, ref) => {
    return (
      <span
        ref={ref}
        className={cn(badgeVariants({ variant }), className)}
        {...props}
      />
    )
  }
)
Badge.displayName = "Badge"

export { Badge, badgeVariants }
