import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { Clock, CheckCircle, AlertTriangle, DollarSign, type LucideIcon } from 'lucide-react';
import { formatCurrency } from '@/utils/formatters';

interface ValidationError {
  employeeId: number;
  totalHours: number;
  message: string;
}

interface TimesheetSummary {
  totalEmployees: number;
  totalHours: number;
  entriesCompleted: number;
  totalCost: number;
  averageHours: number;
  validationErrors: ValidationError[];
}

interface TimesheetSummaryCardsProps {
  summary: TimesheetSummary;
}

// Compose the shared KpiHeroCard; each card's descriptive `sub` becomes the
// `unit` slot so "5/10 người", "8.5 giờ/người" still read inline with the value.
export function TimesheetSummaryCards({ summary }: TimesheetSummaryCardsProps) {
  const warningCount = summary.validationErrors.length;

  const cards: Array<{
    label: string;
    value: string | number;
    unit?: string;
    icon: LucideIcon;
    color: 'blue' | 'emerald' | 'amber' | 'teal' | 'rose';
  }> = [
    {
      label: 'Đã nhập',
      value: `${summary.entriesCompleted}/${summary.totalEmployees}`,
      unit: 'người',
      icon: CheckCircle,
      color: 'emerald',
    },
    {
      label: 'Cảnh báo',
      value: warningCount,
      unit: 'nhân viên',
      icon: AlertTriangle,
      color: warningCount > 0 ? 'rose' : 'blue',
    },
    {
      label: 'Tổng chi phí',
      value: formatCurrency(summary.totalCost),
      icon: DollarSign,
      color: 'blue',
    },
    {
      label: 'Giờ trung bình',
      value: summary.averageHours.toFixed(1),
      unit: 'giờ/người',
      icon: Clock,
      color: 'teal',
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      {cards.map(({ label, value, unit, icon, color }) => (
        <KpiHeroCard
          key={label}
          label={label}
          value={value}
          unit={unit}
          icon={icon}
          color={color}
          className="h-full"
        />
      ))}
    </div>
  );
}
