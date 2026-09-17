import { memo, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

export type AccentColor = 'amber' | 'blue' | 'green' | 'red' | 'gray' | 'orange' | 'teal';

export interface AccentStripCardProps {
  accentColor: AccentColor;
  children: ReactNode;
  onClick?: () => void;
  className?: string;
}

/**
 * Card without left-edge colour accent strip (removed for mobile view).
 */
export const AccentStripCard = memo(function AccentStripCard({
  accentColor: _accentColor,
  children,
  onClick,
  className,
}: AccentStripCardProps) {
  return (
    <div
      className={cn(
        'overflow-hidden rounded-2xl border border-border bg-card shadow-none',
        onClick && 'cursor-pointer active:bg-muted/40 transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        className,
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ')) {
                e.preventDefault();
                onClick();
              }
            }
          : undefined
      }
    >
      {children}
    </div>
  );
});
