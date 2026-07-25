export interface BankAccountNameMismatch {
  enteredName: string;
  bankName: string;
}

const BANK_NAME_MISMATCH_PREFIX = "tên chủ tài khoản không khớp";
const BANK_REPORTED_NAME_PATTERN = /\(ngân hàng ghi:\s*(.+?)\)\s*$/i;

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
