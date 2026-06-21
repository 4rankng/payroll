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
      <div className="border-b border-gray-100 p-4">
        <div className="mb-1 flex items-center gap-2">
          <TrendingUp
            className="h-4 w-4 text-employee"
          />
          <span className="text-[18px] font-bold leading-6 text-slate-900">
            Có thể ứng ngay
          </span>
        </div>
        <p className="text-[16px] leading-6 text-slate-600">
          Số tiền còn ứng được: <strong className="font-bold text-employee">{formatCurrency(info.remainingAmount)}</strong>
        </p>
      </div>

      {info.salary !== undefined && (
        <div className="px-4 py-3 bg-emerald-50/60 border-b border-gray-100 space-y-1.5">
          <div className="flex items-center justify-between text-[16px]">
            <span className="text-gray-600">Tiền công đã tính</span>
            <span className="font-semibold tabular-nums text-gray-800">
              {formatCurrency(info.salary)}
            </span>
          </div>
          <div className="flex items-center justify-between text-[16px]">
            <span className="text-gray-600">
              Được ứng tối đa{" "}
              <span className="text-[15px] text-gray-400">(70%)</span>
            </span>
            <span className="font-bold tabular-nums text-employee">
              {formatCurrency(info.maxAdvanceAmount)}
            </span>
          </div>
          {info.disclaimer && (
            <p className="mt-1 border-t border-emerald-100 pt-1.5 text-[15px] leading-5 text-gray-500">
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
    <div className="relative rounded-xl border border-gray-100 bg-gray-50 p-3.5">
      {isOldest && (
        <div className="absolute -top-2.5 -right-2.5 flex items-center gap-1 rounded-full border border-amber-200 bg-amber-100 px-2 py-0.5 text-[13px] font-bold text-amber-700 shadow-sm">
          <AlertCircle className="h-3 w-3" />
          Sắp hết hạn
        </div>
      )}
      
      <div className="flex justify-between items-center mb-3">
        <span className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-[16px] font-bold text-gray-700 shadow-sm">
          Kỳ lương: {quota.forMonth ? `${quota.forMonth.slice(5, 7)}/${quota.forMonth.slice(2, 4)}` : ""}
        </span>
        <div className="text-right">
          <span className="block text-[15px] leading-5 text-gray-500">Còn ứng được</span>
          <span className="text-[18px] font-bold tabular-nums text-employee">
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

      <div className="flex justify-between gap-3 text-[15px]">
        <div className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-employee" />
          <span className="text-gray-600">Đã nhận: <span className="font-bold text-gray-800">{formatCurrency(quota.completedAmount)}</span></span>
        </div>
        <div className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
          <span className="text-gray-600">Đang chờ: <span className="font-bold text-gray-800">{formatCurrency(quota.pendingAmount)}</span></span>
        </div>
      </div>
    </div>
  );
}
