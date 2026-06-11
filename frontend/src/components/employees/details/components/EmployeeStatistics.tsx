import { format } from "date-fns";
import { Calendar, CheckCircle, Clock, DollarSign } from "lucide-react";
import type { SortingState } from "@tanstack/react-table";
import { memo, useMemo } from "react";
import { GroupedStatCard } from "@/components/shared/GroupedStatCard";
import { EmployeeTimesheetHistory } from "./EmployeeTimesheetHistory";
import { formatCurrency } from "@/utils/formatters";
import type { EmployeeIndividualSummary, Employee, EmployeeTimesheetFilters, TimesheetSummary, PayrollSummary } from "@/types/api/employee.types";
import type { EmployeeTimesheetResponse } from "@/types/api/timesheet.types";

interface EmployeeStatisticsProps {
  employee: Employee;
  summaryData: EmployeeIndividualSummary | undefined;
  summaryLoading: boolean;
  timesheetData: EmployeeTimesheetResponse | undefined;
  timesheetLoading: boolean;
  timesheetFilters: EmployeeTimesheetFilters;
  timesheetSorting?: SortingState;
  onTimesheetSortingChange?: (sorting: SortingState) => void;
  onPageChange: (page: number) => void;
  onDateRangeChange: (fromDate?: string, toDate?: string) => void;
}

export const EmployeeStatistics = memo(({
  employee,
  summaryData,
  summaryLoading,
  timesheetData,
  timesheetLoading,
  timesheetFilters,
  timesheetSorting,
  onTimesheetSortingChange,
  onPageChange,
  onDateRangeChange
}: EmployeeStatisticsProps) => {
  const statsConfig = useMemo(() => summaryData ? [
    {
      title: "Tổng thu nhập",
      value: formatCurrency(summaryData.total_earnings_vnd),
      icon: DollarSign,
    },
    {
      title: "Số lần trả lương",
      value: summaryData.total_payroll_payments,
      icon: CheckCircle,
    },
    {
      title: "Trả lương gần nhất",
      value: summaryData.last_payment_date && summaryData.last_payment_date.trim() !== '' ?
        format(new Date(summaryData.last_payment_date), 'dd/MM/yyyy') :
        'Chưa có',
      icon: Calendar,
    },
    {
      title: "Lương TB/tuần",
      value: formatCurrency(summaryData.avg_weekly_earnings_vnd),
      icon: Clock,
    }
  ] : [], [summaryData]);

  return (
    <div className="space-y-6">
      {/* Timesheet Summary from API */}
      {employee.timesheet_summary && (
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-0.5">Chấm công</h3>
          <div className="space-y-1.5">
            <GroupedStatCard
              title="Tổng quan"
              stats={[
                { label: 'Tổng giờ làm', value: (employee.timesheet_summary as TimesheetSummary).total_hours_worked },
                { label: 'Tuần này', value: (employee.timesheet_summary as TimesheetSummary).current_week_hours },
                { label: 'Tổng bảng chấm', value: (employee.timesheet_summary as TimesheetSummary).total_timesheets },
              ]}
            />
            <GroupedStatCard
              title="Trạng thái"
              stats={[
                { label: 'Chờ duyệt', value: (employee.timesheet_summary as TimesheetSummary).pending_timesheets, variant: (employee.timesheet_summary as TimesheetSummary).pending_timesheets > 0 ? 'accent' as const : 'default' as const },
                { label: 'Đã duyệt', value: (employee.timesheet_summary as TimesheetSummary).approved_timesheets },
                { label: 'Bị loại', value: (employee.timesheet_summary as TimesheetSummary).rejected_timesheets },
              ]}
            />
          </div>
          {(employee.timesheet_summary as TimesheetSummary).last_timesheet_date && (
            <p className="text-xs text-muted-foreground px-0.5">
              Bảng chấm gần nhất: {format(new Date((employee.timesheet_summary as TimesheetSummary).last_timesheet_date), 'dd/MM/yyyy')}
            </p>
          )}
        </div>
      )}

      {/* Payroll Summary from API */}
      {employee.payroll_summary && (
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-0.5">Lương</h3>
          <GroupedStatCard
            title="Lương"
            stats={[
              { label: 'Tổng thanh toán', value: (employee.payroll_summary as PayrollSummary).total_payroll_payments },
              { label: 'Tổng thu nhập', value: formatCurrency((employee.payroll_summary as PayrollSummary).total_earnings_vnd) },
              { label: 'Lương TB/tuần', value: formatCurrency((employee.payroll_summary as PayrollSummary).avg_weekly_earnings_vnd) },
              { label: 'Trả gần nhất', value: (() => { const d = (employee.payroll_summary as PayrollSummary).last_payment_date; return d && d.trim() !== '' ? format(new Date(d), 'dd/MM/yyyy') : 'Chưa có'; })() },
            ]}
          />
        </div>
      )}

      {/* Recent Timesheet History */}
      <EmployeeTimesheetHistory
        data={timesheetData}
        loading={timesheetLoading}
        filters={timesheetFilters}
        sorting={timesheetSorting}
        onSortingChange={onTimesheetSortingChange}
        onPageChange={onPageChange}
        onDateRangeChange={onDateRangeChange}
      />

    </div>
  );
});

EmployeeStatistics.displayName = "EmployeeStatistics";
