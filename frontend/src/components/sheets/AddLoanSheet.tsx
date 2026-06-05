import { useState, useEffect, useMemo, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { X } from 'lucide-react';
import { SlideSheetTemplate } from './templates/SlideSheetTemplate';
import { useLenders, useCreateLoan } from '@/hooks/api/useLoans';
import { useLoanCalculations } from '@/hooks/useLoanCalculations';
import { useLoanValidation } from '@/hooks/useLoanValidation';
import { calculateScheduleSummary } from '@/utils/scheduleSummary';
import type { CreateLoanRequest, Lender, FormState } from '@/types/api/loan.types';
import { percentToBps } from '@/utils/loanHelpers';

// Import new components
import { LoanTypeSelector } from './LoanTypeSelector';
import { LenderDisbursementSection } from './LenderDisbursementSection';
import { PrincipalDateFields } from './PrincipalDateFields';
import { AutoInterestFields } from './AutoInterestFields';
import { ScheduleDisplay } from './ScheduleDisplay';
import { CustomScheduleFields } from './CustomScheduleFields';
import { FormField } from './FormField';
import { useMediaQuery } from '@/hooks/use-media-query';

interface AddLoanSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AddLoanSheet({ isOpen, onClose }: AddLoanSheetProps) {
  const { data: lendersResponse } = useLenders({ page: 1, pageSize: 100, sortBy: 'name', sortOrder: 'asc' }, { enabled: isOpen });
  const lenders: Lender[] = useMemo(() => (Array.isArray(lendersResponse?.data) ? lendersResponse!.data as Lender[] : []), [lendersResponse]);

  const createLoan = useCreateLoan();
  const isCompactViewport = useMediaQuery('(max-width: 1000px)');

  const [form, setForm] = useState<FormState>({
    loan_type: 'bullet_loan',
    lender_id: '',
    principal_amount: '',
    start_date: '',
    description: '',
    disburse_now: false,
    disbursement_reference: '',
    interest_rate_percent: '',
    term_months: '',
    payment_day_of_month: '1',
    custom_term_months: '',
    custom_payment_day_of_month: '1',
    custom_monthly_amount: '',
    custom_last_month_amount: '',
    schedules: [],
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState<string>('');

  useEffect(() => {
    if (isOpen) {
      setForm({
        loan_type: 'bullet_loan',
        lender_id: '',
        principal_amount: '',
        start_date: '',
        description: '',
        disburse_now: false,
        disbursement_reference: '',
        interest_rate_percent: '',
        term_months: '',
        payment_day_of_month: '1',
        custom_term_months: '',
        custom_payment_day_of_month: '1',
        custom_monthly_amount: '',
        custom_last_month_amount: '',
        schedules: [],
      });
      setErrors({});
      setSubmitError('');
    }
  }, [isOpen]);

  const updateField = useCallback(<K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm(prev => ({ ...prev, [key]: value }));
    setErrors(prev => {
      const errorKey = key as string;
      const hasFieldError = prev[errorKey];
      const shouldClearSchedules = key !== 'schedules' && prev.schedules;

      if (!hasFieldError && !shouldClearSchedules) {
        return prev;
      }

      const nextErrors = { ...prev };
      if (hasFieldError) {
        nextErrors[errorKey] = '';
      }
      if (shouldClearSchedules) {
        nextErrors.schedules = '';
      }
      return nextErrors;
    });
    if (submitError) {
      setSubmitError('');
    }
  }, [submitError]);

  const handleDisburseNowChange = useCallback((value: boolean) => {
    updateField('disburse_now', value);
    if (value && !form.start_date) {
      const today = new Date().toISOString().split('T')[0];
      updateField('start_date', today);
    }
  }, [form.start_date, updateField]);

  const {
    handleGenerateSchedules: generateCustomSchedules,
    handleGenerateSchedulesFromAutoInterest: generateAutoInterestSchedules,
    handleGenerateSchedulesFromAmortization: generateAmortizationSchedules
  } = useLoanCalculations(updateField);

  const handleGenerateSchedules = useCallback(() => {
    const result = generateCustomSchedules(form);
    if (!result.success) {
      setErrors(prev => {
        const nextErrors = { ...prev, schedules: (result as any).error };
        if ((result as any).fieldErrors) {
          for (const [key, message] of Object.entries((result as any).fieldErrors)) {
            if (message) {
              nextErrors[key] = message;
            }
          }
        }
        return nextErrors;
      });
      return;
    }
    setErrors(prev => ({ ...prev, schedules: '' }));
  }, [form, generateCustomSchedules]);

  const handleGenerateAutoInterestSchedules = useCallback(() => {
    const result = generateAutoInterestSchedules(form);
    if (!result.success) {
      setErrors(prev => {
        const nextErrors = { ...prev, schedules: (result as any).error };
        if ((result as any).fieldErrors) {
          for (const [key, message] of Object.entries((result as any).fieldErrors)) {
            if (message) {
              nextErrors[key] = message;
            }
          }
        }
        return nextErrors;
      });
      return;
    }
    setErrors(prev => ({ ...prev, schedules: '' }));
  }, [form, generateAutoInterestSchedules]);

  const handleGenerateAmortizationSchedules = useCallback(() => {
    const result = generateAmortizationSchedules(form);
    if (!result.success) {
      setErrors(prev => {
        const nextErrors = { ...prev, schedules: (result as any).error };
        if ((result as any).fieldErrors) {
          for (const [key, message] of Object.entries((result as any).fieldErrors)) {
            if (message) {
              nextErrors[key] = message;
            }
          }
        }
        return nextErrors;
      });
      return;
    }
    setErrors(prev => ({ ...prev, schedules: '' }));
  }, [form, generateAmortizationSchedules]);

  const handleScheduleChange = useCallback((id: string, field: 'due_date' | 'amount', value: string) => {
    const updatedSchedules = form.schedules.map(s =>
      s.id === id ? { ...s, [field]: value } : s
    );
    updateField('schedules', updatedSchedules);
  }, [form.schedules, updateField]);

  // Calculate summary for custom schedule
  const customScheduleSummary = useMemo(() => {
    if (form.loan_type !== 'custom_schedule') {
      return null;
    }
    return calculateScheduleSummary(form);
  }, [form]);

  // Calculate summary for auto interest
  const autoInterestScheduleSummary = useMemo(() => {
    if (form.loan_type !== 'bullet_loan') {
      return null;
    }
    return calculateScheduleSummary(form);
  }, [form]);

  // Calculate summary for validation regardless of loan type (all submit as custom)
  const validationScheduleSummary = useMemo(() => {
    return calculateScheduleSummary(form);
  }, [form]);

  const { validate } = useLoanValidation(validationScheduleSummary);

  const handleSubmit = useCallback(async () => {
    // For bullet_loan and amortization types, ensure schedules are calculated
    if (form.loan_type === 'bullet_loan' && form.schedules.length === 0) {
      const result = generateAutoInterestSchedules(form);
      if (!result.success) {
        setSubmitError((result as any).error || 'Vui lòng tạo lịch trả trước khi tạo khoản vay');
        return;
      }
    } else if (form.loan_type === 'amortization' && form.schedules.length === 0) {
      const result = generateAmortizationSchedules(form);
      if (!result.success) {
        setSubmitError((result as any).error || 'Vui lòng tạo lịch trả trước khi tạo khoản vay');
        return;
      }
    } else if (form.loan_type === 'custom_schedule' && form.schedules.length === 0) {
      setSubmitError('Vui lòng tạo lịch trả trước khi tạo khoản vay');
      return;
    }

    const nextErrors = validate(form);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      setSubmitError('Vui lòng kiểm tra và điền đầy đủ thông tin bắt buộc');
      return;
    }

    const disbursementReference = form.disbursement_reference?.trim();

    const payload: CreateLoanRequest = {
      lender_id: Number(form.lender_id),
      principal_amount: Number(form.principal_amount),
      start_date: form.start_date,
      description: form.description?.trim() ? form.description.trim() : undefined,
      disburse_now: form.disburse_now || undefined,
      disbursement_reference: form.disburse_now && disbursementReference ? disbursementReference : undefined,
      // All loan types submit as custom_schedule with calculated schedules
      interest_rate_bps: 0,
      schedules: form.schedules.map(s => ({
        due_date: s.due_date,
        amount: Number(s.amount),
      })),
    };

    try {
      await createLoan.mutateAsync(payload);
      onClose();
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'Có lỗi xảy ra khi tạo khoản vay';
      setSubmitError(errorMessage);
      console.error('Error creating loan:', e);
      // Error handled by hook notifications
    }
  }, [createLoan, form, onClose, validate, generateAutoInterestSchedules, generateAmortizationSchedules]);

  const handleLoanTypeChange = useCallback((type: FormState['loan_type']) => {
    setForm(prev => ({
      ...prev,
      loan_type: type,
      schedules: []
    }));
    setErrors(prev => ({ ...prev, schedules: '' }));
    if (submitError) {
      setSubmitError('');
    }
  }, [submitError]);

  const handleLenderChange = useCallback((id: string) => {
    updateField('lender_id', id);
  }, [updateField]);

  const handlePrincipalChange = useCallback((value: string) => {
    updateField('principal_amount', value);
  }, [updateField]);

  const handleStartDateChange = useCallback((value: string) => {
    updateField('start_date', value);
  }, [updateField]);

  const handleDescriptionChange = useCallback((value: string) => {
    updateField('description', value);
  }, [updateField]);

  const handleDisbursementReferenceChange = useCallback((value: string) => {
    updateField('disbursement_reference', value);
  }, [updateField]);

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      title="Tạo khoản vay"
      description="Nhập thông tin khoản vay mới"
      size={isCompactViewport ? 'full' : 'large'}
      compact
      footer={
        <div className="w-full px-4 space-y-3">
          {submitError && (
            <div className="error-container">
              <span className="error-icon">⚠️</span>
              <span className="error-text">{submitError}</span>
            </div>
          )}
          <div className="grid gap-3 sm:grid-cols-2">
            <Button variant="outline" onClick={onClose} className="w-full min-h-[44px] typography-label-large">
              <X className="w-4 h-4 mr-2" />
              Hủy
            </Button>
            <Button variant="default" onClick={handleSubmit} disabled={createLoan.isPending} className="w-full min-h-[44px] typography-label-large">
              {createLoan.isPending ? 'Đang tạo...' : 'Tạo'}
            </Button>
          </div>
        </div>
      }
    >
      <div className="space-y-4 sm:space-y-5 mt-2 pb-4">
        <div className="space-y-4 sm:space-y-5">
          <div className="section-card p-4 sm:p-5 space-y-4">
            <div className="grid gap-4 sm:grid-cols-12 items-start">
              <div className="sm:col-span-7">
                <LoanTypeSelector
                  loanType={form.loan_type}
                  onChange={handleLoanTypeChange}
                />
              </div>
              <div className="sm:col-span-5">
                <LenderDisbursementSection
                  lenderId={form.lender_id}
                  onLenderChange={handleLenderChange}
                  lenders={lenders}
                  errors={errors}
                />
              </div>
            </div>
          </div>

          <div className="section-card p-4 sm:p-5 space-y-4">
            <PrincipalDateFields
              principalAmount={form.principal_amount}
              startDate={form.start_date}
              onPrincipalChange={handlePrincipalChange}
              onDateChange={handleStartDateChange}
              disburseNow={form.disburse_now}
              onDisburseNowChange={handleDisburseNowChange}
              errors={errors}
            />
          </div>
        </div>

        {form.loan_type === 'bullet_loan' && (
          <div className="section-card p-4 sm:p-5 space-y-4">
            <AutoInterestFields
              form={form}
              onFieldChange={updateField}
              onGenerateSchedules={handleGenerateAutoInterestSchedules}
              errors={errors}
            />

            {form.schedules.length > 0 && (
              <ScheduleDisplay
                schedules={form.schedules}
                loanType={form.loan_type}
                summary={validationScheduleSummary}
              />
            )}
          </div>
        )}

        {form.loan_type === 'custom_schedule' && (
          <div className="section-card p-4 sm:p-5 space-y-4">
            <CustomScheduleFields
              form={form}
              onFieldChange={updateField}
              onGenerateSchedules={handleGenerateSchedules}
              errors={errors}
            />

            {form.schedules.length > 0 && (
              <ScheduleDisplay
                schedules={form.schedules}
                loanType={form.loan_type}
                onScheduleChange={handleScheduleChange}
                summary={validationScheduleSummary}
              />
            )}
          </div>
        )}

        {form.loan_type === 'amortization' && (
          <div className="section-card p-4 sm:p-5 space-y-4">
            <AutoInterestFields
              form={form}
              onFieldChange={updateField}
              onGenerateSchedules={handleGenerateAmortizationSchedules}
              errors={errors}
              isAmortization={true}
            />

            {form.schedules.length > 0 && (
              <ScheduleDisplay
                schedules={form.schedules}
                loanType={form.loan_type}
                summary={validationScheduleSummary}
              />
            )}
          </div>
        )}

        <div className="section-card p-4 sm:p-5 space-y-4">
          <FormField
            label="Ghi chú"
            value={form.description}
            onChange={handleDescriptionChange}
            placeholder="Mô tả khoản vay (nếu có)"
            className="min-h-[80px]"
          />
        </div>

        {form.disburse_now && (
          <div className="section-card p-4 sm:p-5 space-y-4">
            <FormField
              label="Tham chiếu giải ngân"
              value={form.disbursement_reference}
              onChange={handleDisbursementReferenceChange}
              placeholder="Số chứng từ / ghi chú giải ngân (tùy chọn)"
              errors={errors}
              errorKey="disbursement_reference"
            />
          </div>
        )}
      </div>
    </SlideSheetTemplate>
  );
}

export default AddLoanSheet;
