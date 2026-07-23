import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { CalendarDays, ClipboardList, LoaderCircle } from "lucide-react";
import type { AggregatedDay } from "@/utils/employeePortal/timesheetGrouping";
import type { EmployeeMonth } from "@/hooks/useEmployeeMonth";
import { cn } from "@/lib/utils";
import { formatCurrency, formatNumber } from "@/utils/formatters";
import { getDayPaymentStatus } from "@/utils/employeePortal/paymentStatus";
import { EmployeeMonthNavigator } from "./EmployeeMonthNavigator";

interface EmployeeTimesheetPanelProps {
  month: EmployeeMonth;
  days: AggregatedDay[];
  totalRecords: number;
  bulkTransferPercentage: number;
  isLoading: boolean;
  isFetchingNextPage: boolean;
  observerRef: (node: HTMLElement | null) => void;
  className?: string;
}

const paymentStatusStyles = {
  full: {
    dot: "bg-success",
    label: "Đã trả",
    badge: "border-success/20 bg-success/10 text-success",
    amount: "text-success",
  },
  partial: {
    dot: "bg-warning",
    label: "Đã trả một phần",
    badge: "border-warning/20 bg-warning/10 text-warning",
    amount: "text-warning",
  },
  none: {
    dot: "bg-base-content/25",
    label: "Chưa trả",
    badge: "border-base-300 bg-base-200 text-base-content/55",
    amount: "text-base-content/40",
  },
} as const;

export function EmployeeTimesheetPanel({
  month,
  days,
  totalRecords,
  bulkTransferPercentage,
  isLoading,
  isFetchingNextPage,
  observerRef,
  className,
}: EmployeeTimesheetPanelProps) {
  return (
    <section id="employee-timesheets" className={cn("scroll-mt-4", className)} aria-labelledby="employee-timesheets-title">
      <EmployeeMonthNavigator month={month} className="mb-3" />

      <div className="ct-card employee-surface-card overflow-hidden rounded-[var(--employee-radius-feature)] bg-base-100">
        <div className="flex items-center justify-between gap-3 border-b border-base-300 px-4 py-4 sm:px-5">
          <div className="flex min-w-0 items-center gap-2.5">
            <span className="grid h-9 w-9 shrink-0 place-items-center rounded-[10px] border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]">
              <ClipboardList className="h-4.5 w-4.5" strokeWidth={1.9} aria-hidden="true" />
            </span>
            <h2 id="employee-timesheets-title" className="employee-type-card-title text-base-content">
              Bảng công
            </h2>
          </div>
          <span className="ct-badge ct-badge-ghost employee-type-pill h-auto shrink-0 px-2.5 py-1 text-base-content/60">
            {totalRecords} mục
          </span>
        </div>

        <div className="p-2.5 sm:p-4">
          {isLoading ? (
            <div className="space-y-3" aria-label="Đang tải bảng công">
              {[1, 2, 3].map((item) => <div key={item} className="h-28 animate-pulse rounded-[18px] bg-base-200" />)}
            </div>
          ) : days.length === 0 ? (
            <div className="relative isolate overflow-hidden rounded-[var(--employee-radius-card)] border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-5 py-7 text-center">
              <div className="absolute inset-0 -z-10 opacity-70 [background-image:radial-gradient(#b7e4ca_1px,transparent_1px)] [background-size:14px_14px]" />
              <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-[16px] bg-base-100 text-success ring-1 ring-inset ring-success/10">
                <CalendarDays className="h-6 w-6" aria-hidden="true" />
              </span>
              <p className="employee-type-card-title mt-3 text-base-content">Chưa có bảng công</p>
              <p className="employee-type-body-sm mx-auto mt-1 max-w-[18rem] text-base-content/50">
                Bảng công của bạn sẽ xuất hiện tại đây sau khi được ghi nhận.
              </p>
            </div>
          ) : (
            <div className="relative space-y-2.5 before:absolute before:bottom-4 before:left-[1.2rem] before:top-4 before:w-px before:bg-base-300">
              {days.map((day) => {
                const paymentStatus = getDayPaymentStatus(
                  day.totalAmount,
                  day.totalPaidAmount,
                  bulkTransferPercentage
                );
                const status = paymentStatusStyles[paymentStatus];

                return (
                  <article key={day.date} className="relative overflow-hidden rounded-[18px] border border-base-300 bg-base-100 transition-colors hover:border-base-content/20">
                    <div className="flex items-center justify-between gap-3 px-3.5 py-3 sm:px-4">
                      <div className="flex min-w-0 items-center gap-2.5">
                        <span className={cn("relative z-10 h-2.5 w-2.5 shrink-0 rounded-full ring-4 ring-base-100", status.dot)} aria-hidden="true" />
                        <span className="employee-type-row-amount truncate capitalize text-base-content">
                          {format(new Date(day.date), "EEEE, dd/MM", { locale: vi })}
                        </span>
                      </div>
                      {paymentStatus !== "full" && (
                        <span className={cn("ct-badge ct-badge-outline employee-type-pill h-auto shrink-0 px-2.5 py-1", status.badge)}>
                          {status.label}
                        </span>
                      )}
                    </div>

                    <div className="grid grid-cols-3 divide-x divide-base-300 border-t border-base-300 bg-base-200/55">
                      {[
                        {
                          label: "Giờ công",
                          value: `${day.totalHours % 1 === 0 ? day.totalHours : formatNumber(day.totalHours, 1)} giờ`,
                          color: "text-base-content",
                        },
                        { label: "Tổng lương", value: formatCurrency(day.totalAmount), color: "text-base-content" },
                        { label: "Đã nhận", value: formatCurrency(day.totalPaidAmount), color: status.amount },
                      ].map(({ label, value, color }) => (
                        <div key={label} className="min-w-0 px-2 py-2.5 text-center">
                          <p className="employee-type-label mb-1 text-base-content/40">{label}</p>
                          <p className={cn("employee-type-row-amount truncate tabular-nums", color)}>{value}</p>
                        </div>
                      ))}
                    </div>
                  </article>
                );
              })}
            </div>
          )}

          {isFetchingNextPage && (
            <div className="flex items-center justify-center gap-2 py-4 text-sm text-base-content/50">
              <LoaderCircle className="h-4 w-4 animate-spin text-success" aria-hidden="true" />
              Đang tải thêm...
            </div>
          )}
          <div ref={observerRef} className="h-3" />
        </div>
      </div>
    </section>
  );
}
