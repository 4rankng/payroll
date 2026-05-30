import { memo, type ReactNode } from 'react';
import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export type AlertSeverity = 'info' | 'warning' | 'error' | 'success';

export interface InlineAlertProps {
  severity: AlertSeverity;
  icon?: LucideIcon;
  message: string;
  action?: ReactNode;
  className?: string;
}

const severityStyles: Record<AlertSeverity, string> = {
  info: 'bg-blue-50 border-blue-200 text-blue-700',
  warning: 'bg-amber-50 border-amber-200 text-amber-700',
  error: 'bg-red-50 border-red-200 text-red-700',
  success: 'bg-emerald-50 border-emerald-200 text-emerald-700',
};

/**
 * Compact single-line alert banner for inline contextual feedback.
 * Implements the inline alert pattern from Requirement 6 / TimesheetEntryModal.
 */
export const InlineAlert = memo(function InlineAlert({
  severity,
  icon: Icon,
  message,
  action,
  className,
}: InlineAlertProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between gap-3 px-3 py-2 rounded-xl border text-xs',
        severityStyles[severity],
        className,
      )}
    >
      <div className="flex items-center gap-2 min-w-0">
        {Icon && <Icon className="h-3.5 w-3.5 shrink-0" />}
        <span>{message}</span>
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
});
