import { useState } from "react";
import { Clock, CheckCircle, XCircle, Ban, History, ChevronRight } from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Skeleton } from "@/components/ui/skeleton";
import { formatCurrency } from "@/utils/formatters";
import { getVietnameseAdvancePaymentStatus } from "@/utils/advancePaymentHelpers";
import type {
  AdvancePaymentHistoryItem,
  AdvancePaymentStatus,
} from "@/types/api/advance-payment.types";

const STATUS_CONFIG = {
  PENDING: {
    icon: Clock,
    pill: "bg-amber-50 text-amber-600 border-amber-200",
  },
  APPROVED: {
    icon: CheckCircle,
    pill: "bg-green-50 text-green-700 border-green-200",
  },
  COMPLETED: {
    icon: CheckCircle,
    pill: "bg-green-50 text-green-700 border-green-200",
  },
  FAILED: { icon: XCircle, pill: "bg-red-50 text-red-600 border-red-200" },
  CANCELLED: {
    icon: Ban,
    pill: "bg-gray-100 text-gray-500 border-gray-200",
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
const HistoryItemCard = ({
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
    <div className="flex items-center gap-3 py-3 px-1">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-0.5">
          <span className="text-sm font-bold text-gray-900 tabular-nums">
            {safeFormat(item.requestAmount)}
          </span>
          <span
            className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold border ${cfg.pill}`}
          >
            <StatusIcon className="h-2.5 w-2.5" />
            {getVietnameseAdvancePaymentStatus(item.status)}
          </span>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-gray-500 flex-wrap">
          <span>{safeDate(item.createdAt)}</span>
          <span className="text-gray-300">·</span>
          <span>
            Nhận{" "}
            <span className="font-semibold text-gray-700">
              {safeFormat(item.netAmount)}
            </span>
          </span>
          <span className="text-gray-300">·</span>
          <span>Phí {safeFormat(item.fee)}</span>
        </div>
      </div>
      {item.status === "PENDING" && (
        <div className="shrink-0">
          {showConfirmCancel ? (
            <div className="flex items-center gap-2">
              <button
                onClick={() => setShowConfirmCancel(false)}
                className="text-xs font-medium text-gray-500 hover:text-gray-700 transition-colors"
              >
                Không
              </button>
              <button
                onClick={handleCancelClick}
                className="text-xs font-bold text-red-500 hover:text-red-700 transition-colors"
              >
                Xác nhận
              </button>
            </div>
          ) : (
            <button
              onClick={handleCancelClick}
              className="text-xs font-semibold text-red-500 hover:text-red-600 border border-red-200 bg-red-50 hover:bg-red-100 px-2.5 py-1 rounded-lg transition-colors"
            >
              Hủy
            </button>
          )}
        </div>
      )}
    </div>
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
      <div className="px-4 pt-3.5 pb-1 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <History className="h-3.5 w-3.5 text-gray-400" />
          <h2 className="text-sm font-bold text-gray-800">
            Lịch sử yêu cầu
          </h2>
        </div>
        {history.length > 0 && (
          <span className="text-xs text-gray-400 flex items-center gap-0.5">
            {history.length} yêu cầu <ChevronRight className="h-3 w-3" />
          </span>
        )}
      </div>

      <div className="px-4 pb-3">
        {isLoading ? (
          <div className="space-y-3 pt-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-12 w-full rounded-xl" />
            ))}
          </div>
        ) : history.length === 0 ? (
          <div className="text-center py-8">
            <div className="w-10 h-10 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-2.5">
              <History className="h-4 w-4 text-gray-300" />
            </div>
            <p className="text-sm font-medium text-gray-400">
              Chưa có yêu cầu nào
            </p>
            <p className="text-xs text-gray-300 mt-0.5">
              Các yêu cầu ứng lương sẽ hiển thị ở đây
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {history.map((item) => (
              <HistoryItemCard
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
