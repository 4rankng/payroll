import { useCallback } from 'react';
import type { FormState, ScheduleRow } from '@/types/api/loan.types';

// --- START REFACTORED HELPERS ---

/**
 * Parses a 'YYYY-MM-DD' string into a Date object in a way that avoids timezone issues.
 * It treats the date string as being in the local timezone.
 */
const parseDate = (dateString: string): Date => {
  const [year, month, day] = dateString.split('-').map(Number);
  // Month is 0-indexed in JavaScript Date
  return new Date(year, month - 1, day);
};

/**
 * Formats a Date object into a 'YYYY-MM-DD' string.
 */
const formatDate = (date: Date): string => {
  const year = date.getFullYear();
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');
  return `${year}-${month}-${day}`;
};

/**
 * Creates a due date by adding months to a start date and setting a specific day.
 * Handles month rollovers correctly.
 */
const createDueDate = (startDate: string, offsetMonths: number, paymentDay: number): Date => {
  const base = parseDate(startDate);
  // Set to the first day of the month to avoid issues with months that have different numbers of days.
  base.setDate(1);
  base.setMonth(base.getMonth() + offsetMonths);
  
  // Now set the day, ensuring we don't skip a month if the paymentDay is past the end of the new month.
  const year = base.getFullYear();
  const month = base.getMonth();
  const lastDayOfMonth = new Date(year, month + 1, 0).getDate();
  
  const day = Math.min(paymentDay, lastDayOfMonth);
  return new Date(year, month, day);
};


// --- END REFACTORED HELPERS ---


type FieldErrors = Partial<Record<keyof FormState, string>>;

type ScheduleGenerationResult =
  | { success: true }
  | { success: false; error: string; fieldErrors?: FieldErrors };

type UpdateField = <K extends keyof FormState>(key: K, value: FormState[K]) => void;

const REQUIRED_FIELD_ERROR = (label: string) => `Vui lòng nhập ${label}`;

function generateFlatLoanSchedule(
  principal: number,
  annualRatePercent: number,
  termMonths: number,
  startDate: string,
  paymentDay: number,
): ScheduleRow[] {
  // Keep calculations in integers (VND) to match backend expectations
  const sanitizedPrincipal = Math.round(principal);
  const monthlyInterestRate = annualRatePercent / 100 / 12;

  // Calculate total interest for the entire loan period
  const totalInterest = Math.round(sanitizedPrincipal * monthlyInterestRate * termMonths);

  // Calculate standard monthly interest payment
  const monthlyInterest = Math.round(sanitizedPrincipal * monthlyInterestRate);

  const schedules: ScheduleRow[] = [];

  // Months 1 to termMonths-1: Pay only monthly interest
  for (let i = 1; i < termMonths; i++) {
    schedules.push({
      id: crypto.randomUUID(),
      due_date: formatDate(createDueDate(startDate, i, paymentDay)),
      amount: monthlyInterest.toString(),
    });
  }

  // Final month: Pay principal + remaining interest (to offset any rounding errors)
  const remainingInterest = totalInterest - (monthlyInterest * (termMonths - 1));
  schedules.push({
    id: crypto.randomUUID(),
    due_date: formatDate(createDueDate(startDate, termMonths, paymentDay)),
    amount: (sanitizedPrincipal + remainingInterest).toString(),
  });

  return schedules;
}

function generateAmortizedSchedule(
  principal: number,
  annualRatePercent: number,
  termMonths: number,
  startDate: string,
  paymentDay: number,
): ScheduleRow[] {
  // Using user-provided formula which is a simple interest calculation
  const sanitizedPrincipal = Math.round(principal);
  const annualRate = annualRatePercent / 100;

  // total payment = (1 + annual rate / 12 * term months) * principal
  const totalPayment = Math.round(sanitizedPrincipal * (1 + (annualRate / 12) * termMonths));
  
  // monthly payment = ROUND(total payment / term months)
  const monthlyPayment = Math.round(totalPayment / termMonths);

  // last month payment = total payment - monthly payment * (term months - 1)
  const lastMonthPayment = totalPayment - (monthlyPayment * (termMonths - 1));

  const schedules: ScheduleRow[] = [];

  for (let i = 0; i < termMonths; i++) {
    const amount = (i === termMonths - 1) ? lastMonthPayment : monthlyPayment;
    schedules.push({
      id: crypto.randomUUID(),
      due_date: formatDate(createDueDate(startDate, i + 1, paymentDay)),
      amount: amount.toString(),
    });
  }

  return schedules;
}

export function useLoanCalculations(updateField: UpdateField) {
  const handleGenerateSchedules = useCallback((form: FormState): ScheduleGenerationResult => {
    const missingFields: Array<{ key: keyof FormState; label: string }> = [];
    if (!form.start_date) missingFields.push({ key: 'start_date', label: 'Ngày giải ngân' });
    if (!form.custom_term_months) missingFields.push({ key: 'custom_term_months', label: 'Số tháng' });
    if (!form.custom_payment_day_of_month) missingFields.push({ key: 'custom_payment_day_of_month', label: 'Ngày thanh toán' });
    if (!form.custom_monthly_amount) missingFields.push({ key: 'custom_monthly_amount', label: 'Trả hàng tháng' });

    if (missingFields.length > 0) {
      const fieldErrors = missingFields.reduce<FieldErrors>((acc, field) => {
        acc[field.key] = REQUIRED_FIELD_ERROR(field.label);
        return acc;
      }, {});
      return {
        success: false,
        error: `Vui lòng nhập: ${missingFields.map(field => field.label).join(', ')}`,
        fieldErrors,
      };
    }

    const termMonths = Number(form.custom_term_months);
    const paymentDay = Number(form.custom_payment_day_of_month);
    const monthlyAmount = Number(form.custom_monthly_amount);
    const lastMonthAmount = form.custom_last_month_amount ? Number(form.custom_last_month_amount) : monthlyAmount;

    if (termMonths <= 0 || monthlyAmount <= 0) {
      return {
        success: false,
        error: 'Số tháng và số tiền phải lớn hơn 0',
        fieldErrors: {
          custom_term_months: 'Số tháng phải lớn hơn 0',
          custom_monthly_amount: 'Số tiền phải lớn hơn 0',
        },
      };
    }

    if (paymentDay < 1 || paymentDay > 28) {
      return {
        success: false,
        error: 'Ngày thanh toán phải từ 1 đến 28',
        fieldErrors: {
          custom_payment_day_of_month: 'Ngày thanh toán phải từ 1 đến 28',
        },
      };
    }

    const newSchedules: ScheduleRow[] = [];

    for (let i = 0; i < termMonths; i++) {
      const dueDate = createDueDate(form.start_date, i + 1, paymentDay);
      const amount = i === termMonths - 1 ? lastMonthAmount : monthlyAmount;

      newSchedules.push({
        id: crypto.randomUUID(),
        due_date: formatDate(dueDate),
        amount: amount.toString(),
      });
    }

    updateField('schedules', newSchedules);
    return { success: true };
  }, [updateField]);

  const handleGenerateSchedulesFromAutoInterest = useCallback((formState: FormState): ScheduleGenerationResult => {
    const missingFields: Array<{ key: keyof FormState; label: string }> = [];
    if (!formState.start_date) missingFields.push({ key: 'start_date', label: 'Ngày giải ngân' });
    if (!formState.interest_rate_percent) missingFields.push({ key: 'interest_rate_percent', label: 'Lãi suất' });
    if (!formState.term_months) missingFields.push({ key: 'term_months', label: 'Kỳ hạn (tháng)' });
    if (!formState.payment_day_of_month) missingFields.push({ key: 'payment_day_of_month', label: 'Ngày trả' });
    if (!formState.principal_amount) missingFields.push({ key: 'principal_amount', label: 'Số tiền vay' });

    if (missingFields.length > 0) {
      const fieldErrors = missingFields.reduce<FieldErrors>((acc, field) => {
        acc[field.key] = REQUIRED_FIELD_ERROR(field.label);
        return acc;
      }, {});
      return {
        success: false,
        error: `Vui lòng nhập: ${missingFields.map(field => field.label).join(', ')}`,
        fieldErrors,
      };
    }

    const annualInterestRate = Number(formState.interest_rate_percent);
    const termMonths = Number(formState.term_months);
    const paymentDay = Number(formState.payment_day_of_month);
    const principalAmount = Number(formState.principal_amount);

    if (annualInterestRate <= 0) {
      return {
        success: false,
        error: 'Lãi suất phải lớn hơn 0%',
        fieldErrors: {
          interest_rate_percent: 'Lãi suất phải lớn hơn 0%',
        },
      };
    }
    
    if (termMonths <= 0) {
      return {
        success: false,
        error: 'Kỳ hạn (tháng) phải lớn hơn 0',
        fieldErrors: {
          term_months: 'Kỳ hạn (tháng) phải lớn hơn 0',
        },
      };
    }
    
    if (paymentDay < 1 || paymentDay > 28) {
      return {
        success: false,
        error: 'Ngày thanh toán phải từ 1 đến 28',
        fieldErrors: {
          payment_day_of_month: 'Ngày thanh toán phải từ 1 đến 28',
        },
      };
    }
    
    if (principalAmount <= 0) {
      return {
        success: false,
        error: 'Số tiền vay phải lớn hơn 0',
        fieldErrors: {
          principal_amount: 'Số tiền vay phải lớn hơn 0',
        },
      };
    }

    try {
      const schedule = generateFlatLoanSchedule(
        principalAmount,
        annualInterestRate,
        termMonths,
        formState.start_date,
        paymentDay,
      );
      updateField('schedules', schedule);
      return { success: true };
    } catch {
      return {
        success: false,
        error: 'Lỗi khi tạo lịch trả nợ (flat rate)',
      };
    }
  }, [updateField]);

  const handleGenerateSchedulesFromAmortization = useCallback((formState: FormState): ScheduleGenerationResult => {
    const missingFields: Array<{ key: keyof FormState; label: string }> = [];
    if (!formState.start_date) missingFields.push({ key: 'start_date', label: 'Ngày giải ngân' });
    if (!formState.interest_rate_percent) missingFields.push({ key: 'interest_rate_percent', label: 'Lãi suất' });
    if (!formState.term_months) missingFields.push({ key: 'term_months', label: 'Kỳ hạn (tháng)' });
    if (!formState.payment_day_of_month) missingFields.push({ key: 'payment_day_of_month', label: 'Ngày trả' });
    if (!formState.principal_amount) missingFields.push({ key: 'principal_amount', label: 'Số tiền vay' });

    if (missingFields.length > 0) {
      const fieldErrors = missingFields.reduce<FieldErrors>((acc, field) => {
        acc[field.key] = REQUIRED_FIELD_ERROR(field.label);
        return acc;
      }, {});
      return {
        success: false,
        error: `Vui lòng nhập: ${missingFields.map(field => field.label).join(', ')}`,
        fieldErrors,
      };
    }

    const annualInterestRate = Number(formState.interest_rate_percent);
    const termMonths = Number(formState.term_months);
    const paymentDay = Number(formState.payment_day_of_month);
    const principalAmount = Number(formState.principal_amount);

    if (annualInterestRate <= 0) {
      return {
        success: false,
        error: 'Lãi suất phải lớn hơn 0%',
        fieldErrors: {
          interest_rate_percent: 'Lãi suất phải lớn hơn 0%',
        },
      };
    }

    if (termMonths <= 0) {
      return {
        success: false,
        error: 'Kỳ hạn (tháng) phải lớn hơn 0',
        fieldErrors: {
          term_months: 'Kỳ hạn (tháng) phải lớn hơn 0',
        },
      };
    }

    if (paymentDay < 1 || paymentDay > 28) {
      return {
        success: false,
        error: 'Ngày thanh toán phải từ 1 đến 28',
        fieldErrors: {
          payment_day_of_month: 'Ngày thanh toán phải từ 1 đến 28',
        },
      };
    }

    if (principalAmount <= 0) {
      return {
        success: false,
        error: 'Số tiền vay phải lớn hơn 0',
        fieldErrors: {
          principal_amount: 'Số tiền vay phải lớn hơn 0',
        },
      };
    }

    try {
      const schedule = generateAmortizedSchedule(
        principalAmount,
        annualInterestRate,
        termMonths,
        formState.start_date,
        paymentDay,
      );
      updateField('schedules', schedule);
      return { success: true };
    } catch {
      return {
        success: false,
        error: 'Lỗi khi tính toán lịch trả (amortization)',
      };
    }
  }, [updateField]);

  return {
    handleGenerateSchedules,
    handleGenerateSchedulesFromAutoInterest,
    handleGenerateSchedulesFromAmortization
  };
}
