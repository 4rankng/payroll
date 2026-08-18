import type { Employee } from '@/types/api/employee.types';

type BankInformationWarningInput = Pick<
  Employee,
  | 'bank'
  | 'bank_account_number'
  | 'bank_account_name'
  | 'bank_account_status'
  | 'bank_account_invalid_reason'
>;

const GENERIC_INVALID_REASON = 'Tài khoản ngân hàng không hợp lệ';
const UNSAFE_REASON_PATTERN =
  /\b(?:onepay|9pay|provider|error|invalid|account|timeout|failed|internal)\b|[_{}[\]]/iu;

const RECOGNIZED_INVALID_REASONS = [
  /^Số tài khoản không hợp lệ$/iu,
  /^Tên chủ tài khoản không khớp$/iu,
  /^Tên chủ tài khoản không khớp với ngân hàng(?:\s*\(ngân hàng ghi:\s*[^)]+\))?$/iu,
  /^Không có mã SWIFT của ngân hàng, không thể kiểm tra$/iu,
];

function getConfirmedInvalidReason(reason?: string | null): string {
  const normalizedReason = reason?.trim();
  if (!normalizedReason || UNSAFE_REASON_PATTERN.test(normalizedReason)) {
    return GENERIC_INVALID_REASON;
  }

  return RECOGNIZED_INVALID_REASONS.some((pattern) =>
    pattern.test(normalizedReason),
  )
    ? normalizedReason
    : GENERIC_INVALID_REASON;
}

export function getBankInformationWarningReason(
  employee: BankInformationWarningInput,
): string {
  if (!employee.bank) return 'Thiếu ngân hàng';
  if (!employee.bank_account_number?.trim()) return 'Thiếu số tài khoản';
  if (!employee.bank_account_name?.trim()) return 'Thiếu tên chủ tài khoản';

  if (employee.bank_account_status === 'invalid') {
    return getConfirmedInvalidReason(employee.bank_account_invalid_reason);
  }

  return GENERIC_INVALID_REASON;
}

/**
 * Classifies why an employee appears in the missing-bank-details warning
 * list: OnePay confirmed the stored info is wrong ('invalid') vs the info
 * is simply incomplete ('missing'). Mirrors the backend SQL predicate in
 * buildMissingBankDetailsBaseQuery, where an 'invalid' status is listed
 * unconditionally while missing fields also require pending work.
 */
export function getBankInformationWarningKind(
  employee: BankInformationWarningInput,
): 'invalid' | 'missing' {
  return employee.bank_account_status === 'invalid' ? 'invalid' : 'missing';
}
