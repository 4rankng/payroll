import * as React from "react"
import * as AlertDialogPrimitive from "@radix-ui/react-alert-dialog"

import { cn } from "@/lib/utils"

const AlertDialog = AlertDialogPrimitive.Root

const AlertDialogTrigger = AlertDialogPrimitive.Trigger

const AlertDialogPortal = AlertDialogPrimitive.Portal

const AlertDialogOverlay = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <AlertDialogPrimitive.Overlay
    className={cn(
      "fixed inset-0 z-50 bg-neutral/60 backdrop-blur-[2px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      className
    )}
    {...props}
    ref={ref}
  />
))
AlertDialogOverlay.displayName = AlertDialogPrimitive.Overlay.displayName

type AlertDialogContentProps =
  React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Content> & {
    overlayClassName?: string
  }

const AlertDialogContent = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Content>,
  AlertDialogContentProps
>(({ className, overlayClassName, ...props }, ref) => (
  <AlertDialogPortal>
    <AlertDialogOverlay className={overlayClassName} />
    <AlertDialogPrimitive.Content
      ref={ref}
      className={cn(
        // Note: intentionally NOT using `ct-modal-box` or `bg-base-100` here.
        // Both reference daisyUI's `--b1` which is scoped to
        // `[data-admin-ui]/[data-employee-ui]/[data-partner-ui]` via
        // `themeRoot`, but Radix renders this in a Portal at the end of
        // <body> — outside those containers — so the variable is undefined
        // and the surface renders transparent. `bg-card` uses our own
        // `--card` token (defined on `:root`) so it stays opaque everywhere.
        "fixed left-[50%] top-[50%] z-50 flex max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-lg translate-x-[-50%] translate-y-[-50%] scale-100 flex-col gap-0 overflow-hidden rounded-2xl border border-border bg-card p-0 text-card-foreground shadow-2xl shadow-neutral/20 duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:w-full",
        className
      )}
      {...props}
    />
  </AlertDialogPortal>
))
AlertDialogContent.displayName = AlertDialogPrimitive.Content.displayName

const AlertDialogHeader = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      // `bg-card` (not `bg-base-100`) — see AlertDialogContent comment.
      "flex flex-col space-y-1.5 bg-card px-5 pb-3 pt-5 text-left sm:px-6 sm:pt-6",
      className
    )}
    {...props}
  />
)
AlertDialogHeader.displayName = "AlertDialogHeader"

const AlertDialogFooter = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      // `bg-card` (not `bg-base-100`) — see AlertDialogContent comment.
      "flex flex-col-reverse gap-2 border-t border-border bg-card px-5 py-4 sm:flex-row sm:justify-end sm:px-6",
      className
    )}
    {...props}
  />
)
AlertDialogFooter.displayName = "AlertDialogFooter"

const AlertDialogTitle = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <AlertDialogPrimitive.Title
    ref={ref}
    // `text-card-foreground` (not `text-base-content`) — see AlertDialogContent
    // comment about daisyUI scoped variables and the Radix portal.
    className={cn("text-lg font-semibold leading-tight tracking-tight text-card-foreground", className)}
    {...props}
  />
))
AlertDialogTitle.displayName = AlertDialogPrimitive.Title.displayName

const AlertDialogDescription = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <AlertDialogPrimitive.Description
    ref={ref}
    // `text-muted-foreground` (not `text-base-content/60`) — see comment above.
    className={cn("mt-1 text-sm leading-relaxed text-muted-foreground", className)}
    {...props}
  />
))
AlertDialogDescription.displayName =
  AlertDialogPrimitive.Description.displayName

interface AlertDialogActionProps
  extends React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Action> {
  variant?: "default" | "destructive"
}

const AlertDialogAction = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Action>,
  AlertDialogActionProps
>(({ className, variant = "default", ...props }, ref) => (
  <AlertDialogPrimitive.Action
    ref={ref}
    className={cn(
      // Avoid daisyUI's `ct-btn`/`ct-btn-primary`/`ct-btn-error` — those
      // resolve through scoped `--p`/`--er` variables that are undefined
      // inside the Radix portal (see AlertDialogContent comment).
      "inline-flex items-center justify-center gap-2 min-h-11 rounded-md border-0 px-4 text-sm font-semibold normal-case shadow-none transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
      variant === "destructive"
        ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
        : "bg-primary text-primary-foreground hover:bg-primary/90",
      className
    )}
    {...props}
  />
))
AlertDialogAction.displayName = AlertDialogPrimitive.Action.displayName

const AlertDialogCancel = React.forwardRef<
  React.ElementRef<typeof AlertDialogPrimitive.Cancel>,
  React.ComponentPropsWithoutRef<typeof AlertDialogPrimitive.Cancel>
>(({ className, ...props }, ref) => (
  <AlertDialogPrimitive.Cancel
    ref={ref}
    className={cn(
      // Avoid daisyUI's `ct-btn-ghost`/`bg-base-100`/`hover:bg-base-200` —
      // scoped variables are undefined in the portal.
      "inline-flex items-center justify-center gap-2 min-h-11 rounded-md border border-border bg-card px-4 text-sm font-semibold normal-case text-card-foreground shadow-none transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
      className
    )}
    {...props}
  />
))
AlertDialogCancel.displayName = AlertDialogPrimitive.Cancel.displayName

export {
  AlertDialog,
  AlertDialogPortal,
  AlertDialogOverlay,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
}
