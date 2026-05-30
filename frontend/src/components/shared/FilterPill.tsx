import * as React from 'react';
import * as SelectPrimitive from '@radix-ui/react-select';
import { Check, ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface FilterOption {
  value: string;
  label: string;
}

export interface FilterPillProps {
  value: string;
  onChange: (value: string) => void;
  defaultValue?: string;
  placeholder: string;
  options: FilterOption[];
  icon?: React.ReactNode;
  className?: string;
}

export const FilterPill = React.memo(function FilterPill({
  value,
  onChange,
  defaultValue = 'all',
  placeholder,
  options,
  icon,
  className,
}: FilterPillProps) {
  const isActive = value !== defaultValue;
  const activeLabel = options.find(o => o.value === value)?.label;

  return (
    <SelectPrimitive.Root value={value} onValueChange={onChange}>
      <SelectPrimitive.Trigger
        className={cn(
          'group inline-flex items-center gap-1 h-7 px-2.5 py-0 rounded-xl text-xs font-medium whitespace-nowrap',
          'border transition-colors duration-100 select-none outline-none',
          'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
          !isActive && 'border-border/60 bg-background text-muted-foreground hover:border-border hover:text-foreground hover:bg-accent/40',
          isActive && 'border-primary/30 bg-primary/8 text-primary hover:bg-primary/12',
          className,
        )}
        aria-label={placeholder}
      >
        {icon && <span className="shrink-0 opacity-70">{icon}</span>}
        <span>{isActive ? activeLabel : placeholder}</span>
        <SelectPrimitive.Icon asChild>
          <ChevronDown className="h-3 w-3 shrink-0 opacity-50 group-data-[state=open]:rotate-180 transition-transform duration-150" />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>

      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          position="popper"
          sideOffset={4}
          className={cn(
            'z-50 min-w-[10rem] overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-sm',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[side=bottom]:slide-in-from-top-2',
          )}
        >
          <SelectPrimitive.Viewport className="p-1">
            <FilterPillItem value={defaultValue} label={placeholder} />
            {options.map(o => (
              <FilterPillItem key={o.value} value={o.value} label={o.label} />
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
});

const FilterPillItem = React.memo(function FilterPillItem({
  value,
  label,
}: {
  value: string;
  label: string;
}) {
  return (
    <SelectPrimitive.Item
      value={value}
      className={cn(
        'relative flex w-full cursor-default select-none items-center rounded-sm py-1 pl-6 pr-3 text-xs outline-none whitespace-nowrap',
        'focus:bg-accent focus:text-accent-foreground',
        'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      )}
    >
      <span className="absolute left-1.5 flex h-3.5 w-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <Check className="h-3 w-3" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <SelectPrimitive.ItemText>{label}</SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  );
});
