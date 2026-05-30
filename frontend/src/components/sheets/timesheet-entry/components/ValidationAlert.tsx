import { memo } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle } from 'lucide-react';

interface ValidationAlertProps {
  isValid: boolean;
  errors: string[];
}

export const ValidationAlert = memo(({ isValid, errors }: ValidationAlertProps) => {
  if (isValid || errors.length === 0) return null;

  return (
    <Alert variant="destructive">
      <AlertCircle className="h-4 w-4" />
      <AlertDescription>
        {typeof errors[0] === 'string' ? errors[0] : 'Có lỗi xảy ra'}
      </AlertDescription>
    </Alert>
  );
});

ValidationAlert.displayName = 'ValidationAlert';