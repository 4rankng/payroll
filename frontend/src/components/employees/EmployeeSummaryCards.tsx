import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { Users, UserCheck, UserPlus, DollarSign, type LucideIcon } from 'lucide-react';
import type { EmployeeSummary } from '@/types/api/employee.types';

interface EmployeeSummaryCardsProps {
  summary: EmployeeSummary;
  formatCurrency: (amount: number) => string;
}

type EmployeeCardKey =
  | 'total_employees'
  | 'total_working_employees'
  | 'employees_hired_this_month'
  | 'salary_month_to_date';

// Compose the shared KpiHeroCard instead of a bespoke watermark clone, dropping
// the inline boxShadow/backdrop-blur in favor of the system's shadow-soft token.
const cards: Array<{
  key: EmployeeCardKey;
  label: string;
  icon: LucideIcon;
  color: 'blue' | 'emerald' | 'amber' | 'teal';
  isCurrency: boolean;
}> = [
  { key: 'total_employees', label: 'Tổng nhân viên', icon: Users, color: 'blue', isCurrency: false },
  { key: 'total_working_employees', label: 'Đang làm việc', icon: UserCheck, color: 'emerald', isCurrency: false },
  { key: 'employees_hired_this_month', label: 'Tuyển tháng này', icon: UserPlus, color: 'amber', isCurrency: false },
  { key: 'salary_month_to_date', label: 'Lương tháng này', icon: DollarSign, color: 'teal', isCurrency: true },
];

export const EmployeeSummaryCards = ({ summary, formatCurrency }: EmployeeSummaryCardsProps) => (
  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
    {cards.map(({ key, label, icon, color, isCurrency }) => {
      const raw = summary[key];
      const value = isCurrency ? formatCurrency(raw) : raw;
      return (
        <KpiHeroCard key={key} label={label} value={value} icon={icon} color={color} className="h-full" />
      );
    })}
  </div>
);
