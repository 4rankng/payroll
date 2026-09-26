import { cn } from '@/lib/utils';

export interface FilterChip {
  value: string;
  label: string;
  count?: number;
  dotClass?: string;
}

interface FilterChipBarProps {
  chips: FilterChip[];
  value: string;
  onChange: (value: string) => void;
  className?: string;
}

export function FilterChipBar({ chips, value, onChange, className }: FilterChipBarProps) {
  return (
    <div role="tablist" className={cn('flex gap-1.5 overflow-x-auto px-4 sm:px-6 py-2 scrollbar-none', className)}>
      {chips.map((chip) => {
        const isActive = chip.value === value;
        return (
          <button
            key={chip.value}
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(chip.value)}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium whitespace-nowrap transition-colors shrink-0',
              isActive
                ? 'bg-foreground text-background'
                : 'bg-muted text-muted-foreground hover:bg-muted/80 border border-border'
            )}
          >
            {chip.dotClass && (
              <span
                className={cn(
                  'w-1.5 h-1.5 rounded-full shrink-0',
                  isActive ? 'bg-card/70' : chip.dotClass
                )}
              />
            )}
            {chip.label}
            {chip.count != null && (
              <span className="font-mono text-xs opacity-60">{chip.count}</span>
            )}
          </button>
        );
      })}
    </div>
  );
}
