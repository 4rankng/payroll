// Pure helpers for the fee-schedule form. Kept in a .ts file so the dialog
// .tsx stays focused on UI; logic here is unit-testable in isolation.

import type {
  CreateFeeScheduleRequest,
  FeeScheduleEntry,
} from "@/types/api/advance-payment-fee-schedule.types";

import type {
  FeeScheduleFormMode,
  FeeScheduleFormState,
  TierInput,
} from "./types";

export const DEFAULT_MIN_FEE_VND = 10_000;

export function todayISO(): string {
  const d = new Date();
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, "0"),
    String(d.getDate()).padStart(2, "0"),
  ].join("-");
}

export function tomorrowISO(): string {
  const d = new Date();
  d.setDate(d.getDate() + 1);
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, "0"),
    String(d.getDate()).padStart(2, "0"),
  ].join("-");
}

export function buildInitialFormState(
  mode: FeeScheduleFormMode,
): FeeScheduleFormState {
  if (mode.kind === "edit") {
    const e = mode.entry;
    return {
      effectiveDate: e.effectiveDate,
      structure: e.tiers.length > 1 ? "tiered" : "flat",
      tiers: e.tiers.map((t) => ({
        minAmount: t.minAmount,
        percentage: t.percentage,
      })),
      minFeeVnd: e.minFeeVnd,
      notes: e.notes ?? "",
    };
  }
  return {
    effectiveDate: tomorrowISO(),
    structure: "flat",
    tiers: [{ minAmount: 0, percentage: 2 }],
    minFeeVnd: DEFAULT_MIN_FEE_VND,
    notes: "",
  };
}

export function setStructure(
  state: FeeScheduleFormState,
  structure: "flat" | "tiered",
): FeeScheduleFormState {
  if (state.structure === structure) return state;
  if (structure === "flat") {
    return {
      ...state,
      structure,
      tiers: [state.tiers[0] ?? { minAmount: 0, percentage: 2 }],
    };
  }
  // Switching from flat to tiered seeds a sensible second tier.
  return {
    ...state,
    structure,
    tiers:
      state.tiers.length >= 2
        ? state.tiers
        : [
            state.tiers[0] ?? { minAmount: 0, percentage: 2 },
            { minAmount: 3_500_000, percentage: 1.3 },
          ],
  };
}

export function addTier(state: FeeScheduleFormState): FeeScheduleFormState {
  const last = state.tiers[state.tiers.length - 1];
  return {
    ...state,
    tiers: [
      ...state.tiers,
      {
        minAmount: Math.max(0, last.minAmount) + 1_000_000,
        percentage: Math.max(0, last.percentage - 0.5),
      },
    ],
  };
}

export function removeTier(
  state: FeeScheduleFormState,
  index: number,
): FeeScheduleFormState {
  // Cannot remove the first (min_amount = 0) tier; it's a backend invariant.
  if (index === 0 || index >= state.tiers.length) return state;
  return { ...state, tiers: state.tiers.filter((_, i) => i !== index) };
}

export function updateTier(
  state: FeeScheduleFormState,
  index: number,
  patch: Partial<TierInput>,
): FeeScheduleFormState {
  return {
    ...state,
    tiers: state.tiers.map((t, i) => (i === index ? { ...t, ...patch } : t)),
  };
}

export type FormValidationError = string;

export function validateFormState(
  state: FeeScheduleFormState,
): FormValidationError | null {
  if (!state.effectiveDate) {
    return "Vui lòng chọn ngày hiệu lực";
  }
  if (state.effectiveDate < todayISO()) {
    return "Ngày hiệu lực phải là hôm nay hoặc tương lai";
  }
  if (state.tiers.length === 0) {
    return "Cần ít nhất một bậc phí";
  }
  if (state.tiers[0].minAmount !== 0) {
    return "Bậc đầu tiên phải có giá trị từ 0";
  }
  let prev = -1;
  for (const tier of state.tiers) {
    if (tier.percentage < 0 || tier.percentage > 100) {
      return "Phần trăm phí phải nằm trong khoảng 0–100";
    }
    if (tier.minAmount <= prev) {
      return "Các bậc phải tăng dần theo số tiền tối thiểu";
    }
    prev = tier.minAmount;
  }
  // minFeeVnd = 0 is a deliberate admin choice (no minimum-fee floor, e.g. a
  // fully free advance schedule). Negative input is rejected here because a
  // typed minus sign passes <input type="number" min={0}>, and the backend's
  // uint64 field would fail with an opaque binding error.
  if (state.minFeeVnd < 0) {
    return "Phí tối thiểu không được âm";
  }
  return null;
}

// Per-field validation surfaces inline errors as the user types, instead of
// only after Save. Returns null = field is OK; returning a string = error.
export interface FieldErrors {
  effectiveDate?: string;
  minFee?: string;
  // Tier-level errors keyed by tier index for percentage and minAmount.
  tierPercentage: Record<number, string>;
  tierMinAmount: Record<number, string>;
}

export interface FieldWarnings {
  // Soft warnings — not blocking, but visually flagged. Likely-typo cases.
  tierPercentage: Record<number, string>;
  minFee?: string;
}

export function computeFieldErrors(
  state: FeeScheduleFormState,
): FieldErrors {
  const errors: FieldErrors = {
    tierPercentage: {},
    tierMinAmount: {},
  };

  if (!state.effectiveDate) {
    errors.effectiveDate = "Vui lòng chọn ngày";
  } else if (state.effectiveDate < todayISO()) {
    errors.effectiveDate = "Phải là hôm nay hoặc tương lai";
  }

  // minFeeVnd = 0 is allowed (no floor — free advance schedule); only a
  // typed negative is invalid (input min={0} does not block the minus key).
  if (state.minFeeVnd < 0) {
    errors.minFee = "Phải bằng 0 hoặc lớn hơn 0";
  }

  let prevMin = -1;
  state.tiers.forEach((tier, idx) => {
    if (tier.percentage < 0 || tier.percentage > 100) {
      errors.tierPercentage[idx] = "Phải từ 0 đến 100";
    }
    if (idx === 0) {
      // First tier is fixed at 0 (UI disables it). Not validated here.
    } else if (tier.minAmount <= prevMin) {
      errors.tierMinAmount[idx] = "Phải lớn hơn bậc trước";
    }
    prevMin = tier.minAmount;
  });

  return errors;
}

export function computeFieldWarnings(
  state: FeeScheduleFormState,
): FieldWarnings {
  const warnings: FieldWarnings = { tierPercentage: {} };

  state.tiers.forEach((tier, idx) => {
    if (tier.percentage > 0 && tier.percentage < 0.5) {
      warnings.tierPercentage[idx] = "Tỷ lệ rất thấp — có chắc không?";
    } else if (tier.percentage > 10) {
      warnings.tierPercentage[idx] = "Tỷ lệ rất cao — kiểm tra lại";
    }
  });

  if (state.minFeeVnd > 100_000) {
    warnings.minFee = "Phí tối thiểu khá cao — kiểm tra lại";
  }

  return warnings;
}

export function hasFieldErrors(errors: FieldErrors): boolean {
  return (
    !!errors.effectiveDate ||
    !!errors.minFee ||
    Object.keys(errors.tierPercentage).length > 0 ||
    Object.keys(errors.tierMinAmount).length > 0
  );
}

export function summarizeBlockingIssues(
  state: FeeScheduleFormState,
): string[] {
  const issues: string[] = [];
  const errors = computeFieldErrors(state);
  if (errors.effectiveDate) {
    issues.push(`Ngày hiệu lực: ${errors.effectiveDate}`);
  }
  if (errors.minFee) {
    issues.push(`Phí tối thiểu: ${errors.minFee}`);
  }
  const pctErrIdxs = Object.keys(errors.tierPercentage);
  if (pctErrIdxs.length > 0) {
    issues.push(`Phần trăm bậc ${pctErrIdxs.map((i) => Number(i) + 1).join(", ")}: sai khoảng`);
  }
  const minErrIdxs = Object.keys(errors.tierMinAmount);
  if (minErrIdxs.length > 0) {
    issues.push(`Mức bậc ${minErrIdxs.map((i) => Number(i) + 1).join(", ")}: phải tăng dần`);
  }
  return issues;
}

export function toCreateRequest(
  state: FeeScheduleFormState,
): CreateFeeScheduleRequest {
  return {
    effectiveDate: state.effectiveDate,
    tiers: state.tiers.map((t) => ({
      minAmount: t.minAmount,
      percentage: t.percentage,
    })),
    minFeeVnd: state.minFeeVnd,
    notes: state.notes.trim() || undefined,
  };
}

export function isEntryEditable(entry: FeeScheduleEntry): boolean {
  // Past or currently-active entries are immutable. The backend enforces this
  // too; the UI mirrors it so disabled actions are obvious.
  return entry.isPending;
}

// --- Draft persistence -------------------------------------------------------

const DRAFT_STORAGE_KEY = "fee-schedule-form-draft-v1";

export function loadDraft(): FeeScheduleFormState | null {
  try {
    const raw = sessionStorage.getItem(DRAFT_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as FeeScheduleFormState;
    if (
      !parsed ||
      typeof parsed !== "object" ||
      !Array.isArray(parsed.tiers) ||
      parsed.tiers.length === 0
    ) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function saveDraft(state: FeeScheduleFormState): void {
  try {
    sessionStorage.setItem(DRAFT_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // sessionStorage may be unavailable (private mode etc.); silently skip.
  }
}

export function clearDraft(): void {
  try {
    sessionStorage.removeItem(DRAFT_STORAGE_KEY);
  } catch {
    // ignore
  }
}

// --- Days-from-now helper ----------------------------------------------------

export function daysFromTodayISO(iso: string): number {
  if (!iso) return 0;
  const today = new Date(todayISO());
  const target = new Date(iso);
  const ms = target.getTime() - today.getTime();
  return Math.round(ms / (1000 * 60 * 60 * 24));
}
