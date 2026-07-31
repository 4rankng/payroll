import { useState, useEffect } from 'react';
import { ExternalLink, AlertCircle, Check, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { assetService } from '@/services/api/asset.service';

interface UrlInputProps {
  value?: string;
  onChange: (url: string) => void;
  label?: string;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  required?: boolean;
  error?: string;
  helperText?: string;
}

export function UrlInput({
  value = '',
  onChange,
  label = 'Link chứng từ',
  placeholder = 'https://example.com/receipt.jpg',
  className,
  disabled = false,
  required = false,
  error: externalError,
  helperText = 'Nhập link đầy đủ tới file chứng từ (ví dụ: Google Drive, Dropbox, hoặc website khác)',
}: UrlInputProps) {
  const [inputValue, setInputValue] = useState(value);
  const [validationError, setValidationError] = useState<string>('');
  const [isValid, setIsValid] = useState<boolean | null>(null);

  // Update input when external value changes
  useEffect(() => {
    setInputValue(value);
    if (value) {
      validateUrl(value);
    } else {
      setIsValid(null);
      setValidationError('');
    }
  }, [value]);

  const validateUrl = (url: string) => {
    if (!url.trim()) {
      setIsValid(null);
      setValidationError('');
      return;
    }

    const validation = assetService.validateEvidenceUrl(url);
    if (validation.isValid) {
      setIsValid(true);
      setValidationError('');
    } else {
      setIsValid(false);
      setValidationError(validation.error || 'URL không hợp lệ');
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setInputValue(newValue);
    validateUrl(newValue);
  };

  const handleBlur = () => {
    onChange(inputValue.trim());
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onChange(inputValue.trim());
    }
  };

  const clearUrl = () => {
    setInputValue('');
    setIsValid(null);
    setValidationError('');
    onChange('');
  };

  const openUrl = () => {
    if (inputValue && isValid) {
      window.open(inputValue, '_blank', 'noopener,noreferrer');
    }
  };

  const displayError = externalError || validationError;

  return (
    <div className={cn("space-y-2", className)}>
      {label && (
        <Label htmlFor="url-input" className="typography-body-medium text-gray-700">
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </Label>
      )}

      <div className="relative">
        <Input
          id="url-input"
          type="url"
          value={inputValue}
          onChange={handleInputChange}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          className={cn(
            "min-h-11 pr-28",
            isValid === true && "border-green-300 focus:border-green-400 focus:ring-green-100",
            isValid === false && "border-red-300 focus:border-red-400 focus:ring-red-100",
            displayError && "border-red-300 focus:border-red-400 focus:ring-red-100"
          )}
        />

        {/* Action buttons */}
        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
          {/* Validation status */}
          {isValid === true && (
            <div className="text-green-500" title="URL hợp lệ">
              <Check className="w-4 h-4" />
            </div>
          )}
          
          {isValid === false && (
            <div className="text-red-500" title="URL không hợp lệ">
              <AlertCircle className="w-4 h-4" />
            </div>
          )}

          {/* Open URL button */}
          {inputValue && isValid && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={openUrl}
              disabled={disabled}
              className="h-11 w-11 p-0 hover:bg-blue-50 hover:text-blue-600"
              title="Mở link trong tab mới"
              aria-label="Mở đường dẫn chứng từ"
            >
              <ExternalLink className="w-3 h-3" />
            </Button>
          )}

          {/* Clear button */}
          {inputValue && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={clearUrl}
              disabled={disabled}
              className="h-11 w-11 p-0 hover:bg-gray-100 hover:text-gray-600"
              title="Xóa URL"
              aria-label="Xóa đường dẫn chứng từ"
            >
              <X className="w-3 h-3" />
            </Button>
          )}
        </div>
      </div>

      {/* Error message */}
      {displayError && (
        <div className="flex items-center gap-1 typography-body-medium text-red-600">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <p>{displayError}</p>
        </div>
      )}

      {/* Help text */}
      {!displayError && (
        <p className="typography-body-small text-gray-500">
          {helperText}
        </p>
      )}

      {/* URL preview */}
      {inputValue && isValid && (
        <div className="p-2 bg-gray-50 rounded-md">
          <p className="typography-body-small text-gray-600 font-medium mb-1">Xem trước:</p>
          <div className="flex items-center gap-2">
            <ExternalLink className="w-3 h-3 text-gray-400 flex-shrink-0" />
            <p className="typography-body-small text-gray-700 truncate" title={inputValue}>
              {inputValue}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
