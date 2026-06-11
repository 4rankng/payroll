import { Timesheet } from '@/types/api/timesheet.types';
import { formatCurrency } from '@/utils/formatters';

interface TimesheetPaymentInfoProps {
  timesheet: Timesheet;
}

export function TimesheetPaymentInfo({ timesheet }: TimesheetPaymentInfoProps) {
  return (
    <div className="bg-green-50 rounded-xl p-4">
      <div className="text-center">
        <p className="typography-body-medium text-green-700 mb-1">Tổng tiền</p>
        <p className="typography-display-small text-green-600">
          {formatCurrency(timesheet.amount)}
        </p>
        <p className="typography-body-small text-green-600 mt-2">
          {timesheet.hours_worked} giờ × {formatCurrency(timesheet.payrate)}
        </p>
      </div>
    </div>
  );
}
