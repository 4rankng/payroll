import { useCallback } from 'react';
import type { FormState } from '@/types/api/loan.types';

type ScheduleSummary = {
  principal: number;
  totalRepayment: number;
  totalInterest: number;
  effectiveAnnualRate: number;
} | null;

export function useLoanValidation(customScheduleSummary: ScheduleSummary) {
  const validate = useCallback((form: FormState) => {
    const nextErrors: Record<string, string> = {};

    if (!form.lender_id) nextErrors.lender_id = 'Vui lòng chọn chủ nợ';
    if (!form.principal_amount || Number(form.principal_amount) <= 0) {
      nextErrors.principal_amount = 'Số tiền vay phải lớn hơn 0';
    }
    if (!form.start_date) nextErrors.start_date = 'Vui lòng chọn ngày giải ngân';

    if (form.loan_type === 'custom_schedule') {
      // Custom schedule validation
      if (form.schedules.length === 0) {
        nextErrors.schedules = 'Vui lòng tạo lịch trả';
      } else {
        const invalidSchedules = form.schedules.some(s => !s.due_date || !s.amount || Number(s.amount) <= 0);
        if (invalidSchedules) {
          nextErrors.schedules = 'Vui lòng điền đầy đủ thông tin cho tất cả các lịch trả';
        }
        if (!nextErrors.schedules && customScheduleSummary) {
          const { totalRepayment, principal } = customScheduleSummary;
          if (totalRepayment < principal) {
            nextErrors.schedules = 'Tổng tiền trả phải lớn hơn hoặc bằng số tiền vay';
          }
        }
      }
    } else {
      // For bullet_loan and amortization (both need interest parameters)
      if (!form.interest_rate_percent || Number(form.interest_rate_percent) <= 0) {
        nextErrors.interest_rate_percent = 'Lãi suất phải lớn hơn 0%';
      }
      if (!form.term_months || Number(form.term_months) <= 0) {
        nextErrors.term_months = 'Kỳ hạn (tháng) phải lớn hơn 0';
      }
      const paymentDay = Number(form.payment_day_of_month);
      if (!paymentDay || paymentDay < 1 || paymentDay > 28) {
        nextErrors.payment_day_of_month = 'Ngày thanh toán từ 1 đến 28';
      }
    }
    return nextErrors;
  }, [customScheduleSummary]);

  return { validate };
}
