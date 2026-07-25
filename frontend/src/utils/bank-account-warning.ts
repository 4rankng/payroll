export interface BankAccountNameMismatch {
  enteredName: string;
  bankName: string;
}

export interface BankAccountWarning {
  accountName?: string;
  reason: string;
}

const BANK_NAME_MISMATCH_PREFIX = "tên chủ tài khoản không khớp";
const INVALID_ACCOUNT_NUMBER_PREFIX = "số tài khoản không hợp lệ";
const MISSING_SWIFT_CODE_PREFIX = "không có mã swift";
const BANK_REPORTED_NAME_PATTERN = /\(ngân hàng ghi:\s*(.+?)\)\s*$/i;
const BANK_ACCOUNT_INVALID_ERROR_CODE = "BANK_ACCOUNT_INVALID";

export function getRejectedBankAccountWarning(
  error: unknown,
  accountName?: string,
): BankAccountWarning | null {
  if (!error || typeof error !== "object") {
    return null;
  }

  const apiError = error as { code?: unknown; message?: unknown };
  if (apiError.code !== BANK_ACCOUNT_INVALID_ERROR_CODE) {
    return null;
  }

  return {
    accountName,
    reason:
      typeof apiError.message === "string" && apiError.message.trim()
        ? apiError.message
        : "Tài khoản ngân hàng cần kiểm tra",
  };
}

export function getBankAccountWarningTitle(reason: string): string {
  const normalizedReason = reason.trim().toLocaleLowerCase("vi-VN");

  if (normalizedReason.startsWith(BANK_NAME_MISMATCH_PREFIX)) {
    return "Tên chủ tài khoản không khớp";
  }

  if (normalizedReason.startsWith(INVALID_ACCOUNT_NUMBER_PREFIX)) {
    return "Số tài khoản không hợp lệ";
  }

  if (normalizedReason.startsWith(MISSING_SWIFT_CODE_PREFIX)) {
    return "Không thể kiểm tra tài khoản";
  }

  return "Tài khoản ngân hàng cần kiểm tra";
}

export function getBankAccountNameMismatch(
  reason: string,
  enteredName?: string,
): BankAccountNameMismatch | null {
  if (
    !enteredName ||
    !reason.trim().toLocaleLowerCase("vi-VN").startsWith(BANK_NAME_MISMATCH_PREFIX)
  ) {
    return null;
  }

  const bankName = reason.match(BANK_REPORTED_NAME_PATTERN)?.[1]?.trim();

  if (!bankName) {
    return null;
  }

  return {
    enteredName: enteredName.trim(),
    bankName,
  };
}
