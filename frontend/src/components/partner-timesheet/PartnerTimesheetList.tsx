import { format } from "date-fns";
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar, Clock, User, AlertCircle } from 'lucide-react';
import { useTimesheetModals } from '@/hooks/useModalNavigation';
import type { Timesheet } from '@/types/api/timesheet.types';

interface PartnerTimesheetListProps {
  timesheetData: Timesheet[];
  isLoading: boolean;
}

export const PartnerTimesheetList = ({ timesheetData, isLoading }: PartnerTimesheetListProps) => {
  const { openTimesheetDetails } = useTimesheetModals();

  const handleTimesheetClick = (timesheet: Timesheet) => {
    openTimesheetDetails(timesheet.id);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'approved':
        return 'bg-success text-white';
      case 'pending':
        return 'bg-warning text-white';
      case 'rejected':
        return 'bg-destructive text-white';
      default:
        return 'bg-secondary text-foreground';
    }
  };

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardContent className="p-4 space-y-3">
              <div className="h-5 bg-muted rounded" />
              <div className="h-4 bg-muted rounded w-3/4" />
              <div className="flex gap-2">
                <div className="h-6 bg-muted rounded w-16" />
                <div className="h-6 bg-muted rounded w-12" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (timesheetData.length === 0) {
    return (
      <Card className="text-center py-12">
        <CardContent>
          <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="typography-headline-medium text-foreground mb-2">
            Chưa có bảng công
          </h3>
          <p className="typography-body-medium text-muted-foreground">
            Hiện tại chưa có bảng công nào trong tháng này
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {timesheetData.map((timesheet) => (
        <Card 
          key={timesheet.id}
          className="cursor-pointer transition-shadow"
          onClick={() => handleTimesheetClick(timesheet)}
        >
          <CardContent className="p-4 space-y-3">
            <div className="space-y-1">
              <h3 className="typography-headline-small text-foreground">
                {timesheet.employeeName}
              </h3>
              <p className="typography-body-medium text-muted-foreground">
                {timesheet.projectName}
              </p>
            </div>
            
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <div className="flex items-center gap-1">
                <Calendar className="h-4 w-4" />
                <span>{timesheet.workDays} ngày</span>
              </div>
              <div className="flex items-center gap-1">
                <Clock className="h-4 w-4" />
                <span>{timesheet.overtime}h OT</span>
              </div>
            </div>
            
            <div className="flex items-center justify-between">
              <Badge 
                variant="secondary"
                className={getStatusColor(timesheet.status)}
              >
                {timesheet.status}
              </Badge>
              <span className="typography-label-small text-muted-foreground">
                {format(new Date(timesheet.date), 'dd/MM/yyyy')}
              </span>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};