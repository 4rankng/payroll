// Helpers for the admin "Chuyển tiền" (Manual Disbursement) page —
// validation, formatting, and the curated bank-code list supported by the provider.

import type {
  ManualDisbursementResponse,
  ManualDisbursementStatus,
} from "@/types/api/manual-disbursement.types";

export interface BankOption {
  code: string;
  label: string;
  swiftCode: string;
}

export interface ManualDisbursementFormState {
  amount: string; // kept as string while user types; parsed on submit
  description: string;
  bankCode: string;
  accountNo: string;
  accountName: string;
  accountType: "0" | "1";
}

export const initialFormState: ManualDisbursementFormState = {
  amount: "",
  description: "",
  bankCode: "",
  accountNo: "",
  accountName: "",
  accountType: "0",
};

export const MIN_AMOUNT = 50_000;
export const MAX_AMOUNT = 2_000_000_000;

// validateForm runs the same constraints as the backend (no hyphens,
// digits-only account_no, etc.) so the user sees inline errors before
// the round-trip. Returns the first error message keyed by field, or
// an empty object when the form is fully valid.
export function validateForm(
  state: ManualDisbursementFormState,
): Partial<Record<keyof ManualDisbursementFormState, string>> {
  const errors: Partial<Record<keyof ManualDisbursementFormState, string>> = {};
  const amount = Number(state.amount);
  if (!state.amount || Number.isNaN(amount) || amount < MIN_AMOUNT) {
    errors.amount = `Số tiền phải từ ${MIN_AMOUNT.toLocaleString("vi-VN")} ₫ trở lên`;
  } else if (amount > MAX_AMOUNT) {
    errors.amount = `Số tiền tối đa ${MAX_AMOUNT.toLocaleString("vi-VN")} ₫`;
  }
  if (!state.description.trim()) {
    errors.description = "Vui lòng nhập nội dung chuyển khoản";
  } else if (!/^[A-Za-z0-9 ]+$/.test(state.description)) {
    errors.description =
      "Nội dung chỉ chứa chữ, số và khoảng trắng (không dấu, không ký tự đặc biệt)";
  }
  if (!state.bankCode.trim()) {
    errors.bankCode = "Vui lòng chọn hoặc nhập mã ngân hàng";
  }
  if (!state.accountNo.trim()) {
    errors.accountNo = "Vui lòng nhập số tài khoản";
  } else if (!/^\d{6,20}$/.test(state.accountNo)) {
    errors.accountNo = "Số tài khoản phải có 6–20 chữ số";
  }
  if (!state.accountName.trim()) {
    errors.accountName = "Vui lòng nhập tên chủ tài khoản";
  } else if (state.accountName.trim().length < 3) {
    errors.accountName = "Tên chủ tài khoản phải có ít nhất 3 ký tự";
  } else if (state.accountName.trim().length > 100) {
    errors.accountName = "Tên chủ tài khoản tối đa 100 ký tự";
  }
  return errors;
}

const VND_FORMATTER = new Intl.NumberFormat("vi-VN");

export function formatAmount(value: number | string): string {
  const n = typeof value === "string" ? Number(value) : value;
  if (!Number.isFinite(n)) return "0";
  return VND_FORMATTER.format(n);
}

// parseAmountInput accepts user keystrokes and normalizes to a digits-
// only string. Strips commas, dots, and any non-numeric character so the
// formatter can reapply grouping deterministically on each render.
export function parseAmountInput(raw: string): string {
  return raw.replace(/[^\d]/g, "");
}

// Verification expires after 5 minutes of inactivity per spec — after
// that the user must re-verify before the confirm modal can open.
export const VERIFICATION_TTL_MS = 5 * 60 * 1000;

export interface VerifiedAccount {
  bankCode: string;
  accountNo: string;
  verifiedName: string;
  verifiedAt: number;
}

export function isVerificationValid(
  v: VerifiedAccount | null,
  state: ManualDisbursementFormState,
): boolean {
  if (!v) return false;
  if (Date.now() - v.verifiedAt > VERIFICATION_TTL_MS) return false;
  return v.bankCode === state.bankCode && v.accountNo === state.accountNo;
}

const STATUS_LABEL: Record<ManualDisbursementStatus, string> = {
  pending: "Đang chờ",
  authorised: "Đã duyệt",
  completed: "Thành công",
  failed: "Thất bại",
};

export function statusLabel(status: ManualDisbursementStatus): string {
  return STATUS_LABEL[status] ?? status;
}

export function statusBadgeClass(status: ManualDisbursementStatus): string {
  switch (status) {
    case "completed":
      return "bg-success/15 text-success border border-success/30";
    case "failed":
      return "bg-destructive/15 text-destructive border border-destructive/30";
      case "pending":
    case "authorised":
    default:
      return "bg-muted text-muted-foreground border border-border";
  }
}

// 9Pay uses "000"/"0", OnePay uses "00" as their success codes.
// When stored in error_code these must NOT be rendered as errors in the UI.
const PROVIDER_SUCCESS_CODES = new Set(["000", "0", "00", ""]);

export function isRealErrorCode(code: string | null | undefined): boolean {
  if (!code) return false;
  return !PROVIDER_SUCCESS_CODES.has(code.trim());
}

// Provider returned a success code (e.g. "00") but the backend hasn't moved
// the status to "completed" yet (waiting for IPN). Treat as success in the UI.
export function isProviderSuccess(row: ManualDisbursementResponse): boolean {
  if (row.status === "completed" || row.status === "failed") return false;
  if (!row.error_code || row.error_code.trim() === "") return false;
  return !isRealErrorCode(row.error_code);
}

export function bankLabel(code: string, options: BankOption[]): string {
  return options.find((b) => b.code === code)?.label ?? code;
}

// formatTimestamp renders a backend ISO string in vi-VN locale. Backend
// emits "...Z" UTC; the browser converts to the user's timezone.
export function formatTimestamp(iso: string | null | undefined): string {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleString("vi-VN", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return iso;
  }
}

// summarizeRow is used by the recent-transfers list and the inline
// confirmation panel — both want a stable terse summary of one row.
export function summarizeRow(row: ManualDisbursementResponse): string {
  return `${formatAmount(row.requested_amount)} ₫ → ${row.recipient_name} (${row.recipient_bank})`;
}
