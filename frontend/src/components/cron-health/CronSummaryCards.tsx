import { Activity, Power, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export interface CronSummaryCardsProps {
  total: number;
  enabled: number;
  failed: number;
}

export function CronSummaryCards({ total, enabled, failed }: CronSummaryCardsProps) {
  const disabled = total - enabled;
  return (
    <div className="grid grid-cols-3 gap-2">
      <div className="bg-white rounded-xl border border-gray-200 px-3 py-2.5 flex items-center gap-2 shadow-sm">
        <Activity className="h-4 w-4 text-primary shrink-0" />
        <span className="text-base font-bold text-foreground">{total}</span>
        <span className="text-xs text-muted-foreground">Tổng</span>
      </div>
      <div className="bg-white rounded-xl border border-emerald-100 px-3 py-2.5 flex items-center gap-2 shadow-sm">
        <Power className="h-4 w-4 text-emerald-700 shrink-0" />
        <span className="text-base font-bold text-emerald-700">{enabled}</span>
        <span className="text-xs text-muted-foreground">Bật</span>
      </div>
      {failed > 0 ? (
        <div className="bg-red-50 rounded-xl border border-red-100 px-3 py-2.5 flex items-center gap-2 shadow-sm">
          <AlertCircle className="h-4 w-4 text-red-600 shrink-0" />
          <span className="text-base font-bold text-red-600">{failed}</span>
          <span className="text-xs text-red-700">Lỗi</span>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 px-3 py-2.5 flex items-center gap-2 shadow-sm">
          <Power className="h-4 w-4 text-gray-300 shrink-0" />
          <span className="text-base font-bold text-gray-600">{disabled}</span>
          <span className="text-xs text-muted-foreground">Tắt</span>
        </div>
      )}
    </div>
  );
}
