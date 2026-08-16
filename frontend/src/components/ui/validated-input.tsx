import { useState, useEffect, forwardRef } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { AlertCircle, CheckCircle } from 'lucide-react';
import {
  validateCCCD,
  validatePhoneNumber,
  validateEmail,
  validateDate,
  validateWorkingHours,
  validateCurrency,
  formatCurrency,
  validateProjectCode,
  validateBankAccount,
  validateSWIFTCode
} from '@/lib/validation';

export type ValidationRule =
  | 'cccd'
  | 'phone'
  | 'email'
  | 'date'
  | 'workingHours'
  | 'currency'
  | 'projectCode'
  | 'bankAccount'
  | 'swift'
  | 'required'
  | 'custom';

interface ValidatedInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type' | 'value' | 'onChange'> {
  label?: string;
  value: string | number;
  onChange: (value: string | number) => void;
  validationRule?: ValidationRule;
  customValidator?: (value: unknown) => { valid: boolean; error?: string };
  showValidation?: boolean;
  formatOnBlur?: boolean;
  debounceMs?: number;
  helperText?: string;
  required?: boolean;
}

export const ValidatedInput = forwardRef<HTMLInputElement, ValidatedInputProps>(
  ({
    label,
    value,
    onChange,
    validationRule,
    customValidator,
    showValidation = true,
    formatOnBlur = false,
    debounceMs = 300,
    helperText,
    required = false,
    className,
    disabled,
    ...props
  }, ref) => {
    const [localValue, setLocalValue] = useState(value?.toString() || '');
    const [validation, setValidation] = useState<{ valid: boolean; error?: string }>({ valid: true });
    const [isTouched, setIsTouched] = useState(false);
    const [debounceTimer, setDebounceTimer] = useState<NodeJS.Timeout | null>(null);

    useEffect(() => {
      setLocalValue(value?.toString() || '');
    }, [value]);

    const validateValue = (val: string) => {
      if (!val && required) {
        return { valid: false, error: `${label || 'Trường này'} là bắt buộc` };
      }

      if (!val && !required) {
        return { valid: true };
      }

      switch (validationRule) {
        case 'cccd':
          return validateCCCD(val);
        case 'phone':
          return validatePhoneNumber(val);
        case 'email':
          return validateEmail(val);
        case 'date':
          return validateDate(val);
        case 'workingHours':
          return validateWorkingHours(parseFloat(val));
        case 'currency':
          return validateCurrency(val);
        case 'projectCode':
          return validateProjectCode(val);
        case 'bankAccount':
          return validateBankAccount(val);
        case 'swift':
          return validateSWIFTCode(val);
        case 'custom':
          return customValidator ? customValidator(val) : { valid: true };
        case 'required':
          return val ? { valid: true } : { valid: false, error: `${label || 'Trường này'} là bắt buộc` };
        default:
          return { valid: true };
      }
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setLocalValue(newValue);

      if (debounceTimer) {
        clearTimeout(debounceTimer);
      }

      const timer = setTimeout(() => {
        onChange(newValue);
        if (isTouched) {
          setValidation(validateValue(newValue));
        }
      }, debounceMs);

      setDebounceTimer(timer);
    };

    const handleBlur = () => {
      setIsTouched(true);
      setValidation(validateValue(localValue));

      if (formatOnBlur && validationRule === 'currency' && localValue) {
        const numValue = parseInt(localValue.replace(/[.,]/g, ''));
        if (!isNaN(numValue)) {
          const formatted = formatCurrency(numValue);
          setLocalValue(formatted);
          onChange(numValue);
        }
      }

      if (formatOnBlur && validationRule === 'cccd' && localValue) {
        // Format CCCD with spaces for readability
        const cleaned = localValue.replace(/\s/g, '');
        if (cleaned.length === 12) {
          const formatted = `${cleaned.slice(0, 3)} ${cleaned.slice(3, 6)} ${cleaned.slice(6, 9)} ${cleaned.slice(9)}`;
          setLocalValue(formatted);
        }
      }

      if (formatOnBlur && validationRule === 'phone' && localValue) {
        // Format phone number for readability
        const cleaned = localValue.replace(/[\s.-]/g, '');
        if (cleaned.length === 10 && cleaned.startsWith('0')) {
          const formatted = `${cleaned.slice(0, 4)} ${cleaned.slice(4, 7)} ${cleaned.slice(7)}`;
          setLocalValue(formatted);
        }
      }
    };

    const getInputType = () => {
      switch (validationRule) {
        case 'email':
          return 'email';
        case 'workingHours':
        case 'currency':
          return 'number';
        case 'date':
          return 'text';
        default:
          return 'text';
      }
    };

    const getPlaceholder = () => {
      if (props.placeholder) return props.placeholder;

      switch (validationRule) {
        case 'cccd':
          return '001 234 567 890';
        case 'phone':
          return '0901 234 567';
        case 'email':
          return 'example@company.com';
        case 'date':
          return 'DD/MM/YYYY';
        case 'workingHours':
          return '0-24';
        case 'currency':
          return '1.000.000 ₫';
        case 'projectCode':
          return 'PRJ-2025-001';
        case 'bankAccount':
          return '1234567890';
        case 'swift':
          return 'ABCDVNVX';
        default:
          return '';
      }
    };

    const showError = showValidation && isTouched && !validation.valid;
    const showSuccess = showValidation && isTouched && validation.valid && localValue;

    return (
      <div className="space-y-2">
        {label && (
          <Label htmlFor={props.id} className={cn(required && "after:content-['*'] after:ml-0.5 after:text-destructive")}>
            {label}
          </Label>
        )}
        <div className="relative">
          <Input
            ref={ref}
            type={getInputType()}
            value={localValue}
            onChange={handleChange}
            onBlur={handleBlur}
            placeholder={getPlaceholder()}
            className={cn(
              showError && 'border-destructive focus-visible:ring-destructive',
              showSuccess && 'border-success focus-visible:ring-success',
              'pr-10',
              className
            )}
            disabled={disabled}
            {...props}
          />
          {showValidation && isTouched && localValue && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2">
              {showError ? (
                <AlertCircle className="h-4 w-4 text-destructive" />
              ) : showSuccess ? (
                <CheckCircle className="h-4 w-4 text-success" />
              ) : null}
            </div>
          )}
        </div>
        {(showError || helperText) && (
          <p className={cn(
            'typography-body-medium',
            showError ? 'text-destructive' : 'text-muted-foreground'
          )}>
            {showError ? validation.error : helperText}
          </p>
        )}
      </div>
    );
  }
);

ValidatedInput.displayName = 'ValidatedInput';
