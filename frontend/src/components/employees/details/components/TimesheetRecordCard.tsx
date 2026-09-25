import { Badge } from "@/components/ui/badge";
import { DollarSign } from "lucide-react";
import { memo } from "react";
import { 
  getTimesheetStatusIcon, 
  getTimesheetStatusLabel, 
  getTimesheetStatusColor 
} from "@/utils/employees/employeeStatusHelpers";
import { getTimesheetDisplayData } from "@/utils/employees/timesheetHelpers";
import type { EmployeeTimesheetEntry } from "@/types/api/employee.types";

interface TimesheetRecordCardProps {
  record: EmployeeTimesheetEntry;
}

export const TimesheetRecordCard = memo(({ record }: TimesheetRecordCardProps) => {
  const displayData = getTimesheetDisplayData(record);
  const StatusIcon = getTimesheetStatusIcon(record.status);
  const statusLabel = getTimesheetStatusLabel(record.status);
  const statusColor = getTimesheetStatusColor(record.status);

  return (
    <div className="bg-muted/30 rounded-xl p-4 space-y-3">
      <div className="flex items-start justify-between">
        <div className="space-y-1 flex-1">
          <div className="flex items-center gap-2">
            <DollarSign className="w-4 h-4 text-green-700" />
            <span className="typography-title-large">
              {displayData.amount}
            </span>
          </div>
          <p className="typography-body-medium text-muted-foreground">{displayData.projectName}</p>
        </div>
        <div className="text-right">
          <Badge className={`${statusColor} mb-2`} variant="secondary">
            <div className="flex items-center gap-1">
              <StatusIcon className="w-4 h-4" />
              <span className="typography-body-small">{statusLabel}</span>
            </div>
          </Badge>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-4 typography-body-medium">
        <div>
          <span className="text-muted-foreground">Ngày làm:</span>
          <p className="font-medium">{displayData.date}</p>
        </div>
        <div>
          <span className="text-muted-foreground">Số giờ:</span>
          <p className="font-medium">{displayData.hours}</p>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-4 typography-body-medium">
        <div>
          <span className="text-muted-foreground">Loại ca:</span>
          <p className="typography-body-small">{displayData.paytype}</p>
        </div>
        <div>
          <span className="text-muted-foreground">Đơn giá:</span>
          <p className="font-medium">{displayData.rate}</p>
        </div>
      </div>
      
      <div className="grid grid-cols-2 gap-4 typography-body-medium">
        <div>
          <span className="text-muted-foreground">Loại giờ:</span>
          <p className="typography-body-small">{displayData.hourType}</p>
        </div>
        <div>
          <span className="text-muted-foreground">Loại ngày:</span>
          <p className="typography-body-small">{displayData.dayType}</p>
        </div>
      </div>
    </div>
  );
});

TimesheetRecordCard.displayName = "TimesheetRecordCard";