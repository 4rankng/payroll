import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { formatVND } from '@/utils/loanHelpers';

interface PrincipalDateFieldsProps {
  principalAmount: string;
  startDate: string;
  onPrincipalChange: (value: string) => void;
  onDateChange: (value: string) => void;
  disburseNow: boolean;
  onDisburseNowChange: (value: boolean) => void;
  errors: Record<string, string>;
}

export function PrincipalDateFields({
  principalAmount,
  startDate,
  onPrincipalChange,
  onDateChange,
  disburseNow,
  onDisburseNowChange,
  errors
}: PrincipalDateFieldsProps) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 sm:gap-6">
      <div className="w-full space-y-2">
        <Label className="typography-label-medium">Số tiền vay (đ) *</Label>
        <Input
          inputMode="numeric"
          value={principalAmount}
          onChange={(e) => onPrincipalChange(e.target.value.replace(/[^0-9]/g, ''))}
          placeholder="Ví dụ: 100000000"
          className={errors.principal_amount ? 'input-error' : ''}
        />
        {principalAmount && Number(principalAmount) > 0 && (
          <p className="typography-body-small text-muted-foreground">{formatVND(Number(principalAmount))}</p>
        )}
        {errors.principal_amount && <p className="typography-body-small text-financial-negative mt-1">{errors.principal_amount}</p>}
      </div>
      <div className="w-full space-y-2">
        <Label htmlFor="loan-start-date" className="typography-label-medium">Ngày giải ngân *</Label>
        <Input
          id="loan-start-date"
          type="date"
          value={startDate}
          onChange={(e) => onDateChange(e.target.value)}
          className={`h-11 ${errors.start_date ? 'input-error' : ''}`}
        />
        {errors.start_date && <p className="typography-body-small text-financial-negative mt-1">{errors.start_date}</p>}
      </div>
      <div className="w-full space-y-2">
        <Label className="typography-label-medium opacity-0 pointer-events-none">Tùy chọn</Label>
        <label
          htmlFor="loan-disburse-now"
          className="flex cursor-pointer select-none items-center gap-2.5 rounded-xl px-3 py-0.5 typography-label-small text-muted-foreground transition-all"
        >
          <Switch
            id="loan-disburse-now"
            checked={disburseNow}
            onCheckedChange={(checked) => onDisburseNowChange(checked === true)}
          />
          <span className="font-medium">Giải ngân ngay</span>
        </label>
      </div>
    </div>
  );
}
