import * as React from "react"

import { cn } from "@/lib/utils"

// UU PRO restyle (W3): same treatment as ui/input — white surface, gray-300
// border, rounded-lg + shadow-xs, UU focus-visible outline, invalid via
// utility-error-300 border + error outline (vendored v7 ring tokens are not
// bridged). Height floor unchanged for data-dense screens.
export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[80px] w-full rounded-lg border border-input bg-card px-3 py-2 typography-body-medium shadow-xs placeholder:text-muted-foreground outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 aria-[invalid=true]:border-utility-error-300 aria-[invalid=true]:focus-visible:outline-error disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Textarea.displayName = "Textarea"

export { Textarea }
