import { useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { toast } from 'sonner';
import { AlertTriangle, ChevronLeft, ChevronRight, ClipboardCheck, Loader2, RefreshCw } from 'lucide-react';

import { ApprovalDecisionModal, type ApprovalDecisionMode } from '@/components/approvals/ApprovalDecisionModal';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { getConsoleKicker } from '@/lib/console';
import { cn } from '@/lib/utils';
import {
  useApprovalsList,
  useApproveApproval,
  useFileApproval,
  useCancelApproval,
  useRejectApproval,
} from '@/hooks/api/useApprovals';
import type { ApprovalRequest, ApprovalStatus } from '@/types/api/approval.types';

const PAGE_SIZE = 15;

const FILTERS: { key: string; label: string; statuses: ApprovalStatus[] }[] = [
  { key: 'waiting', label: 'Chờ duyệt', statuses: ['pending', 'rescheduled'] },
  { key: 'done', label: 'Đã duyệt', statuses: ['approved', 'executed'] },
  { key: 'rejected', label: 'Từ chối', statuses: ['rejected'] },
  { key: 'escalated', label: 'Leo thang', statuses: ['escalated'] },
  { key: 'all', label: 'Tất cả', statuses: [] },
];

const STATUS_BADGE: Record<string, string> = {
  pending: 'border-indigo-200 bg-indigo-50 text-indigo-800',
  rescheduled: 'border-amber-200 bg-amber-50 text-amber-800',
  escalated: 'border-red-200 bg-red-50 text-red-800',
  executed: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  approved: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  rejected: 'border-border bg-muted/50 text-muted-foreground',
  cancelled: 'border-border bg-muted/50 text-muted-foreground',
  expired: 'border-border bg-muted/50 text-muted-foreground',
};

const canDecide = (item: ApprovalRequest) => item.status === 'pending' || item.status === 'rescheduled';

const formatDateTime = (iso?: string | null) => {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('vi-VN', { hour12: false });
};

/** Mobile approval queue: one-column list, 44px targets, chip filters. */
const ApprovalsPageMobile = () => {
  const location = useLocation();
  const [filterKey, setFilterKey] = useState('waiting');
  const [page, setPage] = useState(1);
  const [modal, setModal] = useState<{ mode: ApprovalDecisionMode; request: ApprovalRequest } | null>(null);
  const [prepareDate, setPrepareDate] = useState('');
  const fileApproval = useFileApproval();

  const handlePrepareSaoKe = () => {
    const reportAtDate = prepareDate.trim();
    if (!reportAtDate) {
      toast.error('Chọn ngày sao kê trước khi chuẩn bị.');
      return;
    }
    fileApproval.mutate(
      { type: 'send_sao_ke_email', params: { reportAtDate } },
      {
        onSuccess: (filed) => {
          toast.success(`Đã gửi yêu cầu #${filed.id} chờ duyệt.`);
          setPrepareDate('');
        },
        onError: () => toast.error('Không thể chuẩn bị — kiểm tra ngày sao kê.'),
      }
    );
  };

  const filter = FILTERS.find((f) => f.key === filterKey) ?? FILTERS[0];
  const listQuery = useApprovalsList({ statuses: filter.statuses, page, pageSize: PAGE_SIZE });
  const waitingQuery = useApprovalsList({ statuses: ['pending', 'rescheduled'], pageSize: 1 });
  const escalatedQuery = useApprovalsList({ statuses: ['escalated'], pageSize: 1 });

  const approveMutation = useApproveApproval();
  const rejectMutation = useRejectApproval();
  const cancelMutation = useCancelApproval();
  const activeMutation = approveMutation.isPending || rejectMutation.isPending || cancelMutation.isPending;

  const escalatedCount = escalatedQuery.data?.pagination?.totalRecords ?? null;
  const items = listQuery.data?.items ?? [];
  const pagination = listQuery.data?.pagination ?? null;

  const headerSummary = useMemo(() => {
    const waiting = waitingQuery.data?.pagination?.totalRecords ?? null;
    if (waiting === null) return 'Đang tải…';
    return waiting > 0 ? `${waiting.toLocaleString('vi-VN')} yêu cầu chờ duyệt` : 'Không có yêu cầu chờ duyệt';
  }, [waitingQuery.data]);

  const handleConfirm = (id: number, note: string) => {
    if (!modal) return;
    const finish = () => setModal(null);
    if (modal.mode === 'approve') approveMutation.mutate({ id, note }, { onSettled: finish });
    else if (modal.mode === 'reject') rejectMutation.mutate({ id, note }, { onSettled: finish });
    else cancelMutation.mutate({ id, note }, { onSettled: finish });
  };

  return (
    <div className="flex min-h-[100dvh] flex-col overflow-x-clip bg-background pb-[env(safe-area-inset-bottom)]">
      <div className="sticky top-0 z-20 border-b border-border bg-card">
        <MobilePageHeader
          title="Yêu cầu phê duyệt"
          kicker={getConsoleKicker(location.pathname)}
          subtitle={headerSummary}
          icon={ClipboardCheck}
          actions={
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-11 shrink-0"
              aria-label="Làm mới"
              onClick={() => {
                void listQuery.refetch();
                void waitingQuery.refetch();
                void escalatedQuery.refetch();
              }}
            >
              <RefreshCw aria-hidden="true" className="size-4" />
            </Button>
          }
        />
        <div role="tablist" aria-label="Lọc theo trạng thái" className="console-filter-bar overflow-x-auto px-4 pb-2">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              role="tab"
              aria-selected={filterKey === f.key}
              onClick={() => {
                setFilterKey(f.key);
                setPage(1);
              }}
              className={cn(
                'tap-target-44 h-9 shrink-0 rounded-full border px-3.5 text-sm transition-colors',
                filterKey === f.key
                  ? 'border-indigo-300 bg-indigo-50 font-medium text-indigo-800'
                  : 'border-border bg-background text-muted-foreground'
              )}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 space-y-3 p-4 pb-[calc(4rem+env(safe-area-inset-bottom))]">
        <section aria-label="Chuẩn bị sao kê" className="console-panel">
          <div className="console-panel-body flex items-end gap-2">
            <div className="min-w-0 flex-1">
              <label htmlFor="sao-ke-date-mobile" className="block text-sm font-medium text-foreground">
                Chuẩn bị sao kê
              </label>
              <input
                id="sao-ke-date-mobile"
                type="date"
                value={prepareDate}
                onChange={(event) => setPrepareDate(event.target.value)}
                disabled={fileApproval.isPending}
                className="mt-1.5 h-11 w-full rounded-md border border-input bg-background px-3 text-sm"
                aria-label="Ngày sao kê"
              />
            </div>
            <Button
              type="button"
              className="h-11 shrink-0"
              disabled={fileApproval.isPending}
              onClick={handlePrepareSaoKe}
            >
              {fileApproval.isPending ? 'Đang…' : 'Chuẩn bị'}
            </Button>
          </div>
        </section>

        {escalatedCount ? (
          <div role="alert" className="flex items-start gap-3 rounded-xl border border-utility-error-200 bg-utility-error-50 px-4 py-3">
            <AlertTriangle aria-hidden="true" className="mt-0.5 size-5 shrink-0 text-utility-error-700" />
            <p className="text-sm leading-5 text-fg-error-primary">
              {escalatedCount.toLocaleString('vi-VN')} yêu cầu đã leo thang sau khi hết lượt thử lại — cần quyết định của bạn.
            </p>
          </div>
        ) : null}

        {listQuery.isLoading ? (
          <div role="status" aria-live="polite" className="console-panel px-4 py-12 text-center">
            <Loader2 aria-hidden="true" className="mx-auto mb-3 size-6 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Đang tải…</p>
          </div>
        ) : listQuery.isError ? (
          <div role="alert" className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-center">
            <p className="text-sm font-medium text-destructive">Không thể tải danh sách.</p>
            <Button type="button" variant="outline" className="mt-3 h-11" onClick={() => void listQuery.refetch()}>
              Tải lại
            </Button>
          </div>
        ) : items.length === 0 ? (
          <div className="console-panel px-4 py-12 text-center">
            <p className="text-sm text-muted-foreground">Không có yêu cầu trong mục này.</p>
          </div>
        ) : (
          items.map((item) => (
            <article key={item.id} className="console-panel px-4 py-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium text-foreground">
                  {item.summary || item.type_label}
                </span>
                <Badge variant="outline" className={cn('h-6 px-2 text-xs', STATUS_BADGE[item.status] ?? '')}>
                  {item.status_label}
                </Badge>
              </div>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Người gửi: {item.requester_name || '—'} · {formatDateTime(item.created_at)}
                {item.retry_count > 0 ? ` · đã thử lại ${item.retry_count} lần` : ''}
              </p>
              {item.approve_effect ? (
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  Khi duyệt: <span className="text-foreground">{item.approve_effect}</span>
                </p>
              ) : null}
              {item.decision_note ? (
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  Ghi chú: <span className="text-foreground">{item.decision_note}</span>
                </p>
              ) : null}
              {item.outcome ? (
                <p className="mt-1 text-xs leading-5 text-muted-foreground">
                  Kết quả: <span className="text-foreground">{item.outcome}</span>
                </p>
              ) : null}
              {canDecide(item) ? (
                <div className="mt-3 flex gap-2">
                  <Button
                    type="button"
                    className="h-11 flex-1 bg-emerald-700 text-white hover:bg-emerald-800"
                    disabled={activeMutation}
                    onClick={() => setModal({ mode: 'approve', request: item })}
                  >
                    Duyệt
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    className="h-11 flex-1 border-amber-300 text-amber-800 hover:bg-amber-50"
                    disabled={activeMutation}
                    onClick={() => setModal({ mode: 'reject', request: item })}
                  >
                    Từ chối
                  </Button>
                </div>
              ) : null}
            </article>
          ))
        )}

        {pagination && pagination.totalPages > 1 ? (
          <nav className="flex items-center justify-between gap-2 pt-1" aria-label="Phân trang">
            <Button
              type="button"
              variant="outline"
              className="h-11 shrink-0 gap-1"
              disabled={page <= 1 || listQuery.isFetching}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              <ChevronLeft aria-hidden="true" className="size-4" />
              Trước
            </Button>
            <p className="text-xs text-muted-foreground">
              Trang {pagination.page} / {pagination.totalPages} · {pagination.totalRecords.toLocaleString('vi-VN')} yêu cầu
            </p>
            <Button
              type="button"
              variant="outline"
              className="h-11 shrink-0 gap-1"
              disabled={page >= pagination.totalPages || listQuery.isFetching}
              onClick={() => setPage((p) => p + 1)}
            >
              Sau
              <ChevronRight aria-hidden="true" className="size-4" />
            </Button>
          </nav>
        ) : null}
      </div>

      <ApprovalDecisionModal
        open={modal !== null}
        mode={modal?.mode ?? null}
        request={modal?.request ?? null}
        isPending={activeMutation}
        onClose={() => setModal(null)}
        onConfirm={handleConfirm}
      />
    </div>
  );
};

export default ApprovalsPageMobile;
