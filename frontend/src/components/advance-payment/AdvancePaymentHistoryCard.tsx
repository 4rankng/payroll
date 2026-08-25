import { useCallback, useEffect, useId, useRef, useState } from "react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  Ban,
  CheckCircle,
  ChevronDown,
  Clock,
  ReceiptText,
  RefreshCw,
  XCircle,
} from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";
import { getVietnameseAdvancePaymentStatus } from "@/utils/advancePaymentHelpers";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";

// Each status pairs a colour rail with an icon and a text label so payment
// state never depends on colour alone.
const STATUS_CONFIG = {
  PENDING: {
    icon: Clock,
    rail: "bg-[var(--employee-warning)]",
    text: "text-[var(--employee-warning-strong)]",
  },
  APPROVED: {
    icon: CheckCircle,
    rail: "bg-[var(--employee-info)]",
    text: "text-[var(--employee-info)]",
  },
  COMPLETED: {
    icon: CheckCircle,
    rail: "bg-[var(--employee-accent)]",
    text: "text-[var(--employee-accent)]",
  },
  FAILED: {
    icon: XCircle,
    rail: "bg-[var(--employee-error)]",
    text: "text-[var(--employee-error)]",
  },
  CANCELLED: {
    icon: Ban,
    rail: "bg-[var(--employee-border-strong)]",
    text: "text-[var(--employee-text-muted)]",
  },
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
      <div className="flex min-h-12 items-center justify-end gap-2">
        <button
          ref={dismissButtonRef}
          type="button"
          onClick={handleDismissCancel}
          disabled={isCancelling}
          className="h-12 rounded-xl px-4 text-[0.875rem] font-medium text-slate-500 transition-colors hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 disabled:opacity-50"
        >
          Không
        </button>
        <button
          type="button"
          onClick={() => void handleCancelClick()}
          disabled={isCancelling}
          aria-busy={isCancelling}
          className="h-12 rounded-xl bg-red-50 px-5 text-[0.875rem] font-semibold text-red-600 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 disabled:opacity-60"
        >
          {isCancelling ? "Đang hủy…" : "Xác nhận"}
        </button>
      </div>
    );
  }

  return (
    <button
      ref={cancelButtonRef}
      type="button"
      onClick={() => void handleCancelClick()}
      disabled={isCancelling}
      aria-busy={isCancelling}
      className="flex h-12 w-full items-center justify-center rounded-xl border border-red-200 bg-white text-[0.875rem] font-semibold text-red-600 transition-all duration-150 hover:border-red-300 hover:bg-red-50 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 disabled:opacity-60"
    >
      {isCancelling ? "Đang hủy…" : "Hủy yêu cầu"}
    </button>
  );
}

function HistoryItem({
  item,
  onCancel,
  cancellingRequestId,
}: {
  item: AdvancePaymentHistoryItem;
  onCancel?: CancelRequestHandler;
  cancellingRequestId?: number;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();
  const config = STATUS_CONFIG[item.status] ?? STATUS_CONFIG.CANCELLED;
  const StatusIcon = config.icon;

  return (
    <article className="transition-colors duration-150 hover:bg-[var(--employee-page)]">
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="w-full px-3 py-3 text-left transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--employee-accent)]"
      >
        <div className="flex items-center gap-2.5">
          {/* Short status marker, inside the summary row so it stays aligned
              with it when the detail panel opens. */}
          <span
            className={cn("h-5 w-[3px] shrink-0 rounded-full", config.rail)}
            aria-hidden="true"
          />

          {/* Status & date */}
          <div className="min-w-0 flex-1">
            {/* The icon carries the status colour; the label stays neutral so a
                list of completed rows does not read as a wall of green. */}
            <span className="flex items-center gap-1.5">
              <StatusIcon
                className={cn("h-3.5 w-3.5 shrink-0", config.text)}
                strokeWidth={2.5}
                aria-hidden="true"
              />
              <span className="employee-type-body truncate text-[var(--employee-text)]">
                {getVietnameseAdvancePaymentStatus(item.status)}
              </span>
            </span>
            <span className="employee-type-pill mt-0.5 block tabular-nums text-[var(--employee-text-muted)]">
              <span className="sr-only">Yêu cầu ngày </span>
              {safeDate(item.createdAt)}
            </span>
          </div>

          {/* Amount — the row's primary value, so it stays neutral and strong
              while the status carries the colour. */}
          <span className="employee-type-row-amount shrink-0 tabular-nums text-[var(--employee-text)]">
            {safeFormat(item.requestAmount)}
          </span>

          <ChevronDown
            className={cn(
              "h-4 w-4 shrink-0 text-[var(--employee-text-muted)] transition-transform duration-200",
              isOpen && "rotate-180",
            )}
            aria-hidden="true"
          />
          <span className="sr-only">
            {isOpen ? "Thu gọn chi tiết" : "Xem phí và chi tiết"}
          </span>
        </div>
      </button>

      {/* Expandable detail panel */}
      {isOpen && (
        <div
          id={panelId}
          className="border-t border-[var(--employee-border)] bg-[var(--employee-page)] py-3.5 pl-[1.375rem] pr-3"
        >
          <div className="space-y-2.5">
            <div className="flex items-center justify-between gap-4">
              <span className="employee-type-pill text-[var(--employee-text-secondary)]">Thực nhận</span>
              <span className="employee-type-body font-semibold tabular-nums text-[var(--employee-accent)]">
                {safeFormat(item.netAmount)}
              </span>
            </div>
            <div className="flex items-center justify-between gap-4">
              <span className="employee-type-pill text-[var(--employee-text-secondary)]">Phí giao dịch</span>
              <span className="employee-type-body font-semibold tabular-nums text-[var(--employee-text)]">
                {safeFormat(item.fee)}
              </span>
            </div>
          </div>

          {item.status === "PENDING" && (
            <div className="mt-3.5 border-t border-[var(--employee-border)] pt-3.5">
              <CancelPendingAction
                itemId={item.id}
                onCancel={onCancel}
                isCancelling={cancellingRequestId === item.id}
              />
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
    <div
      className={cn(
        "overflow-hidden rounded-[var(--employee-radius-card)] border border-[var(--employee-border)] bg-[var(--employee-surface)] shadow-[var(--employee-shadow)]",
        className,
      )}
      style={style}
      aria-labelledby="employee-advance-history-title"
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-3 border-b border-[var(--employee-border)] px-4 py-3">
        <h2
          id="employee-advance-history-title"
          className="employee-type-card-title min-w-0 truncate text-[var(--employee-text)]"
        >
          Lịch sử yêu cầu
        </h2>
        {!isLoading && !isError && (
          <span className="employee-type-pill inline-flex shrink-0 items-center rounded-full border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-2.5 py-1 tabular-nums text-[var(--employee-accent)]">
            {requestCount} yêu cầu
          </span>
        )}
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="space-y-3 p-5" aria-label="Đang tải lịch sử yêu cầu">
          {[1, 2, 3].map((index) => (
            <div key={index} className="flex items-center gap-3.5 rounded-xl p-4">
              <Skeleton className="h-11 w-11 shrink-0 rounded-xl" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-3 w-16" />
              </div>
              <div className="space-y-1.5 text-right">
                <Skeleton className="ml-auto h-4 w-20" />
                <Skeleton className="ml-auto h-3 w-16" />
              </div>
            </div>
          ))}
        </div>
      ) : isError ? (
        <div className="px-5 py-10 text-center" role="alert">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-red-50 text-red-400">
            <RefreshCw className="h-6 w-6" />
          </div>
          <p className="mt-4 text-[1rem] font-semibold text-slate-700">
            Chưa tải được lịch sử
          </p>
          <p className="mt-1 text-[0.8125rem] text-slate-400">
            Kiểm tra kết nối rồi thử lại.
          </p>
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="mt-4 inline-flex h-11 items-center gap-2 rounded-xl border border-slate-200 bg-white px-5 text-[0.875rem] font-semibold text-slate-600 transition-all duration-150 hover:border-emerald-300 hover:bg-emerald-50 hover:text-emerald-700 active:scale-95"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Tải lại
            </button>
          )}
        </div>
      ) : history.length === 0 ? (
        <div className="flex min-h-44 flex-col items-center justify-center bg-[var(--employee-accent-soft)] px-5 py-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-[16px] bg-[var(--employee-surface)] text-[var(--employee-accent)] ring-1 ring-inset ring-[var(--employee-accent-border)]">
            <ReceiptText className="h-6 w-6" aria-hidden="true" />
          </div>
          <p className="employee-type-card-title mt-3 text-[var(--employee-text)]">
            Chưa có yêu cầu ứng lương
          </p>
          <p className="employee-type-body-sm mt-1 max-w-xs text-[var(--employee-text-secondary)]">
            Yêu cầu mới sẽ xuất hiện tại đây.
          </p>
        </div>
      ) : (
        <div
          className={cn(
            "divide-y divide-[var(--employee-border)]",
            isScrollable
              ? "max-h-[330px] overflow-y-auto overscroll-contain"
              : "",
          )}
          tabIndex={isScrollable ? 0 : undefined}
          aria-label={isScrollable ? "Lịch sử yêu cầu, cuộn để xem thêm" : undefined}
        >
          {history.map((item) => (
            <HistoryItem
              key={item.id}
              item={item}
              onCancel={onCancel}
              cancellingRequestId={cancellingRequestId}
            />
          ))}
        </div>
      )}
    </div>
  );
}
