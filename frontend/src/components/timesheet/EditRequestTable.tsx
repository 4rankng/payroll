import { useMemo, useState } from 'react';
import { DataTable } from '@/components/ui/data-table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Check, Loader2, Calendar } from 'lucide-react';
import { ColumnDef, SortingState } from '@tanstack/react-table';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { useEditRequests, useApproveEditRequest, usePartnerTimesheetsWithEditRequests } from '@/hooks/api/useTimesheetEditRequests';
import type { Timesheet, TimesheetEditRequest } from '@/types/api/timesheet.types';
import { PaytypeHierarchy } from './components/PaytypeHierarchy';
import { EmployeeCell } from './components/EmployeeCell';
import { useAuth } from '@/contexts/AuthContext';

interface EditRequestTableProps {
  userRole?: 'admin' | 'partner';
  onRowClick?: (timesheet: Timesheet) => void;
}

type FilterStatus = 'all' | 'pending' | 'approved';

const EMPTY_REQUESTS: TimesheetEditRequest[] = [];

const SORT_FIELD_MAP: Record<string, string> = {
  employee: 'employee_name',
  project: 'project_name',
  date: 'date',
  hours: 'hours_worked',
  paytype: 'paytype',
  amount: 'amount',
  requester: 'requested_by_name',
};

const transformTimesheetToEditRequest = (
  timesheet: Timesheet,
  fallbackRequesterName?: string
): TimesheetEditRequest => {
  const status = timesheet.request_edit_status
    ?? (timesheet.allowed_edit ? 'approved' : 'pending');

  return {
    id: timesheet.request_edit_id ?? timesheet.id,
    timesheet_id: timesheet.id,
    status,
    requested_by: timesheet.request_edit_requested_by_id ?? timesheet.created_by,
    requested_by_name:
      timesheet.request_edit_requested_by_name
      ?? fallbackRequesterName
      ?? 'Bạn',
    approved_by: undefined,
    approved_by_name: undefined,
    rejected_by: undefined,
    rejected_by_name: undefined,
    created_at: timesheet.request_edit_requested_at ?? timesheet.updated_at ?? timesheet.created_at,
    updated_at: timesheet.updated_at ?? timesheet.created_at,
    timesheet: {
      id: timesheet.id,
      date: timesheet.date,
      end_date: undefined,
      employee_id: timesheet.employee_id,
      employee_name: timesheet.employeeName,
      employee_code: timesheet.employeeCode,
      employee_cccd: timesheet.employeeCCCD,
      project_id: timesheet.project_id,
      project_name: timesheet.projectName,
      project_code: timesheet.projectCode,
      hours_worked: timesheet.hours_worked,
      overtime_hours: timesheet.overtime ?? timesheet.overtimeHours ?? 0,
      amount: timesheet.amount,
      paytype: timesheet.paytype,
      status: timesheet.status,
      payment_status: timesheet.payment_status ?? 'pending',
    },
  };
};

const normalizeRequestTimesheet = (request: TimesheetEditRequest): Timesheet | null => {
  const source = request.timesheet;
  if (!source) {
    return null;
  }

  const sourceRecord = source as Record<string, unknown>;
  const paytype = typeof sourceRecord.paytype === 'string' ? sourceRecord.paytype : '';
  const paytypeParts = paytype ? paytype.split('.') : [];
  const dayType =
    typeof sourceRecord.day_type === 'string'
      ? (sourceRecord.day_type as string)
      : (paytypeParts[1] ?? '');
  const hourType =
    typeof sourceRecord.hour_type === 'string'
      ? (sourceRecord.hour_type as string)
      : (paytypeParts[2] ?? '');

  const createdAt = (sourceRecord.created_at as string) ?? request.created_at;
  const updatedAt = (sourceRecord.updated_at as string) ?? request.updated_at ?? createdAt;

  return {
    id: source.id ?? request.timesheet_id,
    project_id: (sourceRecord.project_id as number) ?? 0,
    employee_id: (sourceRecord.employee_id as number) ?? 0,
    date: (sourceRecord.date as string) ?? request.created_at,
    hours_worked: (sourceRecord.hours_worked as number) ?? 0,
    paytype,
    hour_type: hourType,
    day_type: dayType,
    payrate_id: (sourceRecord.payrate_id as number) ?? 0,
    payrate: (sourceRecord.payrate as number) ?? 0,
    amount: (sourceRecord.amount as number) ?? 0,
    status: ((sourceRecord.status as Timesheet['status']) ?? 'pending_approval'),
    payment_status: ((sourceRecord.payment_status as Timesheet['payment_status']) ?? 'pending'),
    allowed_edit: sourceRecord.allowed_edit as boolean | undefined,
    request_edit_id:
      (sourceRecord.request_edit_id as number | null | undefined) ?? request.id ?? request.timesheet_id,
    request_edit_status:
      (sourceRecord.request_edit_status as Timesheet['request_edit_status']) ?? request.status,
    request_edit_requested_at:
      (sourceRecord.request_edit_requested_at as string) ?? request.created_at,
    request_edit_requested_by_id:
      (sourceRecord.request_edit_requested_by_id as number) ?? request.requested_by,
    request_edit_requested_by_name:
      (sourceRecord.request_edit_requested_by_name as string) ?? request.requested_by_name,
    force_payroll: sourceRecord.force_payroll as boolean | undefined,
    paid_amount: (sourceRecord.paid_amount as number) ?? 0,
    paid_at: (sourceRecord.paid_at as string | null | undefined) ?? null,
    created_by: (sourceRecord.created_by as number) ?? request.requested_by,
    approved_by: sourceRecord.approved_by as number | null | undefined,
    approved_at: (sourceRecord.approved_at as string | null | undefined) ?? null,
    rejection_reason: sourceRecord.rejection_reason as string | undefined,
    created_at: createdAt,
    updated_at: updatedAt,
    projectName:
      (sourceRecord.projectName as string) ?? (sourceRecord.project_name as string) ?? '',
    employeeName:
      (sourceRecord.employeeName as string) ?? (sourceRecord.employee_name as string) ?? '',
    employeeCode:
      (sourceRecord.employeeCode as string) ?? (sourceRecord.employee_code as string) ?? '',
    employeeCCCD:
      (sourceRecord.employeeCCCD as string) ?? (sourceRecord.employee_cccd as string) ?? undefined,
    projectCode:
      (sourceRecord.projectCode as string) ?? (sourceRecord.project_code as string) ?? undefined,
    period: sourceRecord.period as string | undefined,
    weekRange: sourceRecord.weekRange as string | undefined,
    totalHours: sourceRecord.totalHours as number | undefined,
    regularHours: sourceRecord.regularHours as number | undefined,
    overtimeHours:
      (sourceRecord.overtimeHours as number | undefined) ??
      (sourceRecord.overtime_hours as number | undefined),
    totalAmount:
      (sourceRecord.totalAmount as number | undefined) ??
      (sourceRecord.amount as number | undefined),
    submittedAt: sourceRecord.submittedAt as string | undefined,
    workDays: sourceRecord.workDays as number | undefined,
    overtime:
      (sourceRecord.overtime as number | undefined) ??
      (sourceRecord.overtime_hours as number | undefined),
  };
};

export const EditRequestTable = ({ userRole = 'admin', onRowClick }: EditRequestTableProps) => {
  const { user } = useAuth();
  const isPartnerView = userRole === 'partner';

  const [sorting, setSorting] = useState<SortingState>([]);

  const { sortBy, sortOrder } = useMemo(() => {
    if (sorting.length === 0) return { sortBy: 'created_at', sortOrder: 'desc' as const };
    const col = sorting[0];
    return {
      sortBy: SORT_FIELD_MAP[col.id] ?? col.id,
      sortOrder: (col.desc ? 'desc' : 'asc') as const,
    };
  }, [sorting]);

  const partnerFilters = useMemo(() => {
    if (!isPartnerView || !user?.id) {
      return null;
    }
    return {
      page: 1,
      pageSize: 200,
      sortBy,
      sortOrder,
      created_by: Number(user.id),
      has_request_edit: true,
    };
  }, [isPartnerView, user?.id, sortBy, sortOrder]);

  const { data: editRequests, isLoading: isLoadingEditRequests } = useEditRequests({
    page: 1,
    pageSize: 100,
    sortBy,
    sortOrder,
  }, { enabled: !isPartnerView });

  const { data: partnerTimesheets, isLoading: isLoadingPartnerTimesheets } = usePartnerTimesheetsWithEditRequests(
    partnerFilters ?? undefined,
    isPartnerView && !!partnerFilters
  );
  const partnerTimesheetMap = useMemo(() => {
    const map = new Map<number, Timesheet>();
    if (partnerTimesheets?.data) {
      partnerTimesheets.data.forEach((timesheet) => {
        map.set(timesheet.id, timesheet);
      });
    }
    return map;
  }, [partnerTimesheets?.data]);

  const isLoading = isPartnerView ? isLoadingPartnerTimesheets : isLoadingEditRequests;
  const approveMutation = useApproveEditRequest();
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all');

  const partnerRequests = useMemo(
    () => {
      if (!isPartnerView) {
        return [];
      }
      const timesheets = partnerTimesheets?.data ?? [];
      return timesheets
        .filter((timesheet) => timesheet.request_edit_id !== null && timesheet.request_edit_id !== undefined)
        .map((timesheet) => transformTimesheetToEditRequest(timesheet, user?.name));
    },
    [isPartnerView, partnerTimesheets, user?.name]
  );

  const editRequestData = editRequests?.data;
  const allRequests = useMemo<TimesheetEditRequest[]>(() => {
    if (isPartnerView) {
      return partnerRequests;
    }
    return editRequestData ?? EMPTY_REQUESTS;
  }, [isPartnerView, partnerRequests, editRequestData]);

  // Filter records based on user role and selected filter
  const requests = useMemo(() => {
    if (userRole === 'admin') {
      // Admin always sees only pending and rejected (no approved)
      return allRequests.filter(request => request.status !== 'approved');
    }

    // Partner can filter by status
    if (filterStatus === 'all') {
      return allRequests;
    }
    return allRequests.filter(request => request.status === filterStatus);
  }, [allRequests, userRole, filterStatus]);

  const totalCount = requests.length;

  const columns: ColumnDef<TimesheetEditRequest>[] = useMemo(() => [
    {
      accessorKey: 'stt',
      header: 'STT',
      enableSorting: false,
      cell: ({ row }) => {
        return (
          <div className="text-center typography-data-medium">
            {row.index + 1}
          </div>
        );
      },
      size: 60,
    },
    {
      accessorKey: 'employee',
      header: 'Nhân viên',
      cell: ({ row }) => <EmployeeCell request={row.original} />,
    },
    {
      accessorKey: 'project',
      header: 'Dự án',
      cell: ({ row }) => {
        const request = row.original;
        const timesheet = request.timesheet;
        if (!timesheet) return <span className="text-muted-foreground">N/A</span>;

        return (
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
            <div>
              <p className="typography-body-medium text-foreground">
                {timesheet.project_name || '-'}
              </p>
              {timesheet.project_code && (
                <p className="typography-body-small text-muted-foreground">
                  {timesheet.project_code}
                </p>
              )}
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: 'date',
      header: 'Ngày',
      cell: ({ row }) => {
        const request = row.original;
        const timesheet = request.timesheet;
        if (!timesheet) return <span className="text-muted-foreground">N/A</span>;

        const formatDate = (dateString: string) => {
          try {
            return format(new Date(dateString), 'dd/MM/yyyy', { locale: vi });
          } catch {
            return dateString;
          }
        };

        return (
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <div className="text-center">
              <span className="typography-data-medium font-medium text-foreground">
                {formatDate(timesheet.date)}
              </span>
              {timesheet.end_date && (
                <span className="typography-body-small text-muted-foreground ml-1">
                  - {formatDate(timesheet.end_date)}
                </span>
              )}
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: 'hours',
      header: 'Giờ công',
      cell: ({ row }) => {
        const request = row.original;
        const timesheet = request.timesheet;
        if (!timesheet) return <span className="text-muted-foreground">N/A</span>;

        const hoursWorked = timesheet.hours_worked || 0;
        const overtimeHours = timesheet.overtime_hours || 0;

        return (
          <div className="text-center space-y-1">
            <div className="typography-data-medium font-semibold text-primary-600">
              {hoursWorked} giờ
            </div>
            {overtimeHours > 0 && (
              <div className="typography-data-small text-muted-foreground">
                TC: {overtimeHours} giờ
              </div>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'paytype',
      header: 'Loại ca',
      cell: ({ row }) => {
        const request = row.original;
        const timesheet = request.timesheet;
        if (!timesheet || !timesheet.paytype) {
          return <span className="text-muted-foreground">N/A</span>;
        }

        return (
          <PaytypeHierarchy
            paytype={timesheet.paytype}
            className="min-w-0"
          />
        );
      },
    },
    {
      accessorKey: 'amount',
      header: 'Số tiền',
      cell: ({ row }) => {
        const request = row.original;
        const timesheet = request.timesheet;
        if (!timesheet) return <span className="text-muted-foreground">N/A</span>;

        const amount = timesheet.amount || 0;

        return (
          <div className="text-right">
            <span className="typography-data-medium font-semibold text-foreground">
              {amount.toLocaleString('vi-VN')} đ
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: 'requester',
      header: 'Người yêu cầu',
      cell: ({ row }) => {
        const request = row.original;
        const formatDateTime = (dateString: string) => {
          try {
            return format(new Date(dateString), 'dd/MM/yyyy HH:mm', { locale: vi });
          } catch {
            return dateString;
          }
        };

        return (
          <div>
            <p className="typography-body-medium font-medium text-foreground">
              {request.requested_by_name}
            </p>
            <p className="typography-body-small text-muted-foreground">
              {formatDateTime(request.created_at)}
            </p>
          </div>
        );
      },
    },
    {
      accessorKey: 'status',
      header: 'Trạng thái',
      enableSorting: true,
      cell: ({ row }) => {
        const request = row.original;
        const getStatusBadge = () => {
          switch (request.status) {
            case 'approved':
              return (
                <Badge variant="success" className="shadow-sm">
                  Đã duyệt
                </Badge>
              );
            case 'rejected':
              return (
                <Badge variant="destructive" className="shadow-sm">
                  Đã từ chối
                </Badge>
              );
            default:
              return (
                <Badge variant="info" className="shadow-sm">
                  Chờ duyệt
                </Badge>
              );
          }
        };

        return <div className="flex justify-center">{getStatusBadge()}</div>;
      },
      size: 150,
    },
  ], []);

  // Don't render if no pending requests
  if (totalCount === 0 && !isLoading) {
    return null;
  }

  const handleRowClick = (request: TimesheetEditRequest) => {
    if (!onRowClick) {
      return;
    }

    const timesheet =
      partnerTimesheetMap.get(request.timesheet_id) ??
      normalizeRequestTimesheet(request);

    if (timesheet) {
      onRowClick(timesheet);
    }
  };

  const handleApproveAll = () => {
    const pendingRequests = requests.filter(r => r.status === 'pending');
    pendingRequests.forEach(request => {
      approveMutation.mutate(request.id);
    });
  };

  const hasPendingRequests = requests.some(r => r.status === 'pending');

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h3 className="text-lg font-semibold text-foreground">
            Yêu cầu sửa bảng công
          </h3>
          <Badge variant="destructive" className="h-6 px-2">
            {totalCount}
          </Badge>
          {userRole === 'partner' && (
            <Select value={filterStatus} onValueChange={(value) => setFilterStatus(value as FilterStatus)}>
              <SelectTrigger className="w-[160px] h-9">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pending">Chờ duyệt</SelectItem>
                <SelectItem value="approved">Đã duyệt</SelectItem>
                <SelectItem value="all">Tất cả</SelectItem>
              </SelectContent>
            </Select>
          )}
        </div>
        {userRole === 'admin' && hasPendingRequests && (
          <Button
            size="sm"
            onClick={handleApproveAll}
            disabled={approveMutation.isPending}
            className="min-w-[120px]"
          >
            {approveMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Đang duyệt...
              </>
            ) : (
              <>
                <Check className="h-4 w-4 mr-2" />
                Duyệt tất cả
              </>
            )}
          </Button>
        )}
      </div>

      <div className="rounded-xl border border-yellow-200 bg-yellow-50/30 p-4">
        <DataTable
          columns={columns}
          data={requests}
          onRowClick={handleRowClick}
          sorting={sorting}
          onSortingChange={setSorting}
        />
      </div>
    </div>
  );
};
