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
}

const paymentStatusStyles = {
  full: {
    dot: "bg-emerald-500",
    label: "Đã trả",
    badge: "bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200",
    amount: "text-emerald-700",
  },
  partial: {
    dot: "bg-amber-400",
    label: "Đã trả một phần",
    badge: "bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-200",
    amount: "text-amber-700",
  },
  none: {
    dot: "bg-slate-300",
    label: "Chưa trả",
    badge: "bg-slate-100 text-slate-600 ring-1 ring-inset ring-slate-200",
    amount: "text-slate-400",
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
}: EmployeeTimesheetPanelProps) {
  return (
    <section id="employee-timesheets" className="scroll-mt-4" aria-labelledby="employee-timesheets-title">
      <EmployeeMonthNavigator month={month} className="mb-3" />

      <div className="overflow-hidden rounded-[20px] border border-slate-200 bg-white shadow-[var(--employee-shadow)]">
        <div className="flex items-center justify-between gap-3 border-b border-slate-100 bg-[linear-gradient(100deg,#ffffff_0%,#f5fbf7_100%)] px-4 py-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-[14px] bg-employee/10 text-employee">
              <ClipboardList className="h-5 w-5" aria-hidden="true" />
            </span>
            <span className="min-w-0">
              <span className="employee-type-label-caps block text-employee">Bảng công</span>
              <h2 id="employee-timesheets-title" className="employee-type-card-title mt-0.5 text-slate-950">
                Nhật ký công việc
              </h2>
            </span>
          </div>
          <span className="employee-type-pill shrink-0 rounded-full bg-white px-2.5 py-1 text-slate-600 ring-1 ring-inset ring-slate-200">
            {totalRecords} mục
          </span>
        </div>

        <div className="p-3">
          {isLoading ? (
            <div className="space-y-3" aria-label="Đang tải bảng công">
              {[1, 2, 3].map((item) => <div key={item} className="h-28 animate-pulse rounded-[18px] bg-slate-100" />)}
            </div>
          ) : days.length === 0 ? (
            <div className="relative isolate overflow-hidden rounded-[18px] border border-[#dbeee3] bg-[#f8fcf9] px-5 py-7 text-center">
              <div className="absolute inset-0 -z-10 opacity-70 [background-image:radial-gradient(#b7e4ca_1px,transparent_1px)] [background-size:14px_14px]" />
              <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-[16px] bg-white text-employee shadow-sm ring-1 ring-inset ring-employee/10">
                <CalendarDays className="h-6 w-6" aria-hidden="true" />
              </span>
              <p className="employee-type-card-title mt-3 text-slate-950">Chưa có bảng công</p>
              <p className="employee-type-body-sm mx-auto mt-1 max-w-[18rem] text-slate-500">
                Bảng công của bạn sẽ xuất hiện tại đây sau khi được ghi nhận.
              </p>
            </div>
          ) : (
            <div className="space-y-2.5">
              {days.map((day) => {
                const paymentStatus = getDayPaymentStatus(
                  day.totalAmount,
                  day.totalPaidAmount,
                  bulkTransferPercentage
                );
                const status = paymentStatusStyles[paymentStatus];

                return (
                  <article key={day.date} className="overflow-hidden rounded-[18px] border border-slate-200 bg-white">
                    <div className="flex items-center justify-between gap-3 px-3.5 py-3">
                      <div className="flex min-w-0 items-center gap-2.5">
                        <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", status.dot)} aria-hidden="true" />
                        <span className="employee-type-row-amount truncate capitalize text-slate-900">
                          {format(new Date(day.date), "EEEE, dd/MM", { locale: vi })}
                        </span>
                      </div>
                      <span className={cn("employee-type-pill shrink-0 rounded-full px-2.5 py-1", status.badge)}>
                        {status.label}
                      </span>
                    </div>

                    <div className="grid grid-cols-3 divide-x divide-slate-100 border-t border-slate-100 bg-slate-50/60">
                      {[
                        {
                          label: "Giờ công",
                          value: `${day.totalHours % 1 === 0 ? day.totalHours : formatNumber(day.totalHours, 1)} giờ`,
                          color: "text-slate-900",
                        },
                        { label: "Tổng lương", value: formatCurrency(day.totalAmount), color: "text-slate-900" },
                        { label: "Đã nhận", value: formatCurrency(day.totalPaidAmount), color: status.amount },
                      ].map(({ label, value, color }) => (
                        <div key={label} className="min-w-0 px-2 py-2.5 text-center">
                          <p className="employee-type-label mb-1 text-slate-400">{label}</p>
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
            <div className="flex items-center justify-center gap-2 py-4 text-sm text-slate-500">
              <LoaderCircle className="h-4 w-4 animate-spin text-employee" aria-hidden="true" />
              Đang tải thêm...
            </div>
          )}
          <div ref={observerRef} className="h-3" />
        </div>
      </div>
    </section>
  );
}
