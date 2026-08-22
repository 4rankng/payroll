import { memo, type ReactNode } from 'react';
import { type LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { EmptyStateIllustration } from '@/components/shared/EmptyStateIllustration';
import { cn } from '@/lib/utils';

export interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  children?: ReactNode;
  size?: 'sm' | 'default';
  className?: string;
}

/**
 * Standard empty state — icon, primary message, optional description, optional action.
 * Implements Requirement 6 criterion 2.
 */
export const EmptyState = memo(function EmptyState({
  title,
  description,
  action,
  children,
  size = 'default',
  className,
}: EmptyStateProps) {
  return (
    <div
      data-slot="empty-state"
      data-admin-surface="empty-state"
      className={cn(
        'admin-empty-state flex flex-col items-center justify-center gap-2.5',
        size === 'default' ? 'py-8 sm:py-10' : 'py-6',
        className,
      )}
    >
      <EmptyStateIllustration className={size === 'default' ? undefined : 'h-14 w-14 sm:h-14 sm:w-14'} />
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
      {children}
    </div>
  );
});
