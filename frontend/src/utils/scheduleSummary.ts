import type { FormState } from '@/types/api/loan.types';

export function calculateScheduleSummary(form: FormState) {
  if (form.schedules.length === 0 || !form.principal_amount) {
    return null;
  }

  const principal = Number(form.principal_amount);
  if (isNaN(principal) || principal <= 0) {
    return null;
  }

  const totalRepayment = form.schedules.reduce((sum, s) => {
    const amount = Number(s.amount);
    return sum + (isNaN(amount) ? 0 : amount);
  }, 0);

  const totalInterest = totalRepayment - principal;
  
  // Use the provided interest rate from the form for bullet_loan and amortization loans
  // since that's the rate the user entered and expects to see
  let effectiveAnnualRate = 0;
  const providedRate = Number(form.interest_rate_percent);
  
  if ((form.loan_type === 'bullet_loan' || form.loan_type === 'amortization') && 
      !isNaN(providedRate) && providedRate > 0) {
    effectiveAnnualRate = providedRate;
  } else {
    // For custom schedules, calculate based on actual payments
    const termMonths = form.schedules.length;
    effectiveAnnualRate = termMonths > 0
      ? (totalInterest / termMonths) * 12 / principal * 100
      : 0;
  }

  return {
    principal,
    totalRepayment,
    totalInterest,
    effectiveAnnualRate,
  };
}