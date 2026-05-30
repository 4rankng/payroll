import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md text-base font-medium tracking-[-0.01em] transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-40 [&_svg]:pointer-events-none [&_svg]:shrink-0 select-none",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground font-semibold hover:bg-primary/90 active:bg-primary/80",

        destructive:
          "bg-red-600 text-white font-semibold hover:bg-red-500 active:bg-red-700",

        secondary:
          "bg-slate-100 text-slate-800 font-semibold hover:bg-slate-200 active:bg-slate-300 border border-slate-200",

        outline:
          "border border-border bg-background text-foreground hover:bg-muted active:bg-muted/80",

        ghost:
          "text-muted-foreground hover:bg-muted hover:text-foreground active:bg-muted/80",

        success:
          "bg-emerald-600 text-white font-semibold hover:bg-emerald-500 active:bg-emerald-700",

        info:
          "bg-sky-600 text-white font-semibold hover:bg-sky-500 active:bg-sky-700",

        link:
          "text-primary underline-offset-4 hover:underline font-normal px-0 h-auto",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-11 rounded-md px-8 text-base",
        xl: "h-13 rounded-lg px-10 text-lg",
        icon: "h-9 w-9",
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
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
