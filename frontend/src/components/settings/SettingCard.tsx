import { Check, X, AlertCircle } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface SettingCardProps {
  title: string;
  description?: string;
  value: string;
  originalValue: string;
  onChange: (value: string) => void;
  onSave: () => void;
  onReset: () => void;
  isDirty: boolean;
  isSaving: boolean;
  type?: 'text' | 'number';
  suffix?: string;
  inputClassName?: string;
  // For type='number': lower/upper bounds and step. min defaults to 0;
  // max defaults to 100 to preserve the legacy "percent" callers; step
  // defaults to 0.01. Pass max={undefined-friendly large value} for
  // open-ended numeric fields like the disbursement fee.
  min?: number;
  max?: number;
  step?: number;
}

const validateValue = (
  value: string,
  type: 'text' | 'number',
  min: number,
  max: number
): { isValid: boolean; errorMessage?: string } => {
  if (!value || value.trim() === '') {
    return { isValid: false, errorMessage: 'Giá trị không được để trống' };
  }
  if (type === 'number') {
    const numValue = parseFloat(value);
    if (isNaN(numValue)) return { isValid: false, errorMessage: 'Giá trị phải là số' };
    if (numValue < min || numValue > max)
      return { isValid: false, errorMessage: `Giá trị phải từ ${min} đến ${max}` };
  }
  return { isValid: true };
};

export const SettingCard = ({
  title,
  description,
  value,
  originalValue,
  onChange,
  onSave,
  onReset,
  isDirty,
  isSaving,
  type = 'text',
  suffix,
  inputClassName,
  min = 0,
  max = 100,
  step = 0.01,
}: SettingCardProps) => {
  const validation = validateValue(value, type, min, max);
  const isInvalid = !validation.isValid;

  return (
    <div className="group flex flex-col gap-3 rounded-xl border bg-card p-5 transition-all hover:border-border/80 hover:shadow-sm">
      <div className="space-y-0.5">
        <p className="text-sm font-semibold text-foreground">{title}</p>
        {description && (
          <p className="text-xs text-muted-foreground leading-relaxed">{description}</p>
        )}
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Input
              type={type}
              value={value}
              onChange={(e) => onChange(e.target.value)}
              className={cn(
                'h-9 text-sm font-medium',
                suffix && 'pr-8',
                isInvalid && 'border-destructive focus-visible:ring-destructive/20',
                isDirty && !isInvalid && 'border-amber-400 focus-visible:ring-amber-400/20',
                inputClassName
              )}
              min={type === 'number' ? String(min) : undefined}
              max={type === 'number' ? String(max) : undefined}
              step={type === 'number' ? String(step) : undefined}
              aria-invalid={isInvalid}
            />
            {suffix && (
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground">
                {suffix}
              </span>
            )}
          </div>
        </div>

        {isInvalid && (
          <div className="flex items-center gap-1.5 text-destructive animate-in fade-in duration-150">
            <AlertCircle className="h-3 w-3 shrink-0" />
            <span className="text-xs">{validation.errorMessage}</span>
          </div>
        )}

        {isDirty && (
          <div className="flex items-center gap-2 animate-in fade-in slide-in-from-bottom-1 duration-150">
            <Button
              size="sm"
              variant="ghost"
              onClick={onReset}
              disabled={isSaving}
              className="h-7 flex-1 gap-1.5 text-xs text-muted-foreground hover:text-foreground"
            >
              <X className="h-3 w-3" />
              Hủy
            </Button>
            <Button
              size="sm"
              variant="default"
              onClick={onSave}
              disabled={isSaving || isInvalid}
              className="h-7 flex-1 gap-1.5 bg-emerald-600 text-xs hover:bg-emerald-700 text-white"
            >
              <Check className="h-3 w-3" />
              Lưu
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};
