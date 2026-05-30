import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface SectionLabelProps {
  children: React.ReactNode;
  icon?: LucideIcon;
  className?: string;
}

export function SectionLabel({ children, icon: Icon, className }: SectionLabelProps) {
  if (Icon) {
    return (
      <div className={cn("flex items-center gap-2 mb-3", className)}>
        <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
          <Icon className="h-3 w-3 text-primary/70" />
        </div>
        <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground/60">
          {children}
        </p>
      </div>
    );
  }

  return (
    <p className={cn("text-xs font-semibold uppercase tracking-widest text-muted-foreground/60 mb-3", className)}>
      {children}
    </p>
  );
}
