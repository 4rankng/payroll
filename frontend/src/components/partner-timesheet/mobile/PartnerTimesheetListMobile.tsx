import { format } from "date-fns";
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar, Clock, AlertCircle, ChevronRight } from 'lucide-react';
import { useTimesheetModals } from '@/hooks/useModalNavigation';
import type { Timesheet } from '@/types/api/timesheet.types';

interface PartnerTimesheetListMobileProps {
  timesheetData: Timesheet[];
  isLoading: boolean;
}

export const PartnerTimesheetListMobile = ({ timesheetData, isLoading }: PartnerTimesheetListMobileProps) => {
  const { openTimesheetDetails } = useTimesheetModals();

  const handleTimesheetClick = (timesheet: Timesheet) => {
    openTimesheetDetails(String(timesheet.id));
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
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
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
          <AlertCircle className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
          <h3 className="text-sm font-semibold text-foreground mb-2">
            Chưa có bảng công
          </h3>
          <p className="text-sm text-muted-foreground">
            Chưa có bảng công nào trong tháng này
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {timesheetData.map((timesheet) => (
        <Card
          key={timesheet.id}
          className="cursor-pointer active:bg-muted/50 transition-colors overflow-hidden"
          onClick={() => handleTimesheetClick(timesheet)}
        >
          <CardContent className="p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0 space-y-2">
                <div className="space-y-1">
                  <h3 className="text-sm font-semibold text-foreground">
                    {timesheet.employeeName}
                  </h3>
                  <p className="text-sm text-muted-foreground line-clamp-1">
                    {timesheet.projectName}
                  </p>
                </div>
                
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <div className="flex items-center gap-1">
                    <Calendar className="h-3.5 w-3.5" />
                    <span>{timesheet.workDays} ngày</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Clock className="h-3.5 w-3.5" />
                    <span>{timesheet.overtime}h OT</span>
                  </div>
                </div>
                
                <div className="flex items-center justify-between">
                  <Badge 
                    variant="secondary"
                    className={`${getStatusColor(timesheet.status)} text-xs font-medium text-muted-foreground`}
                  >
                    {timesheet.status}
                  </Badge>
                  <span className="text-xs font-medium text-muted-foreground">
                    {format(new Date(timesheet.date), 'dd/MM/yyyy')}
                  </span>
                </div>
              </div>
              
              <ChevronRight className="h-5 w-5 text-muted-foreground shrink-0" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};