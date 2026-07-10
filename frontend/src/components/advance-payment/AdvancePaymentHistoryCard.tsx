import { useEffect, useId, useRef, useState } from "react";
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
  PENDING: { icon: Clock, text: "text-[#B54708]" },
  APPROVED: { icon: CheckCircle, text: "text-[#067647]" },
  COMPLETED: { icon: CheckCircle, text: "text-[#067647]" },
  FAILED: { icon: XCircle, text: "text-[#B42318]" },
  CANCELLED: { icon: Ban, text: "text-[#475467]" },
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

function CancelPendingAction({ itemId, onCancel }: { itemId: number; onCancel?: (id: number) => void }) {
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

  useEffect(() => () => {
    if (resetTimer.current) clearTimeout(resetTimer.current);
  }, []);

  const handleCancelClick = () => {
    if (showConfirmCancel) {
      if (resetTimer.current) clearTimeout(resetTimer.current);
      resetTimer.current = null;
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
    if (resetTimer.current) clearTimeout(resetTimer.current);
    resetTimer.current = null;
    setShowConfirmCancel(false);
  };

  if (showConfirmCancel) {
    return (
      <div className="flex min-h-11 items-center justify-end gap-2">
        <button ref={dismissButtonRef} type="button" onClick={handleDismissCancel} className="employee-type-action min-h-11 rounded-[10px] px-3 text-[#475467] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#667085]">
          Không
        </button>
        <button type="button" onClick={handleCancelClick} className="employee-type-action min-h-11 rounded-[10px] bg-[#FEF3F2] px-4 text-[#B42318] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#D92D20]">
          Xác nhận
        </button>
      </div>
    );
  }

  return (
    <button ref={cancelButtonRef} type="button" onClick={handleCancelClick} className="employee-type-action min-h-11 w-full rounded-[10px] border border-[#FDA29B] bg-white text-[#B42318] transition-colors hover:bg-[#FEF3F2] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#D92D20]">
      Hủy yêu cầu
    </button>
  );
}

function HistoryItem({ item, onCancel }: { item: AdvancePaymentHistoryItem; onCancel?: (id: number) => void }) {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();
  const config = STATUS_CONFIG[item.status] ?? STATUS_CONFIG.CANCELLED;
  const StatusIcon = config.icon;

  return (
    <article className="overflow-hidden rounded-xl border border-[#E4E7EC] bg-white">
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="w-full px-4 py-3 text-left transition-colors hover:bg-[#F9FAFB] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[#07883F]"
      >
        <span className="flex items-center justify-between gap-3">
          <span className={cn("employee-type-label inline-flex min-w-0 items-center gap-1.5 font-semibold", config.text)}>
            <StatusIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="truncate">{getVietnameseAdvancePaymentStatus(item.status)}</span>
          </span>
          <span className="employee-type-label shrink-0 text-[#667085] tabular-nums">{safeDate(item.createdAt)}</span>
        </span>

        <span className="mt-3 grid grid-cols-2 gap-4">
          <span className="min-w-0">
            <span className="employee-type-label block text-[#667085]">Số tiền yêu cầu</span>
            <span className="employee-type-card-amount mt-1 block break-words text-[#101828] tabular-nums">{safeFormat(item.requestAmount)}</span>
          </span>
          <span className="min-w-0 text-right">
            <span className="employee-type-label block text-[#667085]">Thực nhận</span>
            <span className="employee-type-card-amount mt-1 block break-words text-[#067647] tabular-nums">{safeFormat(item.netAmount)}</span>
          </span>
        </span>

        <span className="employee-type-body-sm mt-2.5 flex items-center justify-between gap-3 text-[#667085]">
          <span>{isOpen ? "Thu gọn chi tiết" : "Xem phí và chi tiết"}</span>
          <ChevronDown className={cn("h-5 w-5 shrink-0 transition-transform", isOpen && "rotate-180")} aria-hidden="true" />
        </span>
      </button>

      {isOpen && (
        <div id={panelId} className="border-t border-[#EAECF0] bg-[#F9FAFB] px-4 py-3">
          <div className="flex items-center justify-between gap-4">
            <span className="employee-type-label text-[#667085]">Phí giao dịch</span>
            <span className="employee-type-row-amount text-[#344054] tabular-nums">{safeFormat(item.fee)}</span>
          </div>
          {item.status === "PENDING" && (
            <div className="mt-3 border-t border-[#EAECF0] pt-3">
              <CancelPendingAction itemId={item.id} onCancel={onCancel} />
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
  onCancel?: (id: number) => void;
  monthLabel?: string;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentHistoryCard({
  history,
  isLoading,
  isError = false,
  onRetry,
  onCancel,
  monthLabel,
  className,
  style,
}: AdvancePaymentHistoryCardProps) {
  return (
    <div className={className} style={style} aria-labelledby="employee-advance-history-title">
      <div className="mb-3 flex items-end justify-between gap-3 px-0.5">
        <div>
          <h2 id="employee-advance-history-title" className="employee-type-section-title text-[#101828]">Lịch sử yêu cầu</h2>
          <p className="employee-type-body-sm mt-0.5 text-[#667085]">Theo dõi kết quả ứng lương của bạn</p>
        </div>
        {!isLoading && !isError && (
          <span className="employee-type-label shrink-0 text-[#475467]">
            {monthLabel ? `Kỳ ${monthLabel} · ` : ""}{history.length} yêu cầu
          </span>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2.5" aria-label="Đang tải lịch sử yêu cầu">
          {[1, 2].map((index) => <Skeleton key={index} className="h-32 w-full rounded-xl" />)}
        </div>
      ) : isError ? (
        <div className="rounded-xl border border-[#E4E7EC] bg-white px-4 py-5 text-center" role="alert">
          <p className="employee-type-strong text-[#101828]">Chưa tải được lịch sử</p>
          <p className="employee-type-body-sm mt-1 text-[#667085]">Kiểm tra kết nối rồi thử lại.</p>
          {onRetry && (
            <button type="button" onClick={onRetry} className="employee-type-action mt-3 inline-flex min-h-11 items-center gap-2 rounded-[10px] border border-[#D0D5DD] px-4 text-[#344054] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#07883F]">
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Tải lại
            </button>
          )}
        </div>
      ) : history.length === 0 ? (
        <div className="rounded-xl border border-dashed border-[#D0D5DD] bg-white px-4 py-6 text-center">
          <p className="employee-type-strong text-[#344054]">Chưa có yêu cầu trong tháng {monthLabel}</p>
          <p className="employee-type-body-sm mt-1 text-[#667085]">Yêu cầu mới sẽ xuất hiện tại đây.</p>
        </div>
      ) : (
        <div className="space-y-2.5">
          {history.map((item) => <HistoryItem key={item.id} item={item} onCancel={onCancel} />)}
        </div>
      )}
    </div>
  );
}
