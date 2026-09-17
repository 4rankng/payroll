import { useMemo } from "react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Check, Circle, ClipboardList, LoaderCircle } from "lucide-react";
import type { AggregatedDay } from "@/utils/employeePortal/timesheetGrouping";
import type { EmployeeMonth } from "@/hooks/useEmployeeMonth";
import { cn } from "@/lib/utils";
import { formatCurrency, formatNumber } from "@/utils/formatters";
import { getDayPaymentStatus } from "@/utils/employeePortal/paymentStatus";
import { EmployeeMonthNavigator } from "./EmployeeMonthNavigator";
import { EmployeeDataError } from "./EmployeeDataError";

interface EmployeeTimesheetPanelProps {
  month: EmployeeMonth;
  days: AggregatedDay[];
  bulkTransferPercentage: number;
  isLoading: boolean;
  isError: boolean;
  isRetrying: boolean;
  onRetry: () => void;
  isFetchingNextPage: boolean;
  observerRef: (node: HTMLElement | null) => void;
  className?: string;
}

const paymentStatusStyles = {
  full: {
    rail: "bg-[var(--employee-accent)]",
    text: "text-[var(--employee-accent)]",
    icon: Check,
    label: "Đã trả đủ",
  },
  partial: {
    rail: "bg-[var(--employee-warning)]",
    text: "text-[var(--employee-warning-strong)]",
    icon: Circle,
    label: (amount: string) => `Đã trả ${amount}`,
  },
  none: {
    rail: "bg-[var(--employee-border-strong)]",
    text: "text-[var(--employee-text-secondary)]",
    icon: Circle,
    label: "Chưa trả",
  },
} as const;

export function EmployeeTimesheetPanel({
  month,
  days,
  bulkTransferPercentage,
  isLoading,
  isError,
  isRetrying,
  onRetry,
  isFetchingNextPage,
  observerRef,
  className,
}: EmployeeTimesheetPanelProps) {
  const totalHours = useMemo(
    () => days.reduce((sum, day) => sum + day.totalHours, 0),
    [days]
  );

  return (
    <section id="employee-timesheets" className={cn("scroll-mt-4", className)} aria-labelledby="employee-timesheets-title">
      <EmployeeMonthNavigator month={month} className="mb-4" />

      <div className="mb-2.5 flex flex-wrap items-baseline justify-between gap-3">
        <h2 id="employee-timesheets-title" className="employee-type-card-title flex items-center gap-2 text-[var(--employee-text)]">
          <ClipboardList className="h-4 w-4 text-[var(--employee-accent)]" strokeWidth={2} aria-hidden="true" />
          Bảng công
        </h2>
        {!isError && days.length > 0 && (
          <span className="employee-type-body-sm shrink-0 text-[var(--employee-text-secondary)]">
            {days.length} ngày &middot; {totalHours % 1 === 0 ? totalHours : formatNumber(totalHours, 1)} giờ
          </span>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2" aria-label="Đang tải bảng công">
          {[1, 2, 3].map((item) => <div key={item} className="h-16 animate-pulse rounded-[14px] bg-base-200" />)}
        </div>
      ) : isError ? (
        <EmployeeDataError title="Chưa tải được bảng công" onRetry={onRetry} isRetrying={isRetrying} />
      ) : days.length === 0 ? (
        <div className="relative isolate overflow-hidden rounded-[var(--employee-radius-card)] border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-5 py-7 text-center">
          <div className="absolute inset-0 -z-10 opacity-70 [background-image:radial-gradient(#b7e4ca_1px,transparent_1px)] [background-size:14px_14px]" />
          <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-[16px] bg-base-100 text-[var(--employee-accent)] ring-1 ring-inset ring-[var(--employee-accent-border)]">
            <ClipboardList className="h-6 w-6" aria-hidden="true" />
          </span>
          <p className="employee-type-card-title mt-3 text-base-content">Chưa có bảng công</p>
          <p className="employee-type-body-sm mx-auto mt-1 max-w-[18rem] text-base-content/50">
            Bảng công của bạn sẽ xuất hiện tại đây sau khi được ghi nhận.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {days.map((day) => {
            const paymentStatus = getDayPaymentStatus(
              day.totalAmount,
              day.totalPaidAmount,
              bulkTransferPercentage
            );
            const status = paymentStatusStyles[paymentStatus];
            const StatusIcon = status.icon;
            const statusLabel =
              typeof status.label === "function"
                ? status.label(formatCurrency(day.totalPaidAmount))
                : status.label;
            const hoursLabel = `${day.totalHours % 1 === 0 ? day.totalHours : formatNumber(day.totalHours, 1)} giờ`;

            return (
              <article
                key={day.date}
                className="flex overflow-hidden rounded-[13px] border border-[var(--employee-border)] bg-[var(--employee-surface)] shadow-[var(--employee-shadow)] transition-transform active:scale-[0.99]"
              >
                <span className={cn("w-1 shrink-0", status.rail)} aria-hidden="true" />
                <div className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-3 px-3.5 py-3">
                  <div className="min-w-0">
                    <p className="employee-type-row-amount truncate capitalize text-[var(--employee-text)]">
                      {format(new Date(day.date), "EEEE, dd/MM", { locale: vi })}
                    </p>
                    <p className="employee-type-pill mt-0.5 truncate text-[var(--employee-text-secondary)]">
                      {hoursLabel}
                    </p>
                  </div>
                  <div className="ml-auto min-w-0 text-right">
                    <p className="employee-type-row-amount break-words tabular-nums text-[var(--employee-text)]">
                      {formatCurrency(day.totalAmount)}
                    </p>
                    <p className={cn("mt-0.5 flex items-center justify-end gap-1 employee-type-pill", status.text)}>
                      <StatusIcon className="h-3 w-3 shrink-0" strokeWidth={2.5} aria-hidden="true" />
                      {statusLabel}
                    </p>
                  </div>
                </div>
              </article>
            );
          })}
        </div>
      )}

      {!isError && isFetchingNextPage && (
        <div className="flex items-center justify-center gap-2 py-4 text-sm text-base-content/50">
          <LoaderCircle className="h-4 w-4 animate-spin text-[var(--employee-accent)]" aria-hidden="true" />
          Đang tải thêm...
        </div>
      )}
      <div ref={observerRef} className="h-3" />
    </section>
  );
}
