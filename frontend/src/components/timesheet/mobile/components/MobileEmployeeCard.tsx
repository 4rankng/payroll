import { useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { AlertTriangle, User, IdCard } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/utils/formatters';
import { MobileHourEntry } from './MobileHourEntry';
import { DEFAULT_POSITIONS } from '@/types/api/payrate.types';
import { getPositionBadgeStyle, getItemIndex } from '@/utils/badge-styles';

interface Employee {
  id: number;
  fullname: string;
  cccd: string;
  position: string;
  employee_code: string;
}

interface TimesheetEntry {
  employee_id: number;
  position: string;
  hour_entries: Record<string, number>;
}

interface ValidationError {
  employeeId: number;
  totalHours: number;
  message: string;
}

interface MobileEmployeeCardProps {
  employee: Employee;
  entry: TimesheetEntry | undefined;
  hourStatus: 'normal' | 'exceeded' | 'excessive';
  availableHourTypes: string[];
  onHourEntryUpdate: (employeeId: number, hourType: string, hours: number) => void;
  getRate: (position: string, hourType: string) => number;
  calculateTotalHours: (entry: TimesheetEntry | undefined) => number;
  calculateEmployeeTotal: (entry: TimesheetEntry | undefined) => number;
}

export function MobileEmployeeCard({
  employee,
  entry,
  hourStatus,
  availableHourTypes,
  onHourEntryUpdate,
  getRate,
  calculateTotalHours,
  calculateEmployeeTotal
}: MobileEmployeeCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  
  const totalHours = calculateTotalHours(entry);
  const totalAmount = calculateEmployeeTotal(entry);
  const hasHours = totalHours > 0;
  const hourEntries = entry?.hour_entries || {};

  const getPositionBadgeClass = (position: string) => {
    const positionIndex = getItemIndex<string>(DEFAULT_POSITIONS as readonly string[], position);
    return getPositionBadgeStyle(positionIndex);
  };

  return (
    <div
      className={cn(
        "mx-2 sm:mx-4 bg-card border rounded-xl transition-all duration-200 touch-manipulation",
        hourStatus === 'excessive' && "border-red-200 bg-red-50/30", // > 16 hours: red
        hourStatus === 'exceeded' && "border-orange-200 bg-orange-50/30", // > 12 hours: orange
        hasHours && hourStatus === 'normal' && "border-blue-200",
        "shadow-sm min-h-[64px] sm:min-h-[56px]"
      )}
    >
      <div className="p-3 sm:p-4">
        <div 
          className="flex items-center justify-between cursor-pointer min-h-[48px] touch-manipulation active:scale-[0.98] transition-transform"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <div className="flex items-center gap-2 sm:gap-3 min-w-0 flex-1">
            <div className="flex items-center justify-center w-10 h-10 sm:w-12 sm:h-12 bg-muted rounded-full flex-shrink-0">
              <User className="w-4 h-4 sm:w-5 sm:h-5 text-muted-foreground" />
            </div>
            <div className="min-w-0 flex-1">
              <h3 className="typography-body-medium sm:typography-body-large text-foreground truncate">
                {employee.fullname}
              </h3>
              <div className="flex items-center gap-1 sm:gap-2 mt-1">
                <IdCard className="w-3 h-3 text-gray-500 flex-shrink-0" />
                <span className="typography-body-small sm:typography-body-medium text-muted-foreground font-mono truncate">
                  {employee.cccd}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {(hourStatus === 'exceeded' || hourStatus === 'excessive') && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger>
                    <AlertTriangle className={cn(
                      "h-4 w-4",
                      hourStatus === 'excessive' ? "text-red-600" : "text-orange-700"
                    )} />
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>
                      {hourStatus === 'excessive'
                        ? 'Nhân viên đã làm việc quá 16 giờ trong ngày'
                        : 'Nhân viên đã làm việc quá 12 giờ trong ngày'
                      }
                    </p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}

            {hasHours && (
              <div className="text-right">
                <div className={cn(
                  "typography-body-medium",
                  hourStatus === 'excessive' ? "text-red-600" :
                  hourStatus === 'exceeded' ? "text-orange-700" :
                  "text-foreground"
                )}>
                  {totalHours.toFixed(1)}h
                </div>
                {totalAmount > 0 && (
                  <div className="typography-body-small text-muted-foreground">
                    {formatCurrency(totalAmount)}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2 mt-2">
          <Badge 
            variant="outline" 
            className={cn(
              "typography-body-small",
              getPositionBadgeClass(entry?.position || employee.position)
            )}
          >
            {entry?.position || employee.position}
          </Badge>
          <span className="typography-body-small text-gray-500">
            {employee.employee_code}
          </span>
        </div>
      </div>

      {isExpanded && (
        <div className="px-3 sm:px-4 pb-3 sm:pb-4">
          <div className="space-y-3 pt-3 border-t border-border">            {availableHourTypes.map(hourType => (
              <MobileHourEntry
                key={hourType}
                hourType={hourType}
                hours={hourEntries[hourType] || 0}
                rate={getRate(entry?.position || employee.position, hourType)}
                onHoursChange={(hours) => onHourEntryUpdate(employee.id, hourType, hours)}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
