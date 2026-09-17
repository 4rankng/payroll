import { useState } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

interface MonthPickerProps {
  value: string;
  onChange: (value: string) => void;
}

/** Month-only selection for payroll periods; the API contract stays yyyy-MM. */
export function MonthPicker({ value, onChange }: MonthPickerProps) {
  const selectedYear = Number(value.slice(0, 4));
  const [year, setYear] = useState(() => Number.isFinite(selectedYear) && selectedYear > 0
    ? selectedYear
    : new Date().getFullYear());

  return (
    <div className="w-[280px] max-w-[calc(100vw-2rem)] p-3" role="group" aria-label="Chọn tháng và năm">
      <div className="mb-2 flex items-center justify-between gap-2">
        <button type="button" aria-label="Năm trước" onClick={() => setYear(year - 1)} className="flex h-11 w-11 items-center justify-center rounded-lg hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <ChevronLeft className="h-4 w-4" aria-hidden />
        </button>
        <span className="text-sm font-semibold tabular-nums" aria-live="polite">{year}</span>
        <button type="button" aria-label="Năm sau" onClick={() => setYear(year + 1)} className="flex h-11 w-11 items-center justify-center rounded-lg hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <ChevronRight className="h-4 w-4" aria-hidden />
        </button>
      </div>
      <div className="grid grid-cols-3 gap-2">
        {Array.from({ length: 12 }, (_, index) => {
          const month = index + 1;
          const candidate = `${year}-${String(month).padStart(2, '0')}`;
          const selected = candidate === value;
          return (
            <button key={month} type="button" aria-label={`Tháng ${month} năm ${year}`} aria-pressed={selected} onClick={() => onChange(candidate)}
              className={cn('h-11 rounded-lg text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2', selected ? 'bg-primary text-primary-foreground' : 'bg-muted/40 text-foreground hover:bg-muted')}>
              Tháng {month}
            </button>
          );
        })}
      </div>
    </div>
  );
}
