import { useMemo } from "react";
import { AlertCircle, CalendarDays, CheckCircle2, Clock3 } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
import { EmployeePayIllustration } from "@/components/employees/EmployeePayIllustration";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import type { AdvancePaymentInfo, AdvancePaymentQuota } from "@/types/api/advance-payment.types";

interface AdvancePaymentLimitCardProps {
  info: AdvancePaymentInfo;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentLimitCard({
  info,
  className,
  style,
}: AdvancePaymentLimitCardProps) {
  // Use quotas array if available and not empty, otherwise fallback to top-level aggregate
  const hasQuotas = info.quotas && info.quotas.length > 0;

  // Sort quotas in FIFO order (oldest first) based on forMonth string comparison (YYYY-MM)
  const sortedQuotas = useMemo(
    () => hasQuotas ? [...info.quotas!].sort((a, b) => a.forMonth.localeCompare(b.forMonth)) : [],
    [hasQuotas, info.quotas]
  );
  const quotasToDisplay = hasQuotas
    ? sortedQuotas
    : [{
        forMonth: info.forMonth,
        maxAdvanceAmount: info.maxAdvanceAmount,
        completedAmount: info.completedAmount,
        pendingAmount: info.pendingAmount,
        remainingAmount: info.remainingAmount,
      }];
  const showQuotaRemaining = quotasToDisplay.length > 1;

  return (
    <div className={className ?? "bg-white rounded-2xl overflow-hidden"} style={style}>
      <div className="space-y-4 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-[12px] font-semibold uppercase leading-4 tracking-wide text-slate-500">
              Hạn mức ứng lương
            </p>
            <h2 className="mt-1 text-[18px] font-bold leading-7 text-slate-950">
              Còn ứng được
            </h2>
          </div>
          <EmployeePayIllustration size="sm" />
        </div>

        <div className="rounded-2xl border border-employee/10 bg-employee/5 px-4 py-3">
          <p className="text-[32px] font-bold leading-tight tracking-tight text-employee tabular-nums">
            {formatCurrency(info.remainingAmount)}
          </p>
        </div>

        {info.salary !== undefined && (
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div className="rounded-lg bg-slate-50 px-3 py-2">
              <p className="text-[12px] leading-5 text-slate-500">Tiền công đã tính</p>
              <p className="mt-0.5 truncate text-[15px] font-semibold leading-6 text-slate-800 tabular-nums">
                {formatCurrency(info.salary)}
              </p>
            </div>
            <div className="rounded-lg bg-employee/5 px-3 py-2">
              <p className="text-[12px] leading-5 text-slate-500">Hạn mức tối đa</p>
              <p className="mt-0.5 truncate text-[15px] font-semibold leading-6 text-slate-800 tabular-nums">
                {formatCurrency(info.maxAdvanceAmount)}
              </p>
            </div>
          </div>
        )}
        {info.disclaimer && (
          <p className="text-[14px] leading-6 text-slate-500">
            {info.disclaimer}
          </p>
        )}
      </div>

      <div className="space-y-3 border-t border-slate-100 bg-slate-50/70 p-4">
        {quotasToDisplay.map((quota, idx) => (
          <QuotaSection
            key={quota.forMonth || idx}
            quota={quota}
            isOldest={idx === 0 && quotasToDisplay.length > 1}
            showRemaining={showQuotaRemaining}
          />
        ))}
      </div>
    </div>
  );
}

function QuotaSection({
  quota,
  isOldest,
  showRemaining,
}: {
  quota: AdvancePaymentQuota;
  isOldest?: boolean;
  showRemaining?: boolean;
}) {
  const max = quota.maxAdvanceAmount;
  const completedPct = max > 0 ? Math.min((quota.completedAmount / max) * 100, 100) : 0;
  const pendingPct = max > 0 ? Math.min((quota.pendingAmount / max) * 100, 100 - completedPct) : 0;
  const usedAmount = quota.completedAmount + quota.pendingAmount;
  const usedPct = max > 0 ? Math.min((usedAmount / max) * 100, 100) : 0;

  return (
    <div className="relative rounded-2xl border border-slate-100 bg-white p-3.5 shadow-[0_1px_0_rgba(15,23,42,0.03)]">
      {isOldest && (
        <div className="absolute -top-2.5 -right-2.5 flex items-center gap-1 rounded-full border border-amber-200 bg-amber-100 px-2 py-0.5 text-[13px] font-bold text-amber-700 shadow-sm">
          <AlertCircle className="h-3 w-3" />
          Sắp hết hạn
        </div>
      )}

      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <EmployeeIconFrame icon={CalendarDays} size="row" tone="slate" />
          <div className="min-w-0">
            <p className="text-[13px] font-medium leading-5 text-slate-500">Kỳ lương</p>
            <p className="text-[18px] font-bold leading-7 text-slate-900">
              {quota.forMonth ? `${quota.forMonth.slice(5, 7)}/${quota.forMonth.slice(2, 4)}` : "—"}
            </p>
          </div>
        </div>
        {showRemaining && (
          <div className="text-right">
            <p className="text-[13px] leading-5 text-slate-500">Còn lại</p>
            <p className="text-[17px] font-bold leading-6 text-employee tabular-nums">
              {formatCurrency(quota.remainingAmount)}
            </p>
          </div>
        )}
      </div>

      {usedAmount > 0 && (
        <>
          <div className="mb-2 mt-3 flex items-center justify-between gap-3">
            <span className="text-[13px] font-medium leading-5 text-slate-500">
              Đã dùng {Math.round(usedPct)}%
            </span>
          </div>
          <div className="mb-3 flex h-2 overflow-hidden rounded-full bg-slate-200">
            <div
              className="h-full bg-employee transition-all duration-500"
              style={{ width: `${completedPct}%` }}
            />
            <div
              className="h-full transition-all duration-500"
              style={{ width: `${pendingPct}%`, backgroundColor: "hsl(var(--warning))" }}
            />
          </div>
        </>
      )}

      <div className="mt-3 grid grid-cols-2 gap-2">
        <div className="rounded-xl bg-slate-50 px-3 py-2">
          <div className="flex items-center gap-1.5 text-[13px] leading-5 text-slate-500">
            <CheckCircle2 className="h-4 w-4 shrink-0 text-employee" aria-hidden="true" />
            Đã nhận
          </div>
          <p className="mt-0.5 text-[16px] font-bold leading-6 text-slate-950 tabular-nums">
            {formatCurrency(quota.completedAmount)}
          </p>
        </div>
        <div className="rounded-xl bg-amber-50/70 px-3 py-2">
          <div className="flex items-center gap-1.5 text-[13px] leading-5 text-amber-700">
            <Clock3 className="h-4 w-4 shrink-0 text-amber-500" aria-hidden="true" />
            Đang chờ
          </div>
          <p className="mt-0.5 text-[16px] font-bold leading-6 text-slate-950 tabular-nums">
            {formatCurrency(quota.pendingAmount)}
          </p>
        </div>
      </div>
    </div>
  );
}
