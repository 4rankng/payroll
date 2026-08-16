import { TrendingUp } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';

interface EmployeeSummaryData {
  employeeId: number;
  employeeName: string;
  totalHours: Record<string, number>;
  totalAmount: number;
  averageHoursPerDay: number;
  workingDays: number;
  lastEntryDate: string;
  pendingEntries: number;
}

interface TimesheetMonthlySummaryProps {
  summaryData?: EmployeeSummaryData;
}

export function TimesheetMonthlySummary({ summaryData }: TimesheetMonthlySummaryProps) {
  if (!summaryData) return null;

  const totalHours = Object.values(summaryData.totalHours || {}).reduce(
    (sum: number, hours: number) => sum + (Number(hours) || 0),
    0
  );

  return (
    <div className="bg-emerald-50 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-3">
        <TrendingUp className="w-4 h-4 text-emerald-600" />
        <h4 className="typography-body-medium text-emerald-700">
          Tổng quan tháng
        </h4>
      </div>
      <div className="grid grid-cols-2 gap-3 typography-body-medium">
        <div>
          <span className="typography-body-small text-emerald-600">Tổng giờ</span>
          <p className="font-medium">{totalHours} giờ</p>
        </div>
        <div>
          <span className="typography-body-small text-emerald-600">Tổng tiền</span>
          <p className="font-medium">
            {summaryData.totalAmount?.toLocaleString('vi-VN') || '0'}
            <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
          </p>
        </div>
        <div>
          <span className="typography-body-small text-emerald-600">Ngày làm</span>
          <p className="font-medium">{summaryData.workingDays} ngày</p>
        </div>
        <div>
          <span className="typography-body-small text-emerald-600">TB/ngày</span>
          <p className="font-medium">{summaryData.averageHoursPerDay} giờ</p>
        </div>
      </div>
    </div>
  );
}
