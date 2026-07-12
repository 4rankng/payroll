import { memo } from 'react';
import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface ContextStripItem {
  label: string;
  value: string;
  icon?: LucideIcon;
  /** Tailwind bg class for icon badge, e.g. 'bg-blue-100' */
  iconBg?: string;
  /** Tailwind text class for icon color, e.g. 'text-blue-600' */
  iconColor?: string;
  subtitle?: string;
}

export interface ContextStripProps {
  items: ContextStripItem[];
  columns?: number;
  className?: string;
}

/**
 * Grid of labelled context cells — project/employee context, month label, etc.
 * Implements the context strip pattern from TimesheetEntryModal and AdvancePaymentsPage.
 */
export const ContextStrip = memo(function ContextStrip({
  items,
  columns = 2,
  className,
}: ContextStripProps) {
  return (
    <div
      className={cn('grid gap-2', className)}
      style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
    >
      {items.map((item, i) => {
        const Icon = item.icon;
        return (
          <div
            key={i}
            className="flex flex-col gap-0.5 px-3 py-2.5 rounded-xl bg-slate-50 border border-slate-200"
          >
            <span className="text-[11px] font-bold text-slate-500 uppercase tracking-wider mb-1">
              {item.label}
            </span>
            <div className="flex items-center gap-2 min-w-0">
              {Icon && (
                <div className={cn('w-7 h-7 rounded flex items-center justify-center shrink-0', item.iconBg ?? 'bg-slate-100')}>
                  <Icon className={cn('h-3.5 w-3.5', item.iconColor ?? 'text-slate-500')} />
                </div>
              )}
              <span className="text-sm font-semibold text-slate-900 truncate">
                {item.value}
              </span>
            </div>
            {item.subtitle && (
              <span className="text-xs text-slate-500 tabular-nums">
                {item.subtitle}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
});
