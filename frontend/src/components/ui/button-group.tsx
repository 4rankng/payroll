import * as React from 'react';
import { cn } from '@/lib/utils';

export interface ButtonGroupOption<T = string> {
  value: T;
  label: string;
}

interface ButtonGroupProps<T = string> {
  options: ButtonGroupOption<T>[];
  value: T;
  onChange: (value: T) => void;
  className?: string;
  fullWidth?: boolean;
}

export function ButtonGroup<T extends string = string>({
  options,
  value,
  onChange,
  className,
  fullWidth = false,
}: ButtonGroupProps<T>) {
  return (
    <div
      className={cn(
        'inline-flex rounded-lg border border-border p-0.5',
        fullWidth && 'w-full',
        className
      )}
      role="group"
    >
      {options.map((option) => {
        const isSelected = value === option.value;

        return (
          <button
            key={String(option.value)}
            type="button"
            onClick={() => onChange(option.value)}
            className={cn(
              'relative inline-flex items-center justify-center px-3 py-1.5 text-sm font-medium transition-all rounded-md',
              'focus:z-10 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              fullWidth && 'flex-1',
              isSelected
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
