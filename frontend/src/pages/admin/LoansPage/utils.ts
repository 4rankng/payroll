import { formatVND } from '@/utils/loanHelpers';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';
import type { Loan } from '@/types/api/loan.types';

export interface LoansSummary {
  total_borrowed: number;
  total_outstanding: number;
  total_interest_paid: number;
  active_loans_count: number;
}

export function computeLoansSummary(loans: Loan[]): LoansSummary {
  if (!loans.length) {
    return {
      total_borrowed: 0,
      total_outstanding: 0,
      total_interest_paid: 0,
      active_loans_count: 0,
    };
  }
  return {
    total_borrowed: loans.reduce((sum, loan) => sum + loan.principal_amount, 0),
    total_outstanding: loans.reduce((sum, loan) => sum + loan.outstanding_principal, 0),
    total_interest_paid: loans.reduce((sum, loan) => sum + (loan.total_interest_paid ?? 0), 0),
    active_loans_count: loans.filter((loan) => loan.status === 'active').length,
  };
}

export function computeLoanStatItems(summary: LoansSummary) {
  return [
    { label: 'Tổng vay', value: formatVND(summary.total_borrowed) },
    { label: 'Dư nợ hiện tại', value: formatVND(summary.total_outstanding) },
    { label: 'Lãi đã trả', value: formatVND(summary.total_interest_paid) },
    { label: 'Khoản vay', value: summary.active_loans_count },
  ];
}

export function filterLoansBySearch(loans: Loan[], query: string): Loan[] {
  const q = query.trim();
  if (!q) return loans;
  return loans.filter(
    (loan) =>
      vietnameseIncludes(loan.loan_code, q) ||
      vietnameseIncludes(loan.lender.name, q),
  );
}
