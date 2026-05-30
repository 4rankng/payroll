import { AlertTriangle } from 'lucide-react';

interface ValidationError {
  employeeId: number;
  totalHours: number;
  message: string;
}

interface MobileValidationAlertProps {
  validationErrors: ValidationError[];
}

export function MobileValidationAlert({ validationErrors }: MobileValidationAlertProps) {
  if (validationErrors.length === 0) return null;

  return (
    <div className="mx-2 sm:mx-4 p-2 sm:p-3 border border-red-200 bg-red-50 rounded-xl">
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0" />
        <div>
          <div className="font-medium text-red-800 typography-body-medium">
            Có {validationErrors.length} nhân viên vượt quá giới hạn 24 giờ/ngày
          </div>
          <div className="typography-body-small sm:typography-body-medium text-red-700">
            Vui lòng điều chỉnh số giờ làm việc trước khi lưu bảng công.
          </div>
        </div>
      </div>
    </div>
  );
}