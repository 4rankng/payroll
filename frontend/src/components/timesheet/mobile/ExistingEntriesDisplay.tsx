import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Clock, Edit, Trash2, CheckCircle, AlertCircle, XCircle, Calendar } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { formatCurrency } from '@/utils/formatters';

interface TimesheetEntry {
  id: number;
  employee_id: number;
  date: string;
  hours_worked: number;
  hour_type: string;
  day_type: string;
  payrate: number;
  amount: number;
  status: 'draft' | 'pending_approval' | 'approved' | 'rejected';
  employee_name?: string;
  employee_code?: string;
}

interface ExistingEntriesDisplayProps {
  projectId: number;
  date: string;
  existingTimesheets: TimesheetEntry[];
}

export function ExistingEntriesDisplay({
  projectId,
  date,
  existingTimesheets
}: ExistingEntriesDisplayProps) {

  const getStatusInfo = (status: string) => {
    switch (status) {
      case 'approved':
        return {
          label: 'Đã duyệt',
          variant: 'success' as const,
          icon: CheckCircle
        };
      case 'pending_approval':
        return {
          label: 'Chờ duyệt',
          variant: 'warning' as const,
          icon: AlertCircle
        };
      case 'rejected':
        return {
          label: 'Loại',
          variant: 'destructive' as const,
          icon: XCircle
        };
      default:
        return {
          label: 'Nháp',
          variant: 'secondary' as const,
          icon: Calendar
        };
    }
  };

  const getHourTypeIcon = (hourType: string) => {
    switch (hourType.toLowerCase()) {
      case 'ca đêm':
        return '🌙';
      case 'tăng ca':
        return '⚡';
      default:
        return '☀️';
    }
  };

  if (!existingTimesheets || existingTimesheets.length === 0) {
    return (
      <Card className="border-none shadow-sm">
        <CardContent className="pt-6">
          <div className="text-center py-8 text-muted-foreground">
            <Calendar className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p className="typography-body-medium">Chưa có bảng công nào cho ngày này</p>
            <p className="typography-body-small mt-1">
              {format(new Date(date + 'T00:00:00'), 'EEEE, dd/MM/yyyy', { locale: vi })}
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  // Group entries by employee
  const entriesByEmployee = existingTimesheets.reduce<Record<string, {
    employee_id: number;
    employee_name: string;
    employee_code: string;
    entries: TimesheetEntry[];
  }>>((acc, entry) => {
    const key = `${entry.employee_id}-${entry.employee_name || 'Unknown'}`;
    if (!acc[key]) {
      acc[key] = {
        employee_id: entry.employee_id,
        employee_name: entry.employee_name || 'Unknown',
        employee_code: entry.employee_code || '',
        entries: []
      };
    }
    acc[key].entries.push(entry);
    return acc;
  }, {});

  const employeeGroups = Object.values(entriesByEmployee);

  return (
    <Card className="border-none shadow-sm">
      <CardHeader className="pb-3">
        <CardTitle className="typography-title-large flex items-center gap-2">
          <Clock className="h-5 w-5 text-primary" />
          Bảng công hiện tại
          <Badge variant="outline" className="typography-body-small">
            {existingTimesheets.length} bản ghi
          </Badge>
        </CardTitle>
        <p className="typography-body-medium text-muted-foreground">
          {format(new Date(date + 'T00:00:00'), 'EEEE, dd/MM/yyyy', { locale: vi })}
        </p>
      </CardHeader>
      <CardContent className="space-y-4">
        {employeeGroups.map((employeeGroup) => (
          <div key={employeeGroup.employee_id} className="space-y-2">
            {/* Employee header */}
            <div className="flex items-center justify-between">
              <div>
                <p className="typography-body-medium">{employeeGroup.employee_name}</p>
                {employeeGroup.employee_code && (
                  <p className="typography-body-small text-muted-foreground">{employeeGroup.employee_code}</p>
                )}
              </div>
              <Badge variant="outline" className="typography-body-small">
                {employeeGroup.entries.length} ca
              </Badge>
            </div>

            {/* Employee's timesheet entries */}
            {employeeGroup.entries.map((entry: TimesheetEntry) => {
              const statusInfo = getStatusInfo(entry.status);
              const StatusIcon = statusInfo.icon;

              return (
                <div
                  key={entry.id}
                  className="bg-muted/30 rounded-xl p-3 border border-border/50"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <span className="typography-title-large">{getHourTypeIcon(entry.hour_type)}</span>
                      <span className="typography-body-medium">{entry.hour_type}</span>
                    </div>
                    <Badge variant={statusInfo.variant} className="typography-body-small flex items-center gap-1">
                      <StatusIcon className="h-3 w-3" />
                      {statusInfo.label}
                    </Badge>
                  </div>

                  <div className="grid grid-cols-2 gap-3 typography-body-small">
                    <div>
                      <span className="text-muted-foreground">Ca làm việc:</span>
                      <p className="font-medium">{entry.hours_worked} giờ</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Đơn giá:</span>
                      <p className="font-medium">{formatCurrency(entry.payrate)}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Loại ngày:</span>
                      <p className="font-medium capitalize">{entry.day_type}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Thành tiền:</span>
                      <p className="font-bold text-primary">{formatCurrency(entry.amount)}</p>
                    </div>
                  </div>

                  {/* Actions for editable entries */}
                  {(entry.status === 'draft' || entry.status === 'rejected') && (
                    <div className="flex gap-2 mt-3">
                      <Button variant="outline" size="sm" className="flex-1 h-8 typography-body-small">
                        <Edit className="h-3 w-3 mr-1" />
                        Sửa
                      </Button>
                      <Button variant="outline" size="sm" className="h-8 px-2">
                        <Trash2 className="h-3 w-3 text-destructive" />
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ))}

        {/* Summary */}
        <div className="bg-primary/5 border border-primary/20 rounded-xl p-3 mt-4">
          <div className="flex items-center justify-between">
            <span className="typography-body-medium">Tổng cộng:</span>
            <div className="text-right">
              <p className="font-bold text-primary">
                {formatCurrency(existingTimesheets.reduce((sum, entry) => sum + entry.amount, 0))}
              </p>
              <p className="typography-body-small text-muted-foreground">
                {existingTimesheets.reduce((sum, entry) => sum + entry.hours_worked, 0).toFixed(1)}h
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
