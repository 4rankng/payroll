// Pure helpers for the weekly-payment fee schedule UI (phí trả lương tuần).
// Mirrors DisbursementFeeSchedule/helpers.ts but for a percentage-based fee.

import type {
  CreateWeeklyPaymentFeeScheduleRequest,
  WeeklyPaymentFeeScheduleEntry,
} from "@/types/api/weekly-payment-fee-schedule.types";

export interface WeeklyPaymentFeeFormState {
  effectiveDate: string;
  percentage: number;
  notes: string;
}

export type WeeklyPaymentFeeFormMode =
  | { kind: "create" }
  | { kind: "edit"; entry: WeeklyPaymentFeeScheduleEntry };

export const DEFAULT_FEE_PERCENTAGE = 1.8;

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
  mode: WeeklyPaymentFeeFormMode,
): WeeklyPaymentFeeFormState {
  if (mode.kind === "edit") {
    const e = mode.entry;
    return {
      effectiveDate: e.effectiveDate,
      percentage: e.percentage,
      notes: e.notes ?? "",
    };
  }
  return {
    effectiveDate: tomorrowISO(),
    percentage: DEFAULT_FEE_PERCENTAGE,
    notes: "",
  };
}

export function validateFormState(
  state: WeeklyPaymentFeeFormState,
): string | null {
  if (!state.effectiveDate) {
    return "Vui lòng chọn ngày hiệu lực";
  }
  if (state.effectiveDate < todayISO()) {
    return "Ngày hiệu lực phải là hôm nay hoặc tương lai";
  }
  if (!Number.isFinite(state.percentage) || state.percentage < 0 || state.percentage > 100) {
    return "Tỷ lệ phí phải từ 0 đến 100";
  }
  return null;
}

export function toCreateRequest(
  state: WeeklyPaymentFeeFormState,
): CreateWeeklyPaymentFeeScheduleRequest {
  return {
    effectiveDate: state.effectiveDate,
    percentage: state.percentage,
    notes: state.notes.trim() || undefined,
  };
}

// Past or currently-active entries are immutable (backend invariant). UI
// mirrors the same rule so disabled actions are obvious.
export function isEntryEditable(entry: WeeklyPaymentFeeScheduleEntry): boolean {
  return entry.isPending;
}

// formatVietnameseDate: YYYY-MM-DD → DD/MM/YYYY for human-readable display.
export function formatVietnameseDate(iso: string): string {
  if (!iso || iso.length < 10) return iso;
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
}

// formatPercentageVi: 1.8 → "1,8%" (Vietnamese decimal comma).
export function formatPercentageVi(pct: number): string {
  return `${String(pct).replace(".", ",")}%`;
}

export function daysFromTodayISO(iso: string): number {
  if (!iso) return 0;
  const today = new Date(todayISO());
  const target = new Date(iso);
  const ms = target.getTime() - today.getTime();
  return Math.round(ms / (1000 * 60 * 60 * 24));
}

export function formatDaysFromNow(days: number): string {
  if (days === 0) return "hôm nay";
  if (days === 1) return "ngày mai";
  if (days > 0) return `còn ${days} ngày nữa`;
  if (days === -1) return "hôm qua";
  return `${Math.abs(days)} ngày trước`;
}
