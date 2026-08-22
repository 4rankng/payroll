import { format } from "date-fns";
import { memo, useMemo, useCallback, useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import {
  Clock,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
import { getMergedStatusBadge } from "@/components/timesheet/utils/timesheetHelpers";
import { EmptyState } from "@/components/shared/EmptyState";
import type { EmployeeTimesheetResponse } from "@/types/api/timesheet.types";
import type { EmployeeTimesheetEntry } from "@/types/api/employee.types";
import type { EmployeeTimesheetFilters } from "@/types/api/employee.types";
import type { SortingState, OnChangeFn } from "@tanstack/react-table";

interface EmployeeTimesheetHistoryProps {
  data: EmployeeTimesheetResponse | undefined;
  loading?: boolean;
  filters: EmployeeTimesheetFilters;
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  onPageChange: (page: number) => void;
  onDateRangeChange: (fromDate?: string, toDate?: string) => void;
}

const formatPaytype = (paytype: string) => {
  if (!paytype) return '-';
  return paytype
    .split('.')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' · ');
};

const generateMonthOptions = () => {
  const months = [];
  const now = new Date();
  for (let i = 0; i < 12; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const value = `${year}-${month.toString().padStart(2, '0')}`;
    const label = `Tháng ${month}/${year}`;
    months.push({ value, label });
  }
  return months;
};

function TimesheetCard({ entry, index }: { entry: EmployeeTimesheetEntry; index: number }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border rounded-xl overflow-hidden">
      {/* Main row — always visible */}
      <button
        className="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-muted/40 transition-colors"
        onClick={() => setExpanded(v => !v)}
        aria-expanded={expanded}
      >
        <span className="text-xs text-muted-foreground w-5 shrink-0">{index}</span>
        <span className="text-sm font-medium flex-1">
          {format(new Date(entry.date), 'dd/MM/yyyy')}
        </span>
        <span className="text-sm font-semibold text-green-700 shrink-0">
          {formatCurrency(entry.amount)}
        </span>
        <span className="shrink-0">{getMergedStatusBadge(entry.status, entry.payment_status)}</span>
        {expanded
          ? <ChevronUp className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
          : <ChevronDown className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
        }
      </button>

      {/* Expanded detail */}
      {expanded && (
        <div className="border-t bg-muted/20 px-3 py-2.5 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <div>
            <p className="text-[11px] text-muted-foreground font-medium mb-0.5">Loại công</p>
            <p className="text-xs">{formatPaytype(entry.paytype)}</p>
          </div>
          <div>
            <p className="text-[11px] text-muted-foreground font-medium mb-0.5">Ca làm việc</p>
            <p className="text-xs">{entry.hours_worked} giờ</p>
          </div>
          <div>
            <p className="text-[11px] text-muted-foreground font-medium mb-0.5">Đơn giá</p>
            <p className="text-xs">{formatCurrency(entry.payrate)}</p>
          </div>
          <div>
            <p className="text-[11px] text-muted-foreground font-medium mb-0.5">Đã thanh toán</p>
            <p className="text-xs font-semibold text-green-700">
              {entry.paid_amount ? formatCurrency(entry.paid_amount) : '0₫'}
            </p>
          </div>
          {entry.paid_at && (
            <div className="col-span-2">
              <p className="text-[11px] text-muted-foreground font-medium mb-0.5">Ngày thanh toán</p>
              <p className="text-xs">{format(new Date(entry.paid_at), 'dd/MM/yyyy')}</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export const EmployeeTimesheetHistory = memo(({
  data,
  loading = false,
  filters,
  onPageChange,
  onDateRangeChange,
}: EmployeeTimesheetHistoryProps) => {
  const timesheetEntries = data?.data || [];
  const pagination = data?.pagination;

  const monthOptions = useMemo(() => generateMonthOptions(), []);

  const selectedMonth = useMemo(() => {
    if (filters.fromDate) {
      const [year, month] = filters.fromDate.split('-');
      return `${year}-${month}`;
    }
    return 'all';
  }, [filters.fromDate]);

  const handleMonthChange = useCallback((monthValue: string) => {
    if (!monthValue || monthValue === 'all') {
      onDateRangeChange(undefined, undefined);
      return;
    }
    const [year, month] = monthValue.split('-').map(Number);
    const lastDay = new Date(year, month, 0);
    const fromDate = `${year}-${month.toString().padStart(2, '0')}-01`;
    const toDate = `${year}-${month.toString().padStart(2, '0')}-${lastDay.getDate().toString().padStart(2, '0')}`;
    onDateRangeChange(fromDate, toDate);
  }, [onDateRangeChange]);

  const startIndex = (filters.page - 1) * (filters.pageSize || 10) + 1;

  if (loading) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-muted-foreground" />
          <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Lịch sử chấm công</p>
        </div>
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-11 w-full rounded-xl" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-muted-foreground" />
          <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Lịch sử chấm công</p>
        </div>
        <Select value={selectedMonth} onValueChange={handleMonthChange}>
          <SelectTrigger className="h-11 w-40 text-xs">
            <SelectValue placeholder="Chọn tháng" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Tất cả</SelectItem>
            {monthOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {timesheetEntries.length === 0 ? (
        <div className="border rounded-xl px-3">
          <EmptyState title="Chưa có dữ liệu chấm công" size="sm" className="py-4" />
        </div>
      ) : (
        <div className="space-y-1.5">
          {timesheetEntries.map((entry, i) => (
            <TimesheetCard key={entry.id ?? i} entry={entry} index={startIndex + i} />
          ))}
        </div>
      )}

      {pagination && pagination.totalPages > 1 && (
        <div className="flex items-center justify-between pt-1">
          <span className="text-xs text-muted-foreground">
            Trang {pagination.page}/{pagination.totalPages} · {pagination.totalRecords} bản ghi
          </span>
          <div className="flex items-center gap-1.5">
            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => onPageChange(pagination.page - 1)}
              disabled={pagination.page <= 1}
            >
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => onPageChange(pagination.page + 1)}
              disabled={pagination.page >= pagination.totalPages}
            >
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
});

EmployeeTimesheetHistory.displayName = "EmployeeTimesheetHistory";
