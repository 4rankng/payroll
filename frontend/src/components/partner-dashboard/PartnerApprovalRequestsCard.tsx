import { Inbox } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { useApprovalsList } from '@/hooks/api/useApprovals';
import { cn } from '@/lib/utils';

const STATUS_BADGE: Record<string, string> = {
  pending: 'border-indigo-200 bg-indigo-50 text-indigo-800',
  rescheduled: 'border-amber-200 bg-amber-50 text-amber-800',
  escalated: 'border-red-200 bg-red-50 text-red-800',
  executed: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  approved: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  rejected: 'border-slate-200 bg-slate-50 text-slate-700',
  cancelled: 'border-slate-200 bg-slate-50 text-slate-500',
  expired: 'border-slate-200 bg-slate-50 text-slate-500',
};

const formatDateTime = (iso?: string | null) => {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('vi-VN', { hour12: false });
};

/** Partner "Yêu cầu của tôi": read-only status of the caller's own approval
 * requests. Renders nothing when the list is empty (non-zero row law) and
 * nothing while loading — a partner without requests never sees a dead card. */
export const PartnerApprovalRequestsCard = ({ compact = false }: { compact?: boolean }) => {
  const { data, isLoading } = useApprovalsList({ page: 1, pageSize: 5 });
  const items = data?.items ?? [];

  if (isLoading || items.length === 0) return null;

  return (
    <section
      aria-label="Yêu cầu phê duyệt của tôi"
      className={cn(
        'rounded-xl border bg-card text-card-foreground',
        compact ? 'px-4 py-3.5' : 'p-4 sm:p-5'
      )}
    >
      <div className="mb-2.5 flex items-center gap-2">
        <Inbox aria-hidden="true" className="size-4 text-indigo-700" />
        <h2 className="text-sm font-semibold text-foreground">Yêu cầu của tôi</h2>
        <span className="text-xs text-muted-foreground">({items.length})</span>
      </div>
      <ul className="space-y-2">
        {items.map((item) => (
          <li key={item.id} className="flex flex-wrap items-center justify-between gap-1.5 rounded-lg border bg-background px-3 py-2">
            <div className="min-w-0">
              <p className="truncate text-sm text-foreground">
                {item.summary || item.type_label}
              </p>
              <p className="text-xs text-muted-foreground">
                {formatDateTime(item.created_at)}
                {item.scheduled_at ? ` · thử lại ${formatDateTime(item.scheduled_at)}` : ''}
              </p>
              {item.decision_note ? (
                <p className="text-xs text-muted-foreground">
                  Ghi chú: <span className="text-foreground">{item.decision_note}</span>
                </p>
              ) : null}
            </div>
            <Badge variant="outline" className={cn('h-6 shrink-0 px-2 text-xs', STATUS_BADGE[item.status] ?? '')}>
              {item.status_label}
            </Badge>
          </li>
        ))}
      </ul>
      <p className="mt-2.5 text-xs leading-5 text-muted-foreground">
        Yêu cầu chỉ được thực thi sau khi quản trị viên duyệt — kết quả sẽ được thông báo đến bạn.
      </p>
    </section>
  );
};
