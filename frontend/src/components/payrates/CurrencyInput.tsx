import { useState, useCallback, forwardRef } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { AlertTriangle } from 'lucide-react';

interface CurrencyInputProps {
  id?: string;
  label?: string;
  value: number;
  onChange: (value: number) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  required?: boolean;
  min?: number;
  max?: number;
  step?: number;
  error?: string;
  warning?: string;
  'aria-label'?: string;
  'aria-describedby'?: string;
}

export const CurrencyInput = forwardRef<HTMLInputElement, CurrencyInputProps>(
  ({
    id,
    label,
    value,
    onChange,
    placeholder = "Nhập số tiền...",
    className = "",
    disabled = false,
    required = false,
    min = 0,
    max,
    step = 1000,
    error,
    warning,
    'aria-label': ariaLabel,
    'aria-describedby': ariaDescribedBy,
  }, ref) => {
    const formatCurrency = useCallback((amount: number): string => {
      return amount.toLocaleString('vi-VN');
    }, []);

    const [displayValue, setDisplayValue] = useState(formatCurrency(value));
    const [focused, setFocused] = useState(false);

    const parseCurrency = useCallback((value: string): number => {
      const cleanedValue = value.replace(/[^\d]/g, '');
      return parseInt(cleanedValue) || 0;
    }, []);

    const handleFocus = useCallback(() => {
      setFocused(true);
      setDisplayValue(value.toString());
    }, [value]);

    const handleBlur = useCallback(() => {
      setFocused(false);
      setDisplayValue(formatCurrency(value));
    }, [value, formatCurrency]);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
      const rawValue = e.target.value;
      setDisplayValue(rawValue);

      const numericValue = parseCurrency(rawValue);

      // Apply min/max constraints
      let constrainedValue = numericValue;
      if (min !== undefined) {
        constrainedValue = Math.max(constrainedValue, min);
      }
      if (max !== undefined) {
        constrainedValue = Math.min(constrainedValue, max);
      }

      onChange(constrainedValue);
    }, [onChange, parseCurrency, min, max]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
      // Allow navigation keys
      if (['Backspace', 'Delete', 'Tab', 'Escape', 'Enter', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
        return;
      }

      // Allow Ctrl+A, Ctrl+C, Ctrl+V, etc.
      if (e.ctrlKey || e.metaKey) {
        return;
      }

      // Only allow digits
      if (!/^\d$/.test(e.key)) {
        e.preventDefault();
      }
    }, []);

    const inputClassName = `
      ${className}
      ${error ? 'border-red-500 focus:ring-red-500' : ''}
      ${warning ? 'border-yellow-500 focus:ring-yellow-500' : ''}
    `.trim();

    return (
      <div className="space-y-2">
        {label && (
          <Label htmlFor={id} className="typography-body-medium text-foreground">
            {label}
            {required && <span className="text-red-500 ml-1" aria-label="bắt buộc">*</span>}
          </Label>
        )}

        <div className="relative">
          <Input
            ref={ref}
            id={id}
            type="text"
            value={displayValue}
            onChange={handleChange}
            onFocus={handleFocus}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            className={inputClassName}
            disabled={disabled}
            required={required}
            aria-label={ariaLabel || (label ? `${label} (₫)` : 'Số tiền ₫')}
            aria-describedby={ariaDescribedBy}
            aria-invalid={!!error}
          />

          {!focused && (
            <div className="absolute inset-y-0 right-3 flex items-center pointer-events-none">
              <span className="typography-body-medium text-muted-foreground">₫</span>
            </div>
          )}
        </div>

        {error && (
          <div className="flex items-start gap-2 typography-body-medium text-red-600">
            <AlertTriangle className="w-3 h-3 mt-0.5 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {!error && warning && (
          <div className="flex items-start gap-2 typography-body-medium text-yellow-600">
            <AlertTriangle className="w-3 h-3 mt-0.5 flex-shrink-0" />
            <span>{warning}</span>
          </div>
        )}

        {/* Screen reader helper */}
        <div className="sr-only">
          Nhập số tiền bằng ₫. Ví dụ: 30000 cho 30.000 ₫
        </div>
      </div>
    );
  }
);

CurrencyInput.displayName = 'CurrencyInput';
