import * as React from "react"

import { cn } from "@/lib/utils"

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input"> & {
  variant?: 'default' | 'filled' | 'outlined'
}>(({ className, type, variant = 'default', ...props }, ref) => {
  const variants = {
    default: "flex h-11 w-full rounded-md border border-input bg-card px-3 py-2 typography-body-medium ring-offset-background file:border-0 file:bg-transparent file:typography-body-medium file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 transition-colors sm:h-9 sm:px-2.5 sm:py-1.5",
    filled: "flex h-11 w-full rounded-md border-0 bg-muted px-3 py-2 typography-body-medium ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 transition-colors sm:h-9 sm:px-2.5 sm:py-1.5",
    outlined: "flex h-11 w-full rounded-md border-2 border-input bg-transparent px-3 py-2 typography-body-medium ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 transition-colors sm:h-9 sm:px-2.5 sm:py-1.5",
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
