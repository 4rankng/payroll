import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RefreshCw } from 'lucide-react';
import type { FormState } from '@/types/api/loan.types';

type FormFieldUpdater = <K extends keyof FormState>(key: K, value: FormState[K]) => void;

interface AutoInterestFieldsProps {
  form: FormState;
  onFieldChange: FormFieldUpdater;
  onGenerateSchedules: () => void;
  errors: Record<string, string>;
  isAmortization?: boolean;
}

export function AutoInterestFields({
  form,
  onFieldChange,
  onGenerateSchedules,
  errors,
  isAmortization = false
}: AutoInterestFieldsProps) {
  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-3 items-start">
        <div className="space-y-2">
          <Label className="typography-label-medium">Lãi suất (%/năm) *</Label>
          <Input
            inputMode="decimal"
            value={form.interest_rate_percent}
            onChange={(e) => onFieldChange('interest_rate_percent', e.target.value.replace(/[^0-9.]/g, ''))}
            placeholder="Ví dụ: 12"
            className={errors.interest_rate_percent ? 'input-error' : ''}
          />
          {errors.interest_rate_percent && <p className="typography-body-small text-financial-negative mt-1">{errors.interest_rate_percent}</p>}
        </div>
        <div className="space-y-2">
          <Label className="typography-label-medium">Kỳ hạn (tháng) *</Label>
          <Input
            inputMode="numeric"
            value={form.term_months}
            onChange={(e) => onFieldChange('term_months', e.target.value.replace(/[^0-9]/g, ''))}
            placeholder="Ví dụ: 12"
            className={errors.term_months ? 'input-error' : ''}
          />
          {errors.term_months && <p className="typography-body-small text-financial-negative mt-1">{errors.term_months}</p>}
        </div>
        <div className="space-y-2">
          <Label className="typography-label-medium">Ngày trả (1-28) *</Label>
          <Input
            inputMode="numeric"
            value={form.payment_day_of_month}
            onChange={(e) => onFieldChange('payment_day_of_month', e.target.value.replace(/[^0-9]/g, ''))}
            placeholder="1"
            className={errors.payment_day_of_month ? 'input-error' : ''}
          />
          {errors.payment_day_of_month && <p className="typography-body-small text-financial-negative mt-1">{errors.payment_day_of_month}</p>}
        </div>
        <div className="space-y-2">
          <Label className="typography-label-medium opacity-0 select-none pointer-events-none" aria-hidden="true">Button</Label>
          <Button
            type="button"
            variant="default"
            onClick={onGenerateSchedules}
            className="w-full min-h-[44px] typography-label-large"
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Xem lịch trả
          </Button>
        </div>
      </div>

      {errors.schedules && form.schedules.length === 0 && (
        <p className="typography-body-small text-financial-negative">{errors.schedules}</p>
      )}
    </>
  );
}
