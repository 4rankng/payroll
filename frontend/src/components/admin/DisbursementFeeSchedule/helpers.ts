// Pure helpers for the disbursement fee schedule UI. Mirrors
// AdvancePaymentFeeSchedule/form-helpers.ts but for the simpler flat-fee shape.

import type {
  CreateDisbursementFeeScheduleRequest,
  DisbursementFeeScheduleEntry,
} from "@/types/api/disbursement-fee-schedule.types";

export interface DisbursementFeeFormState {
  effectiveDate: string;
  feeVnd: number;
  notes: string;
}

export type DisbursementFeeFormMode =
  | { kind: "create" }
  | { kind: "edit"; entry: DisbursementFeeScheduleEntry };

export const DEFAULT_FEE_VND = 200;

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
  mode: DisbursementFeeFormMode,
): DisbursementFeeFormState {
  if (mode.kind === "edit") {
    const e = mode.entry;
    return {
      effectiveDate: e.effectiveDate,
      feeVnd: e.feeVnd,
      notes: e.notes ?? "",
    };
  }
  return {
    effectiveDate: tomorrowISO(),
    feeVnd: DEFAULT_FEE_VND,
    notes: "",
  };
}

export function validateFormState(
  state: DisbursementFeeFormState,
): string | null {
  if (!state.effectiveDate) {
    return "Vui lòng chọn ngày hiệu lực";
  }
  if (state.effectiveDate < todayISO()) {
    return "Ngày hiệu lực phải là hôm nay hoặc tương lai";
  }
  if (state.feeVnd < 0) {
    return "Mức phí không được âm";
  }
  return null;
}

export function toCreateRequest(
  state: DisbursementFeeFormState,
): CreateDisbursementFeeScheduleRequest {
  return {
    effectiveDate: state.effectiveDate,
    feeVnd: state.feeVnd,
    notes: state.notes.trim() || undefined,
  };
}

// Past or currently-active entries are immutable (backend invariant). UI
// mirrors the same rule so disabled actions are obvious.
export function isEntryEditable(entry: DisbursementFeeScheduleEntry): boolean {
  return entry.isPending;
}

// formatVietnameseDate: YYYY-MM-DD → DD/MM/YYYY for human-readable display.
export function formatVietnameseDate(iso: string): string {
  if (!iso || iso.length < 10) return iso;
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
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
