import { useMemo, useState } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { useFeeSchedulePreview } from "@/hooks/admin/useFeeSchedulePreview";
import { formatCurrency } from "@/utils/formatters";
import type { FeeScheduleTier } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  tiers: FeeScheduleTier[];
  minFeeVnd: number;
}

const PRESET_AMOUNTS = [500_000, 2_000_000, 5_000_000, 10_000_000];

export const FeeScheduleLivePreview = ({ tiers, minFeeVnd }: Props) => {
  const [customAmount, setCustomAmount] = useState<number>(5_000_000);

  const previewInput = useMemo(
    () => ({ amount: customAmount, tiers, minFeeVnd }),
    [customAmount, tiers, minFeeVnd],
  );
  const preview = useFeeSchedulePreview(previewInput);

  const presetRows = useMemo(
    () =>
      PRESET_AMOUNTS.map((amount) => {
        const result = (() => {
          if (tiers.length === 0) return null;
          let applied = tiers[0];
          for (const t of tiers) {
            if (t.minAmount <= amount) applied = t;
            else break;
          }
          const pct = Math.floor((amount * applied.percentage) / 100);
          const fee = Math.max(pct, minFeeVnd);
          return { amount, fee, percentage: applied.percentage };
        })();
        return { amount, result };
      }),
    [tiers, minFeeVnd],
  );

  return (
    <aside
      className="rounded-xl border bg-muted/30 p-4 space-y-4 self-start sticky top-2"
      aria-label="Xem trước phí"
    >
      <div className="space-y-1">
        <h3 className="text-sm font-semibold">Xem trước phí</h3>
        <p className="text-xs text-muted-foreground">
          Tính theo cấu hình hiện tại trên form
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="preview-amount" className="text-xs">
          Số tiền ứng (₫)
        </Label>
        <Input
          id="preview-amount"
          type="number"
          min={0}
          step={100_000}
          value={customAmount}
          onChange={(e) => setCustomAmount(Number(e.target.value) || 0)}
        />
        <div className="rounded-md bg-background border p-3 space-y-1.5 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Phí</span>
            <span className="font-medium">{formatCurrency(preview.fee)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Thực nhận</span>
            <span className="font-semibold">
              {formatCurrency(preview.netAmount)}
            </span>
          </div>
          {preview.appliedTier && (
            <div className="flex justify-between text-xs text-muted-foreground pt-1 border-t">
              <span>Bậc áp dụng</span>
              <span>{preview.appliedTier.percentage}%</span>
            </div>
          )}
        </div>
      </div>

      <div className="space-y-1.5">
        <p className="text-xs font-medium text-muted-foreground">Mức tham khảo</p>
        <ul className="space-y-1 text-xs">
          {presetRows.map(({ amount, result }) => (
            <li key={amount} className="flex justify-between gap-2">
              <span className="text-muted-foreground">
                {formatCurrency(amount)}
              </span>
              <span>
                {result
                  ? `${formatCurrency(result.fee)} (${result.percentage}%)`
                  : "—"}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </aside>
  );
};
