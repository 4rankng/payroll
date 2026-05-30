import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface FilterBarProps {
  children: ReactNode;
  className?: string;
}

/** Standardized filter toolbar — no card wrapper, just a flex row. */
export const FilterBar = ({ children, className }: FilterBarProps) => (
  <div className={cn('flex items-center gap-1.5 flex-wrap', className)}>
    {children}
  </div>
);
