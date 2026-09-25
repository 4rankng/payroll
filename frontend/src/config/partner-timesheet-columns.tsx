import { ColumnDef } from '@tanstack/react-table';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Calendar, Clock, Building2 } from 'lucide-react';
import { formatDate } from '@/utils/formatters';
import { formatCurrency } from '@/utils/formatters';
import { TimesheetStatusBadge, type TimesheetStatus } from '@/components/timesheet/TimesheetStatusBadge';
import type { Timesheet } from '@/types/api/timesheet.types';

export function createPartnerTimesheetColumns(): ColumnDef<Timesheet>[] {
  return [
    {
      id: "stt",
      header: "STT",
      size: 40,
      maxSize: 60,
      cell: ({ row }) => (
        <div className="text-center typography-label-medium font-medium text-muted-foreground">
          {row.index + 1}
        </div>
      ),
      enableHiding: false,
    },
    {
      id: "employee",
      accessorKey: "employeeName",
      header: "Nhân viên",
      cell: ({ row }) => {
        const timesheet = row.original;
        return (
          <div className="flex items-center gap-3 max-w-xs">
            <UserAvatar
              name={timesheet.employeeName}
              email={timesheet.employeeCode}
              size="md"
            />
            <div className="min-w-0 flex-1">
              <div className="typography-body-medium font-medium text-foreground line-clamp-1">
                {timesheet.employeeName}
              </div>
              <div className="typography-body-small text-muted-foreground">
                #{timesheet.employeeCode}
              </div>
            </div>
          </div>
        );
      },
    },
    {
      id: "project",
      accessorKey: "projectName",
      header: "Dự án",
      cell: ({ row }) => {
        const timesheet = row.original;
        return (
          <div className="space-y-1 max-w-xs">
            <div className="flex items-center gap-2">
              <Building2 className="h-3 w-3 text-muted-foreground flex-shrink-0" />
              <span className="typography-body-medium text-foreground line-clamp-1">
                {timesheet.projectName}
              </span>
            </div>
          </div>
        );
      },
    },
    {
      id: "date",
      accessorKey: "date",
      header: "Ngày làm",
      cell: ({ row }) => {
        const timesheet = row.original;
        return (
          <div className="flex items-center gap-1">
            <Calendar className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="typography-body-medium text-foreground">
              {formatDate(timesheet.date, 'short')}
            </span>
          </div>
        );
      },
    },
    {
      id: "hours",
      accessorKey: "hours_worked",
      header: "Ca làm việc",
      cell: ({ row }) => {
        const timesheet = row.original;
        return (
          <div className="flex items-center gap-1">
            <Clock className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="typography-data-medium tabular-nums font-semibold text-foreground">
              {timesheet.hours_worked} giờ
            </span>
          </div>
        );
      },
    },
    {
      id: "paytype",
      accessorKey: "paytype",
      header: "Loại công",
      cell: ({ row }) => {
        const timesheet = row.original;
        return (
          <div className="typography-body-small text-muted-foreground max-w-xs">
            {timesheet.paytype}
          </div>
        );
      },
    },
    {
      id: "amount",
      accessorKey: "amount",
      header: "Tổng tiền",
      cell: ({ row }) => {
        const timesheet = row.original;
        const amount = timesheet.amount;
        if (!amount) return (
          <span className="typography-body-small text-muted-foreground">
            Chưa tính
          </span>
        );

        return (
          <div className="typography-data-medium tabular-nums font-semibold text-foreground">
            {formatCurrency(amount)}
          </div>
        );
      },
    },
    {
      id: "paid_amount",
      accessorKey: "paid_amount",
      header: "Tiền đã trả",
      cell: ({ row }) => {
        const timesheet = row.original;
        const paidAmount = timesheet.paid_amount;
        if (!paidAmount) return (
          <span className="typography-body-small text-muted-foreground">
            0₫
          </span>
        );

        return (
          <div className="typography-data-medium tabular-nums font-semibold text-emerald-700">
            {formatCurrency(paidAmount)}
          </div>
        );
      },
    },
    {
      id: "paid_at",
      accessorKey: "paid_at",
      header: "Trả lúc",
      cell: ({ row }) => {
        const timesheet = row.original;
        const paidAt = timesheet.paid_at;
        if (!paidAt) return (
          <span className="typography-body-small text-muted-foreground">
            -
          </span>
        );

        return (
          <div className="typography-body-small text-muted-foreground">
            {formatDate(paidAt, 'short')}
          </div>
        );
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: "Trạng thái",
      cell: ({ row }) => {
        const timesheet = row.original;
        const status = timesheet.status as TimesheetStatus;
        const paymentStatus = timesheet.payment_status;

        return (
          <TimesheetStatusBadge
            status={status}
            paymentStatus={paymentStatus}
            showTooltip={true}
          />
        );
      },
    },
    {
      id: "created_at",
      accessorKey: "created_at",
      header: "Ngày tạo",
      cell: ({ row }) => {
        const timesheet = row.original;
        const createdAt = timesheet.created_at;
        if (!createdAt) return (
          <span className="typography-body-small text-muted-foreground">
            -
          </span>
        );

        return (
          <div className="typography-body-small text-muted-foreground">
            {formatDate(createdAt, 'short')}
          </div>
        );
      },
    },
  ];
}
