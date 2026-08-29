// Plain-Vietnamese formatters for fee-schedule rules. Used by the
// "currently active" and "upcoming" summary cards so admins can read the
// rule at a glance without parsing the schedule table.

import type {
  FeeScheduleEntry,
  FeeScheduleTier,
} from "@/types/api/advance-payment-fee-schedule.types";

import { formatCurrency } from "@/utils/formatters";

// "2" → "2%", "1.3" → "1,3%" (Vietnamese decimal mark).
export function formatPercentVi(percentage: number): string {
  const fixed = Number.isInteger(percentage)
    ? percentage.toString()
    : percentage.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
  return `${fixed.replace(".", ",")}%`;
}

export interface RuleFragment {
  type: "text" | "emphasis";
  value: string;
}

// Plain Vietnamese rendering of the fee rule, returned as fragments so the
// component can bold the numerical parts. The rendered string is the same
// across both representations — a renderer either joins the fragments or
// applies <strong> on emphasis fragments.
export function describeRuleFragments(
  tiers: FeeScheduleTier[],
  minFeeVnd: number,
): RuleFragment[] {
  if (!tiers.length) {
    return [{ type: "text", value: "Không có bậc nào" }];
  }

  if (tiers.length === 1) {
    // minFeeVnd = 0 means no floor (e.g. a fully free schedule) — omit the
    // misleading "mức tối thiểu 0 ₫" clause entirely.
    if (minFeeVnd === 0) {
      return [{ type: "emphasis", value: formatPercentVi(tiers[0].percentage) }];
    }
    return [
      { type: "emphasis", value: formatPercentVi(tiers[0].percentage) },
      { type: "text", value: " với mức tối thiểu " },
      { type: "emphasis", value: formatCurrency(minFeeVnd) },
    ];
  }

  // Tiered: render each tier in plain Vietnamese.
  const parts: RuleFragment[] = [];
  tiers.forEach((tier, idx) => {
    if (idx > 0) parts.push({ type: "text", value: " — " });
    parts.push({ type: "emphasis", value: formatPercentVi(tier.percentage) });

    if (idx === 0 && tiers.length > 1) {
      parts.push({ type: "text", value: " cho khoản dưới " });
      parts.push({
        type: "emphasis",
        value: formatCurrency(tiers[1].minAmount),
      });
    } else if (idx === tiers.length - 1) {
      parts.push({ type: "text", value: " cho khoản từ " });
      parts.push({
        type: "emphasis",
        value: formatCurrency(tier.minAmount),
      });
      parts.push({ type: "text", value: " trở lên" });
    } else {
      parts.push({ type: "text", value: " từ " });
      parts.push({
        type: "emphasis",
        value: formatCurrency(tier.minAmount),
      });
      parts.push({ type: "text", value: " đến " });
      parts.push({
        type: "emphasis",
        value: formatCurrency(tiers[idx + 1].minAmount),
      });
    }
  });

  // minFeeVnd = 0 means no floor — the "mức tối thiểu 0 ₫" clause would be
  // misleading, so omit it (same as the single-tier branch above).
  if (minFeeVnd === 0) {
    return parts;
  }

  parts.push({ type: "text", value: " (mức tối thiểu " });
  parts.push({ type: "emphasis", value: formatCurrency(minFeeVnd) });
  parts.push({ type: "text", value: ")" });
  return parts;
}

export function describeEntryFragments(entry: FeeScheduleEntry): RuleFragment[] {
  return describeRuleFragments(entry.tiers, entry.minFeeVnd);
}

// Heuristic: a single-flat 2% with min 10k VND created on/before backend
// install is the bootstrap config. We can't be sure without metadata, so
// only flag when the entry was created without a user (createdByUserId
// undefined) AND the values match the historical bootstrap defaults.
export function isLikelyBootstrap(entry: FeeScheduleEntry): boolean {
  if (entry.createdByUserId !== undefined) return false;
  if (entry.tiers.length !== 1) return false;
  const t = entry.tiers[0];
  return t.minAmount === 0 && t.percentage === 2 && entry.minFeeVnd === 10_000;
}

// Format the date range in Vietnamese, e.g. "Từ 01/01/2020 đến nay".
export function formatActiveRange(
  start: string,
  next: FeeScheduleEntry | null,
): string {
  const startDate = formatVietnameseDate(start);
  if (!next) return `Từ ${startDate} đến nay`;
  return `Từ ${startDate} đến ${formatVietnameseDate(next.effectiveDate)}`;
}

export function formatVietnameseDate(iso: string): string {
  const d = new Date(iso);
  const day = String(d.getDate()).padStart(2, "0");
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const year = d.getFullYear();
  return `${day}/${month}/${year}`;
}

export function formatDaysFromNow(days: number): string {
  if (days === 0) return "hôm nay";
  if (days === 1) return "ngày mai";
  if (days > 1) return `sau ${days} ngày`;
  if (days === -1) return "hôm qua";
  return `${Math.abs(days)} ngày trước`;
}
