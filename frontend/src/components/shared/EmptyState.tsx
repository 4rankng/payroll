import { memo } from 'react';
import { type LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  size?: 'sm' | 'default';
  className?: string;
}

/**
 * Standard empty state — icon, primary message, optional description, optional action.
 * Implements Requirement 6 criterion 2.
 */
export const EmptyState = memo(function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  size = 'default',
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3',
        size === 'default' ? 'py-12' : 'py-10',
        className,
      )}
    >
      <div
        className={cn(
          'flex items-center justify-center rounded-2xl bg-muted/60',
          size === 'default' ? 'h-14 w-14' : 'h-10 w-10',
        )}
      >
        {Icon && (
          <Icon
            className={cn(
              'text-muted-foreground/50',
              size === 'default' ? 'h-7 w-7' : 'h-5 w-5',
            )}
          />
        )}
      </div>
      <div className="text-center">
        <p className="text-sm font-medium text-foreground">{title}</p>
        {description && (
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        )}
      </div>
      {action && (
        <Button variant="outline" size="sm" onClick={action.onClick} className="mt-1">
          {action.label}
        </Button>
      )}
    </div>
  );
});
