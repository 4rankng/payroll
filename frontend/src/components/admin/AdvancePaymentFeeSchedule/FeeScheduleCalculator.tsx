import { useMemo, useState } from "react";
import { Calculator } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { formatVndDigits } from "@/components/settings/SettingCard";
import { resolveFeeLocal } from "@/hooks/admin/useFeeSchedulePreview";
import { formatCurrency, formatDate } from "@/utils/formatters";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  entries: FeeScheduleEntry[];
}

function pickEntryForDate(
  entries: FeeScheduleEntry[],
  iso: string,
): FeeScheduleEntry | null {
  // Newest entry whose effectiveDate <= iso.
  const eligible = entries
    .filter((e) => e.effectiveDate <= iso)
    .sort((a, b) => (a.effectiveDate < b.effectiveDate ? 1 : -1));
  return eligible[0] ?? null;
}

function pickNextEntryAfter(
  entries: FeeScheduleEntry[],
  iso: string,
): FeeScheduleEntry | null {
  const upcoming = entries
    .filter((e) => e.effectiveDate > iso)
    .sort((a, b) => (a.effectiveDate < b.effectiveDate ? -1 : 1));
  return upcoming[0] ?? null;
}

export const FeeScheduleCalculator = ({ entries }: Props) => {
  const today = useMemo(() => {
    const d = new Date();
    return [
      d.getFullYear(),
      String(d.getMonth() + 1).padStart(2, "0"),
      String(d.getDate()).padStart(2, "0"),
    ].join("-");
  }, []);

  const [amount, setAmount] = useState<number>(5_000_000);
  const [date, setDate] = useState<string>(today);

  const activeEntry = useMemo(
    () => pickEntryForDate(entries, date),
    [entries, date],
  );
  const nextEntry = useMemo(
    () => pickNextEntryAfter(entries, date),
    [entries, date],
  );

  const result = useMemo(() => {
    if (!activeEntry) return null;
    return resolveFeeLocal({
      amount,
      tiers: activeEntry.tiers,
      minFeeVnd: activeEntry.minFeeVnd,
    });
  }, [activeEntry, amount]);

  const futureResult = useMemo(() => {
    if (!nextEntry) return null;
    return resolveFeeLocal({
      amount,
      tiers: nextEntry.tiers,
      minFeeVnd: nextEntry.minFeeVnd,
    });
  }, [nextEntry, amount]);

  return (
    <div
      className="rounded-xl border bg-card p-4 space-y-3"
      aria-label="Máy tính phí thử"
    >
      <div className="flex items-center gap-2">
        <Calculator className="h-4 w-4 text-muted-foreground" />
        <h3 className="text-sm font-semibold">Tính thử phí</h3>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label htmlFor="calc-amount" className="text-xs text-muted-foreground">
            Số tiền ứng (₫)
          </Label>
          <Input
            id="calc-amount"
            type="text"
            inputMode="numeric"
            autoComplete="off"
            value={formatVndDigits(String(amount))}
            onChange={(e) => {
              const digits = e.target.value.replace(/\D/g, '');
              setAmount(digits ? Number(digits) : 0);
            }}
            className="h-12 text-base font-semibold tabular-nums"
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="calc-date" className="text-xs text-muted-foreground">
            Ngày giao dịch
          </Label>
          <Input
            id="calc-date"
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
          />
        </div>
      </div>

      <div className="rounded-md bg-muted/40 p-3 space-y-2 text-sm">
        {!activeEntry && (
          <p className="text-muted-foreground">
            Chưa có cấu hình áp dụng cho ngày này.
          </p>
        )}
        {activeEntry && result && (
          <>
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-muted-foreground text-xs">Phí áp dụng</span>
              <span className="font-semibold">{formatCurrency(result.fee)}</span>
            </div>
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-muted-foreground text-xs">Thực nhận</span>
              <span>{formatCurrency(result.netAmount)}</span>
            </div>
            <div className="flex items-baseline justify-between gap-2 pt-1.5 border-t border-border/60 text-xs text-muted-foreground">
              <span>Bậc áp dụng</span>
              <span>
                {result.appliedTier?.percentage}% · phí tối thiểu{" "}
                {formatCurrency(activeEntry.minFeeVnd)}
              </span>
            </div>
            <div className="text-xs text-muted-foreground">
              Cấu hình hiệu lực từ {formatDate(activeEntry.effectiveDate)}
            </div>
          </>
        )}
      </div>

      {nextEntry && futureResult && (
        <div className="rounded-md border border-dashed border-border/70 p-3 text-xs text-muted-foreground space-y-1">
          <p className="font-medium text-foreground/80">
            Sau {formatDate(nextEntry.effectiveDate)}:
          </p>
          <p>
            Phí sẽ là {formatCurrency(futureResult.fee)} (
            {futureResult.appliedTier?.percentage}%)
          </p>
        </div>
      )}
    </div>
  );
};
