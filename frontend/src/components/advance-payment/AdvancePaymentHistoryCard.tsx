import { useState } from "react";
import {
  Ban,
  CheckCircle,
  Clock,
  History,
  XCircle,
} from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Skeleton } from "@/components/ui/skeleton";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import { cn } from "@/lib/utils";
import { formatCurrency } from "@/utils/formatters";
import { getVietnameseAdvancePaymentStatus } from "@/utils/advancePaymentHelpers";
import type {
  AdvancePaymentHistoryItem,
  AdvancePaymentStatus,
} from "@/types/api/advance-payment.types";

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

const safeFormat = (v: number | null | undefined) =>
  v == null || isNaN(v) ? "—" : formatCurrency(v);

const safeDate = (d: string | null | undefined) => {
  if (!d) return "—";
  const dt = new Date(d);
  return isNaN(dt.getTime()) ? "—" : format(dt, "dd/MM/yyyy", { locale: vi });
};

// Internal sub-component
const HistoryItemRow = ({
  item,
  onCancel,
}: {
  item: AdvancePaymentHistoryItem;
  onCancel?: (id: number) => void;
}) => {
  const [showConfirmCancel, setShowConfirmCancel] = useState(false);
  const cfg =
    STATUS_CONFIG[item.status as keyof typeof STATUS_CONFIG] ??
    STATUS_CONFIG.CANCELLED;
  const StatusIcon = cfg.icon;

  const handleCancelClick = () => {
    if (showConfirmCancel) {
      onCancel?.(item.id);
      setShowConfirmCancel(false);
    } else {
      setShowConfirmCancel(true);
      setTimeout(() => setShowConfirmCancel(false), 3000);
    }
  };

  return (
    <article
      className={cn(
        "px-4 py-3",
        item.status === "PENDING" && "bg-amber-50/25"
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <span
          className={cn(
            "employee-type-pill inline-flex min-h-8 shrink-0 items-center gap-1.5 rounded-full border px-2.5",
            cfg.pill
          )}
        >
          <StatusIcon className="h-4 w-4" />
          {getVietnameseAdvancePaymentStatus(item.status)}
        </span>
        <span className="employee-type-label text-slate-400">
          {safeDate(item.createdAt)}
        </span>
      </div>

      <div className="mt-2.5 flex items-end justify-between gap-4">
        <span className="employee-type-label-caps pb-0.5 text-slate-500">
          Yêu cầu
        </span>
        <span className="employee-type-card-amount shrink-0 text-slate-950 tabular-nums">
          {safeFormat(item.requestAmount)}
        </span>
      </div>

      <div className="mt-2.5 flex items-center justify-between gap-4 border-t border-slate-100 pt-2.5">
        <div className="min-w-0">
          <span className="employee-type-label block text-slate-500">
            Thực nhận
          </span>
          <span className="employee-type-row-amount mt-0.5 block whitespace-nowrap text-emerald-600 tabular-nums">
            {safeFormat(item.netAmount)}
          </span>
        </div>
        <div className="shrink-0 text-right">
          <span className="employee-type-label block text-slate-500">
            Phí giao dịch
          </span>
          <span className="employee-type-row-amount mt-0.5 block text-slate-700 tabular-nums">
            {safeFormat(item.fee)}
          </span>
        </div>
      </div>

      {item.status === "PENDING" && (
        <div className="mt-3 border-t border-amber-100 pt-3">
          {showConfirmCancel ? (
            <div className="flex items-center justify-end gap-2">
              <button
                onClick={() => setShowConfirmCancel(false)}
                className="employee-type-action min-h-11 rounded-full px-3 text-slate-500 transition-colors hover:text-slate-700"
              >
                Không
              </button>
              <button
                onClick={handleCancelClick}
                className="employee-type-action min-h-11 rounded-full bg-red-50 px-4 text-red-600 transition-colors hover:bg-red-100"
              >
                Xác nhận
              </button>
            </div>
          ) : (
            <button
              onClick={handleCancelClick}
              className="employee-type-action min-h-11 w-full rounded-2xl border border-red-200 bg-red-50 text-red-600 transition-colors hover:bg-red-100"
            >
              Hủy
            </button>
          )}
        </div>
      )}
    </article>
  );
};

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
  return (
    <div
      className={className ?? "bg-white rounded-2xl overflow-hidden"}
      style={style}
    >
      <div className="flex items-center justify-between gap-3 px-4 pb-2 pt-4">
        <div className="flex items-center gap-2">
          <EmployeeIconFrame icon={History} size="row" tone="slate" />
          <h2 className="employee-type-hero-title text-slate-950">
            Lịch sử yêu cầu
          </h2>
        </div>
        {history.length > 0 && (
          <span className="employee-type-pill rounded-full bg-slate-100 px-3 py-1.5 text-slate-500">
            {history.length} yêu cầu
          </span>
        )}
      </div>

      <div className="pb-3">
        {isLoading ? (
          <div className="space-y-3 px-4 pt-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-28 w-full rounded-[22px]" />
            ))}
          </div>
        ) : history.length === 0 ? (
          <div className="px-4 pb-9 pt-6 text-center">
            <img
              src="/advance-payment-empty-state.png"
              alt=""
              aria-hidden="true"
              loading="lazy"
              className="mx-auto mb-3 h-32 w-32 object-contain"
            />
            <p className="employee-type-card-title text-gray-500">
              Chưa có lần ứng lương nào
            </p>
            <p className="employee-type-body mt-1 text-gray-400">
              Khi bạn gửi yêu cầu, trạng thái sẽ hiện ở đây.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-slate-100 border-t border-slate-100">
            {history.map((item) => (
              <HistoryItemRow
                key={item.id}
                item={item}
                onCancel={onCancel}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
