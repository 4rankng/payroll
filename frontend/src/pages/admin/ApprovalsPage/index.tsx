import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AlertTriangle, ChevronLeft, ChevronRight, ClipboardCheck, Loader2, RefreshCw } from 'lucide-react';

import { ApprovalDecisionModal, type ApprovalDecisionMode } from '@/components/approvals/ApprovalDecisionModal';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import {
  useApprovalsList,
  useApproveApproval,
  useFileApproval,
  useCancelApproval,
  useRejectApproval,
} from '@/hooks/api/useApprovals';
import type { ApprovalRequest, ApprovalStatus } from '@/types/api/approval.types';

const PAGE_SIZE = 20;

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
  rejected: 'border-slate-200 bg-slate-50 text-slate-700',
  cancelled: 'border-slate-200 bg-slate-50 text-slate-500',
  expired: 'border-slate-200 bg-slate-50 text-slate-500',
};

const canDecide = (item: ApprovalRequest) => item.status === 'pending' || item.status === 'rescheduled';

const formatDateTime = (iso?: string | null) => {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString('vi-VN', { hour12: false });
};

/** Dedicated approval-request surface: Tổng Quan links here for the full
 * queue with decisions (approve / reject-with-note / terminal cancel). */
const ApprovalsPage = () => {
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
          toast.success(`Đã gửi yêu cầu #${filed.id} chờ duyệt — bản thảo email đính kèm yêu cầu.`);
          setPrepareDate('');
        },
        onError: () => toast.error('Không thể chuẩn bị yêu cầu — kiểm tra ngày sao kê (ngày 11–23 không hợp lệ).'),
      }
    );
  };

  const filter = FILTERS.find((f) => f.key === filterKey) ?? FILTERS[0];
  const waitingQuery = useApprovalsList({ statuses: ['pending', 'rescheduled'], pageSize: 1 });
  const escalatedQuery = useApprovalsList({ statuses: ['escalated'], pageSize: 1 });
  const listQuery = useApprovalsList({
    statuses: filter.statuses,
    page,
    pageSize: PAGE_SIZE,
  });

  const approveMutation = useApproveApproval();
  const rejectMutation = useRejectApproval();
  const cancelMutation = useCancelApproval();
  const activeMutation = approveMutation.isPending || rejectMutation.isPending || cancelMutation.isPending;

  const waitingCount = waitingQuery.data?.pagination?.totalRecords ?? null;
  const escalatedCount = escalatedQuery.data?.pagination?.totalRecords ?? null;
  const items = listQuery.data?.items ?? [];
  const pagination = listQuery.data?.pagination ?? null;

  const summaryLine = useMemo(() => {
    const parts: string[] = [];
    if (waitingCount !== null) parts.push(`${waitingCount.toLocaleString('vi-VN')} chờ duyệt`);
    if (escalatedCount) parts.push(`${escalatedCount.toLocaleString('vi-VN')} leo thang`);
    return parts.length ? parts.join(' · ') : 'Không có yêu cầu đang chờ';
  }, [waitingCount, escalatedCount]);

  const handleConfirm = (id: number, note: string) => {
    if (!modal) return;
    const finish = () => setModal(null);
    if (modal.mode === 'approve') approveMutation.mutate({ id, note }, { onSettled: finish });
    else if (modal.mode === 'reject') rejectMutation.mutate({ id, note }, { onSettled: finish });
    else cancelMutation.mutate({ id, note }, { onSettled: finish });
  };

  return (
    <div className="approvals-page p-3 sm:p-4 lg:p-6">
      <header className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex size-10 items-center justify-center rounded-lg border bg-card">
            <ClipboardCheck aria-hidden="true" className="size-5 text-indigo-700" />
          </div>
          <div>
            <h1 className="text-lg font-semibold text-foreground">Yêu cầu phê duyệt</h1>
            <p className="text-sm text-muted-foreground">{summaryLine}</p>
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          className="h-10 gap-2"
          onClick={() => {
            void waitingQuery.refetch();
            void escalatedQuery.refetch();
            void listQuery.refetch();
          }}
        >
          <RefreshCw aria-hidden="true" className="size-4" />
          Làm mới
        </Button>
      </header>

      <section
        aria-label="Chuẩn bị sao kê"
        className="mb-4 flex flex-wrap items-end gap-3 rounded-xl border bg-card px-4 py-3.5"
      >
        <div className="min-w-0">
          <label htmlFor="sao-ke-date" className="block text-sm font-medium text-foreground">
            Chuẩn bị sao kê
          </label>
          <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
            Chọn ngày sao kê — hệ thống kiểm tra dữ liệu, soạn bản thảo email bằng AI và gửi vào hàng chờ duyệt.
          </p>
        </div>
        <input
          id="sao-ke-date"
          type="date"
          value={prepareDate}
          onChange={(event) => setPrepareDate(event.target.value)}
          disabled={fileApproval.isPending}
          className="h-11 rounded-md border border-input bg-background px-3 text-sm"
          aria-label="Ngày sao kê"
        />
        <Button
          type="button"
          className="h-11 gap-2"
          disabled={fileApproval.isPending}
          onClick={handlePrepareSaoKe}
        >
          {fileApproval.isPending ? 'Đang chuẩn bị…' : 'Chuẩn bị sao kê'}
        </Button>
      </section>

      {escalatedCount ? (
        <div
          role="alert"
          className="mb-4 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3"
        >
          <AlertTriangle aria-hidden="true" className="mt-0.5 size-5 shrink-0 text-red-700" />
          <div className="min-w-0 text-sm text-red-800">
            <p className="font-medium">
              {escalatedCount.toLocaleString('vi-VN')} yêu cầu đã leo thang sau khi hết lượt thử lại
            </p>
            <p className="mt-0.5">
              Các yêu cầu này đã bị từ chối hoặc thiếu dữ liệu sau 3 lần — cần quyết định thủ công của bạn.
            </p>
          </div>
        </div>
      ) : null}

      <Tabs value={filterKey} onValueChange={(next) => { setFilterKey(next); setPage(1); }} className="mb-4">
        <TabsList className="h-11 w-full justify-start overflow-x-auto sm:w-auto">
          {FILTERS.map((f) => (
            <TabsTrigger key={f.key} value={f.key} className="h-9 px-3 text-sm">
              {f.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {listQuery.isLoading ? (
        <div role="status" aria-live="polite" className="rounded-xl border bg-card px-4 py-12 text-center">
          <Loader2 aria-hidden="true" className="mx-auto mb-3 size-6 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Đang tải danh sách yêu cầu…</p>
        </div>
      ) : listQuery.isError ? (
        <div role="alert" className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-center">
          <p className="text-sm font-medium text-destructive">Không thể tải danh sách yêu cầu.</p>
          <Button type="button" variant="outline" className="mt-4 h-11" onClick={() => void listQuery.refetch()}>
            <RefreshCw aria-hidden="true" className="size-4" />
            Tải lại
          </Button>
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-xl border bg-card px-4 py-12 text-center">
          <p className="text-sm text-muted-foreground">Không có yêu cầu trong mục này.</p>
        </div>
      ) : (
        <ul className="space-y-2">
          {items.map((item) => (
            <li key={item.id} className="rounded-xl border bg-card px-4 py-3">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium text-foreground">
                      #{item.id} · {item.type_label}
                    </span>
                    <Badge variant="outline" className={cn('h-6 px-2 text-xs', STATUS_BADGE[item.status] ?? '')}>
                      {item.status_label}
                    </Badge>
                    {item.retry_count > 0 ? (
                      <Badge variant="outline" className="h-6 px-2 text-xs text-muted-foreground">
                        đã thử lại {item.retry_count} lần
                      </Badge>
                    ) : null}
                  </div>
                  <p className="mt-1 text-xs leading-5 text-muted-foreground">
                    Gửi lúc {formatDateTime(item.created_at)}
                    {item.scheduled_at ? ` · thử lại lúc ${formatDateTime(item.scheduled_at)}` : ''}
                  </p>
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
                  {typeof item.payload?.llmDraftBody === 'string' && item.payload.llmDraftBody ? (
                    <details className="mt-2 rounded-lg border bg-background px-3 py-2">
                      <summary className="cursor-pointer text-xs font-medium text-foreground">
                        Bản thảo email sẽ gửi
                        {typeof item.payload?.llmDraftSubject === 'string' && item.payload.llmDraftSubject
                          ? ` — ${item.payload.llmDraftSubject}`
                          : ''}
                      </summary>
                      <p className="mt-2 whitespace-pre-line text-xs leading-5 text-muted-foreground">
                        {String(item.payload.llmDraftBody)}
                      </p>
                    </details>
                  ) : null}
                </div>
                {canDecide(item) ? (
                  <div className="flex shrink-0 gap-2">
                    <Button
                      type="button"
                      size="sm"
                      className="h-9 bg-emerald-600 text-white hover:bg-emerald-700"
                      disabled={activeMutation}
                      onClick={() => setModal({ mode: 'approve', request: item })}
                    >
                      Duyệt
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className="h-9 border-amber-300 text-amber-800 hover:bg-amber-50"
                      disabled={activeMutation}
                      onClick={() => setModal({ mode: 'reject', request: item })}
                    >
                      Từ chối
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className="h-9 text-red-700 hover:bg-red-50"
                      disabled={activeMutation}
                      onClick={() => setModal({ mode: 'cancel', request: item })}
                    >
                      Huỷ
                    </Button>
                  </div>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}

      {pagination && pagination.totalPages > 1 ? (
        <nav className="mt-4 flex items-center justify-between" aria-label="Phân trang">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-10 gap-1"
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
            size="sm"
            className="h-10 gap-1"
            disabled={page >= (pagination?.totalPages ?? 1) || listQuery.isFetching}
            onClick={() => setPage((p) => p + 1)}
          >
            Sau
            <ChevronRight aria-hidden="true" className="size-4" />
          </Button>
        </nav>
      ) : null}

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

export default ApprovalsPage;
