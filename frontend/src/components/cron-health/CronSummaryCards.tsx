// W13c: lucide → verified @untitledui/icons; Power → Power01.
import { Activity, AlertCircle, Power01 } from "@untitledui/icons";
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
      <div className="console-panel px-3 py-2.5 flex items-center gap-2 border-utility-gray-200">
        <Activity className="h-4 w-4 text-primary shrink-0" />
        <span className="text-base font-bold text-foreground">{total}</span>
        <span className="text-xs text-muted-foreground">Tổng</span>
      </div>
      <div className="console-panel px-3 py-2.5 flex items-center gap-2 border-utility-success-100">
        <Power01 className="h-4 w-4 text-fg-success-primary shrink-0" />
        <span className="text-base font-bold text-fg-success-primary">{enabled}</span>
        <span className="text-xs text-muted-foreground">Bật</span>
      </div>
      {failed > 0 ? (
        <div className="console-panel px-3 py-2.5 flex items-center gap-2 border-utility-error-100 bg-utility-error-50">
          <AlertCircle className="h-4 w-4 text-utility-error-700 shrink-0" />
          <span className="text-base font-bold text-utility-error-700">{failed}</span>
          <span className="text-xs text-fg-error-primary">Lỗi</span>
        </div>
      ) : (
        <div className="console-panel px-3 py-2.5 flex items-center gap-2 border-utility-gray-200">
          <Power01 className="h-4 w-4 text-utility-gray-500 shrink-0" />
          <span className="text-base font-bold text-utility-gray-600">{disabled}</span>
          <span className="text-xs text-muted-foreground">Tắt</span>
        </div>
      )}
    </div>
  );
}
