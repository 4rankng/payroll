import React from 'react';
import { cn } from '@/lib/utils';

export interface HexagonBadgeProps {
  label: string;
  value: string;
  accent: 'teal' | 'green' | 'gold';
  ariaLabel?: string;
}

const ACCENT: Record<HexagonBadgeProps['accent'], { bg: string; text: string; ring: string; icon: string }> = {
  teal:  { bg: 'bg-teal/10',  text: 'text-teal',  ring: 'ring-teal/20',  icon: 'bg-teal' },
  green: { bg: 'bg-green/10', text: 'text-green', ring: 'ring-green/20', icon: 'bg-green' },
  gold:  { bg: 'bg-gold/10',  text: 'text-gold',  ring: 'ring-gold/20',  icon: 'bg-gold' },
};

// Hexagon SVG shape as a decorative background element
function HexagonShape({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 48 56"
      fill="currentColor"
      className={cn('absolute inset-0 w-full h-full', className)}
      aria-hidden="true"
    >
      <path d="M24 2 L46 14 L46 42 L24 54 L2 42 L2 14 Z" />
    </svg>
  );
}

export function HexagonBadge({ label, value, accent, ariaLabel }: HexagonBadgeProps) {
  const a = ACCENT[accent];

  return (
    <div
      className="flex flex-col items-center gap-1.5 min-w-0"
      aria-label={ariaLabel ?? `${label}: ${value}`}
    >
      {/* Hexagon icon container */}
      <div className="relative w-12 h-14 flex items-center justify-center flex-shrink-0">
        <HexagonShape className={cn('opacity-15', a.text)} />
        <HexagonShape className={cn('opacity-100 scale-[0.82]', a.text)} />
        {/* Inner dot accent */}
        <span className={cn('relative z-10 w-2 h-2 rounded-full', a.icon)} />
      </div>

      {/* Label + value below the hexagon */}
      <div className="text-center min-w-0 w-full px-1">
        <p className={cn('break-words text-xs font-bold tabular-nums leading-tight', a.text)}>
          {value}
        </p>
        <p className="mt-0.5 line-clamp-2 text-[10px] leading-tight text-muted-foreground">
          {label}
        </p>
      </div>
    </div>
  );
}

export default HexagonBadge;
