import { format } from "date-fns";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CalendarRange } from "lucide-react";

interface SalaryPeriodFieldsProps {
  salaryPeriodFrom: number | null;
  salaryPeriodTo: number | null;
  onSalaryPeriodFromChange: (value: number | null) => void;
  onSalaryPeriodToChange: (value: number | null) => void;
  disabled?: boolean;
  variant?: 'detailed' | 'compact';
}

export function SalaryPeriodFields({
  salaryPeriodFrom,
  salaryPeriodTo,
  onSalaryPeriodFromChange,
  onSalaryPeriodToChange,
  disabled = false,
  variant = 'detailed'
}: SalaryPeriodFieldsProps) {
  const handleFromChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    if (value === '') {
      onSalaryPeriodFromChange(null);
    } else {
      const num = parseInt(value);
      onSalaryPeriodFromChange(isNaN(num) ? null : num);
    }
  };

  const handleToChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    if (value === '') {
      onSalaryPeriodToChange(null);
    } else {
      const num = parseInt(value);
      onSalaryPeriodToChange(isNaN(num) ? null : num);
    }
  };

  // Use only the compact variant with enhanced examples
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 pb-2 border-b">
        <CalendarRange className="h-4 w-4 text-primary" />
        <h4 className="typography-body-large font-medium">Kỳ lương tháng</h4>
      </div>

      <p className="typography-body-medium text-muted-foreground">
      Nhập 0 = cuối tháng. Để trống = từ ngày 01 đến cuối tháng.
      </p>

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label htmlFor="salary_period_from" className="text-sm font-medium">
            Từ ngày
          </Label>
          <Input
            id="salary_period_from"
            type="number"
            min="0"
            max="31"
            value={salaryPeriodFrom ?? ''}
            onChange={handleFromChange}
            className="transition-colors"
            placeholder="0 = đầu tháng"
            disabled={disabled}
          />
          {salaryPeriodFrom !== null && salaryPeriodFrom >= 0 && salaryPeriodFrom <= 28 ? (
            <p className="text-xs text-muted-foreground">
              {salaryPeriodFrom === 0
                ? `Ngày đầu tháng này, ví dụ: ${format(new Date(new Date().getFullYear(), new Date().getMonth(), 1), 'dd/MM/yyyy')}`
                : `Ngày tháng trước, ví dụ: ${format(new Date(new Date().getFullYear(), new Date().getMonth() - 1, salaryPeriodFrom), 'dd/MM/yyyy')}`
              }
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">Ngày đầu tháng này, ví dụ: 01/{String(new Date().getMonth() + 1).padStart(2, '0')}/{new Date().getFullYear()}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="salary_period_to" className="text-sm font-medium">
            Đến ngày
          </Label>
          <Input
            id="salary_period_to"
            type="number"
            min="0"
            max="31"
            value={salaryPeriodTo ?? ''}
            onChange={handleToChange}
            className="transition-colors"
            placeholder="0 = cuối tháng"
            disabled={disabled}
          />
          {salaryPeriodTo !== null && salaryPeriodTo >= 0 && salaryPeriodTo <= 28 ? (
            <p className="text-xs text-muted-foreground">
              {salaryPeriodTo === 0
                ? `Ngày cuối tháng này, ví dụ: ${format(new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0), 'dd/MM/yyyy')}`
                : `Ngày tháng này, ví dụ: ${format(new Date(new Date().getFullYear(), new Date().getMonth(), salaryPeriodTo), 'dd/MM/yyyy')}`
              }
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">Ngày cuối tháng này, ví dụ: {format(new Date(new Date().getFullYear(), new Date().getMonth() + 1, 0), 'dd/MM/yyyy')}</p>
          )}
        </div>
      </div>
    </div>
  );
}
