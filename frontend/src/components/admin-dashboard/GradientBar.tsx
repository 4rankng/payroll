import React, { useId } from 'react';
import { cn } from '@/lib/utils';

export interface GradientBarProps {
  label: string;
  count: number;
  maxCount: number;
  binRange: string;
}

function computeWidth(count: number, maxCount: number): number {
  if (maxCount === 0) return 4;
  return Math.max(4, Math.min(100, (count / maxCount) * 100));
}

export function GradientBar({ label, count, maxCount, binRange }: GradientBarProps) {
  const tooltipId = useId();
  const widthPercent = computeWidth(count, maxCount);
  const isEmpty = count === 0;

  return (
    <div className="group relative" aria-label={`${binRange}: ${count} nhân viên`}>
      {/* Tooltip */}
      <div
        id={tooltipId}
        role="tooltip"
        className="pointer-events-none absolute -top-8 left-0 z-20 opacity-0 group-hover:opacity-100 transition-opacity duration-150"
      >
        <div className="bg-foreground/90 text-background text-xs font-medium rounded-md px-2.5 py-1 whitespace-nowrap">
          {binRange}: <span className="font-bold">{count}</span> nhân viên
        </div>
        {/* Arrow */}
        <div className="w-2 h-2 bg-foreground/90 rotate-45 mx-3 -mt-1" />
      </div>

      {/* Row: label | bar | count */}
      <div className="flex items-center gap-2">
        {/* Label — fixed width so bars align */}
        <span className="text-xs text-muted-foreground w-[72px] flex-shrink-0 truncate leading-none">
          {label}
        </span>

        {/* Track */}
        <div className="flex-1 relative h-[8px] rounded-full bg-muted/40 overflow-hidden">
          <div
            role="img"
            aria-describedby={tooltipId}
            className={cn(
              'h-full rounded-full transition-all duration-500 ease-out',
              isEmpty
                ? 'bg-muted/60'
                : 'bg-gradient-to-r from-teal to-green motion-safe:group-hover:brightness-110',
            )}
            style={{ width: `${widthPercent}%` }}
          />
        </div>

        {/* Count */}
        <span className={cn(
          'text-xs font-semibold tabular-nums w-7 text-right flex-shrink-0 leading-none',
          isEmpty ? 'text-muted-foreground/50' : 'text-foreground',
        )}>
          {count}
        </span>
      </div>
    </div>
  );
}

export default GradientBar;
