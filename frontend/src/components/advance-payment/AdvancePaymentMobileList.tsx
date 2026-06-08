import { memo, useCallback, useState, useEffect, useRef } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Wallet, Clock, Banknote } from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { formatCurrency } from "@/utils/formatters";
import {
  getVietnameseAdvancePaymentStatus,
  getAdvancePaymentStatusColor,
} from "@/utils/advancePaymentHelpers";
import { cn } from "@/lib/utils";
import { AccentStripCard, type AccentColor } from "@/components/shared/AccentStripCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { MobilePagination, type PaginationInfo } from "@/components/shared/MobilePagination";
import type {
  AdvancePaymentListItem,
  AdvancePaymentStatus,
} from "@/types/api/advance-payment.types";

const statusAccentColorMap: Record<AdvancePaymentStatus, AccentColor> = {
  PENDING: "amber",
  APPROVED: "blue",
  COMPLETED: "green",
  FAILED: "red",
  CANCELLED: "gray",
};

const RequestCard = memo(function RequestCard({
  item,
  onCancel,
  isCancelling,
  onRetry,
  isRetrying,
  isPolling,
}: {
  item: AdvancePaymentListItem;
  onCancel: (id: number) => void;
  isCancelling: boolean;
  onRetry?: (id: number) => void;
  isRetrying?: boolean;
  isPolling?: boolean;
}) {
  const isCancellable = item.status === "PENDING" || item.status === "APPROVED";
  const isRetryable = item.status === "APPROVED" || item.status === "FAILED";
  const [showConfirm, setShowConfirm] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-dismiss confirm prompt after 3 s of inaction
  useEffect(() => {
    if (showConfirm) {
      timeoutRef.current = setTimeout(() => setShowConfirm(false), 3000);
    }
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, [showConfirm]);

  const handleCancelClick = useCallback(() => {
    if (showConfirm) {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      setShowConfirm(false);
      onCancel(item.id);
    } else {
      setShowConfirm(true);
    }
  }, [showConfirm, onCancel, item.id]);

  const handleDismiss = useCallback(() => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    setShowConfirm(false);
  }, []);

  return (
    <AccentStripCard accentColor={statusAccentColorMap[item.status] ?? "gray"}>
      <div className="px-4 pt-3.5 pb-3">
        <div className="flex items-start justify-between gap-2 mb-2">
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-foreground truncate">
              {item.employeeName}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {item.employeeCCCD}
            </p>
          </div>
          <div className="flex items-center gap-1.5">
            {isPolling && (
              <span className="h-3.5 w-3.5 shrink-0 animate-spin rounded-full border-2 border-blue-300 border-t-blue-600" />
            )}
            <Badge
              variant="outline"
              className={cn(
                getAdvancePaymentStatusColor(item.status),
                "shrink-0 text-xs",
              )}
            >
              {getVietnameseAdvancePaymentStatus(item.status)}
            </Badge>
          </div>
        </div>
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1.5">
            <Banknote className="h-3 w-3 text-muted-foreground shrink-0" />
            <span className="font-semibold text-foreground tabular-nums">
              {formatCurrency(item.requestAmount)}
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Thực nhận:</span>
            <span className="font-semibold text-emerald-700 tabular-nums">
              {formatCurrency(item.netAmount)}
            </span>
          </div>
        </div>
        <div className="flex items-center justify-between mt-2">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3 w-3 shrink-0" />
            <span className="tabular-nums">
              {format(new Date(item.createdAt), "dd/MM/yyyy", { locale: vi })}
            </span>
          </div>
          {isCancellable && (
            showConfirm ? (
              <div className="flex items-center gap-2">
                <button
                  onClick={handleDismiss}
                  className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors px-1"
                >
                  Không
                </button>
                <Button
                  variant="destructive"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={handleCancelClick}
                  disabled={isCancelling}
                >
                  {isCancelling ? "Đang hủy..." : "Xác nhận hủy"}
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-1.5">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={handleCancelClick}
                  disabled={isCancelling}
                >
                  Hủy
                </Button>
                {isRetryable && onRetry && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2 text-xs text-blue-600 hover:text-blue-700 hover:bg-blue-50"
                    onClick={() => onRetry(item.id)}
                    disabled={isRetrying}
                  >
                    {isRetrying ? "Đang gửi..." : "Thử lại"}
                  </Button>
                )}
              </div>
            )
          )}
          {!isCancellable && isRetryable && onRetry && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-blue-600 hover:text-blue-700 hover:bg-blue-50"
              onClick={() => onRetry(item.id)}
              disabled={isRetrying}
            >
              {isRetrying ? "Đang gửi..." : "Thử lại"}
            </Button>
          )}
        </div>
      </div>
    </AccentStripCard>
  );
});

interface AdvancePaymentMobileListProps {
  requests: AdvancePaymentListItem[];
  isCancelling: boolean;
  cancellingId?: number;
  onCancel: (id: number) => void;
  onRetry?: (id: number) => void;
  isRetrying?: boolean;
  retryingId?: number;
  pollingIds?: Set<number>;
  pagination?: PaginationInfo | null;
  onPageChange?: (page: number) => void;
  /** @deprecated — confirm state is now managed locally inside each card */
  cancelConfirmId?: number | null;
}

export function AdvancePaymentMobileList({
  requests,
  isCancelling,
  cancellingId,
  onCancel,
  onRetry,
  isRetrying,
  retryingId,
  pollingIds,
  pagination,
  onPageChange,
}: AdvancePaymentMobileListProps) {
  if (requests.length === 0) {
    return (
      <EmptyState
        icon={Wallet}
        title="Chưa có yêu cầu nào"
        description="Yêu cầu ứng lương sẽ hiển thị tại đây"
      />
    );
  }

  return (
    <>
      <div className="space-y-2.5">
        {requests.map((item) => (
          <RequestCard
            key={item.id}
            item={item}
            onCancel={onCancel}
            isCancelling={isCancelling && cancellingId === item.id}
            onRetry={onRetry}
            isRetrying={isRetrying && retryingId === item.id}
            isPolling={pollingIds?.has(item.id)}
          />
        ))}
      </div>

      {pagination && onPageChange && (
        <MobilePagination pagination={pagination} onPageChange={onPageChange} />
      )}
    </>
  );
}
