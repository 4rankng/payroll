import * as React from "react"

import { cn } from "@/lib/utils"

// UU PRO restyle (W3): white surface + gray-300 border + rounded-lg + shadow-xs
// after the vendored UU v7 input (surface/border read from the role-scoped
// input tokens so admin/partner themes keep applying). Focus uses the UU
// focus-visible outline treatment matching the vendored button. Vendored
// invalid styling (ring-error_subtle / ring-error) has no bridged tokens, so
// invalid maps to a utility-error-300 border + error outline.
const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input"> & {
  variant?: 'default' | 'filled' | 'outlined'
}>(({ className, type, variant = 'default', ...props }, ref) => {
  const variants = {
    default: "flex h-11 w-full rounded-lg border border-input bg-card px-3 py-2 typography-body-medium shadow-xs file:border-0 file:bg-transparent file:typography-body-medium file:font-medium file:text-foreground placeholder:text-muted-foreground outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 aria-[invalid=true]:border-utility-error-300 aria-[invalid=true]:focus-visible:outline-error disabled:cursor-not-allowed disabled:opacity-50 sm:h-9 sm:px-2.5 sm:py-1.5",
    filled: "flex h-11 w-full rounded-lg border-0 bg-muted px-3 py-2 typography-body-medium shadow-xs placeholder:text-muted-foreground outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 aria-[invalid=true]:border aria-[invalid=true]:border-utility-error-300 aria-[invalid=true]:focus-visible:outline-error disabled:cursor-not-allowed disabled:opacity-50 sm:h-9 sm:px-2.5 sm:py-1.5",
    outlined: "flex h-11 w-full rounded-lg border-2 border-input bg-transparent px-3 py-2 typography-body-medium shadow-xs placeholder:text-muted-foreground outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 aria-[invalid=true]:border-utility-error-300 aria-[invalid=true]:focus-visible:outline-error disabled:cursor-not-allowed disabled:opacity-50 sm:h-9 sm:px-2.5 sm:py-1.5",
  }

  return (
    <input
      type={type}
      lang={type === 'date' ? 'vi-VN' : undefined}
      className={cn(variants[variant], className)}
      ref={ref}
      {...props}
    />
  )
})
Input.displayName = "Input"

export { Input }
