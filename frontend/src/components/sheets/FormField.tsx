import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { formatVND } from '@/utils/loanHelpers';

interface FormFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: 'text' | 'number' | 'date' | 'decimal';
  errors?: Record<string, string>;
  errorKey?: string;
  formatAsVND?: boolean;
  className?: string;
  inputMode?: 'numeric' | 'decimal' | 'text';
  disabled?: boolean;
}

export function FormField({
  label,
  value,
  onChange,
  placeholder,
  type = 'text',
  errors,
  errorKey,
  formatAsVND = false,
  className = '',
  inputMode = 'text',
  disabled = false
}: FormFieldProps) {
  const error = errorKey ? errors?.[errorKey] : undefined;
  
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let newValue = e.target.value;
    
    if (type === 'number') {
      newValue = newValue.replace(/[^0-9]/g, '');
    } else if (type === 'decimal') {
      newValue = newValue.replace(/[^0-9.]/g, '');
    }
    
    onChange(newValue);
  };

  return (
    <div className={`space-y-2 ${className}`}>
      <Label className="typography-label-medium">{label}</Label>
      <Input
        inputMode={inputMode}
        value={value}
        onChange={handleChange}
        placeholder={placeholder}
        type={type === 'date' ? 'date' : 'text'}
        className={error ? 'input-error' : ''}
        disabled={disabled}
      />
      {formatAsVND && value && Number(value) > 0 && (
        <p className="typography-body-small text-muted-foreground">{formatVND(Number(value))}</p>
      )}
      {error && <p className="typography-body-small text-financial-negative mt-1">{error}</p>}
    </div>
  );
}