import { useId, useState } from 'react';
import { Check, X, AlertCircle, RefreshCw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
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
  displayMode?: 'default' | 'currency-vnd';
  errorMessage?: string | null;
  unavailableMessage?: string | null;
  onRetry?: () => void;
  // For type='number': lower/upper bounds and step. min defaults to 0;
  // max defaults to 100 to preserve the legacy "percent" callers; step
  // defaults to 0.01. Pass max={undefined-friendly large value} for
  // open-ended numeric fields like the disbursement fee.
  min?: number | string;
  max?: number | string;
  step?: number;
}

const VND_FORMATTER = new Intl.NumberFormat('vi-VN');

const normalizeIntegerDigits = (value: string) => {
  const normalized = value.replace(/^0+(?=\d)/, '');
  return normalized || (value ? '0' : '');
};

export const formatVndDigits = (value: string) => {
  if (!/^\d+$/.test(value)) return value;
  return VND_FORMATTER.format(BigInt(value));
};

const validateValue = (
  value: string,
  type: 'text' | 'number',
  min: number | string,
  max: number | string,
  displayMode: 'default' | 'currency-vnd',
): { isValid: boolean; errorMessage?: string } => {
  if (!value || value.trim() === '') {
    return { isValid: false, errorMessage: 'Giá trị không được để trống' };
  }
  if (displayMode === 'currency-vnd') {
    if (!/^\d+$/.test(value)) {
      return { isValid: false, errorMessage: 'Chỉ nhập số nguyên dương, không nhập số lẻ' };
    }
    const integerValue = BigInt(value);
    const minimum = BigInt(min);
    const maximum = BigInt(max);
    if (integerValue < minimum || integerValue > maximum) {
      return {
        isValid: false,
        errorMessage: `Giá trị phải từ ${formatVndDigits(String(min))} đ đến ${formatVndDigits(String(max))} đ`,
      };
    }
    return { isValid: true };
  }
  if (type === 'number') {
    const numValue = parseFloat(value);
    const numericMin = Number(min);
    const numericMax = Number(max);
    if (isNaN(numValue)) return { isValid: false, errorMessage: 'Giá trị phải là số' };
    if (numValue < numericMin || numValue > numericMax)
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
  displayMode = 'default',
  errorMessage,
  unavailableMessage,
  onRetry,
  min = 0,
  max = 100,
  step = 0.01,
}: SettingCardProps) => {
  const generatedId = useId().replace(/:/g, '');
  const inputId = `setting-${generatedId}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const [currencyInputError, setCurrencyInputError] = useState<string | null>(null);
  const validation = validateValue(value, type, min, max, displayMode);
  const visibleError =
    unavailableMessage ?? errorMessage ?? currencyInputError ?? validation.errorMessage;
  const isUnavailable = Boolean(unavailableMessage);
  const isInvalid = isUnavailable || Boolean(currencyInputError) || !validation.isValid;
  const hasVisibleError = Boolean(visibleError);
  const displayValue = displayMode === 'currency-vnd' ? formatVndDigits(value) : value;
  const visibleSuffix = displayMode === 'currency-vnd' ? 'đ' : suffix;

  const handleValueChange = (nextValue: string) => {
    if (displayMode !== 'currency-vnd') {
      onChange(nextValue);
      return;
    }

    const trimmedValue = nextValue.trim();
    if (trimmedValue === '') {
      setCurrencyInputError(null);
      onChange('');
      return;
    }

    if (/^\d+$/.test(trimmedValue)) {
      setCurrencyInputError(null);
      onChange(normalizeIntegerDigits(trimmedValue));
      return;
    }

    if (/^\d{1,3}(?:\.\d{3})+$/.test(trimmedValue)) {
      setCurrencyInputError(null);
      onChange(normalizeIntegerDigits(trimmedValue.replace(/\./g, '')));
      return;
    }

    setCurrencyInputError('Chỉ nhập số nguyên dương, không nhập số lẻ');
  };

  return (
    <div className="group min-w-0 overflow-hidden rounded-xl border bg-card p-4 transition-colors hover:border-border/80 sm:p-5">
      <div className="space-y-0.5">
        <Label htmlFor={inputId} className="block break-words text-sm font-semibold text-foreground">
          {title}
        </Label>
        {description && (
          <p
            id={descriptionId}
            className="break-words text-xs leading-relaxed text-muted-foreground"
          >
            {description}
          </p>
        )}
      </div>

      <div className="mt-3 space-y-2">
        <div className="flex items-center gap-2">
          <div className="relative min-w-0 flex-1">
            <Input
              id={inputId}
              name={inputId}
              autoComplete="off"
              type={displayMode === 'currency-vnd' ? 'text' : type}
              inputMode={displayMode === 'currency-vnd' ? 'numeric' : undefined}
              value={displayValue}
              onChange={(event) => handleValueChange(event.target.value)}
              disabled={isSaving || isUnavailable}
              className={cn(
                'h-11 min-w-0 text-base font-medium',
                visibleSuffix && 'pr-8',
                displayMode === 'currency-vnd' && 'text-right tabular-nums',
                hasVisibleError && 'border-destructive focus-visible:ring-destructive/20',
                isDirty && !hasVisibleError && 'border-amber-400 focus-visible:ring-amber-400/20',
                inputClassName
              )}
              min={type === 'number' && displayMode === 'default' ? String(min) : undefined}
              max={type === 'number' && displayMode === 'default' ? String(max) : undefined}
              step={type === 'number' && displayMode === 'default' ? String(step) : undefined}
              aria-invalid={hasVisibleError}
              aria-describedby={[
                description ? descriptionId : null,
                visibleError ? errorId : null,
              ].filter(Boolean).join(' ') || undefined}
            />
            {visibleSuffix && (
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground">
                {visibleSuffix}
              </span>
            )}
          </div>
        </div>

        {visibleError && (
          <div
            id={errorId}
            role="alert"
            aria-live="polite"
            className="flex min-w-0 items-start gap-1.5 text-destructive"
          >
            <AlertCircle aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span className="min-w-0 break-words text-xs">{visibleError}</span>
          </div>
        )}

        {isUnavailable && onRetry ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onRetry}
            disabled={isSaving}
            className="h-11 w-full gap-1.5 text-xs"
          >
            <RefreshCw aria-hidden="true" className="h-3.5 w-3.5" />
            Thử lại
          </Button>
        ) : isDirty ? (
          <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() => {
                setCurrencyInputError(null);
                onReset();
              }}
              disabled={isSaving}
              className="h-11 w-full gap-1.5 text-xs text-muted-foreground hover:text-foreground"
            >
              <X aria-hidden="true" className="h-3 w-3" />
              Hủy
            </Button>
            <Button
              type="button"
              size="sm"
              variant="default"
              onClick={onSave}
              disabled={isSaving || isInvalid}
              className="h-11 w-full gap-1.5 bg-emerald-600 text-xs text-white hover:bg-emerald-700"
            >
              <Check aria-hidden="true" className="h-3 w-3" />
              <span aria-live="polite">{isSaving ? 'Đang lưu…' : 'Lưu'}</span>
            </Button>
          </div>
        ) : null}
      </div>
    </div>
  );
};
