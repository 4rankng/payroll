import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { DayType, ValidationResult, PayrateStructure } from '../../types';

type DiffState = 'new' | 'modified' | 'deleted' | 'unchanged';

function getDiffState(
  current: number,
  original: number | undefined,
): DiffState {
  const orig = original ?? 0;
  if (orig === current) return 'unchanged';
  if (orig === 0 && current > 0) return 'new';
  if (orig > 0 && current === 0) return 'deleted';
  return 'modified';
}

interface RateCellProps {
  position: string;
  dayType: DayType;
  hourType: string;
  rate: number;
  readOnly?: boolean;
  validation?: ValidationResult;
  originalRates?: PayrateStructure;
  onChange: (position: string, dayType: DayType, hourType: string, value: string) => void;
}

export function RateCell({ position, dayType, hourType, rate, readOnly = false, validation, originalRates, onChange }: RateCellProps) {
  const hasError = validation?.errors.some(e => e.includes(position) && e.includes(dayType) && e.includes(hourType));
  const isEmpty = rate === 0;

  // Compute diff state only when originalRates is provided (edit mode)
  const diffState: DiffState | null = originalRates
    ? getDiffState(rate, (originalRates[position]?.[dayType]?.[hourType] as number) ?? 0)
    : null;

  const diffClasses: Record<DiffState, string> = {
    new:       'bg-emerald-50 border-emerald-400 text-emerald-900 focus:border-emerald-500 focus:ring-emerald-500/20',
    modified:  'bg-amber-50  border-amber-400  text-amber-900  focus:border-amber-500  focus:ring-amber-500/20',
    deleted:   'bg-red-50    border-red-400    text-red-900    focus:border-red-500    focus:ring-red-500/20',
    unchanged: '',
  };

  if (readOnly) {
    return (
      <div className={cn(
        "text-right tabular-nums text-sm px-2 py-1 rounded",
        isEmpty && !diffState ? "text-muted-foreground/40" : "",
        diffState && diffState !== 'unchanged' ? diffClasses[diffState] : "",
      )}>
        {isEmpty ? '—' : rate.toLocaleString('vi-VN')}
      </div>
    );
  }

  const hasDiff = diffState && diffState !== 'unchanged';

  return (
    <div className="relative group">
      <Input
        type="text"
        aria-label={`Mức lương ${position}, ${dayType}, ${hourType}`}
        value={isEmpty ? '' : rate.toLocaleString('vi-VN')}
        onChange={e => onChange(position, dayType, hourType, e.target.value)}
        placeholder="0"
        className={cn(
          "h-11 min-w-20 rounded-md px-3 text-right text-sm font-medium tabular-nums transition-colors lg:h-10",
          hasError && "border-destructive",
          hasDiff
            ? diffClasses[diffState]
            : isEmpty
              ? "border-transparent bg-muted/25 text-muted-foreground placeholder:text-muted-foreground/40 focus:bg-background"
              : "border-transparent bg-muted/35 text-foreground",
          !hasDiff && "hover:border-primary/60 focus:border-primary focus:ring-1 focus:ring-primary/20"
        )}
        onFocus={e => e.target.select()}
        onKeyDown={e => {
          if (e.key === 'Enter' || e.key === 'Tab') e.currentTarget.blur();
          if (e.key === 'Escape') e.currentTarget.blur();
        }}
      />
      {/* Diff indicator dot */}
      {hasDiff && (
        <span className={cn(
          "absolute top-0.5 right-0.5 h-1.5 w-1.5 rounded-full",
          diffState === 'new'      && "bg-emerald-500",
          diffState === 'modified' && "bg-amber-500",
          diffState === 'deleted'  && "bg-red-500",
        )} />
      )}
    </div>
  );
}
