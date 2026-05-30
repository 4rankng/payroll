/**
 * Compact read-only rate grid for the project details sheet.
 * Renders: position → day type rows → hour-type chips with rates.
 * Much denser than the full PayrateMatrixEditor table.
 */
import { cn } from '@/lib/utils';
import type { PayrateStructure } from '../types';
import { getPositionsFromRates, getAllHourTypes } from '../types';

const DAY_TYPES = ['ngày thường', 'ngày nghỉ', 'ngày lễ'] as const;
const DAY_LABELS: Record<string, string> = {
  'ngày thường': 'Thường',
  'ngày nghỉ': 'Nghỉ',
  'ngày lễ': 'Lễ',
};
const DAY_COLORS: Record<string, string> = {
  'ngày thường': 'bg-blue-50 text-blue-700 border-blue-200',
  'ngày nghỉ':   'bg-amber-50 text-amber-700 border-amber-200',
  'ngày lễ':     'bg-red-50 text-red-700 border-red-200',
};

interface PayrateRateGridProps {
  rates: PayrateStructure;
}

export function PayrateRateGrid({ rates }: PayrateRateGridProps) {
  const positions = getPositionsFromRates(rates);
  const hourTypes = getAllHourTypes(rates);

  if (positions.length === 0) return null;

  return (
    <div className="space-y-3">
      {/* Column header row */}
      <div className="grid gap-x-2" style={{ gridTemplateColumns: `6rem repeat(${hourTypes.length}, 1fr)` }}>
        <div /> {/* day label column */}
        {hourTypes.map(ht => (
          <div key={ht} className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wide text-center truncate px-1">
            {ht}
          </div>
        ))}
      </div>

      {positions.map((position, pi) => (
        <div key={position} className={cn("rounded-xl border border-border/60 overflow-hidden", pi > 0 && "mt-2")}>
          {/* Position header */}
          <div className="px-3 py-1.5 bg-muted/40 border-b border-border/40 flex items-center gap-2">
            <span className="text-xs font-semibold text-foreground capitalize">{position}</span>
          </div>

          {/* Day type rows */}
          <div className="divide-y divide-border/30">
            {DAY_TYPES.map(dayType => {
              const dayRates = rates[position]?.[dayType] ?? {};
              const hasAnyRate = Object.values(dayRates).some(r => (r as number) > 0);

              return (
                <div
                  key={dayType}
                  className="grid items-center gap-x-2 px-3 py-1.5"
                  style={{ gridTemplateColumns: `6rem repeat(${hourTypes.length}, 1fr)` }}
                >
                  {/* Day type badge */}
                  <span className={cn(
                    "inline-flex items-center justify-center text-[10px] font-semibold px-1.5 py-0.5 rounded border w-fit",
                    DAY_COLORS[dayType]
                  )}>
                    {DAY_LABELS[dayType]}
                  </span>

                  {/* Rate chips per hour type */}
                  {hourTypes.map(hourType => {
                    const rate = (dayRates[hourType] as number) ?? 0;
                    const active = rate > 0;
                    return (
                      <div key={hourType} className="text-center">
                        {active ? (
                          <span className="text-xs font-semibold tabular-nums text-foreground">
                            {rate.toLocaleString('vi-VN')}
                            <span className="text-[9px] font-normal text-muted-foreground ml-0.5">đ</span>
                          </span>
                        ) : (
                          <span className="text-[10px] text-muted-foreground/50">—</span>
                        )}
                      </div>
                    );
                  })}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
