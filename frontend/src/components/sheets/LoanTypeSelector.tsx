import { useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import type { FormLoanType } from '@/types/api/loan.types';

interface LoanTypeSelectorProps {
  loanType: FormLoanType;
  onChange: (type: FormLoanType) => void;
}

export function LoanTypeSelector({ loanType, onChange }: LoanTypeSelectorProps) {
  const handleSelectAutoInterest = useCallback(() => {
    onChange('bullet_loan');
  }, [onChange]);

  const handleSelectAmortization = useCallback(() => {
    onChange('amortization');
  }, [onChange]);

  const handleSelectCustomSchedule = useCallback(() => {
    onChange('custom_schedule');
  }, [onChange]);

  return (
    <div className="space-y-2">
      <Label className="typography-label-medium">Loại khoản vay *</Label>
      <div
        className="flex w-full flex-col overflow-hidden rounded-xl border border-border bg-card/80 sm:flex-row"
        role="group"
        aria-label="Chọn loại khoản vay"
      >
        <Button
          type="button"
          variant={loanType === 'bullet_loan' ? 'default' : 'ghost'}
          onClick={handleSelectAutoInterest}
          aria-pressed={loanType === 'bullet_loan'}
          className={`flex-1 min-h-[44px] rounded-none font-medium transition-colors ${
            loanType === 'bullet_loan'
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-card text-muted-foreground hover:text-foreground'
          }`}
        >
          Lãi trước gốc sau
        </Button>
        <Button
          type="button"
          variant={loanType === 'amortization' ? 'default' : 'ghost'}
          onClick={handleSelectAmortization}
          aria-pressed={loanType === 'amortization'}
          className={`flex-1 min-h-[44px] rounded-none font-medium transition-colors border-t border-border sm:border-t-0 sm:border-l ${
            loanType === 'amortization'
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-card text-muted-foreground hover:text-foreground'
          }`}
        >
          Dư nợ giảm dần
        </Button>
        <Button
          type="button"
          variant={loanType === 'custom_schedule' ? 'default' : 'ghost'}
          onClick={handleSelectCustomSchedule}
          aria-pressed={loanType === 'custom_schedule'}
          className={`flex-1 min-h-[44px] rounded-none font-medium transition-colors border-t border-border sm:border-t-0 sm:border-l ${
            loanType === 'custom_schedule'
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-card text-muted-foreground hover:text-foreground'
          }`}
        >
          Tự thiết lập
        </Button>
      </div>
    </div>
  );
}
