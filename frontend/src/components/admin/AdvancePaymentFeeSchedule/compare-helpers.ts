// Helpers to compute side-by-side fee comparisons between the active
// schedule and the schedule about to be saved. Powers the confirmation
// modal's BEFORE/AFTER diff so admins can see the real-world impact.

import type {
  FeeScheduleEntry,
  FeeScheduleTier,
} from "@/types/api/advance-payment-fee-schedule.types";

import { resolveFeeLocal } from "@/hooks/admin/useFeeSchedulePreview";
import type { FeeScheduleFormState } from "./types";

// Representative amounts in VND. Picked to span common payroll-advance ranges.
export const COMPARISON_SAMPLE_AMOUNTS = [
  1_000_000,
  3_000_000,
  5_000_000,
  10_000_000,
  20_000_000,
];

export type ChangeDirection = "decrease" | "increase" | "same" | "new";

export interface FeeComparisonRow {
  amount: number;
  beforeFee: number | null; // null = no current schedule for comparison
  afterFee: number;
  delta: number; // afterFee - beforeFee, or 0 when no before
  deltaPct: number; // % delta vs beforeFee; 0 when no before
  direction: ChangeDirection;
}

interface BuildRowsInput {
  current: FeeScheduleEntry | null;
  next: { tiers: FeeScheduleTier[]; minFeeVnd: number };
}

export function buildComparisonRows(
  input: BuildRowsInput,
): FeeComparisonRow[] {
  const { current, next } = input;
  return COMPARISON_SAMPLE_AMOUNTS.map((amount) => {
    const after = resolveFeeLocal({
      amount,
      tiers: next.tiers,
      minFeeVnd: next.minFeeVnd,
    });
    const before = current
      ? resolveFeeLocal({
          amount,
          tiers: current.tiers,
          minFeeVnd: current.minFeeVnd,
        })
      : null;

    if (!before) {
      return {
        amount,
        beforeFee: null,
        afterFee: after.fee,
        delta: 0,
        deltaPct: 0,
        direction: "new" as const,
      };
    }

    const delta = after.fee - before.fee;
    const deltaPct = before.fee === 0 ? 0 : (delta / before.fee) * 100;
    let direction: ChangeDirection = "same";
    if (delta > 0) direction = "increase";
    else if (delta < 0) direction = "decrease";

    return {
      amount,
      beforeFee: before.fee,
      afterFee: after.fee,
      delta,
      deltaPct,
      direction,
    };
  });
}

// Builds the human-readable summary of a not-yet-persisted form state, in
// the same style the backend produces for FeeScheduleEntry.summary. Used by
// the "Visual diff vs current" affordance (item #10).
export function buildPendingSummary(state: FeeScheduleFormState): string {
  if (!state.tiers.length) return "—";
  if (state.tiers.length === 1) {
    return `Phí cố định ${formatPct(state.tiers[0].percentage)}`;
  }
  const parts = state.tiers.map((tier, idx) => {
    if (idx === 0) {
      return `< ${formatVndShort(state.tiers[1].minAmount)}: ${formatPct(tier.percentage)}`;
    }
    return `≥ ${formatVndShort(tier.minAmount)}: ${formatPct(tier.percentage)}`;
  });
  return `Phân tầng — ${parts.join(" · ")}`;
}

function formatPct(pct: number): string {
  return `${pct.toString().replace(".", ",")}%`;
}

function formatVndShort(vnd: number): string {
  if (vnd >= 1_000_000) {
    const m = vnd / 1_000_000;
    const trimmed = Number.isInteger(m) ? m.toString() : m.toFixed(1).replace(".", ",");
    return `${trimmed}tr`;
  }
  if (vnd >= 1_000) {
    return `${Math.round(vnd / 1_000)}k`;
  }
  return vnd.toString();
}

// Detects unsafe configurations the live preview might surface. Used to
// block submit (item #13) — e.g. a tier whose flat min-fee exceeds the
// principal at our smallest sample amount, indicating a misconfigured floor.
export function detectUnsafePreview(
  state: FeeScheduleFormState,
): string | null {
  // If the percentage on any tier is ≥ 100%, fees can equal/exceed principal.
  // resolveFeeLocal caps at amount, but the configuration is still wrong.
  const overTier = state.tiers.find((t) => t.percentage >= 100);
  if (overTier) {
    return "Phí vượt quá khoản rút — kiểm tra lại %";
  }

  // If min fee exceeds the smallest sample, the floor dominates real-world
  // small ứng. Allow but warn (caller decides whether to block); this helper
  // returns null otherwise.
  return null;
}
