import { useEffect, useId, useRef, useState } from "react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  Ban,
  CheckCircle,
  ChevronDown,
  Clock,
  History,
  XCircle,
} from "lucide-react";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import { Skeleton } from "@/components/ui/skeleton";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";
import { getVietnameseAdvancePaymentStatus } from "@/utils/advancePaymentHelpers";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";

const STATUS_CONFIG = {
  PENDING: {
    icon: Clock,
    pill: "border-amber-200 bg-amber-50 text-amber-700",
  },
  APPROVED: {
    icon: CheckCircle,
    pill: "border-emerald-200 bg-emerald-50 text-emerald-700",
  },
  COMPLETED: {
    icon: CheckCircle,
    pill: "border-emerald-200 bg-emerald-50 text-emerald-700",
  },
  FAILED: {
    icon: XCircle,
    pill: "border-red-200 bg-red-50 text-red-600",
  },
  CANCELLED: {
    icon: Ban,
    pill: "border-slate-200 bg-slate-100 text-slate-500",
  },
} as const;

const safeFormat = (value: number | null | undefined) =>
  value == null || isNaN(value) ? "—" : formatCurrency(value);

const safeDate = (date: string | null | undefined) => {
  if (!date) return "—";
  const parsedDate = new Date(date);
  return isNaN(parsedDate.getTime())
    ? "—"
    : format(parsedDate, "dd/MM/yyyy", { locale: vi });
};

function StatusPill({ item }: { item: AdvancePaymentHistoryItem }) {
  const config =
    STATUS_CONFIG[item.status as keyof typeof STATUS_CONFIG] ??
    STATUS_CONFIG.CANCELLED;
  const StatusIcon = config.icon;

  return (
    <span
      className={cn(
        "employee-type-pill inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-2.5",
        config.pill
      )}
    >
      <StatusIcon className="h-4 w-4" />
      {getVietnameseAdvancePaymentStatus(item.status)}
    </span>
  );
}

function CancelPendingAction({
  itemId,
  onCancel,
}: {
  itemId: number;
  onCancel?: (id: number) => void;
}) {
  const [showConfirmCancel, setShowConfirmCancel] = useState(false);
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const cancelButtonRef = useRef<HTMLButtonElement | null>(null);
  const dismissButtonRef = useRef<HTMLButtonElement | null>(null);
  const confirmationWasOpen = useRef(false);

  useEffect(() => {
    if (showConfirmCancel) {
      confirmationWasOpen.current = true;
      dismissButtonRef.current?.focus();
      return;
    }

    if (confirmationWasOpen.current) {
      confirmationWasOpen.current = false;
      cancelButtonRef.current?.focus();
    }
  }, [showConfirmCancel]);

  useEffect(
    () => () => {
      if (resetTimer.current) clearTimeout(resetTimer.current);
    },
    []
  );

  const handleCancelClick = () => {
    if (showConfirmCancel) {
      if (resetTimer.current) {
        clearTimeout(resetTimer.current);
        resetTimer.current = null;
      }
      onCancel?.(itemId);
      setShowConfirmCancel(false);
      return;
    }

    setShowConfirmCancel(true);
    resetTimer.current = setTimeout(() => {
      resetTimer.current = null;
      setShowConfirmCancel(false);
    }, 3000);
  };

  const handleDismissCancel = () => {
    if (resetTimer.current) {
      clearTimeout(resetTimer.current);
      resetTimer.current = null;
    }
    setShowConfirmCancel(false);
  };

  if (showConfirmCancel) {
    return (
      <div className="flex min-h-11 items-center justify-end gap-2">
        <button
          ref={dismissButtonRef}
          type="button"
          onClick={handleDismissCancel}
          className="employee-type-action min-h-11 rounded-full px-3 text-slate-500 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          Không
        </button>
        <button
          type="button"
          onClick={handleCancelClick}
          className="employee-type-action min-h-11 rounded-full bg-red-50 px-4 text-red-600 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
        >
          Xác nhận
        </button>
      </div>
    );
  }

  return (
    <button
      ref={cancelButtonRef}
      type="button"
      onClick={handleCancelClick}
      className="employee-type-action min-h-11 w-full rounded-xl border border-red-200 bg-red-50 text-red-600 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
    >
      Hủy yêu cầu
    </button>
  );
}

function HistoryItemRow({
  item,
  onCancel,
}: {
  item: AdvancePaymentHistoryItem;
  onCancel?: (id: number) => void;
}) {
  return (
    <article
      className={cn(
        "px-4 py-3",
        item.status === "PENDING" && "bg-amber-50/25"
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <StatusPill item={item} />
        <span className="employee-type-label text-slate-600">
          {safeDate(item.createdAt)}
        </span>
      </div>

      <div className="mt-2.5 flex items-end justify-between gap-4">
        <div className="min-w-0">
          <span className="employee-type-label block text-slate-500">
            Số tiền yêu cầu
          </span>
          <span className="employee-type-card-amount mt-1 block text-slate-950 tabular-nums">
            {safeFormat(item.requestAmount)}
          </span>
        </div>
        <div className="shrink-0 text-right">
          <span className="employee-type-label block text-slate-500">
            Thực nhận
          </span>
          <span className="employee-type-row-amount mt-1 block text-emerald-600 tabular-nums">
            {safeFormat(item.netAmount)}
          </span>
        </div>
      </div>

      <p className="employee-type-body-sm mt-2 text-slate-500">
        Phí giao dịch: <span className="font-semibold text-slate-700 tabular-nums">{safeFormat(item.fee)}</span>
      </p>

      {item.status === "PENDING" && (
        <div className="mt-3 border-t border-amber-100 pt-3">
          <CancelPendingAction itemId={item.id} onCancel={onCancel} />
        </div>
      )}
    </article>
  );
}

interface AdvancePaymentHistoryCardProps {
  history: AdvancePaymentHistoryItem[];
  isLoading: boolean;
  onCancel?: (id: number) => void;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentHistoryCard({
  history,
  isLoading,
  onCancel,
  className,
  style,
}: AdvancePaymentHistoryCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();
  const newestPendingRequest = history.find((item) => item.status === "PENDING");

  return (
    <div
      className={className ?? "overflow-hidden rounded-2xl bg-white"}
      style={style}
    >
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="flex min-h-14 w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-slate-50/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500"
      >
        <span className="flex min-w-0 items-center gap-2.5">
          <EmployeeIconFrame icon={History} size="row" tone="slate" />
          <span className="min-w-0">
            <span className="employee-type-hero-title block text-slate-950">
              Lịch sử yêu cầu
            </span>
            <span className="employee-type-body-sm mt-0.5 block text-slate-500">
              {isLoading
                ? "Đang tải lịch sử"
                : history.length > 0
                  ? `${history.length} yêu cầu`
                  : "Chưa có yêu cầu"}
            </span>
          </span>
        </span>
        <span className="flex shrink-0 items-center gap-2">
          {newestPendingRequest && (
            <span className="employee-type-pill rounded-full bg-amber-50 px-2.5 py-1.5 text-amber-700">
              Đang chờ
            </span>
          )}
          <ChevronDown
            className={cn(
              "h-5 w-5 text-slate-400 transition-transform duration-200",
              isOpen && "rotate-180"
            )}
          />
        </span>
      </button>

      {isOpen && (
        <div id={panelId} className="border-t border-slate-100 pb-3">
          {isLoading ? (
            <div className="space-y-3 px-4 pt-3">
              {[1, 2, 3].map((index) => (
                <Skeleton key={index} className="h-28 w-full rounded-xl" />
              ))}
            </div>
          ) : history.length === 0 ? (
            <div className="px-4 pb-7 pt-5 text-center">
              <img
                src="/advance-payment-empty-state.png"
                alt=""
                aria-hidden="true"
                loading="lazy"
                className="mx-auto mb-3 h-28 w-28 object-contain"
              />
              <p className="employee-type-card-title text-slate-700">
                Chưa có lần ứng lương nào
              </p>
              <p className="employee-type-body mt-1 text-slate-600">
                Khi bạn gửi yêu cầu, trạng thái sẽ hiện ở đây.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {history.map((item) => (
                <HistoryItemRow key={item.id} item={item} onCancel={onCancel} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
