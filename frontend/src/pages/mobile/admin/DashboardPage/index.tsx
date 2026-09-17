import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { addMonths, format, parse, startOfMonth, subMonths } from 'date-fns';
import { vi } from 'date-fns/locale';
import {
  Activity,
  AlertCircle,
  ArrowRightLeft,
  BarChart2,
  BarChart3,
  Building2,
  Calendar as CalendarIcon,
  ChevronLeft,
  ChevronRight,
  Clock,
  FolderKanban,
  Loader2,
  RefreshCcw,
  TrendingUp,
  UserPlus,
  Users,
  type LucideIcon,
} from 'lucide-react';

import { MonthPicker } from '@/components/ui/month-picker';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { SalaryDistributionChartMobile } from '@/components/admin-dashboard/SalaryDistributionChartMobile';
import { GroupedStatCard } from '@/components/shared/GroupedStatCard';
import { ProjectProfitabilityMobile } from '@/components/admin-dashboard/ProjectProfitabilityMobile';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useBankUsageAllProjects, useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { useEmployeeModals, useTimesheetModals } from '@/hooks/useModalNavigation';
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
import { formatFullCurrency as formatVND } from '@/utils/formatters';

interface DashboardDisclosureProps {
  value: string;
  eyebrow: string;
  title: string;
  summary: string;
  meta?: ReactNode;
  icon: LucideIcon;
  children: ReactNode;
}

function DashboardDisclosureSection({
  value,
  eyebrow,
  title,
  summary,
  meta,
  icon: Icon,
  children,
}: DashboardDisclosureProps) {
  return (
    <AccordionItem
      value={value}
      className="admin-dashboard-mobile-disclosure overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white shadow-none"
    >
      <AccordionTrigger className="min-h-11 px-3 py-3 no-underline hover:no-underline">
        <div className="flex min-w-0 items-start gap-2 pr-1 text-left">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-primary/10 bg-primary/5">
            <Icon className="h-4 w-4 text-primary/70" />
          </span>
          <span className="min-w-0">
            <span className="block text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {eyebrow}
            </span>
            <span className="mt-1 block text-sm font-semibold text-foreground">{title}</span>
            <span className="sr-only">{summary}</span>
            {meta && (
              <span className="mt-1 inline-flex items-center text-xs font-semibold text-foreground tabular-nums">
                {meta}
              </span>
            )}
          </span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="ct-card-body gap-3 px-3 pb-3 pt-0">
        {children}
      </AccordionContent>
    </AccordionItem>
  );
}

const AdminDashboardMobile = () => {
  const navigate = useNavigate();
  const [monthPickerOpen, setMonthPickerOpen] = useState(false);
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);

  const { data: bankData } = useBankUsageAllProjects();

  const monthParam = selectedMonth === 'all' ? undefined : selectedMonth;
  const monthLabel =
    selectedMonth === 'all'
      ? 'Tất cả'
      : (() => {
          const [year, month] = selectedMonth.split('-');
          return `${month}/${year}`;
        })();

  const selectedDate =
    selectedMonth !== 'all'
      ? startOfMonth(parse(`${selectedMonth}-01`, 'yyyy-MM-dd', new Date()))
      : startOfMonth(new Date());

  const {
    data,
    loading,
    employees,
    refetch,
    isRefreshing: isDashboardDataRefreshing,
  } = useDashboardData(monthParam);
  const actions = useDashboardActions();
  const {
    data: pendingData,
    isError: hasPendingError,
    isFetching: isPendingFetching,
    refetch: refetchPending,
  } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });
  const {
    data: activityData,
    isLoading: isActivityLoading,
    isError: hasActivityError,
    isFetching: isActivityFetching,
    refetch: refetchActivity,
  } = useEmployeeActivityStats(monthParam);
  const dashboardNav = useDashboardNavigation();
  const { openTimesheetEntry } = useTimesheetModals();
  const { openAddEmployee } = useEmployeeModals();

  const pendingApprovals = pendingData?.pagination?.totalRecords ?? null;

  const handleRefresh = useCallback(async () => {
    setIsManualRefreshing(true);
    try {
      await Promise.all([
        refetch(),
        refetchPending(),
        refetchActivity(),
        Promise.resolve(actions.handleRefresh?.()),
      ]);
    } finally {
      setIsManualRefreshing(false);
    }
  }, [actions, refetch, refetchActivity, refetchPending]);

  const isDashboardRefreshing =
    isManualRefreshing ||
    isDashboardDataRefreshing ||
    isPendingFetching ||
    isActivityFetching;

  const { employeeStats, salaryStats, profitStats, activityStats } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingApprovals ?? 0,
    dashboardNav,
    activityStats: activityData ?? null,
    onActivityClick: (schedule) => {
      const params = new URLSearchParams();
      if (schedule) params.set('schedule', schedule);
      if (monthParam) params.set('month', monthParam);
      navigate(`/admin/dashboard/activity?${params.toString()}`);
    },
  });

  const resolvedSalaryStats = useMemo(
    () =>
      pendingApprovals === null
        ? salaryStats.filter((item) => item.label !== 'Chờ duyệt')
        : salaryStats,
    [pendingApprovals, salaryStats],
  );

  const quickActions = useMemo<MobileOperationAction[]>(
    () => [
      {
        label: 'Chấm công',
        icon: CalendarIcon,
        onClick: () => openTimesheetEntry(),
      },
      {
        label: 'Duyệt công',
        icon: Clock,
        onClick: dashboardNav.navigateToPendingApprovals,
        badge: pendingApprovals !== null && pendingApprovals > 0 ? pendingApprovals : undefined,
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
    ],
    [dashboardNav.navigateToPendingApprovals, navigate, openAddEmployee, openTimesheetEntry, pendingApprovals],
  );

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
  }, [dashboardNav, data.dashboardSummary, monthLabel]);

  const priorityRows = useMemo<MobileTaskRow[]>(() => {
    return [
      {
        title: 'Duyệt bảng công',
        description: hasPendingError
          ? 'Không thể tải hàng đợi toàn hệ thống. Làm mới để thử lại.'
          : pendingApprovals === null
            ? 'Đang tải hàng đợi bảng công trên toàn hệ thống'
            : pendingApprovals > 0
              ? 'Xem hàng đợi bảng công trên toàn hệ thống'
              : 'Không có bảng công đang chờ xử lý',
        value: pendingApprovals === null ? '--' : pendingApprovals.toLocaleString('vi-VN'),
        icon: AlertCircle,
        tone:
          hasPendingError || pendingApprovals === null
            ? 'neutral'
            : pendingApprovals > 0
              ? 'warning'
              : 'success',
        onClick:
          pendingApprovals !== null && pendingApprovals > 0
            ? dashboardNav.navigateToPendingApprovals
            : undefined,
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
    ];
  }, [bankData, dashboardNav, hasPendingError, navigate, pendingApprovals]);

  if (loading) {
    return (
      <MobilePageShell className="admin-dashboard-page-mobile">
        <DashboardLoadingSkeleton />
      </MobilePageShell>
    );
  }

  const dashboardSummary = data.dashboardSummary;

  return (
    <MobilePageShell className="admin-dashboard-page-mobile space-y-2 overflow-x-hidden">
      <MobilePageHeader
        title="Tổng quan"
        subtitle={format(new Date(), 'EEEE, dd/MM', { locale: vi })}
        icon={BarChart3}
        sticky={false}
        bordered={false}
        className="px-0 pt-0 [&_.shadow-sm]:shadow-none"
      />

      <div className="admin-dashboard-mobile-monthbar overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white">
        <div className="grid grid-cols-[auto_44px_minmax(0,1fr)_44px_44px] items-center gap-1 px-1 py-1">
          <button
            onClick={() => setSelectedMonth('all')}
            className={cn(
              'ct-btn ct-btn-sm h-11 min-h-11 rounded-full border-0 px-3.5 text-xs font-semibold normal-case shadow-none',
              selectedMonth === 'all'
                ? 'ct-btn-active bg-primary text-primary-foreground'
                : 'bg-muted/60 text-muted-foreground hover:bg-muted',
            )}
          >
            Tất cả
          </button>
          <button
            onClick={() => setSelectedMonth(format(subMonths(selectedDate, 1), 'yyyy-MM'))}
            className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square h-11 w-11 min-h-11 rounded-full border-0 shadow-none"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <Popover open={monthPickerOpen} onOpenChange={setMonthPickerOpen}>
            <PopoverTrigger asChild>
              <button type="button" aria-label="Chọn tháng" className="flex min-h-11 min-w-0 items-center justify-center gap-1.5 rounded-full bg-muted/60 px-2 text-xs font-semibold text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                <CalendarIcon className="h-3 w-3 shrink-0 text-muted-foreground" />
                {selectedMonth === 'all' ? 'Chọn tháng' : format(selectedDate, 'MM/yyyy', { locale: vi })}
              </button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="center">
              <MonthPicker value={selectedMonth} onChange={(month) => { setSelectedMonth(month); setMonthPickerOpen(false); }} />
            </PopoverContent>
          </Popover>
          <button
            onClick={() => setSelectedMonth(format(addMonths(selectedDate, 1), 'yyyy-MM'))}
            className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square h-11 w-11 min-h-11 rounded-full border-0 shadow-none"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isDashboardRefreshing}
            aria-busy={isDashboardRefreshing}
            className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square h-11 w-11 min-h-11 rounded-full border-0 shadow-none"
            aria-label="Làm mới số liệu"
          >
            {isDashboardRefreshing ? (
              <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
            ) : (
              <RefreshCcw className="h-4 w-4" aria-hidden="true" />
            )}
          </button>
        </div>
      </div>

      {dashboardSummary && (
        <MobileOperationsPanel
          eyebrow={`Kỳ ${monthLabel}`}
          title="Điều hành lương"
          primaryLabel="Chờ trả"
          primaryValue={formatVND(dashboardSummary.pending_salary_this_month)}
          primaryHint={`Đã trả ${formatVND(dashboardSummary.paid_salary_this_month)}`}
          metrics={operationMetrics}
          actions={quickActions}
        />
      )}

      <MobileTaskList
        title="Cần xử lý"
        items={priorityRows}
      />

      <Accordion
        type="multiple"
        className="space-y-3 motion-reduce:[&_[data-state=open]]:animate-none motion-reduce:[&_[data-state=closed]]:animate-none"
      >
        <DashboardDisclosureSection
          value="salary-workforce"
          eyebrow="Sổ vận hành"
          title="Lương và nhân sự"
          summary="Theo dõi bảng công, nhân viên đang làm và hồ sơ mới."
          meta={`${dashboardSummary?.total_working_employees.toLocaleString('vi-VN') ?? '--'} đang làm`}
          icon={Users}
        >
          <div className="space-y-3">
            {resolvedSalaryStats.length > 0 && (
              <GroupedStatCard title="Bảng công" icon={Clock} stats={resolvedSalaryStats} />
            )}
            {employeeStats.length > 0 && <GroupedStatCard title="Nhân viên" icon={Users} stats={employeeStats} />}
            {isActivityLoading && !activityData ? (
              <p className="rounded-xl border border-border/60 bg-muted/30 px-3 py-3 text-xs text-muted-foreground">
                Đang tải hoạt động nhân sự…
              </p>
            ) : hasActivityError ? (
              <p className="rounded-xl border border-border/60 bg-muted/30 px-3 py-3 text-xs text-muted-foreground">
                Không thể tải hoạt động nhân sự. Hãy làm mới để thử lại.
              </p>
            ) : activityData && activityStats.length > 0 ? (
              <GroupedStatCard title={`Hoạt động (${monthLabel})`} icon={Activity} stats={activityStats} />
            ) : null}
          </div>

          <div className="space-y-2">
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
        </DashboardDisclosureSection>

        <DashboardDisclosureSection
          value="payout-analytics"
          eyebrow="Phân tích"
          title="Chi trả và biến động"
          summary="Đối chiếu phân bổ lương, lợi nhuận và người nhận lương cao nhất."
          meta={`${formatVND(dashboardSummary?.total_profit_this_month ?? 0)} lợi nhuận`}
          icon={TrendingUp}
        >
          <div className="space-y-3">
            {profitStats.length > 0 && <GroupedStatCard title="Tài chính" icon={BarChart2} stats={profitStats} />}
          </div>

          <div className="space-y-2">
            <SectionHeader icon={BarChart3} title="Phân bổ lương" />
            <SalaryDistributionChartMobile />
          </div>

          <div className="space-y-2">
            <SectionHeader icon={Users} title={`Chi trả theo nhân viên — ${monthLabel}`} />
            <TopPaidEmployeesCard month={monthParam} />
          </div>

          <div className="space-y-2">
            <SectionHeader icon={TrendingUp} title="Lịch sử tài chính" />
            <MonthlyFinancialTable />
          </div>
        </DashboardDisclosureSection>

        <DashboardDisclosureSection
          value="projects-health"
          eyebrow="Kiểm soát"
          title="Dự án, ngân hàng và sức khỏe hệ thống"
          summary="Rà soát tài khoản nhận lương, chấm công và hạn mức ứng."
          meta={`${bankData?.projects.length.toLocaleString('vi-VN') ?? '--'} dự án`}
          icon={Building2}
        >
          <div className="space-y-2">
            <SectionHeader icon={FolderKanban} title="Lợi nhuận dự án" />
            <ProjectProfitabilityMobile />
          </div>

          <div className="space-y-2">
            <SectionHeader icon={Building2} title="Ngân hàng">
              {bankData && bankData.projects.length > 0 ? (
                <div className="ml-auto">
                  <ProjectSelector
                    projects={bankData.projects}
                    selectedId={bankProjectId}
                    onSelect={setBankProjectId}
                  />
                </div>
              ) : undefined}
            </SectionHeader>
            <BankTransferBreakdownCard selectedProjectId={bankProjectId} />
          </div>

          <div className="space-y-2">
            <SectionHeader icon={Activity} title="Sức khỏe vận hành" />
            <CheckInHealthStrip month={monthParam} />
          </div>
        </DashboardDisclosureSection>
      </Accordion>
    </MobilePageShell>
  );
};

export default AdminDashboardMobile;
