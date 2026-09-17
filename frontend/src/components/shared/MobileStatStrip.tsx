import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface MobileStatItem {
  /** Stable identity for React keys and test queries. */
  key: string;
  label: string;
  value: ReactNode;
  /** Renders the cell as the selected filter state. */
  active?: boolean;
  /**
   * Supplying a handler turns the cell into a filter toggle. Omit it for
   * read-only metrics so the cell is not focusable or announced as a control.
   */
  onClick?: () => void;
}

interface MobileStatStripProps {
  items: readonly MobileStatItem[];
  /**
   * Cells per row. Defaults to one row of all items; pass 2 for strips whose
   * values are long (currency totals) and need wider cells.
   */
  columns?: number;
  className?: string;
}

/**
 * One bordered surface holding equally sized metric cells separated by
 * hairlines — the compact mobile summary strip.
 *
 * Rendered as a single rounded container rather than N free-floating bordered
 * cards, so the strip reads as one control and sibling pages stay visually
 * consistent. Cells are 64px tall, comfortably above the 44px touch minimum.
 */
export const MobileStatStrip = ({
  items,
  columns,
  className,
}: MobileStatStripProps) => {
  if (items.length === 0) return null;

  const columnCount = columns ?? items.length;
  // Cells after the first in each row get a hairline; the rest get a top hairline
  // so a multi-row strip still reads as one divided surface.
  const hasMultipleRows = items.length > columnCount;

  return (
    <div
      className={cn(
        "grid overflow-hidden rounded-xl border border-border bg-card",
        className,
      )}
      style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))` }}
    >
      {items.map((item, index) => {
        const interactive = typeof item.onClick === "function";
        const isFirstInRow = index % columnCount === 0;
        const content = (
          <>
            <span
              className={cn(
                "text-sm font-bold leading-none tabular-nums",
                item.active ? "text-primary" : "text-foreground",
              )}
            >
              {item.value}
            </span>
            <span
              className={cn(
                "text-xs leading-tight",
                item.active ? "text-primary" : "text-muted-foreground",
              )}
            >
              {item.label}
            </span>
          </>
        );
        const cellClass = cn(
          "flex min-h-16 min-w-0 flex-col items-center justify-center gap-1 px-1.5 py-2.5 text-center transition-colors",
          hasMultipleRows
            ? cn(
                !isFirstInRow && "border-l border-border",
                index >= columnCount && "border-t border-border",
              )
            : !isFirstInRow && "border-l border-border",
          item.active && "bg-primary/5",
          interactive && "active:bg-muted/60",
        );

        return interactive ? (
          <button
            key={item.key}
            type="button"
            onClick={item.onClick}
            aria-pressed={Boolean(item.active)}
            className={cellClass}
          >
            {content}
          </button>
        ) : (
          <div key={item.key} className={cellClass}>
            {content}
          </div>
        );
      })}
    </div>
  );
};
