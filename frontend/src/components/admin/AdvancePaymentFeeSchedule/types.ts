// Form-level types shared between FeeSchedulePage, FeeScheduleFormDialog, and
// the live preview panel. These aren't re-exporting the API types verbatim
// because the form keeps tier rows in a structure that allows blank user
// input (empty string) without forcing premature numeric coercion.

import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

export type FeeStructure = "flat" | "tiered";

export interface TierInput {
  minAmount: number; // 0 for the first tier; user-edited for the rest
  percentage: number;
}

export interface FeeScheduleFormState {
  effectiveDate: string;       // YYYY-MM-DD (required)
  structure: FeeStructure;     // controls tier UI; flat = exactly one tier
  tiers: TierInput[];          // always non-empty; tiers[0].minAmount = 0
  minFeeVnd: number;
  notes: string;
}

export type FeeScheduleFormMode =
  | { kind: "create" }
  | { kind: "edit"; entry: FeeScheduleEntry };
