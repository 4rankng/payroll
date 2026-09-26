import { memo, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

export interface SheetFooterProps {
  /** Optional item count label, e.g. "3 mục" */
  itemCount?: number;
  itemLabel?: string;
  actions: ReactNode;
  legend?: ReactNode;
  className?: string;
}

export const SheetFooter = memo(function SheetFooter({
  itemCount,
  itemLabel = 'mục',
  actions,
  legend,
  className,
}: SheetFooterProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between px-4 sm:px-6 py-3 border-t bg-card flex-shrink-0',
        className,
      )}
    >
      <div className="flex items-center gap-4">
        {legend}
        {itemCount != null && itemCount > 0 && (
          <span className="text-sm text-muted-foreground">
            <span className="font-medium text-foreground">{itemCount}</span>{' '}
            {itemLabel}
          </span>
        )}
      </div>
      <div className="flex items-center gap-2 sm:gap-3">{actions}</div>
    </div>
  );
});
