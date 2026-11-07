import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface FilterBarProps {
  children: ReactNode;
  className?: string;
}

/** Standardized filter toolbar — one console row, no card wrapper. */
export const FilterBar = ({ children, className }: FilterBarProps) => (
  <div
    data-slot="filter-bar"
    data-admin-surface="filter-bar"
    className={cn(
      'admin-filter-bar console-filter-bar',
      className,
    )}
  >
    {children}
  </div>
);
