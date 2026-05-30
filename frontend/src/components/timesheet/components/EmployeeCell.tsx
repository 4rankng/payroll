import { UserAvatar } from '@/components/ui/user-avatar';
import type { TimesheetEditRequest } from '@/types/api/timesheet.types';

interface EmployeeCellProps {
  request: TimesheetEditRequest;
}

export const EmployeeCell = ({ request }: EmployeeCellProps) => {
  const timesheet = request.timesheet;

  if (!timesheet) return <span className="text-muted-foreground">N/A</span>;

  return (
    <div className="flex items-center gap-3">
      <UserAvatar
        email={timesheet.employee_code || timesheet.employee_id.toString()}
        name={timesheet.employee_name || ''}
        size="sm"
      />
      <div>
        <p className="typography-body-medium font-medium text-foreground">
          {timesheet.employee_name || '-'}
        </p>
        <p className="typography-body-small text-muted-foreground">
          CCCD: {timesheet.employee_code || '-'}
        </p>
      </div>
    </div>
  );
};
