/**
 * Shared design system components for info/detail panels.
 * Replaces the old `bg-muted/40 rounded-xl px-3 py-2.5` card boxes
 * with a compact, professional row-based layout.
 */
import { cn } from "@/lib/utils";

/** A titled card section with an optional icon badge */
export function InfoSection({
  icon: Icon,
  title,
  children,
  className,
}: {
  icon?: React.ElementType;
  title?: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("space-y-2", className)}>
      {(Icon || title) && (
        <div className="flex items-center gap-2">
          {Icon && (
            <div className="h-5 w-5 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
              <Icon className="h-3 w-3 text-primary" />
            </div>
          )}
          {title && <span className="text-xs font-semibold text-foreground">{title}</span>}
        </div>
      )}
      <div className="rounded-xl border bg-card">{children}</div>
    </div>
  );
}

/** A single label→value row inside an InfoSection */
export function InfoRow({
  label,
  value,
  mono,
  className,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-baseline justify-between gap-3 px-3 py-2 border-b border-border/50 last:border-0",
        className
      )}
    >
      <span className="text-xs text-muted-foreground shrink-0">{label}</span>
      <span className={cn("text-xs font-medium text-right break-all", mono && "font-mono")}>
        {value ?? "-"}
      </span>
    </div>
  );
}

/** Two-column grid of InfoRows inside an InfoSection */
export function InfoGrid({
  children,
  cols = 2,
}: {
  children: React.ReactNode;
  cols?: 2 | 3;
}) {
  return (
    <div
      className={cn(
        "grid divide-x divide-border/50",
        cols === 2 ? "grid-cols-2" : "grid-cols-1 sm:grid-cols-3"
      )}
    >
      {children}
    </div>
  );
}

/** A single cell inside an InfoGrid */
export function InfoCell({
  label,
  value,
  mono,
  accent,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  accent?: string;
}) {
  return (
    <div className="flex flex-col gap-0.5 px-3 py-2.5">
      <span className="text-[10px] text-muted-foreground">{label}</span>
      <span className={cn("text-sm font-semibold", mono && "font-mono", accent)}>
        {value ?? "-"}
      </span>
    </div>
  );
}

/** Footer action bar — destructive left, primary right */
export function ActionBar({
  left,
  right,
}: {
  left?: React.ReactNode;
  right?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between w-full gap-2">
      <div className="flex items-center gap-1">{left}</div>
      <div className="flex items-center gap-2">{right}</div>
    </div>
  );
}
