import { format } from "date-fns";
import { XCircle } from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';

interface TimesheetStatusInfoProps {
  timesheet: Timesheet;
}

export function TimesheetStatusInfo({ timesheet }: TimesheetStatusInfoProps) {
  return (
    <>
      {/* Rejection Reason */}
      {timesheet.status === 'rejected' && timesheet.rejection_reason && (
        <div className="bg-red-50 rounded-xl p-3">
          <div className="flex items-center gap-2 mb-2">
            <XCircle className="w-4 h-4 text-red-600" />
            <span className="typography-body-medium text-red-700">
              Lý do loại
            </span>
          </div>
          <p className="typography-body-medium text-red-600">
            {timesheet.rejection_reason}
          </p>
        </div>
      )}

      {/* Approval Date */}
      {timesheet.status === 'approved' && timesheet.approved_at && (
        <div className="text-center typography-body-medium text-green-700">
          Đã duyệt ngày {format(new Date(timesheet.approved_at), 'dd/MM/yyyy')}
        </div>
      )}
    </>
  );
}
