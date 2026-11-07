import { useApprovalsList } from '@/hooks/api/useApprovals';
import { cn } from '@/lib/utils';
import { ConsolePanel } from '@/components/shared/console-panel';

const STATUS_TONE: Record<string, string> = {
  pending: 'text-info',
  rescheduled: 'text-warning',
  escalated: 'text-destructive',
  executed: 'text-success',
  approved: 'text-success',
  rejected: 'text-destructive',
  cancelled: 'text-muted-foreground',
  expired: 'text-muted-foreground',
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
    <ConsolePanel
      kicker="Phê duyệt"
      title="Yêu cầu của tôi"
      className="text-card-foreground"
      actions={
        <span className="console-value text-[11px] text-muted-foreground">
          {items.length} yêu cầu
        </span>
      }
      isFlush={compact}
      bodyClassName={compact ? 'px-3 py-2' : undefined}
    >
      <ul className="divide-y divide-border/60">
        {items.map((item) => (
          <li key={item.id} className="flex flex-wrap items-center justify-between gap-1.5 py-1.5">
            <div className="min-w-0">
              <p className="truncate text-[13px] text-foreground">
                {item.summary || item.type_label}
              </p>
              <p className="text-[11px] text-muted-foreground">
                {formatDateTime(item.created_at)}
                {item.scheduled_at ? ` · thử lại ${formatDateTime(item.scheduled_at)}` : ''}
              </p>
              {item.decision_note ? (
                <p className="text-[11px] text-muted-foreground">
                  Ghi chú: <span className="text-foreground">{item.decision_note}</span>
                </p>
              ) : null}
            </div>
            <span
              className={cn(
                'console-kicker shrink-0',
                STATUS_TONE[item.status] ?? 'text-muted-foreground',
              )}
            >
              {item.status_label}
            </span>
          </li>
        ))}
      </ul>
    </ConsolePanel>
  );
};
