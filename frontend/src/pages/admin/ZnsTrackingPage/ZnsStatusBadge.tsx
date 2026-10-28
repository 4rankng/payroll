import { Badge } from '@/components/ui/badge';
import { ZNS_STATUS_LABELS } from '@/types/api/zns.types';
import { cn } from '@/lib/utils';

function statusClass(status: string): string {
  switch (status) {
    case 'sent':
      return 'border-success-200 bg-success-50 text-success-700';
    case 'failed':
      return 'border-error-200 bg-error-50 text-error-700';
    case 'processing':
      return 'border-brand-200 bg-brand-50 text-brand-700';
    case 'suppressed':
      return 'border-warning-200 bg-warning-50 text-warning-700';
    default:
      return 'border-utility-gray-200 bg-utility-gray-50 text-fg-secondary';
  }
}

/** Shared status badge — one styling source for desktop table and mobile cards. */
export function ZnsStatusBadge({ status }: { status: string }) {
  return (
    <Badge variant="outline" className={cn('whitespace-nowrap', statusClass(status))}>
      {ZNS_STATUS_LABELS[status] ?? status}
    </Badge>
  );
}
