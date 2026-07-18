import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface FilterBarProps {
  children: ReactNode;
  className?: string;
}

/** Standardized filter toolbar — no card wrapper, just a flex row. */
export const FilterBar = ({ children, className }: FilterBarProps) => (
  <div
    data-slot="filter-bar"
    data-admin-surface="filter-bar"
    className={cn(
      'admin-filter-bar flex items-center gap-1.5 flex-wrap',
      className,
    )}
  >
    {children}
  </div>
);
