import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface MobilePageShellProps {
  children: ReactNode;
  className?: string;
}

export function MobilePageShell({ children, className }: MobilePageShellProps) {
  return (
    <div
      className={cn(
        "min-h-[100dvh] max-w-full overflow-x-clip bg-[hsl(var(--surface-page))]",
        "[--mobile-nonsticky-header-top-padding:1rem]",
        "px-4 pb-[var(--mobile-page-bottom-padding,calc(5.75rem+env(safe-area-inset-bottom)))] pt-[var(--mobile-page-top-padding,calc(0.75rem+env(safe-area-inset-top)))]",
        className,
      )}
    >
      {children}
    </div>
  );
}

interface MobileSurfaceProps {
  children: ReactNode;
  className?: string;
}

export function MobileSurface({ children, className }: MobileSurfaceProps) {
  return (
    <section
      className={cn(
        "overflow-hidden rounded-[28px] border border-[hsl(var(--surface-border))] bg-white shadow-none",
        className,
      )}
    >
      {children}
    </section>
  );
}
