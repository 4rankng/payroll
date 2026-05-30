import { memo, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

export type AccentColor = 'amber' | 'blue' | 'green' | 'red' | 'gray' | 'orange' | 'purple';

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
        'bg-card border border-border rounded-xl overflow-hidden shadow-sm',
        onClick && 'cursor-pointer active:bg-muted/40 transition-colors touch-manipulation',
        className,
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={
        onClick
          ? (e) => {
              if (e.key === 'Enter' || e.key === ' ') {
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
