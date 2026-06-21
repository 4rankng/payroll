import { useMemo } from "react";
import { TrendingUp, AlertCircle } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
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

  return (
    <div className={className ?? "bg-white rounded-2xl overflow-hidden"} style={style}>
      <div className="p-4 border-b border-gray-100">
        <div className="flex items-center gap-1.5 mb-1">
          <TrendingUp
            className="h-4 w-4 text-employee"
          />
          <span className="font-semibold text-gray-800 uppercase tracking-wide text-sm">
            Hạn mức ứng lương
          </span>
        </div>
        <p className="text-xs text-gray-500">
          Tổng có thể ứng: <strong className="text-employee">{formatCurrency(info.remainingAmount)}</strong>
        </p>
      </div>

      {info.salary !== undefined && (
        <div className="px-4 py-3 bg-emerald-50/60 border-b border-gray-100 space-y-1.5">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">Tiền công thực tế</span>
            <span className="font-semibold tabular-nums text-gray-800">
              {formatCurrency(info.salary)}
            </span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600">
              Tiền công được ứng{" "}
              <span className="text-[10px] text-gray-400">(70%)</span>
            </span>
            <span className="font-bold tabular-nums text-employee">
              {formatCurrency(info.maxAdvanceAmount)}
            </span>
          </div>
          {info.disclaimer && (
            <p className="text-[11px] text-gray-500 leading-snug pt-1.5 mt-1 border-t border-emerald-100">
              {info.disclaimer}
            </p>
          )}
        </div>
      )}

      <div className="p-4 space-y-4">
        {hasQuotas ? (
          sortedQuotas.map((quota, idx) => (
            <QuotaSection key={quota.forMonth} quota={quota} isOldest={idx === 0 && sortedQuotas.length > 1} />
          ))
        ) : (
          <QuotaSection 
            quota={{
              forMonth: info.forMonth,
              maxAdvanceAmount: info.maxAdvanceAmount,
              completedAmount: info.completedAmount,
              pendingAmount: info.pendingAmount,
              remainingAmount: info.remainingAmount
            }} 
          />
        )}
      </div>
    </div>
  );
}

function QuotaSection({ quota, isOldest }: { quota: AdvancePaymentQuota, isOldest?: boolean }) {
  const max = quota.maxAdvanceAmount;
  const completedPct = max > 0 ? Math.min((quota.completedAmount / max) * 100, 100) : 0;
  const pendingPct = max > 0 ? Math.min((quota.pendingAmount / max) * 100, 100 - completedPct) : 0;

  return (
    <div className="bg-gray-50 rounded-xl p-3 border border-gray-100 relative">
      {isOldest && (
        <div className="absolute -top-2.5 -right-2.5 bg-amber-100 text-amber-700 text-[10px] font-bold px-2 py-0.5 rounded-full flex items-center gap-1 shadow-sm border border-amber-200">
          <AlertCircle className="h-3 w-3" />
          Sắp hết hạn
        </div>
      )}
      
      <div className="flex justify-between items-center mb-3">
        <span className="text-xs font-semibold text-gray-700 bg-white px-2 py-1 rounded-md shadow-sm border border-gray-200">
          Kỳ lương: {quota.forMonth ? `${quota.forMonth.slice(5, 7)}/${quota.forMonth.slice(2, 4)}` : ""}
        </span>
        <div className="text-right">
          <span className="text-[10px] text-gray-400 block leading-tight">Còn lại</span>
          <span className="text-sm font-bold tabular-nums text-employee">
            {formatCurrency(quota.remainingAmount)}
          </span>
        </div>
      </div>

      <div className="h-1.5 bg-gray-200 rounded-full overflow-hidden mb-3 flex">
        <div
          className="h-full bg-employee transition-all duration-500"
          style={{ width: `${completedPct}%` }}
        />
        <div
          className="h-full transition-all duration-500"
          style={{ width: `${pendingPct}%`, background: "#F59E0B" }}
        />
      </div>

      <div className="flex justify-between text-[11px]">
        <div className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-employee" />
          <span className="text-gray-500">Đã ứng: <span className="font-semibold text-gray-700">{formatCurrency(quota.completedAmount)}</span></span>
        </div>
        <div className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
          <span className="text-gray-500">Đang chờ: <span className="font-semibold text-gray-700">{formatCurrency(quota.pendingAmount)}</span></span>
        </div>
      </div>
    </div>
  );
}
