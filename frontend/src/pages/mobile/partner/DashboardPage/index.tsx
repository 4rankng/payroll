import { useState, useMemo, useCallback } from 'react';
import { format, startOfMonth } from 'date-fns';
import { Users, UserCheck, UserX, Banknote, Trophy, BarChart3 } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { UserAvatar } from '@/components/ui/user-avatar';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { usePartnerDashboard } from '@/hooks/api/useDashboard';
import { PartnerEmployeeListSheet } from '@/components/partner-dashboard/PartnerEmployeeListSheet';
import { generateMonthOptions } from '@/utils/dateHelpers';
import type { TopPaidEmployeeItem, PartnerEmployeeListType } from '@/types/api/dashboard.types';

const ALL_VALUE = 'all';

const monthOptions = [
  { value: ALL_VALUE, label: 'Tất cả' },
  ...generateMonthOptions(12).map((o) => ({
    value: o.value,
    label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
  })),
];

function formatVND(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(1)}tỷ đ`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(0)}K đ`;
  return `${sign}${abs.toLocaleString('vi-VN')} đ`;
}

function MonthSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="flex items-center gap-1.5 overflow-x-auto pb-1 -mx-4 px-4 scrollbar-none">
      {monthOptions.slice(0, 5).map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={`px-3 py-1.5 rounded-full text-xs font-semibold transition-colors whitespace-nowrap shrink-0 ${
            value === opt.value
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground hover:bg-muted/80'
          }`}
        >
          {opt.label}
        </button>
      ))}
      <select
        value={monthOptions.slice(5).some((o) => o.value === value) ? value : ''}
        onChange={(e) => e.target.value && onChange(e.target.value)}
        className="px-2 py-1.5 rounded-full text-xs font-semibold bg-muted text-muted-foreground border-0 outline-none cursor-pointer shrink-0"
      >
        <option value="">Khác…</option>
        {monthOptions.slice(5).map((opt) => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
    </div>
  );
}

const RANK_COLORS = ['text-amber-500', 'text-slate-400', 'text-amber-700'];
const RANK_BG = ['bg-amber-50 border-amber-200', 'bg-slate-50 border-slate-200', 'bg-amber-50/60 border-amber-100'];

function TopEmployeeRow({ item, maxPaid }: { item: TopPaidEmployeeItem; maxPaid: number }) {
  const pct = maxPaid > 0 ? (item.total_paid_vnd / maxPaid) * 100 : 0;
  const rankColor = RANK_COLORS[item.rank - 1] ?? 'text-muted-foreground';
  const rankBg = RANK_BG[item.rank - 1] ?? '';
  return (
    <div className="flex items-center gap-3 py-3 border-b border-border/40 last:border-0">
      <div className={`flex items-center justify-center w-7 h-7 rounded-full text-xs font-bold shrink-0 border ${rankBg || 'bg-muted/40 border-border/40'} ${rankColor}`}>
        {item.rank <= 3 ? <Trophy className="w-3 h-3" /> : <span>{item.rank}</span>}
      </div>
      <UserAvatar name={item.employee_name} size="sm" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="text-sm font-medium text-foreground truncate">{item.employee_name}</span>
          {item.is_active ? (
            <Badge variant="outline" className="text-[9px] px-1 py-0 h-3.5 border-emerald-300 text-emerald-700 bg-emerald-50 shrink-0">
              <UserCheck className="w-2.5 h-2.5 mr-0.5" />Đang làm
            </Badge>
          ) : (
            <Badge variant="outline" className="text-[9px] px-1 py-0 h-3.5 border-rose-300 text-rose-600 bg-rose-50 shrink-0">
              <UserX className="w-2.5 h-2.5 mr-0.5" />Nghỉ
            </Badge>
          )}
        </div>
        <div className="mt-1.5 h-1.5 rounded-full bg-muted/60 overflow-hidden">
          <div className="h-full rounded-full bg-primary/70 transition-all duration-500" style={{ width: `${pct}%` }} />
        </div>
      </div>
      <span className="text-sm font-semibold tabular-nums text-foreground shrink-0">
        {formatVND(item.total_paid_vnd)}
      </span>
    </div>
  );
}

const PartnerDashboardMobile = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(startOfMonth(new Date()), 'yyyy-MM')
  );
  const [sheetType, setSheetType] = useState<PartnerEmployeeListType | null>(null);
  const handleMonthChange = useCallback((v: string) => setSelectedMonth(v), []);
  const openSheet = useCallback((type: PartnerEmployeeListType) => setSheetType(type), []);
  const closeSheet = useCallback(() => setSheetType(null), []);

  const apiMonth = selectedMonth === ALL_VALUE ? undefined : selectedMonth;
  const { data, isLoading } = usePartnerDashboard({ month: apiMonth });

  const periodLabel = useMemo(() => {
    if (selectedMonth === ALL_VALUE) return 'Tất cả';
    const [y, m] = selectedMonth.split('-');
    return `T${m}/${y}`;
  }, [selectedMonth]);

  const topEmployees = useMemo(() => data?.top_paid_employees ?? [], [data?.top_paid_employees]);
  const maxPaid = topEmployees[0]?.total_paid_vnd ?? 0;

  const momEmployeesSublabel = useMemo(() => {
    if (!data?.mom_paid_employees || selectedMonth === ALL_VALUE) return periodLabel;
    const pct = data.mom_paid_employees.change_pct;
    return `${pct >= 0 ? '+' : ''}${pct.toFixed(0)}% so tháng trước`;
  }, [data?.mom_paid_employees, periodLabel, selectedMonth]);

  const momAmountSublabel = useMemo(() => {
    if (!data?.mom_paid_amount || selectedMonth === ALL_VALUE) return periodLabel;
    const pct = data.mom_paid_amount.change_pct;
    return `${pct >= 0 ? '+' : ''}${pct.toFixed(0)}% so tháng trước`;
  }, [data?.mom_paid_amount, periodLabel, selectedMonth]);

  return (
    <div className="p-4 pb-20 space-y-4">
      <MobilePageHeader
        title="Tổng quan"
        subtitle="Theo dõi nhân viên và thanh toán"
        icon={BarChart3}
        sticky={false}
        bordered={false}
      />

      <MonthSelector value={selectedMonth} onChange={handleMonthChange} />

      <div className="grid grid-cols-2 gap-3">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="rounded-xl border border-border/40 bg-card p-4 space-y-2">
              <Skeleton className="h-7 w-7 rounded-lg" />
              <Skeleton className="h-6 w-14" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))
        ) : (
          <>
            <KpiHeroCard label="Đang làm việc" value={data?.active_employees ?? 0} icon={UserCheck} color="emerald" sublabel="Có bảng công 14 ngày qua" onClick={() => openSheet('active')} />
            <KpiHeroCard label="Có thể nghỉ việc" value={data?.dropped_employees ?? 0} icon={UserX} color="amber" sublabel="Không bảng công 14 ngày qua" onClick={(data?.dropped_employees ?? 0) > 0 ? () => openSheet('dropped') : undefined} />
            <KpiHeroCard label="Được trả lương" value={data?.paid_employees ?? 0} icon={Users} color="blue" sublabel={momEmployeesSublabel} onClick={(data?.paid_employees ?? 0) > 0 ? () => openSheet('paid') : undefined} />
            <KpiHeroCard label="Tổng chi trả" value={data?.total_paid_vnd ?? 0} formattedValue={formatVND(data?.total_paid_vnd ?? 0)} icon={Banknote} color="violet" sublabel={momAmountSublabel} />
          </>
        )}
      </div>

      <div>
        <div className="flex items-center gap-2 mb-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
            <Trophy className="h-3.5 w-3.5 text-primary/70" />
          </div>
          <h2 className="text-sm font-semibold text-foreground">Top nhân viên — {periodLabel}</h2>
        </div>
        <Card>
          <CardContent className="pt-0 px-4">
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3 py-3 border-b border-border/40 last:border-0">
                  <Skeleton className="w-7 h-7 rounded-full" />
                  <Skeleton className="w-8 h-8 rounded-full" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-3.5 w-32" />
                    <Skeleton className="h-1.5 w-full rounded-full" />
                  </div>
                  <Skeleton className="h-3.5 w-16" />
                </div>
              ))
            ) : topEmployees.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-10 text-center">
                <Trophy className="w-9 h-9 text-muted-foreground/30 mb-3" />
                <p className="text-sm font-semibold">Chưa có dữ liệu thanh toán</p>
                <p className="text-xs text-muted-foreground mt-1">Chọn tháng khác để xem</p>
              </div>
            ) : (
              topEmployees.map((item) => (
                <TopEmployeeRow key={item.employee_id} item={item} maxPaid={maxPaid} />
              ))
            )}
          </CardContent>
        </Card>
      </div>

      <PartnerEmployeeListSheet type={sheetType} month={apiMonth} onClose={closeSheet} />
    </div>
  );
};

export default PartnerDashboardMobile;
