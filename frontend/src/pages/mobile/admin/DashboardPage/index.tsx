import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth, addMonths, subMonths, parse } from 'date-fns';
import { vi } from 'date-fns/locale';
import {
  Users, Clock, BarChart2, Activity, ChevronLeft, ChevronRight,
  Calendar as CalendarIcon, TrendingUp, FolderKanban, BarChart3,
  UserPlus, Building2, ArrowRightLeft,
  AlertCircle,
} from 'lucide-react';
import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { SalaryDistributionChartMobile } from '@/components/admin-dashboard/SalaryDistributionChartMobile';
import { GroupedStatCard } from '@/components/shared/GroupedStatCard';
import { ProjectProfitabilityMobile } from '@/components/admin-dashboard/ProjectProfitabilityMobile';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
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
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import {
  MobileOperationsPanel,
  MobileTaskList,
  type MobileOperationAction,
  type MobileOperationMetric,
  type MobileTaskRow,
} from '@/components/shared/MobileOperationsPanel';
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

  const quickActions = useMemo<MobileOperationAction[]>(() => [
    {
      label: 'Chấm công',
      icon: CalendarIcon,
      onClick: () => openTimesheetEntry(),
    },
    {
      label: 'Duyệt công',
      icon: Clock,
      onClick: dashboardNav.navigateToPendingApprovals,
      badge: pendingApprovals > 0 ? pendingApprovals : undefined,
    },
    {
      label: 'Ví tiền',
      icon: ArrowRightLeft,
      onClick: () => navigate('/admin/wallet'),
    },
    {
      label: 'Thêm NV',
      icon: UserPlus,
      onClick: () => openAddEmployee(),
    },
  ], [dashboardNav.navigateToPendingApprovals, navigate, openTimesheetEntry, openAddEmployee, pendingApprovals]);

  const operationMetrics = useMemo<MobileOperationMetric[]>(() => {
    if (!data.dashboardSummary) return [];

    return [
      {
        label: 'Nhân viên đang làm',
        value: data.dashboardSummary.total_working_employees.toLocaleString('vi-VN'),
        helper: `Tổng ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} nhân viên`,
        icon: Users,
        tone: 'primary',
        onClick: dashboardNav.navigateToActiveEmployees,
      },
      {
        label: 'Bảng công chờ duyệt',
        value: pendingApprovals.toLocaleString('vi-VN'),
        helper: pendingApprovals > 0 ? 'Cần xử lý trước trả lương' : 'Không có mục đang chờ',
        icon: AlertCircle,
        tone: pendingApprovals > 0 ? 'warning' : 'success',
        onClick: pendingApprovals > 0 ? dashboardNav.navigateToPendingApprovals : undefined,
      },
      {
        label: 'Đã trả kỳ này',
        value: formatVND(data.dashboardSummary.paid_salary_this_month),
        helper: monthLabel,
        icon: ArrowRightLeft,
        tone: 'success',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
      {
        label: 'Nhân viên mới',
        value: data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN'),
        helper: 'Trong kỳ đang xem',
        icon: UserPlus,
        tone: 'neutral',
        onClick: dashboardNav.navigateToNewEmployees,
      },
    ];
  }, [dashboardNav, data.dashboardSummary, monthLabel, pendingApprovals]);

  const priorityRows = useMemo<MobileTaskRow[]>(() => [
    {
      title: 'Duyệt bảng công',
      description: pendingApprovals > 0
        ? 'Xem các bảng công đang chờ xác nhận'
        : 'Không có bảng công cần xử lý',
      value: pendingApprovals.toLocaleString('vi-VN'),
      icon: AlertCircle,
      tone: pendingApprovals > 0 ? 'warning' : 'success',
      onClick: pendingApprovals > 0 ? dashboardNav.navigateToPendingApprovals : undefined,
    },
    {
      title: 'Đối soát ví trả lương',
      description: 'Kiểm tra số dư và lịch chuyển tiền',
      value: 'Ví',
      icon: ArrowRightLeft,
      tone: 'primary',
      onClick: () => navigate('/admin/wallet'),
    },
    {
      title: 'Dự án & tài khoản ngân hàng',
      description: 'Kiểm tra dữ liệu nhận tiền theo dự án',
      value: bankData ? bankData.projects.length.toLocaleString('vi-VN') : '--',
      icon: Building2,
      tone: 'neutral',
      onClick: () => navigate('/admin/projects'),
    },
  ], [bankData, dashboardNav, navigate, pendingApprovals]);

  if (loading) {
    return (
      <MobilePageShell>
        <DashboardLoadingSkeleton />
      </MobilePageShell>
    );
  }

  const dashboardSummary = data.dashboardSummary;

  return (
    <MobilePageShell className="space-y-4">
      <MobilePageHeader
        title="Tổng quan"
        subtitle={format(new Date(), 'EEEE, dd/MM', { locale: vi })}
        icon={BarChart3}
        sticky={false}
        bordered={false}
        actions={
          pendingApprovals > 0 ? (
            <button
              onClick={dashboardNav.navigateToPendingApprovals}
              className="flex min-h-11 items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-3 text-xs font-bold text-amber-700 transition-transform active:scale-95"
            >
              <AlertCircle className="h-3.5 w-3.5" />
              {pendingApprovals} chờ duyệt
            </button>
          ) : undefined
        }
      />

      <div className="overflow-hidden rounded-[28px] border border-[hsl(var(--surface-border))] bg-white">
        <div className="flex items-center gap-1.5 overflow-x-auto px-2 py-2">
          <button
            onClick={() => setSelectedMonth('all')}
            className={cn(
              'min-h-11 rounded-full px-3.5 text-xs font-semibold transition-colors',
              selectedMonth === 'all'
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted/60 text-muted-foreground',
            )}
          >
            Tất cả
          </button>
          <button
            onClick={() => setSelectedMonth(format(subMonths(selectedDate, 1), 'yyyy-MM'))}
            className="flex h-11 w-11 items-center justify-center rounded-full bg-muted/60 text-muted-foreground active:bg-muted"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <div className={cn(
            'flex min-h-11 items-center gap-1.5 rounded-full bg-muted/60 px-3 text-xs font-semibold text-foreground',
            selectedMonth === 'all' && 'opacity-50',
          )}>
            <CalendarIcon className="h-3 w-3 text-muted-foreground" />
            {selectedMonth === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
          </div>
          <button
            onClick={() => setSelectedMonth(format(addMonths(selectedDate, 1), 'yyyy-MM'))}
            className="flex h-11 w-11 items-center justify-center rounded-full bg-muted/60 text-muted-foreground active:bg-muted"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      {dashboardSummary && (
        <MobileOperationsPanel
          eyebrow={`Kỳ ${monthLabel}`}
          title="Điều hành lương"
          subtitle="Theo dõi bảng công, ví trả lương và dữ liệu nhân sự"
          primaryLabel="Chờ trả"
          primaryValue={formatVND(dashboardSummary.pending_salary_this_month)}
          primaryHint={`Đã trả ${formatVND(dashboardSummary.paid_salary_this_month)}`}
          metrics={operationMetrics}
          actions={quickActions}
        />
      )}

      <MobileTaskList
        title="Cần xử lý"
        subtitle="Các việc ảnh hưởng trực tiếp đến kỳ lương"
        items={priorityRows}
      />

      <div className="space-y-3">
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

      <div>
        <SectionHeader icon={BarChart3} title="Phân bổ lương" />
        <SalaryDistributionChartMobile />
      </div>

      <div>
        <SectionHeader icon={FolderKanban} title="Lợi nhuận dự án" />
        <ProjectProfitabilityMobile />
      </div>

      <div>
        <SectionHeader icon={Users} title={`Chi trả theo nhân viên — ${monthLabel}`} />
        <TopPaidEmployeesCard month={monthParam} />
      </div>

      <div>
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

      <div>
        <SectionHeader icon={TrendingUp} title="Lịch sử tài chính" />
        <MonthlyFinancialTable />
      </div>

      <div>
        <CheckInHealthStrip month={monthParam} />
      </div>

      <div className="pb-2">
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
    </MobilePageShell>
  );
};

export default AdminDashboardMobile;
