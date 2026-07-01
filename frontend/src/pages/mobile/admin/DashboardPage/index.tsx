import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth, addMonths, subMonths, parse } from 'date-fns';
import { vi } from 'date-fns/locale';
import {
  Users, Clock, BarChart2, Activity, ChevronLeft, ChevronRight,
  Calendar as CalendarIcon, TrendingUp, FolderKanban, BarChart3,
  UserPlus, Trophy, Building2, ArrowRightLeft,
  AlertCircle,
} from 'lucide-react';
import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { SalaryDistributionChartMobile } from '@/components/admin-dashboard/SalaryDistributionChartMobile';
import { GroupedStatCard } from '@/components/shared/GroupedStatCard';
import { ProjectProfitabilityMobile } from '@/components/admin-dashboard/ProjectProfitabilityMobile';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';

import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { useBankUsageAllProjects } from '@/hooks/api/useDashboard';
import { useTimesheetModals } from '@/hooks/useModalNavigation';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import { cn } from '@/lib/utils';
import { MobileSectionHeader as SectionHeader } from '@/components/shared/MobileSectionHeader';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { formatCompactCurrency as formatVND } from '@/utils/formatters';

// ── helpers ──────────────────────────────────────────────────────────────────

const AdminDashboardMobile = () => {
  const navigate = useNavigate();
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);

  const { data: bankData } = useBankUsageAllProjects();

  const monthParam = selectedMonth === 'all' ? undefined : selectedMonth;
  const monthLabel = selectedMonth === 'all' ? 'Tất cả' : (() => {
    const [y, m] = selectedMonth.split('-');
    return `${m}/${y}`;
  })();

  const selectedDate = selectedMonth !== 'all'
    ? startOfMonth(parse(selectedMonth + '-01', 'yyyy-MM-dd', new Date()))
    : startOfMonth(new Date());

  const { data, loading, employees } = useDashboardData(monthParam);
  const actions = useDashboardActions();
  const { data: pendingData } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });
  const { data: activityData } = useEmployeeActivityStats(monthParam);
  const dashboardNav = useDashboardNavigation();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openAddEmployee } = useEmployeeModals();

  const pendingApprovals = pendingData?.pagination?.totalRecords || 0;

  const { employeeStats, salaryStats, profitStats, activityStats } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingApprovals,
    dashboardNav,
    activityStats: activityData ?? null,
    onActivityClick: (schedule) => {
      const params = new URLSearchParams();
      if (schedule) params.set('schedule', schedule);
      if (monthParam) params.set('month', monthParam);
      navigate(`/admin/dashboard/activity?${params.toString()}`);
    },
  });

  // ── Quick actions ──
  const quickActions = useMemo(() => [
    {
      label: 'Chấm công',
      icon: CalendarIcon,
      color: 'text-blue-600',
      bg: 'bg-blue-50',
      onClick: () => openTimesheetEntry(),
    },
    {
      label: 'Chuyển tiền',
      icon: ArrowRightLeft,
      color: 'text-emerald-600',
      bg: 'bg-emerald-50',
      onClick: () => navigate('/admin/wallet'),
    },
    {
      label: 'Thêm NV',
      icon: UserPlus,
      color: 'text-violet-600',
      bg: 'bg-violet-50',
      onClick: () => openAddEmployee(),
    },
    {
      label: 'Dự án',
      icon: FolderKanban,
      color: 'text-amber-600',
      bg: 'bg-amber-50',
      onClick: () => navigate('/admin/projects'),
    },
  ], [navigate, openTimesheetEntry, openAddEmployee]);

  if (loading) {
    return <div className="p-4"><DashboardLoadingSkeleton /></div>;
  }

  return (
    <div className="pb-20">
      {/* ── Sticky Header ── */}
      <MobilePageHeader
        title="Tổng quan"
        subtitle={format(new Date(), 'EEEE, dd/MM', { locale: vi })}
        icon={BarChart3}
        actions={
          pendingApprovals > 0 ? (
            <button
              onClick={dashboardNav.navigateToPendingApprovals}
              className="flex items-center gap-1.5 px-3 h-9 rounded-full bg-amber-50 border border-amber-200 text-amber-700 text-xs font-bold active:scale-95 transition-transform"
            >
              <AlertCircle className="h-3.5 w-3.5" />
              {pendingApprovals} chờ duyệt
            </button>
          ) : undefined
        }
      />

      {/* ── Compact month picker ── */}
      <div className="px-4 pt-3 pb-1">
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => setSelectedMonth('all')}
            className={cn(
              'h-9 px-3.5 rounded-lg text-xs font-semibold transition-colors',
              selectedMonth === 'all'
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted/60 text-muted-foreground',
            )}
          >
            Tất cả
          </button>
          <button
            onClick={() => setSelectedMonth(format(subMonths(selectedDate, 1), 'yyyy-MM'))}
            className="h-9 w-9 flex items-center justify-center rounded-lg bg-muted/60 text-muted-foreground active:bg-muted"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <div className={cn(
            'flex items-center gap-1.5 h-9 px-3 rounded-lg bg-muted/60 text-xs font-semibold text-foreground',
            selectedMonth === 'all' && 'opacity-50',
          )}>
            <CalendarIcon className="h-3 w-3 text-muted-foreground" />
            {selectedMonth === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
          </div>
          <button
            onClick={() => setSelectedMonth(format(addMonths(selectedDate, 1), 'yyyy-MM'))}
            className="h-9 w-9 flex items-center justify-center rounded-lg bg-muted/60 text-muted-foreground active:bg-muted"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* ── KPI Hero Grid ── */}
      {data.dashboardSummary && (
        <div className="px-4 pt-4 grid grid-cols-2 gap-2.5">
          <KpiHeroCard
            label="NV đang làm"
            value={data.dashboardSummary.total_working_employees}
            icon={Users}
            color="blue"
            sublabel={`Tổng: ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')}`}
            onClick={dashboardNav.navigateToActiveEmployees}
          />
          <KpiHeroCard
            label="Lợi nhuận"
            value={data.dashboardSummary.total_profit_this_month}
            formattedValue={formatVND(data.dashboardSummary.total_profit_this_month)}
            icon={TrendingUp}
            color="emerald"
            sublabel={`Tổng: ${formatVND(data.dashboardSummary.total_profit)}`}
          />
          <KpiHeroCard
            label="Chờ duyệt"
            value={pendingApprovals}
            icon={Clock}
            color="amber"
            sublabel="Bảng công"
            onClick={pendingApprovals > 0 ? dashboardNav.navigateToPendingApprovals : undefined}
          />
          <KpiHeroCard
            label="NV mới"
            value={data.dashboardSummary.employees_hired_this_month}
            icon={UserPlus}
            color="violet"
            sublabel={monthLabel}
            onClick={dashboardNav.navigateToNewEmployees}
          />
        </div>
      )}

      {/* ── Quick Actions ── */}
      <div className="px-4 pt-4">
        <div className="flex gap-2">
          {quickActions.map((action) => {
            const Icon = action.icon;
            return (
              <button
                key={action.label}
                onClick={action.onClick}
                className="flex-1 flex flex-col items-center gap-1.5 py-3 rounded-xl bg-card border border-border/50 active:scale-[0.97] transition-transform"
              >
                <div className={cn('p-2 rounded-xl', action.bg)}>
                  <Icon className={cn('h-4 w-4', action.color)} />
                </div>
                <span className="text-xs font-semibold text-muted-foreground leading-none">
                  {action.label}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* ── Stat Sections ── */}
      <div className="px-4 pt-5 space-y-3">
        {salaryStats.length > 0 && (
          <GroupedStatCard title="Bảng công" icon={Clock} stats={salaryStats} />
        )}
        {profitStats.length > 0 && (
          <GroupedStatCard title="Tài chính" icon={BarChart2} stats={profitStats} />
        )}
        {employeeStats.length > 0 && (
          <GroupedStatCard title="Nhân viên" icon={Users} stats={employeeStats} />
        )}
        {activityStats.length > 0 && (
          <GroupedStatCard title={`Hoạt động (${monthLabel})`} icon={Activity} stats={activityStats} />
        )}
      </div>

      {/* ── Salary Distribution Chart ── */}
      <div className="px-4 pt-6">
        <SectionHeader icon={BarChart3} title="Phân bổ lương" />
        <SalaryDistributionChartMobile />
      </div>

      {/* ── Project Profitability ── */}
      <div className="px-4 pt-6">
        <SectionHeader icon={FolderKanban} title="Lợi nhuận dự án" />
        <ProjectProfitabilityMobile />
      </div>

      {/* ── Top Paid Employees ── */}
      <div className="px-4 pt-6">
        <SectionHeader icon={Trophy} title={`Top nhân viên — ${monthLabel}`} />
        <TopPaidEmployeesCard month={monthParam} />
      </div>

      {/* ── Bank Transfer Breakdown ── */}
      <div className="px-4 pt-6">
        <SectionHeader icon={Building2} title="Ngân hàng">
          {bankData && bankData.projects.length > 0 && (
            <div className="ml-auto">
              <ProjectSelector
                projects={bankData.projects}
                selectedId={bankProjectId}
                onSelect={setBankProjectId}
              />
            </div>
          )}
        </SectionHeader>
        <BankTransferBreakdownCard selectedProjectId={bankProjectId} />
      </div>

      {/* ── Monthly Financial History ── */}
      <div className="px-4 pt-6">
        <SectionHeader icon={TrendingUp} title="Lịch sử tài chính" />
        <MonthlyFinancialTable />
      </div>

      {/* ── Check-in / Advance Health ── */}
      <div className="px-4 pt-6">
        <CheckInHealthStrip month={monthParam} />
      </div>

      {/* ── Recent Employees ── */}
      <div className="px-4 pt-6 pb-2">
        <SectionHeader icon={Users} title="Nhân viên mới nhất" />
        <RecentEmployeesCard
          employees={employees.employees}
          isLoading={employees.isLoading}
          isLoadingMore={employees.isLoadingMore}
          totalEmployees={employees.totalEmployees}
          weeks={employees.weeks}
          onEmployeeClick={(employee) => actions.handleEmployeeClick?.(employee)}
          onLoadMore={employees.loadMore}
        />
      </div>
    </div>
  );
};

export default AdminDashboardMobile;
