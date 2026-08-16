import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RefreshCw } from 'lucide-react';
import type { FormState } from '@/types/api/loan.types';
import { formatVND } from '@/utils/loanHelpers';

type FormFieldUpdater = <K extends keyof FormState>(key: K, value: FormState[K]) => void;

interface CustomScheduleFieldsProps {
  form: FormState;
  onFieldChange: FormFieldUpdater;
  onGenerateSchedules: () => void;
  errors: Record<string, string>;
}

export function CustomScheduleFields({
  form,
  onFieldChange,
  onGenerateSchedules,
  errors
}: CustomScheduleFieldsProps) {
  return (
    <div className="space-y-4">
      {/* Simplified Inputs */}
      <div className="space-y-3 bg-gradient-to-br from-primary-50 to-primary-100/30 border border-primary/20 rounded-xl p-4 shadow-sm">
        <div className="typography-label-medium text-foreground font-semibold">Thiết lập lịch trả</div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div className="space-y-2">
            <Label className="typography-label-small text-foreground">Số tháng *</Label>
            <Input
              inputMode="numeric"
              value={form.custom_term_months}
              onChange={(e) => onFieldChange('custom_term_months', e.target.value.replace(/[^0-9]/g, ''))}
              placeholder="Ví dụ: 12"
              className={`bg-background ${errors.custom_term_months ? 'input-error' : ''}`}
            />
            {errors.custom_term_months && (
              <p className="typography-body-small text-financial-negative">{errors.custom_term_months}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label className="typography-label-small text-foreground">Ngày thanh toán (1-28) *</Label>
            <Input
              inputMode="numeric"
              value={form.custom_payment_day_of_month}
              onChange={(e) => onFieldChange('custom_payment_day_of_month', e.target.value.replace(/[^0-9]/g, ''))}
              placeholder="1"
              className={`bg-background ${errors.custom_payment_day_of_month ? 'input-error' : ''}`}
            />
            {errors.custom_payment_day_of_month && (
              <p className="typography-body-small text-financial-negative">{errors.custom_payment_day_of_month}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label className="typography-label-small text-foreground">Trả hàng tháng (₫) *</Label>
            <Input
              inputMode="numeric"
              value={form.custom_monthly_amount}
              onChange={(e) => onFieldChange('custom_monthly_amount', e.target.value.replace(/[^0-9]/g, ''))}
              placeholder="Ví dụ: 10000000"
              className={`bg-background ${errors.custom_monthly_amount ? 'input-error' : ''}`}
            />
            {form.custom_monthly_amount && Number(form.custom_monthly_amount) > 0 && (
              <p className="typography-label-small text-muted-foreground">{formatVND(Number(form.custom_monthly_amount))}</p>
            )}
            {errors.custom_monthly_amount && (
              <p className="typography-body-small text-financial-negative">{errors.custom_monthly_amount}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label className="typography-label-small text-foreground">Trả tháng cuối (₫)</Label>
            <Input
              inputMode="numeric"
              value={form.custom_last_month_amount}
              onChange={(e) => onFieldChange('custom_last_month_amount', e.target.value.replace(/[^0-9]/g, ''))}
              placeholder="Tùy chọn"
              className="bg-background"
            />
            {form.custom_last_month_amount && Number(form.custom_last_month_amount) > 0 && (
              <p className="typography-label-small text-muted-foreground">{formatVND(Number(form.custom_last_month_amount))}</p>
            )}
          </div>
        </div>

        <p className="typography-label-small text-muted-foreground">
          * Nếu để trống "Trả tháng cuối", hệ thống sẽ dùng số tiền hàng tháng cho kỳ cuối.
        </p>

        <Button
          type="button"
          variant="default"
          onClick={onGenerateSchedules}
          className="w-full min-h-[44px] typography-label-large"
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Xem lịch trả
        </Button>

        {errors.schedules && form.schedules.length === 0 && (
          <p className="typography-body-small text-financial-negative">{errors.schedules}</p>
        )}
      </div>
    </div>
  );
}
