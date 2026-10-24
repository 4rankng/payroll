import { format } from 'date-fns';
import type { PayRate } from '@/types/api/payrate.types';
import { getPayrateStatus } from '@/types/api/payrate.types';
import { cn } from '@/lib/utils';

const STATUS_STYLES: Record<string, string> = {
  active: 'border-utility-success-300 bg-utility-success-50 text-fg-success-primary',
  upcoming: 'border-utility-brand-200 bg-utility-brand-50 text-fg-brand-primary',
};

// Compact timeline of the project's payrate periods, rendered under the
// effective-date input so the user sees where the new/edited config lands
// before saving. From-date-only model: there is no stored end date — each
// period runs until its successor's fromDate, so the end shown here is
// derived from the timeline itself. The period containing the selected date
// is highlighted so overlaps are visible at a glance.
export function PayratePeriodStrip({ payrates, selectedDate }: {
  payrates: PayRate[];
  selectedDate?: string;
}) {
  if (!payrates.length) return null;

  const sorted = [...payrates].sort((a, b) => a.fromDate.localeCompare(b.fromDate));

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {sorted.map((p, i) => {
        const status = getPayrateStatus(p, payrates);
        // A config's visible end is the day before its successor starts; the
        // latest config is open-ended.
        const successor = i < sorted.length - 1 ? sorted[i + 1] : null;
        const contains = !!selectedDate
          && selectedDate >= p.fromDate
          && (!successor || selectedDate < successor.fromDate);
        const endLabel = successor ? format(new Date(successor.fromDate), 'dd/MM') : 'mở';
        return (
          <span
            key={p.id}
            className={cn(
              'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs',
              contains
                ? 'border-primary bg-primary/5 font-medium text-primary'
                : STATUS_STYLES[status] ?? 'border-border bg-card text-muted-foreground',
            )}
            title={`${p.fromDate} → ${endLabel}`}
          >
            {format(new Date(p.fromDate), 'dd/MM')}
            {' → '}
            {endLabel}
          </span>
        );
      })}
    </div>
  );
}
