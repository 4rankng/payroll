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
        "min-h-screen max-w-full overflow-x-hidden bg-[#EEF3F8]",
        "px-3 pb-[calc(5.75rem+env(safe-area-inset-bottom))] pt-3",
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
        "rounded-2xl border border-[#DCE5EF] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05),0_16px_36px_-30px_rgba(15,49,103,0.55)]",
        className,
      )}
    >
      {children}
    </section>
  );
}
