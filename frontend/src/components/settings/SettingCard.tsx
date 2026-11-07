import { useId, useState } from 'react';
import { Check, X, AlertCircle, RefreshCw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
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
  displayMode?: 'default' | 'currency-vnd' | 'currency-slider' | 'account-number';
  // Step size for displayMode='currency-slider'. Defaults to 10.000.000 ₫ —
  // coarse round stops are the intended interaction for the Chuyển lô cap.
  sliderStep?: number;
  // Optional quick-stop values for displayMode='currency-slider'. Dragging a
  // slider to an exact figure is imprecise on touch, so these are rendered as
  // tappable chips. Values are raw digit strings, same shape as `value`.
  sliderPresets?: readonly string[];
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
  wholeNumber?: boolean;
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
  displayMode: 'default' | 'currency-vnd' | 'currency-slider' | 'account-number',
  wholeNumber: boolean,
): { isValid: boolean; errorMessage?: string } => {
  if (!value || value.trim() === '') {
    return { isValid: false, errorMessage: 'Giá trị không được để trống' };
  }
  if (displayMode === 'account-number') {
    if (!/^\d{6,24}$/.test(value)) {
      return { isValid: false, errorMessage: 'Số tài khoản chỉ gồm chữ số (6-24 số)' };
    }
    return { isValid: true };
  }
  if (displayMode === 'currency-vnd' || displayMode === 'currency-slider') {
    if (!/^\d+$/.test(value)) {
      return { isValid: false, errorMessage: 'Chỉ nhập số nguyên dương, không nhập số lẻ' };
    }
    const integerValue = BigInt(value);
    const minimum = BigInt(min);
    const maximum = BigInt(max);
    if (integerValue < minimum || integerValue > maximum) {
      return {
        isValid: false,
        errorMessage: `Giá trị phải từ ${formatVndDigits(String(min))} ₫ đến ${formatVndDigits(String(max))} ₫`,
      };
    }
    return { isValid: true };
  }
  if (type === 'number') {
    if (wholeNumber && !/^(0|[1-9]\d*)$/.test(value)) {
      return { isValid: false, errorMessage: 'Giá trị phải là số nguyên' };
    }
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
  sliderStep = 10_000_000,
  sliderPresets,
  wholeNumber = false,
}: SettingCardProps) => {
  const generatedId = useId().replace(/:/g, '');
  const inputId = `setting-${generatedId}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const [currencyInputError, setCurrencyInputError] = useState<string | null>(null);
  const validation = validateValue(value, type, min, max, displayMode, wholeNumber);
  const visibleError =
    unavailableMessage ?? errorMessage ?? currencyInputError ?? validation.errorMessage;
  const isUnavailable = Boolean(unavailableMessage);
  const isInvalid = isUnavailable || Boolean(currencyInputError) || !validation.isValid;
  const hasVisibleError = Boolean(visibleError);
  const isSliderMode = displayMode === 'currency-slider';
  const displayValue = displayMode === 'currency-vnd' ? formatVndDigits(value) : value;
  const visibleSuffix = displayMode === 'currency-vnd' ? '₫' : suffix;

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

    // displayValue still holds the previously rendered value here (the
    // component is controlled and React has not re-rendered yet), so an edit
    // that only drops trailing characters is a backspace/delete through the
    // formatted string. Its dots are formatter separators, not decimal
    // points — strip them and keep the remaining digits.
    const isTrailingDeletion =
      trimmedValue.length < displayValue.length && displayValue.startsWith(trimmedValue);
    const groups = trimmedValue.split('.');
    const digits = groups.join('');
    if (isTrailingDeletion && /^\d+$/.test(digits)) {
      setCurrencyInputError(null);
      onChange(normalizeIntegerDigits(digits));
      return;
    }

    // Dots are thousand separators in the vi-VN formatted display. Appending
    // a digit to "2.000" yields "2.0000", which is a legitimate mid-typing
    // state even though no group boundary lines up yet — accept any all-digit
    // value whose separator groups carry at least three digits, and reject
    // shorter tails ("2.5") as decimal intent.
    const isSeparatorForm = /^\d+$/.test(digits) && groups.slice(1).every((g) => g.length >= 3);
    if (isSeparatorForm) {
      setCurrencyInputError(null);
      onChange(normalizeIntegerDigits(digits));
      return;
    }

    setCurrencyInputError('Chỉ nhập số nguyên dương, không nhập số lẻ');
  };

  // Dense row grid (owner density directive 2026-10-12): label column, fixed
  // value column, fixed action column so rows never shift when actions mount.
  const rowGridClass = isSliderMode
    ? 'md:grid-cols-[minmax(0,1fr)_minmax(18rem,1.1fr)_6rem]'
    : 'md:grid-cols-[minmax(0,1fr)_13rem_6rem]';

  return (
    <div
      data-slot="setting-row"
      className={cn(
        'group min-w-0 px-4 py-2.5 transition-colors sm:px-5',
        isDirty && 'bg-warning/5',
      )}
    >
      <div
        className={cn(
          'grid min-w-0 items-center gap-x-4 gap-y-2 md:auto-rows-min',
          rowGridClass,
        )}
      >
        <div className="min-w-0">
          <Label
            htmlFor={inputId}
            className="block text-[13px] font-medium leading-5 text-foreground"
          >
            {title}
          </Label>
          {description && (
            <p
              id={descriptionId}
              className="text-xs leading-4 text-muted-foreground"
            >
              {description}
            </p>
          )}
        </div>

        {isSliderMode ? (
          <div className="min-w-0 space-y-1.5">
            <div className="flex items-center gap-3">
              <Slider
                aria-label={title}
                min={Number(min)}
                max={Number(max)}
                step={sliderStep}
                value={[Number(value) || Number(min)]}
                onValueChange={([next]) => {
                  setCurrencyInputError(null);
                  onChange(String(next));
                }}
                disabled={isSaving || isUnavailable}
                className="min-w-0 flex-1"
              />
              <span className="w-28 shrink-0 text-right text-sm font-semibold tabular-nums text-foreground">
                {formatVndDigits(value)}
                <span className="ml-0.5 text-xs font-medium text-muted-foreground">₫</span>
              </span>
            </div>
            <div className="flex items-center justify-between text-[11px] tabular-nums text-muted-foreground">
              <span>{formatVndDigits(String(min))} ₫</span>
              <span>{formatVndDigits(String(max))} ₫</span>
            </div>
            {/* Quick stops — precise values are hard to hit by dragging */}
            {sliderPresets && sliderPresets.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {sliderPresets.map((preset) => {
                  const isSelected = value === preset;
                  return (
                    <Button
                      key={preset}
                      type="button"
                      variant="outline"
                      size="sm"
                      aria-pressed={isSelected}
                      disabled={isSaving || isUnavailable}
                      onClick={() => {
                        setCurrencyInputError(null);
                        onChange(preset);
                      }}
                      className={cn(
                        'rounded-md px-2 font-semibold tabular-nums',
                        isSelected && 'border-primary bg-primary/5 text-primary',
                      )}
                    >
                      {formatVndDigits(preset)}
                    </Button>
                  );
                })}
              </div>
            )}
          </div>
        ) : (
          <div className="min-w-0 space-y-1">
            <div className="relative min-w-0">
              <Input
                id={inputId}
                name={inputId}
                autoComplete="off"
                type={
                  displayMode === 'currency-vnd' || displayMode === 'account-number'
                    ? 'text'
                    : type
                }
                inputMode={
                  displayMode === 'currency-vnd' || displayMode === 'account-number'
                    ? 'numeric'
                    : undefined
                }
                value={displayValue}
                onChange={(event) => handleValueChange(event.target.value)}
                disabled={isSaving || isUnavailable}
                className={cn(
                  'w-full bg-card tabular-nums',
                  visibleSuffix && 'pr-8 sm:pr-8',
                  displayMode === 'currency-vnd' && 'text-right',
                  displayMode === 'account-number',
                  hasVisibleError && 'border-destructive focus-visible:ring-destructive/20',
                  isDirty &&
                    !hasVisibleError &&
                    'border-warning focus-visible:ring-warning/20',
                  inputClassName,
                )}
                min={type === 'number' && displayMode === 'default' ? String(min) : undefined}
                max={type === 'number' && displayMode === 'default' ? String(max) : undefined}
                step={type === 'number' && displayMode === 'default' ? String(step) : undefined}
                aria-invalid={hasVisibleError}
                aria-describedby={
                  [description ? descriptionId : null, visibleError ? errorId : null]
                    .filter(Boolean)
                    .join(' ') || undefined
                }
              />
              {visibleSuffix && (
                <span className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground">
                  {visibleSuffix}
                </span>
              )}
            </div>
            {visibleError && (
              <div
                id={errorId}
                role="alert"
                aria-live="polite"
                className="flex min-w-0 items-start gap-1 text-destructive"
              >
                <AlertCircle aria-hidden="true" className="mt-0.5 h-3 w-3 shrink-0" />
                <span className="min-w-0 break-words text-xs">{visibleError}</span>
              </div>
            )}
          </div>
        )}

        <div className="flex items-center justify-end gap-1.5 md:min-w-[6rem]">
          {isUnavailable && onRetry ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={onRetry}
              disabled={isSaving}
              className="gap-1.5"
            >
              <RefreshCw aria-hidden="true" data-icon className="h-3.5 w-3.5" />
              Thử lại
            </Button>
          ) : isDirty ? (
            <>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label="Hoàn tác"
                title="Hoàn tác"
                onClick={() => {
                  setCurrencyInputError(null);
                  onReset();
                }}
                disabled={isSaving}
                className="text-muted-foreground hover:text-foreground"
              >
                <X aria-hidden="true" className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                aria-label="Lưu thay đổi"
                onClick={onSave}
                disabled={isSaving || isInvalid}
              >
                <Check aria-hidden="true" data-icon className="h-3.5 w-3.5" />
                <span aria-live="polite">{isSaving ? 'Đang lưu…' : 'Lưu'}</span>
              </Button>
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
};
