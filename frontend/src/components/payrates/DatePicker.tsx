import { memo } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AlertTriangle } from 'lucide-react';

interface DatePickerProps {
  fromDate: string;
  onFromDateChange: (date: string) => void;
  fromErrors?: string[];
  fromWarnings?: string[];
  disabled?: boolean;
  required?: boolean;
  className?: string;
}

export const DatePicker = memo(function DatePicker({
  fromDate,
  onFromDateChange,
  fromErrors = [],
  fromWarnings = [],
  disabled = false,
  required = true,
  className = ""
}: DatePickerProps) {
  const today = new Date().toISOString().split('T')[0];

  const hasFromErrors = fromErrors.length > 0;
  const hasFromWarnings = fromWarnings.length > 0;

  return (
    <div className={`space-y-2 ${className}`}>
      {/* Header Row with Title and Date Field */}
      <div className="flex items-center gap-4">


        <div className="flex items-center gap-2">
          <Label htmlFor="effective-from" className="typography-body-medium text-muted-foreground shrink-0">
            Có hiệu lực từ
            {required && <span className="text-red-600 ml-1" aria-label="bắt buộc">*</span>}
          </Label>
          <Input
            id="effective-from"
            type="date"
            value={fromDate}
            onChange={(e) => onFromDateChange(e.target.value)}
            className={`h-10 w-auto ${hasFromErrors ? 'border-red-500 focus:ring-red-500' : hasFromWarnings ? 'border-yellow-500 focus:ring-yellow-500' : ''}`}
            disabled={disabled}
            aria-describedby="effective-from-help"
            aria-invalid={hasFromErrors}
            min={today}
          />
        </div>
      </div>

      <div id="effective-from-help" className="sr-only">
        Ngày bắt đầu áp dụng mức lương này. Định dạng: DD/MM/YYYY
      </div>

      {hasFromErrors && (
        <div className="space-y-1">
          {fromErrors.map((error, index) => (
            <div key={index} className="flex items-start gap-2 typography-body-medium text-red-600">
              <AlertTriangle className="w-3 h-3 mt-0.5 flex-shrink-0" />
              <span>{error}</span>
            </div>
          ))}
        </div>
      )}

      {!hasFromErrors && hasFromWarnings && (
        <div className="space-y-1">
          {fromWarnings.map((warning, index) => (
            <div key={index} className="flex items-start gap-2 typography-body-medium text-yellow-700">
              <AlertTriangle className="w-3 h-3 mt-0.5 flex-shrink-0" />
              <span>{warning}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
});
