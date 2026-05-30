import { useMemo } from "react";
import { ArrowRight, TrendingDown, TrendingUp, Minus } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { formatCurrency } from "@/utils/formatters";
import {
  describeRuleFragments,
  formatDaysFromNow,
  formatVietnameseDate,
  type RuleFragment,
} from "./rule-formatters";
import { buildComparisonRows } from "./compare-helpers";
import { daysFromTodayISO } from "./form-helpers";
import type { FeeScheduleFormState } from "./types";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending: FeeScheduleFormState;
  current: FeeScheduleEntry | null;
  isSubmitting: boolean;
  isEdit: boolean;
  onConfirm: () => void;
}

const Fragments = ({ items }: { items: RuleFragment[] }) => (
  <span>
    {items.map((f, i) =>
      f.type === "emphasis" ? (
        <strong key={i} className="font-semibold">
          {f.value}
        </strong>
      ) : (
        <span key={i}>{f.value}</span>
      ),
    )}
  </span>
);

export const FeeScheduleConfirmDialog = ({
  open,
  onOpenChange,
  pending,
  current,
  isSubmitting,
  isEdit,
  onConfirm,
}: Props) => {
  const days = daysFromTodayISO(pending.effectiveDate);
  const fragments = useMemo(
    () => describeRuleFragments(pending.tiers, pending.minFeeVnd),
    [pending.tiers, pending.minFeeVnd],
  );
  const rows = useMemo(
    () =>
      buildComparisonRows({
        current,
        next: { tiers: pending.tiers, minFeeVnd: pending.minFeeVnd },
      }),
    [current, pending.tiers, pending.minFeeVnd],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Xác nhận cập nhật cấu hình" : "Xác nhận tạo cấu hình mới"}
          </DialogTitle>
          <DialogDescription>
            Vui lòng xem lại trước khi lưu. Sau khi lưu, mọi giao dịch ứng
            lương từ ngày hiệu lực sẽ áp dụng cấu hình này.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Pending summary */}
          <div className="rounded-lg border bg-muted/20 p-4 space-y-2">
            <p className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Cấu hình mới
            </p>
            <p className="text-sm leading-relaxed">
              <Fragments items={fragments} />
            </p>
            <p className="text-xs text-muted-foreground">
              Hiệu lực từ{" "}
              <strong className="font-medium text-foreground/90">
                {formatVietnameseDate(pending.effectiveDate)}
              </strong>{" "}
              ({formatDaysFromNow(days)})
            </p>
            {pending.notes.trim() && (
              <p className="text-xs text-muted-foreground italic">
                Ghi chú: {pending.notes.trim()}
              </p>
            )}
          </div>

          {/* Comparison */}
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">
              {current
                ? "So sánh phí trên một số khoản mẫu (Hiện tại → Mới):"
                : "Phí dự kiến trên một số khoản mẫu:"}
            </p>
            <div className="rounded-lg border overflow-x-auto">
              <table className="w-full text-xs">
                <thead className="bg-muted/40">
                  <tr>
                    <th className="text-left font-medium px-3 py-2">Khoản ứng</th>
                    {current && (
                      <th className="text-right font-medium px-3 py-2">
                        Hiện tại
                      </th>
                    )}
                    <th className="text-right font-medium px-3 py-2">Mới</th>
                    <th className="text-right font-medium px-3 py-2">
                      Thay đổi
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr
                      key={row.amount}
                      className="border-t border-border/60"
                    >
                      <td className="px-3 py-2 text-muted-foreground">
                        {formatCurrency(row.amount)}
                      </td>
                      {current && (
                        <td className="px-3 py-2 text-right tabular-nums">
                          {row.beforeFee !== null
                            ? formatCurrency(row.beforeFee)
                            : "—"}
                        </td>
                      )}
                      <td className="px-3 py-2 text-right tabular-nums font-medium">
                        {formatCurrency(row.afterFee)}
                      </td>
                      <td
                        className={cn(
                          "px-3 py-2 text-right text-xs whitespace-nowrap",
                          row.direction === "decrease" &&
                            "text-emerald-600 dark:text-emerald-400",
                          row.direction === "increase" &&
                            "text-red-600 dark:text-red-400",
                          row.direction === "same" && "text-muted-foreground",
                          row.direction === "new" && "text-muted-foreground",
                        )}
                      >
                        <span className="inline-flex items-center gap-1 justify-end">
                          {row.direction === "decrease" && (
                            <TrendingDown className="h-3 w-3" />
                          )}
                          {row.direction === "increase" && (
                            <TrendingUp className="h-3 w-3" />
                          )}
                          {row.direction === "same" && (
                            <Minus className="h-3 w-3" />
                          )}
                          {row.direction === "new" && (
                            <ArrowRight className="h-3 w-3" />
                          )}
                          {row.direction === "new"
                            ? "Mới"
                            : row.direction === "same"
                              ? "Không đổi"
                              : `${row.delta > 0 ? "+" : ""}${formatCurrency(row.delta)} (${
                                  row.deltaPct > 0 ? "+" : ""
                                }${row.deltaPct.toFixed(0)}%)`}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {current && (
              <p className="text-xs text-muted-foreground">
                <span className="text-emerald-600 dark:text-emerald-400">
                  Xanh
                </span>{" "}
                = phí giảm,{" "}
                <span className="text-red-600 dark:text-red-400">đỏ</span> =
                phí tăng so với cấu hình hiện tại.
              </p>
            )}
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            Quay lại sửa
          </Button>
          <Button
            type="button"
            onClick={onConfirm}
            disabled={isSubmitting}
          >
            {isSubmitting ? "Đang lưu..." : "Xác nhận lưu"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
