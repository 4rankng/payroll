import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { format, startOfMonth } from 'date-fns';
import { TrendingUp, Users, Clock, Activity, FolderKanban, BarChart3, UserPlus, Trophy, Building2 } from 'lucide-react';

import { Masonry } from 'masonic';

// UI Components
import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';

// Dashboard Components
import { DashboardHeader } from '@/components/admin-dashboard/DashboardHeader';
import { DashboardSectionHeader } from '@/components/admin-dashboard/DashboardSectionHeader';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { InlineStatStrip } from '@/components/shared/InlineStatStrip';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { ActivityUsersSheet } from '@/components/admin-dashboard/ActivityUsersSheet';
import { ProjectProfitabilityCard } from '@/components/admin-dashboard/ProjectProfitabilityCard';
import { SalaryDistributionChart } from '@/components/admin-dashboard/SalaryDistributionChart';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';

// Dashboard Hooks
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { useBankUsageAllProjects } from '@/hooks/api/useDashboard';

import { formatCompactCurrency as formatVND } from '@/utils/formatters';
import { DashboardMasonryCard, type DashboardCardData } from './components/DashboardMasonryCard';

const AdminDashboard = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [activitySchedule, setActivitySchedule] = useState<'weekly' | 'monthly' | 'flexible' | null>(null);
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);

  const { data: bankData } = useBankUsageAllProjects();

  const handleMonthChange = useCallback((value: string) => {
    setSelectedMonth(value);
  }, []);

  const monthParam = selectedMonth === 'all' ? undefined : selectedMonth;
  const monthLabel = selectedMonth === 'all' ? 'Tất cả' : (() => {
    const [y, m] = selectedMonth.split('-');
    return `${m}/${y}`;
  })();

  const { data, loading, employees, refetch } = useDashboardData(monthParam);
  const actions = useDashboardActions();
  const { data: pendingData } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });
  const { data: activityData } = useEmployeeActivityStats(monthParam);
  const dashboardNav = useDashboardNavigation();

  const handleRefresh = useCallback(() => {
    refetch();
    actions.handleRefresh?.();
  }, [refetch, actions]);

  const { salaryStats, activityStats } =
    useDashboardStats({
      dashboardSummary: data.dashboardSummary,
      totalPendingApprovals: pendingData?.pagination?.totalRecords || 0,
      dashboardNav,
      activityStats: activityData ?? null,
      onActivityClick: setActivitySchedule,
    });

  const salaryStatItems = useMemo(
    () => salaryStats.map(s => ({ label: String(s.label), value: s.value, unit: s.unit, highlight: s.variant === 'accent', onClick: s.onClick })),
    [salaryStats]
  );

  const activityStatItems = useMemo(
    () => activityStats.map(s => ({ label: String(s.label), value: s.value, unit: s.unit, highlight: s.variant === 'accent', onClick: s.onClick })),
    [activityStats]
  );

  const dashboardItems = useMemo<DashboardCardData[]>(() => {
    const items: DashboardCardData[] = [];

    if (salaryStatItems.length > 0) {
      items.push({
        id: 'salary-stats',
        content: (
          <div className="space-y-2">
            <div className="flex items-center gap-2 px-0.5">
              <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
                <Clock className="h-3.5 w-3.5 text-primary/70" />
              </div>
              <span className="text-xs font-bold uppercase tracking-wider text-foreground">Bảng công</span>
            </div>
            <InlineStatStrip items={salaryStatItems} variant="premium" direction="vertical" />
          </div>
        ),
      });
    }

    if (activityStatItems.length > 0) {
      items.push({
        id: 'activity-stats',
        content: (
          <div className="space-y-2">
            <div className="flex items-center gap-2 px-0.5">
              <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
                <Activity className="h-3.5 w-3.5 text-primary/70" />
              </div>
              <span className="text-xs font-bold uppercase tracking-wider text-foreground">
                NV Đăng Nhập ({monthLabel})
              </span>
            </div>
            <InlineStatStrip items={activityStatItems} variant="premium" direction="vertical" />
          </div>
        ),
      });
    }

    items.push(
      { id: 'salary-distribution', content: (
        <div className="space-y-3">
          <DashboardSectionHeader title="Phân bổ lương" icon={BarChart3} />
          <SalaryDistributionChart />
        </div>
      )},
      { id: 'monthly-financials', content: (
        <div className="space-y-3">
          <DashboardSectionHeader title="Lịch sử tài chính" icon={TrendingUp} />
          <MonthlyFinancialTable />
        </div>
      )},
      { id: 'project-profitability', content: (
        <div className="space-y-3">
          <DashboardSectionHeader title="Lợi nhuận dự án" icon={FolderKanban} />
          <ProjectProfitabilityCard />
        </div>
      )},
      { id: 'top-paid-employees', content: (
        <div className="space-y-3">
          <DashboardSectionHeader title={`Top Nhân Viên Được Trả Lương Cao Nhất — ${monthLabel}`} icon={Trophy} />
          <TopPaidEmployeesCard month={monthParam} />
        </div>
      )},
      { id: 'bank-breakdown', content: (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <DashboardSectionHeader title="Phân bổ ngân hàng" icon={Building2} />
            {bankData && bankData.projects.length > 0 && (
              <ProjectSelector
                projects={bankData.projects}
                selectedId={bankProjectId}
                onSelect={setBankProjectId}
              />
            )}
          </div>
          <BankTransferBreakdownCard selectedProjectId={bankProjectId} />
        </div>
      )},
    );

    items.push({
      id: 'recent-employees',
      content: (
        <div className="space-y-3">
          <DashboardSectionHeader title="Nhân viên mới nhất" icon={Users} />
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
      ),
    });

    return items;
  }, [salaryStatItems, activityStatItems, monthLabel, monthParam, bankData, bankProjectId, employees, actions]);

  if (loading) {
    return (
      <div className="p-4 lg:p-6 animate-fade-in">
        <DashboardLoadingSkeleton />
      </div>
    );
  }

  return (
    <div className="min-h-full animate-hero-reveal">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">

        {/* Header */}
        <div className="animate-fade-in-up">
          <DashboardHeader value={selectedMonth} onChange={handleMonthChange} />
        </div>

        {/* ── KPI Hero Cards: 3-col grid ── */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {data.dashboardSummary && (
            <div className="opacity-0 animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:forwards]">
              <KpiHeroCard
                label="NV trả lương tuần này"
                value={data.dashboardSummary.total_working_employees}
                icon={Users}
                color="blue"
                sublabel={`Tổng: ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} NV`}
                onClick={dashboardNav.navigateToActiveEmployees}
              />
            </div>
          )}
          {data.dashboardSummary && (
            <div className="opacity-0 animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:forwards]">
              <KpiHeroCard
                label="Lợi nhuận tháng này"
                value={data.dashboardSummary.total_profit_this_month}
                formattedValue={formatVND(data.dashboardSummary.total_profit_this_month)}
                icon={TrendingUp}
                color="emerald"
                sublabel={`Tổng: ${formatVND(data.dashboardSummary.total_profit)}`}
              />
            </div>
          )}
          {data.dashboardSummary && (
            <div className="opacity-0 animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:forwards]">
              <KpiHeroCard
                label={`Nhân viên mới (${monthLabel})`}
                value={data.dashboardSummary.employees_hired_this_month}
                icon={UserPlus}
                color="amber"
                sublabel=""
                onClick={dashboardNav.navigateToNewEmployees}
              />
            </div>
          )}
        </div>

        {/* Masonry layout */}
        <Masonry
          items={dashboardItems}
          render={DashboardMasonryCard}
          columnWidth={340}
          columnGutter={16}
          rowGutter={16}
          maxColumnCount={3}
          overscanBy={Infinity}
          itemKey={data => data.id}
        />

        {/* Check-in / Advance health: bottom full-width responsive row */}
        <div className="opacity-0 animate-fade-in-up [animation-delay:200ms] [animation-fill-mode:forwards]">
          <CheckInHealthStrip month={monthParam} />
        </div>
      </div>

      <ActivityUsersSheet
        schedule={activitySchedule}
        month={monthParam}
        onClose={() => setActivitySchedule(null)}
      />
    </div>
  );
};

export default AdminDashboard;
