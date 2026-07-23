import { Clock, DollarSign, Calendar, TrendingUp, CheckCircle, Clock3, AlertCircle, CreditCard, LucideIcon } from 'lucide-react';
import { EmployeeTimesheetSummary } from '@/types/api/timesheet.types';
import { formatCurrency } from '@/utils/formatters';

export interface TimesheetStatsConfig {
  title: string;
  value: string | number;
  icon: LucideIcon;
  description?: string;
  change?: string;
  trend?: 'up' | 'down' | 'neutral';
  color?: 'blue' | 'green' | 'red' | 'yellow' | 'teal' | 'gray';
}

export const createTimesheetStatsConfig = (summary: EmployeeTimesheetSummary): TimesheetStatsConfig[] => {
  const totalHoursWorked = Object.values(summary.totalHours).reduce((sum, hours) => sum + hours, 0);

  const mainStats: TimesheetStatsConfig[] = [
    {
      title: 'Tổng giờ làm',
      value: `${totalHoursWorked} giờ`,
      icon: Clock,
      description: `Trung bình ${summary.averageHoursPerDay} giờ/ngày`,
      color: 'blue'
    },
    {
      title: 'Tổng tiền',
      value: formatCurrency(summary.totalAmount),
      icon: DollarSign,
      description: 'Tổng thu nhập',
      color: 'green'
    },
    {
      title: 'Ngày làm việc',
      value: summary.workingDays,
      icon: Calendar,
      description: 'Số ngày có chấm công',
      color: 'teal'
    },
    {
      title: 'Chờ duyệt',
      value: summary.pendingEntries,
      icon: AlertCircle,
      description: 'Bản ghi chưa được duyệt',
      color: summary.pendingEntries > 0 ? 'yellow' : 'gray'
    }
  ];

  return mainStats;
};

export const createHourTypeBreakdownStats = (totalHours: Record<string, number>): TimesheetStatsConfig[] => {
  const hourTypeStats: TimesheetStatsConfig[] = [];

  // Color palette for dynamic assignment (7+ colors)
  const colorPalette: Array<'blue' | 'green' | 'red' | 'yellow' | 'teal' | 'gray'> = [
    'blue', 'green', 'yellow', 'teal', 'red', 'gray'
  ];

  // Track color assignments for hour types
  const hourTypeColorMap = new Map<string, 'blue' | 'green' | 'red' | 'yellow' | 'teal' | 'gray'>();
  let colorIndex = 0;

  // Group hour types by category
  const hourTypeGroups: Record<string, { hours: number; color: 'blue' | 'green' | 'red' | 'yellow' | 'teal' | 'gray' }> = {};

  Object.entries(totalHours).forEach(([hourType, hours]) => {
    if (hours === 0) return;

    // Extract hour type from compound string (e.g., "phổ thông.ngày nghỉ.ca ngày" -> "ca ngày")
    const parts = hourType.split('.');
    const simpleHourType = parts[parts.length - 1];

    if (!hourTypeGroups[simpleHourType]) {
      // Assign color dynamically from palette
      if (!hourTypeColorMap.has(simpleHourType)) {
        const assignedColor = colorPalette[colorIndex % colorPalette.length];
        hourTypeColorMap.set(simpleHourType, assignedColor);
        colorIndex++;
      }

      const color = hourTypeColorMap.get(simpleHourType)!;
      hourTypeGroups[simpleHourType] = { hours: 0, color };
    }

    hourTypeGroups[simpleHourType].hours += hours;
  });

  // Convert to stats format
  Object.entries(hourTypeGroups).forEach(([hourType, data]) => {
    hourTypeStats.push({
      title: hourType.charAt(0).toUpperCase() + hourType.slice(1),
      value: `${data.hours}h`,
      icon: Clock3,
      color: data.color
    });
  });

  return hourTypeStats;
};

export const createApprovalStats = (approvalStats?: EmployeeTimesheetSummary['approvalStats']): TimesheetStatsConfig[] => {
  if (!approvalStats) return [];

  return [
    {
      title: 'Đã duyệt',
      value: approvalStats.approved,
      icon: CheckCircle,
      color: 'green'
    },
    {
      title: 'Chờ duyệt',
      value: approvalStats.pending,
      icon: Clock,
      color: 'yellow'
    },
    {
      title: 'Loại',
      value: approvalStats.rejected,
      icon: AlertCircle,
      color: 'red'
    }
  ];
};

export const getPaymentStatusConfig = (paymentStatus?: string) => {
  const statusConfig = {
    pending: { color: 'yellow' as const, icon: Clock, text: 'Chờ thanh toán' },
    processing: { color: 'blue' as const, icon: TrendingUp, text: 'Đang xử lý' },
    paid: { color: 'green' as const, icon: CheckCircle, text: 'Đã thanh toán' },
    overdue: { color: 'red' as const, icon: AlertCircle, text: 'Quá hạn' }
  };

  return statusConfig[paymentStatus as keyof typeof statusConfig] || statusConfig.pending;
};
