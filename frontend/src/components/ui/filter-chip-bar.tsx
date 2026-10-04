import { cn } from '@/lib/utils';

// UU PRO restyle (W8): chips on the W2 badge soft-chip recipe — selected =
// the badge default (soft brand) chip, rest = badge secondary (soft gray)
// with a solid utility-gray-100 hover. Solid surfaces only (repo filter
// law): no translucent tints. Family dots (dotClass) stay the identity mark
// in both states, and the count inherits the chip copy color so text stays
// WCAG-safe on the -50 surfaces.
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
              'inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium whitespace-nowrap transition-colors shrink-0 ring-1 ring-inset',
              isActive
                ? 'bg-utility-brand-50 text-utility-brand-700 ring-utility-brand-200'
                : 'bg-utility-gray-50 text-utility-gray-700 ring-utility-gray-200 hover:bg-utility-gray-100'
            )}
          >
            {chip.dotClass && (
              <span
                className={cn(
                  'w-1.5 h-1.5 rounded-full shrink-0',
                  chip.dotClass
                )}
              />
            )}
            {chip.label}
            {chip.count != null && (
              <span className="font-mono text-xs">{chip.count}</span>
            )}
          </button>
        );
      })}
    </div>
  );
}
