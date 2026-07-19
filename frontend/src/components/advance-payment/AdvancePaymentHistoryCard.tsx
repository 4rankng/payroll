import { useCallback, useEffect, useId, useRef, useState } from "react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  Ban,
  CheckCircle,
  ChevronDown,
  Clock,
  RefreshCw,
  XCircle,
} from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";
import { getVietnameseAdvancePaymentStatus } from "@/utils/advancePaymentHelpers";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";

const STATUS_CONFIG = {
  PENDING: { icon: Clock, tone: "bg-[var(--employee-warning-soft)] text-[var(--employee-warning-strong)] ring-[var(--employee-warning-border)]" },
  APPROVED: { icon: CheckCircle, tone: "bg-[var(--employee-info-soft)] text-[var(--employee-info)] ring-[var(--employee-info-border)]" },
  COMPLETED: { icon: CheckCircle, tone: "bg-[var(--employee-accent-soft)] text-[var(--employee-accent)] ring-[var(--employee-accent-border)]" },
  FAILED: { icon: XCircle, tone: "bg-[var(--employee-error-soft)] text-[var(--employee-error)] ring-[#FECDCA]" },
  CANCELLED: { icon: Ban, tone: "bg-[#F2F4F7] text-[#475467] ring-[#E4E7EC]" },
} as const;

const safeFormat = (value: number | null | undefined) =>
  value == null || Number.isNaN(value) ? "—" : formatCurrency(value);

const safeDate = (date: string | null | undefined) => {
  if (!date) return "—";
  const parsedDate = new Date(date);
  return Number.isNaN(parsedDate.getTime())
    ? "—"
    : format(parsedDate, "dd/MM/yyyy", { locale: vi });
};

type CancelRequestHandler = (id: number) => void | Promise<void>;

function CancelPendingAction({
  itemId,
  onCancel,
  isCancelling,
}: {
  itemId: number;
  onCancel?: CancelRequestHandler;
  isCancelling: boolean;
}) {
  const [isConfirmingCancel, setIsConfirmingCancel] = useState(false);
  const submittingRef = useRef(false);
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cancelButtonRef = useRef<HTMLButtonElement | null>(null);
  const dismissButtonRef = useRef<HTMLButtonElement | null>(null);
  const confirmationWasOpen = useRef(false);
  const clearResetTimer = useCallback(() => {
    if (!resetTimer.current) return;
    clearTimeout(resetTimer.current);
    resetTimer.current = null;
  }, []);

  useEffect(() => {
    if (isConfirmingCancel) {
      confirmationWasOpen.current = true;
      dismissButtonRef.current?.focus();
      return;
    }
    if (confirmationWasOpen.current) {
      confirmationWasOpen.current = false;
      cancelButtonRef.current?.focus();
    }
  }, [isConfirmingCancel]);

  useEffect(() => clearResetTimer, [clearResetTimer]);

  const handleCancelClick = async () => {
    if (isCancelling || submittingRef.current) return;
    if (isConfirmingCancel) {
      clearResetTimer();
      submittingRef.current = true;
      try {
        await onCancel?.(itemId);
      } finally {
        submittingRef.current = false;
        setIsConfirmingCancel(false);
      }
      return;
    }
    setIsConfirmingCancel(true);
    resetTimer.current = setTimeout(() => {
      resetTimer.current = null;
      setIsConfirmingCancel(false);
    }, 3000);
  };

  const handleDismissCancel = () => {
    clearResetTimer();
    setIsConfirmingCancel(false);
  };

  if (isConfirmingCancel) {
    return (
      <div className="flex min-h-11 items-center justify-end gap-2">
        <button ref={dismissButtonRef} type="button" onClick={handleDismissCancel} disabled={isCancelling} className="employee-type-action min-h-11 rounded-[var(--employee-radius-control)] px-3 text-[var(--employee-text-secondary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-text-secondary)] disabled:opacity-50">
          Không
        </button>
        <button type="button" onClick={() => void handleCancelClick()} disabled={isCancelling} aria-busy={isCancelling} className="employee-type-action min-h-11 rounded-[var(--employee-radius-control)] bg-[var(--employee-error-soft)] px-4 text-[var(--employee-error)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-error)] disabled:opacity-60">
          {isCancelling ? "Đang hủy…" : "Xác nhận"}
        </button>
      </div>
    );
  }

  return (
    <button ref={cancelButtonRef} type="button" onClick={() => void handleCancelClick()} disabled={isCancelling} aria-busy={isCancelling} className="employee-type-action min-h-11 w-full rounded-[var(--employee-radius-control)] border border-[#FDA29B] bg-white text-[var(--employee-error)] transition-colors hover:bg-[var(--employee-error-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-error)] disabled:opacity-60">
      {isCancelling ? "Đang hủy…" : "Hủy yêu cầu"}
    </button>
  );
}

function HistoryItem({ item, onCancel, cancellingRequestId }: { item: AdvancePaymentHistoryItem; onCancel?: CancelRequestHandler; cancellingRequestId?: number }) {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();
  const config = STATUS_CONFIG[item.status] ?? STATUS_CONFIG.CANCELLED;
  const StatusIcon = config.icon;

  return (
    <article className="bg-[var(--employee-surface)]">
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="w-full px-4 py-3.5 text-left transition-transform duration-200 active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--employee-accent)]"
      >
        <span className="flex items-center gap-3">
          <span className="min-w-0 flex-1">
            <span className={cn("ct-badge employee-type-pill h-auto max-w-full gap-1.5 rounded-full px-2.5 py-1 ring-1 ring-inset", config.tone)}>
              <StatusIcon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span className="truncate">{getVietnameseAdvancePaymentStatus(item.status)}</span>
            </span>
            <span className="employee-type-body-sm mt-1.5 block text-[var(--employee-text-secondary)] tabular-nums">
              Yêu cầu ngày {safeDate(item.createdAt)}
            </span>
          </span>
          <span className="min-w-0 text-right">
            <span className="employee-type-card-title block break-words text-[var(--employee-text)] tabular-nums">
              {safeFormat(item.requestAmount)}
            </span>
            <span className="employee-type-body-sm mt-1 block text-[var(--employee-text-secondary)]">Số tiền yêu cầu</span>
          </span>
          <ChevronDown className={cn("h-5 w-5 shrink-0 text-[var(--employee-text-muted)] transition-transform", isOpen && "rotate-180")} aria-hidden="true" />
          <span className="sr-only">{isOpen ? "Thu gọn chi tiết" : "Xem phí và chi tiết"}</span>
        </span>
      </button>

      {isOpen && (
        <div id={panelId} className="border-t border-[#EAECF0] bg-[#F9FAFB] px-4 py-3">
          <div className="flex items-center justify-between gap-4">
            <span className="employee-type-label text-[#667085]">Thực nhận</span>
            <span className="employee-type-row-amount text-[var(--employee-accent)] tabular-nums">{safeFormat(item.netAmount)}</span>
          </div>
          <div className="mt-2 flex items-center justify-between gap-4">
            <span className="employee-type-label text-[#667085]">Phí giao dịch</span>
            <span className="employee-type-row-amount text-[#344054] tabular-nums">{safeFormat(item.fee)}</span>
          </div>
          {item.status === "PENDING" && (
            <div className="mt-3 border-t border-[#EAECF0] pt-3">
              <CancelPendingAction itemId={item.id} onCancel={onCancel} isCancelling={cancellingRequestId === item.id} />
            </div>
          )}
        </div>
      )}
    </article>
  );
}

interface AdvancePaymentHistoryCardProps {
  history: AdvancePaymentHistoryItem[];
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  onCancel?: CancelRequestHandler;
  cancellingRequestId?: number;
  totalCount?: number;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentHistoryCard({
  history,
  isLoading,
  isError = false,
  onRetry,
  onCancel,
  cancellingRequestId,
  totalCount,
  className,
  style,
}: AdvancePaymentHistoryCardProps) {
  const requestCount = totalCount ?? history.length;
  const isScrollable = history.length > 5;

  return (
    <div className={className} style={style} aria-labelledby="employee-advance-history-title">
      <div className="mb-3 flex items-start justify-between gap-3 px-0.5">
        <div className="min-w-0">
          <h2 id="employee-advance-history-title" className="employee-type-section-title text-[#101828]">Lịch sử yêu cầu</h2>
          <p className="employee-type-body mt-0.5 text-[var(--employee-text-secondary)]">Theo dõi trạng thái các yêu cầu ứng lương</p>
        </div>
        {!isLoading && !isError && (
          <span className="ct-badge ct-badge-success ct-badge-outline employee-type-pill h-auto shrink-0 rounded-full px-2.5 py-1 tabular-nums">
            {requestCount} yêu cầu
          </span>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2.5" aria-label="Đang tải lịch sử yêu cầu">
          {[1, 2].map((index) => <Skeleton key={index} className="h-32 w-full rounded-xl" />)}
        </div>
      ) : isError ? (
        <div className="ct-card employee-surface-card bg-[var(--employee-surface)] px-4 py-5 text-center" role="alert">
          <p className="employee-type-strong text-[#101828]">Chưa tải được lịch sử</p>
          <p className="employee-type-body-sm mt-1 text-[#667085]">Kiểm tra kết nối rồi thử lại.</p>
          {onRetry && (
            <button type="button" onClick={onRetry} className="ct-btn ct-btn-outline employee-type-action mt-3 h-auto min-h-11 gap-2 rounded-[10px] px-4 normal-case">
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Tải lại
            </button>
          )}
        </div>
      ) : history.length === 0 ? (
        <div className="flex min-h-32 flex-col items-center justify-center rounded-xl border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-4 py-5 text-center">
          <img
            src="/advance-payment-empty-state.png"
            alt=""
            width={72}
            height={72}
            loading="lazy"
            decoding="async"
            className="employee-empty-art mb-2 h-[72px] w-[72px] rounded-2xl object-cover"
            aria-hidden="true"
          />
          <p className="employee-type-card-title text-[#344054]">Chưa có yêu cầu ứng lương</p>
          <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Yêu cầu mới sẽ xuất hiện tại đây.</p>
        </div>
      ) : (
        <div
          className={cn(
            "ct-card employee-surface-card divide-y divide-[var(--employee-border)] bg-[var(--employee-surface)]",
            isScrollable
              ? "max-h-[390px] overflow-y-auto overscroll-contain"
              : "overflow-hidden"
          )}
          tabIndex={isScrollable ? 0 : undefined}
          aria-label={isScrollable ? "Lịch sử yêu cầu, cuộn để xem thêm" : undefined}
        >
          {history.map((item) => <HistoryItem key={item.id} item={item} onCancel={onCancel} cancellingRequestId={cancellingRequestId} />)}
        </div>
      )}
    </div>
  );
}
