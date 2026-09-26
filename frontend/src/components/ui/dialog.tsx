import * as React from "react"
import * as DialogPrimitive from "@radix-ui/react-dialog"
import { X } from "lucide-react"

import { cn } from "@/lib/utils"
import { useIsMobile } from '@/hooks/useBreakpoint'
import { useDialogFocusReturn } from '@/hooks/useDialogFocusReturn'

const Dialog = DialogPrimitive.Root

const DialogTrigger = DialogPrimitive.Trigger

const DialogPortal = DialogPrimitive.Portal

const DialogClose = DialogPrimitive.Close

const DialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay> & { mobileOverlay?: boolean }
>(({ className, mobileOverlay, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    className={cn(
      "fixed inset-0 z-50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      mobileOverlay ? "bg-black/40" : "bg-black/80",
      className
    )}
    {...props}
  />
))
DialogOverlay.displayName = DialogPrimitive.Overlay.displayName

type DialogContentProps = React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
  title?: string;
  description?: string;
  hideCloseButton?: boolean;
  contentPadding?: 'default' | 'none';
};

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  DialogContentProps
>(({ className, children, title, description, hideCloseButton, contentPadding = 'default', onOpenAutoFocus, onCloseAutoFocus, style, ...props }, ref) => {
  const isMobile = useIsMobile();
  const focusReturn = useDialogFocusReturn(onOpenAutoFocus, onCloseAutoFocus);

  return (
    <DialogPortal>
      <DialogOverlay mobileOverlay={isMobile} />
      <DialogPrimitive.Content
        ref={ref}
        className={cn(
          "fixed z-50 flex flex-col gap-4 overflow-hidden border bg-card shadow-2xl",
          contentPadding === 'default' && "p-4 sm:p-6",
          isMobile
            ? /* Mobile: bottom sheet */
              "inset-x-0 bottom-0 w-full max-h-[92dvh] rounded-t-2xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom data-[state=open]:duration-300 data-[state=closed]:duration-200"
            : /* Desktop: centered modal */
              "left-[50%] top-[50%] max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-lg translate-x-[-50%] translate-y-[-50%] rounded-2xl duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:w-full",
          className
        )}
        {...props}
        // Desktop width constraints supplied by callers must not narrow a
        // bottom sheet while it remains anchored to both mobile screen edges.
        style={isMobile ? { ...style, width: '100%', maxWidth: 'none', left: 0, right: 0, marginLeft: 0, marginRight: 0 } : style}
        {...focusReturn}
      >
        {/* Visually-hidden fallbacks so DialogContent stays accessible when a
          caller supplies neither a visible DialogTitle/DialogDescription nor
          the props. When the props are absent we defer to the caller's own
          Title/Description instead of rendering a second, generic "Dialog"
          label that competes for the accessible name (Radix binds
          aria-labelledby to the first Title in the DOM). */}
        {title ? <DialogPrimitive.Title className="sr-only">{title}</DialogPrimitive.Title> : null}
        {description ? <DialogPrimitive.Description className="sr-only">{description}</DialogPrimitive.Description> : null}
        {children}
        {!hideCloseButton && (
          <DialogPrimitive.Close className="absolute right-3 top-2 z-10 flex h-11 w-11 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-white/20 outline-none focus:ring-2 focus:ring-white/30 sm:right-4">
            <X className="w-4 h-4 text-white" />
            <span className="sr-only">Đóng</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </DialogPortal>
  );
})
DialogContent.displayName = DialogPrimitive.Content.displayName

type DialogNavyHeaderProps = {
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  children?: React.ReactNode;
};

const DialogNavyHeader = React.forwardRef<HTMLDivElement, DialogNavyHeaderProps>(
  ({ title, description, action, className, children }, ref) => (
    <div
      ref={ref}
      className={cn("bg-emerald-950 px-5 pt-5 pb-4 text-white flex-shrink-0", className)}
      style={{ '--foreground': '0 0% 100%', '--muted-foreground': '152 30% 78%' } as React.CSSProperties}
    >
      <div className="flex justify-between items-start gap-3">
        <div className="min-w-0 flex-1">
          <DialogPrimitive.Title className="break-words text-base font-semibold tracking-tight text-white leading-tight">{title}</DialogPrimitive.Title>
          {description && (
            <DialogPrimitive.Description className="text-emerald-200/75 text-xs mt-1">{description}</DialogPrimitive.Description>
          )}
        </div>
        {action && <div className="flex items-center flex-shrink-0 pt-0.5">{action}</div>}
        <DialogPrimitive.Close className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-emerald-800 outline-none focus:ring-2 focus:ring-emerald-200/50">
          <X className="w-4 h-4 text-white" />
          <span className="sr-only">Đóng</span>
        </DialogPrimitive.Close>
      </div>
      {children}
    </div>
  )
);
DialogNavyHeader.displayName = "DialogNavyHeader";

const DialogHeader = ({
  className,
  style,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      "-mx-4 -mt-4 flex min-h-14 flex-col space-y-1.5 rounded-t-2xl bg-emerald-950 px-4 pb-4 pr-16 pt-5 text-center text-white sm:-mx-6 sm:-mt-6 sm:px-6 sm:pr-20 sm:text-left",
      className
    )}
    style={{
      // Override CSS variables so text-foreground / text-muted-foreground
      // resolve to light colours inside the dark navy header, fixing
      // invisible-text bugs across all dialogs without touching each one.
      '--foreground': '0 0% 100%',          // white
      '--muted-foreground': '152 30% 78%',  // light emerald
      ...style,
    } as React.CSSProperties}
    {...props}
  />
)
DialogHeader.displayName = "DialogHeader"

const DialogFooter = ({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
      className
    )}
    {...props}
  />
)
DialogFooter.displayName = "DialogFooter"

const DialogTitle = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title
    ref={ref}
    className={cn(
      "text-base font-semibold tracking-tight text-white leading-tight",
      className
    )}
    {...props}
  />
))
DialogTitle.displayName = DialogPrimitive.Title.displayName

const DialogDescription = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description
    ref={ref}
    className={cn("text-sm text-emerald-200/75 mt-1", className)}
    {...props}
  />
))
DialogDescription.displayName = DialogPrimitive.Description.displayName

export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogNavyHeader,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
}
