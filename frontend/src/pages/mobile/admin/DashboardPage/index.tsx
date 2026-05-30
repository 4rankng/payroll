import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth, addMonths, subMonths, parse } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Users, Clock, BarChart2, Activity, ChevronLeft, ChevronRight, Calendar as CalendarIcon, TrendingUp, FolderKanban, BarChart3, ChevronDown, UserPlus, Trophy, Building2 } from 'lucide-react';
import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { SalaryDistributionChartMobile } from '@/components/admin-dashboard/SalaryDistributionChartMobile';
import { GroupedStatCard } from '@/components/shared/GroupedStatCard';
import { DashboardSectionHeader } from '@/components/admin-dashboard/DashboardSectionHeader';
import { ProjectProfitabilityMobile } from '@/components/admin-dashboard/ProjectProfitabilityMobile';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { useBankUsageAllProjects } from '@/hooks/api/useDashboard';
// ── helpers ──────────────────────────────────────────────────────────────────
function formatVND(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(1)}B đ`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(0)}K đ`;
  return `${sign}${abs.toLocaleString('vi-VN')} đ`;
}

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

  const { employeeStats, salaryStats, profitStats, activityStats } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingData?.pagination?.totalRecords || 0,
    dashboardNav,
    activityStats: activityData ?? null,
    onActivityClick: (schedule) => {
      const params = new URLSearchParams();
      if (schedule) params.set('schedule', schedule);
      if (monthParam) params.set('month', monthParam);
      navigate(`/admin/dashboard/activity?${params.toString()}`);
    },
  });


  if (loading) {
    return <div className="p-4"><DashboardLoadingSkeleton /></div>;
  }

  return (
    <div className="space-y-3 pb-20">

      {/* Sticky header */}
      <div className="px-4 pt-4 pb-2 flex items-center justify-between gap-3 border-b border-border/40">
        <div>
          <h1 className="text-xl font-bold text-foreground">Tổng quan</h1>
          <p className="text-xs text-muted-foreground mt-0.5">Quản lý nhân sự & lương</p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => setSelectedMonth('all')}
            className={`h-11 px-3 flex items-center justify-center rounded-xl text-xs font-medium ${selectedMonth === 'all' ? 'bg-muted text-foreground border border-border/60' : 'text-muted-foreground'}`}
          >
            Tất cả
          </button>
          <button
            onClick={() => setSelectedMonth(format(subMonths(selectedDate, 1), 'yyyy-MM'))}
            className="h-11 w-11 flex items-center justify-center rounded-xl border border-border/60 bg-card text-muted-foreground active:bg-muted"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <div className={`flex items-center gap-1.5 h-11 px-3 rounded-xl border border-border/60 bg-card text-sm font-medium text-foreground ${selectedMonth === 'all' ? 'opacity-50' : ''}`}>
            <CalendarIcon className="h-3.5 w-3.5 text-muted-foreground" />
            {selectedMonth === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
          </div>
          <button
            onClick={() => setSelectedMonth(format(addMonths(selectedDate, 1), 'yyyy-MM'))}
            className="h-11 w-11 flex items-center justify-center rounded-xl border border-border/60 bg-card text-muted-foreground active:bg-muted"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* KPI Hero Cards */}
      {data.dashboardSummary && (
        <div className="px-4 grid grid-cols-1 sm:grid-cols-3 gap-3">
          <KpiHeroCard
            label="NV đang làm"
            value={data.dashboardSummary.total_working_employees}
            icon={Users}
            color="blue"
            sublabel={`Tổng: ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} NV`}
            onClick={dashboardNav.navigateToActiveEmployees}
          />
          <KpiHeroCard
            label="Lợi nhuận tháng này"
            value={data.dashboardSummary.total_profit_this_month}
            formattedValue={formatVND(data.dashboardSummary.total_profit_this_month)}
            icon={TrendingUp}
            color="emerald"
            sublabel={`Tổng: ${formatVND(data.dashboardSummary.total_profit)}`}
          />
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

      {/* Stat strips */}
      <div className="px-4 space-y-3">
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
          <GroupedStatCard title={`NV Đăng Nhập (${monthLabel})`} icon={Activity} stats={activityStats} />
        )}
      </div>

      {/* Salary distribution chart */}
      <div className="px-4 space-y-3">
        <DashboardSectionHeader title="Phân Bổ Lương" icon={BarChart3} />
        <SalaryDistributionChartMobile />
      </div>

      {/* Monthly Financial History */}
      <div className="px-4 space-y-3">
        <DashboardSectionHeader title="Lịch Sử Tài Chính" icon={TrendingUp} />
        <MonthlyFinancialTable />
      </div>

      {/* Project profitability — collapsible, default closed */}
      <div className="px-4">
        <Collapsible defaultOpen={true}>
          <CollapsibleTrigger asChild>
            <button className="flex items-center gap-2 w-full min-h-[44px] py-2">
              <DashboardSectionHeader title="Lợi Nhuận Dự Án" icon={FolderKanban} />
              <ChevronDown className="h-3.5 w-3.5 ml-auto text-muted-foreground transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
            </button>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <ProjectProfitabilityMobile />
          </CollapsibleContent>
        </Collapsible>
      </div>

      {/* Top paid employees */}
      <div className="px-4 space-y-3">
        <DashboardSectionHeader title={`Top Nhân Viên Được Trả Lương Cao Nhất — ${monthLabel}`} icon={Trophy} />
        <TopPaidEmployeesCard month={monthParam} />
      </div>

      {/* Bank transfer breakdown */}
      <div className="px-4 space-y-3">
        <div className="flex items-center gap-2">
          <DashboardSectionHeader title="Phân Bổ Ngân Hàng" icon={Building2} />
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

      {/* Recent employees */}
      <div className="px-4 space-y-3">
        <DashboardSectionHeader title="Nhân Viên Mới Nhất" icon={Users} />
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
