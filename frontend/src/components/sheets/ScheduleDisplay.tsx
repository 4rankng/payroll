import { Input } from '@/components/ui/input';
import { formatVND } from '@/utils/loanHelpers';
import type { ScheduleRow } from '@/types/api/loan.types';

interface ScheduleDisplayProps {
  schedules: ScheduleRow[];
  loanType: string;
  onScheduleChange?: (id: string, field: 'due_date' | 'amount', value: string) => void;
  summary: {
    principal: number;
    totalRepayment: number;
    totalInterest: number;
    effectiveAnnualRate: number;
  } | null;
}

export function ScheduleDisplay({
  schedules,
  loanType,
  onScheduleChange,
  summary
}: ScheduleDisplayProps) {
  const cardColorClasses = [
    'bg-muted/50 border-border',
    'bg-muted/50 border-border',
    'bg-muted/50 border-border',
    'bg-muted/50 border-border'
  ];

  return (
    <div className="space-y-3">
      <div className="typography-label-medium text-muted-foreground font-semibold">
        Lịch trả ({schedules.length} kỳ)
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {schedules.map((schedule, index) => (
          <div
            key={schedule.id}
            className={`relative flex flex-col justify-between rounded-xl border p-4 shadow-sm shadow-slate-100/60 transition-all focus-within:border-blue-500 focus-within:bg-card focus-within:shadow-sm ${
              cardColorClasses[index % cardColorClasses.length]
            } ${index === schedules.length - 1 ? 'bg-slate-100 border-border' : ''}`}
          >
            <span className={`absolute right-1 top-1 flex h-7 min-w-[2.5rem] items-center justify-center rounded-full px-3 typography-label-small font-semibold shadow-sm ${
              index === schedules.length - 1 ? 'bg-slate-200 text-foreground' : 'bg-slate-100 text-foreground'
            }`}>
              {index + 1}
            </span>

            <div className="space-y-4">
              <label className="flex flex-col gap-2">
                <span className="typography-label-small text-muted-foreground">Ngày đến hạn</span>
                <Input
                  type="date"
                  value={schedule.due_date}
                  onChange={loanType === 'custom_schedule'
                    ? (e) => onScheduleChange?.(schedule.id, 'due_date', e.target.value)
                    : undefined}
                  disabled={loanType !== 'custom_schedule'}
                  className="h-11 w-full bg-card"
                />
              </label>

              <label className="flex flex-col gap-2">
                <span className="typography-label-small text-muted-foreground">Số tiền trả</span>
                <Input
                  inputMode="numeric"
                  value={Number(schedule.amount).toLocaleString('vi-VN')}
                  onChange={loanType === 'custom_schedule'
                    ? (e) => onScheduleChange?.(schedule.id, 'amount', e.target.value.replace(/[^0-9]/g, ''))
                    : undefined}
                  disabled={loanType !== 'custom_schedule'}
                  className="h-11 w-full bg-card"
                />
              </label>
            </div>

            {schedule.amount && Number(schedule.amount) > 0 && (
              <div className={`mt-3 p-3 rounded-xl ${index === schedules.length - 1 ? 'bg-slate-200 border border-border' : 'bg-slate-100'}`}>
                <p className="text-right">
                  {index === schedules.length - 1 && (
                    <span className="typography-label-small text-foreground font-semibold block mb-1">Gốc + Lãi cuối kỳ</span>
                  )}
                  <span className={`font-bold ${index === schedules.length - 1 ? 'typography-h3 text-foreground' : 'typography-body-large text-foreground'}`}>
                    {formatVND(Number(schedule.amount))}
                  </span>
                </p>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Summary Section */}
      {summary && (
        <div className="bg-muted/50 border border-border rounded-xl p-6 space-y-4">
          <div className="typography-label-medium text-foreground font-semibold uppercase tracking-wide">XÁC NHẬN THÔNG TIN VAY NỢ</div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="flex flex-col gap-2 bg-card p-4 rounded-xl border border-border">              <span className="typography-label-small text-muted-foreground uppercase tracking-wide">Tổng tiền vay</span>
              <span className="typography-h2 font-bold text-foreground">
                {formatVND(summary.principal)}
              </span>
            </div>

            <div className="flex flex-col gap-2 bg-card p-4 rounded-xl border border-border">              <span className="typography-label-small text-muted-foreground uppercase tracking-wide">Tổng tiền trả</span>
              <span className="typography-h2 font-bold text-foreground">
                {formatVND(summary.totalRepayment)}
              </span>
            </div>

            <div className="flex flex-col gap-2 bg-card p-4 rounded-xl border border-border">              <span className="typography-label-small text-muted-foreground uppercase tracking-wide">Tổng tiền lãi</span>
              <span className="typography-h2 font-bold text-foreground">
                {formatVND(summary.totalInterest)}
              </span>
            </div>

            <div className="flex flex-col gap-2 bg-card p-4 rounded-xl border border-border">              <span className="typography-label-small text-muted-foreground uppercase tracking-wide">Lãi suất/năm</span>
              <span className="typography-h3 font-bold text-foreground">
                {summary.effectiveAnnualRate.toFixed(2)}%
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
