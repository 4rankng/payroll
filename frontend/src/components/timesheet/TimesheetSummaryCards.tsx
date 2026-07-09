import { Clock, CheckCircle, AlertTriangle, DollarSign } from 'lucide-react';
import { cn } from '@/lib/utils';
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

export function TimesheetSummaryCards({ summary }: TimesheetSummaryCardsProps) {
  const warningCount = summary.validationErrors.length;

  const cards = [
    {
      label: 'Đã nhập',
      value: `${summary.entriesCompleted}/${summary.totalEmployees}`,
      sub: 'người',
      icon: CheckCircle,
      iconText: 'text-emerald-600',
      watermark: 'text-emerald-500/15',
    },
    {
      label: 'Cảnh báo',
      value: warningCount.toLocaleString('vi-VN'),
      sub: 'nhân viên',
      icon: AlertTriangle,
      iconText: warningCount > 0 ? 'text-rose-600' : 'text-muted-foreground',
      watermark: warningCount > 0 ? 'text-rose-500/15' : 'text-slate-500/10',
    },
    {
      label: 'Tổng chi phí',
      value: formatCurrency(summary.totalCost),
      sub: '',
      icon: DollarSign,
      iconText: 'text-blue-600',
      watermark: 'text-blue-500/15',
    },
    {
      label: 'Giờ trung bình',
      value: summary.averageHours.toFixed(1),
      sub: 'giờ/người',
      icon: Clock,
      iconText: 'text-violet-600',
      watermark: 'text-violet-500/15',
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
      {cards.map(({ label, value, sub, icon: Icon, iconText, watermark }) => (
        <div
          key={label}
          className="group relative rounded-xl border border-border/60 bg-card px-3 py-2.5 overflow-hidden shadow-sm transition-colors hover:bg-muted/40"
        >
          <Icon
            className={cn(
              'absolute right-2 top-1/2 -translate-y-1/2 h-10 w-10 pointer-events-none',
              'transition-transform duration-300 group-hover:scale-105',
              watermark,
            )}
            strokeWidth={1.5}
          />
          <div className="relative min-w-0 pr-10">
            <div className="flex items-center gap-1.5">
              <Icon className={cn('h-3 w-3 shrink-0', iconText)} strokeWidth={2.2} />
              <span className="text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight line-clamp-2">
                {label}
              </span>
            </div>
            <p className="mt-1 break-words text-[15px] font-semibold tabular-nums text-foreground leading-tight">
              {value}
              {sub && <span className="text-[11px] font-normal text-muted-foreground ml-1">{sub}</span>}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}
