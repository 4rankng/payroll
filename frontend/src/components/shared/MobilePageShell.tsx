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
        "px-3 pb-[calc(5.75rem+env(safe-area-inset-bottom))] pt-[calc(0.75rem+env(safe-area-inset-top))]",
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
        "rounded-2xl border border-[hsl(var(--surface-border))] bg-white shadow-[var(--shadow-navy-soft)]",
        className,
      )}
    >
      {children}
    </section>
  );
}
