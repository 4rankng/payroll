import { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { format, startOfMonth } from 'date-fns';
// UU PRO restyle (W13a): lucide → verified @untitledui/icons. The shared
// admin-dashboard icon slots are widened to ComponentType<SVGProps> (W13a
// follow-up), so UU's plain-FC icons fit everywhere on this page.
import {
  AlertTriangle,
  FolderCheck,
  SwitchHorizontal01,
} from '@untitledui/icons';

import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { DashboardHeader } from '@/components/admin-dashboard/DashboardHeader';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { ActivityUsersSheet } from '@/components/admin-dashboard/ActivityUsersSheet';
import { ProjectProfitabilityCard } from '@/components/admin-dashboard/ProjectProfitabilityCard';
import { SalaryDistributionChart } from '@/components/admin-dashboard/SalaryDistributionChart';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';
import {
  DashboardActivityPanel,
  DashboardMetricStrip,
  DashboardPriorityList,
  type DashboardMetricItem,
  type DashboardPriorityItem,
} from '@/components/admin-dashboard/DashboardOverview';
import { ConsoleSectionHeading } from '@/components/shared/console-panel';
import { useApprovalsList } from '@/hooks/api/useApprovals';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { CheckInRosterReminderBanner } from '@/components/admin-dashboard/CheckInRosterReminderBanner';
import { formatFullCurrency as formatVND, formatCompactVND } from '@/utils/formatters';

const AdminDashboard = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [activitySchedule, setActivitySchedule] = useState<'weekly' | 'monthly' | 'flexible' | null>(null);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);

  const handleMonthChange = useCallback((value: string) => {
    setSelectedMonth(value);
  }, []);

  const monthParam = selectedMonth === 'all' ? undefined : selectedMonth;
  const monthLabel =
    selectedMonth === 'all'
      ? 'Tất cả'
      : (() => {
          const [year, month] = selectedMonth.split('-');
          return `${month}/${year}`;
        })();

  const { data, loading, employees, refetch, isRefreshing } = useDashboardData(monthParam);
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
  const navigate = useNavigate();
  const dashboardNav = useDashboardNavigation();
  // Approval-engine queue counts (pending + rescheduled + escalated). The
  // priority row renders only when non-zero (row law: no zero-value rows).
  const approvalsQueue = useApprovalsList({ statuses: ['pending', 'rescheduled'], pageSize: 1 });
  const approvalsEscalated = useApprovalsList({ statuses: ['escalated'], pageSize: 1 });

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
    isManualRefreshing || isRefreshing || isPendingFetching || isActivityFetching;

  const { activityStats } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingApprovals ?? 0,
    dashboardNav,
    activityStats: activityData ?? null,
    onActivityClick: setActivitySchedule,
  });

  const activityItems = useMemo(
    () =>
      activityStats
        .map((item) => ({
          label: typeof item.label === 'string' ? item.label : 'Hoạt động',
          value: item.value,
          onClick: item.onClick,
        }))
        // Non-zero row law: a zero channel is noise, not a signal.
        .filter((item) => Number(item.value) > 0),
    [activityStats],
  );

  const activityProjectItems = useMemo(
    () =>
      (activityData?.active_by_project ?? [])
        .map((project) => ({
          label: project.project_name,
          value: project.count,
        }))
        .filter((project) => Number(project.value) > 0),
    [activityData],
  );

  const metricItems = useMemo<DashboardMetricItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    // Zone ownership: the strip owns period OUTCOMES only. Pending payouts and
    // approval-queue counts belong to DashboardPriorityList below — never
    // repeat a metric between the two adjacent zones.
    return [
      {
        label: 'Đã trả kỳ này',
        value: formatCompactVND(data.dashboardSummary.paid_salary_this_month),
        fullValue: formatVND(data.dashboardSummary.paid_salary_this_month),
        context: `Tổng đã trả ${formatVND(data.dashboardSummary.total_paid_salary)}`,
        tone: 'primary',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
      {
        label: 'Lợi nhuận kỳ này',
        value: formatCompactVND(data.dashboardSummary.total_profit_this_month),
        fullValue: formatVND(data.dashboardSummary.total_profit_this_month),
        context: `Lũy kế ${formatVND(data.dashboardSummary.total_profit)}`,
        tone: 'success',
      },
      {
        label: 'Nhân sự đang làm',
        value: data.dashboardSummary.total_working_employees.toLocaleString('vi-VN'),
        context: `${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} nhân viên · ${data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN')} mới`,
        tone: 'neutral',
        onClick: dashboardNav.navigateToActiveEmployees,
      },
    ];
  }, [dashboardNav, data.dashboardSummary]);

  const priorityItems = useMemo<DashboardPriorityItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    const items: DashboardPriorityItem[] = [];
    const pendingKnown = pendingApprovals !== null;
    const pendingCount = pendingApprovals ?? 0;

    // Non-zero row law: a completed queue is not a task. Only a real value,
    // a load failure, or a pending fetch earns a row.
    if (hasPendingError || !pendingKnown || pendingCount > 0) {
      items.push({
        title: 'Bảng công chờ duyệt',
        detail: hasPendingError
          ? 'Không thể tải hàng đợi toàn hệ thống.'
          : !pendingKnown
            ? 'Đang tải hàng đợi bảng công.'
            : 'Xử lý trước khi khóa sổ lương.',
        value: !pendingKnown ? '--' : pendingCount.toLocaleString('vi-VN'),
        statusLabel: hasPendingError ? 'Lỗi tải' : !pendingKnown ? 'Đang tải' : 'Cần duyệt',
        icon: AlertTriangle,
        tone: hasPendingError || !pendingKnown ? 'neutral' : 'warning',
        onClick:
          pendingKnown && pendingCount > 0
            ? dashboardNav.navigateToPendingApprovals
            : undefined,
      });
    }

    if (data.dashboardSummary.pending_salary_this_month > 0) {
      items.push({
        title: 'Lương chờ giải ngân',
        detail: `Sau khi chốt công kỳ ${monthLabel}.`,
        value: formatVND(data.dashboardSummary.pending_salary_this_month),
        statusLabel: 'Cần theo dõi',
        icon: SwitchHorizontal01,
        tone: 'primary',
        onClick: dashboardNav.navigateToSalaryLedger,
      });
    }

    const approvalsTotal =
      (approvalsQueue.data?.pagination?.totalRecords ?? 0) +
      (approvalsEscalated.data?.pagination?.totalRecords ?? 0);

    if (approvalsTotal > 0) {
      items.push({
        title: 'Yêu cầu phê duyệt',
        detail: 'Đang chờ duyệt hoặc đã leo thang.',
        value: approvalsTotal.toLocaleString('vi-VN'),
        statusLabel:
          (approvalsEscalated.data?.pagination?.totalRecords ?? 0) > 0
            ? 'Có leo thang'
            : 'Cần duyệt',
        icon: FolderCheck,
        tone:
          (approvalsEscalated.data?.pagination?.totalRecords ?? 0) > 0
            ? 'warning'
            : 'primary',
        onClick: () => navigate('/admin/approvals'),
      });
    }

    return items;
  }, [
    dashboardNav,
    data.dashboardSummary,
    hasPendingError,
    monthLabel,
    pendingApprovals,
    approvalsQueue.data,
    approvalsEscalated.data,
    navigate,
  ]);

  if (loading) {
    return (
      <div className="admin-dashboard-page p-3 sm:p-4">
        <DashboardLoadingSkeleton />
      </div>
    );
  }

  return (
    <div className="admin-dashboard-page min-h-full bg-transparent">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-3 overflow-x-hidden p-3 sm:p-4">
        <div className="motion-reduce:animate-none motion-reduce:opacity-100">
          <DashboardHeader value={selectedMonth} onChange={handleMonthChange} />
        </div>

        <CheckInRosterReminderBanner />

        <DashboardMetricStrip items={metricItems} />

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading
            kicker="Ưu tiên"
            title="Vận hành"
            meta={`Kỳ ${monthLabel}`}
          />
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-12 lg:items-stretch">
            <DashboardPriorityList
              items={priorityItems}
              isRefreshing={isDashboardRefreshing}
              onRefresh={handleRefresh}
              className="lg:col-span-7"
            />
            <DashboardActivityPanel
              monthLabel={monthLabel}
              totalActive={activityData?.active_total}
              projectItems={activityProjectItems}
              items={activityItems}
              isLoading={isActivityLoading && !activityData}
              hasError={hasActivityError}
              className="lg:col-span-5"
            />
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading
            kicker="Tiền và hiệu quả"
            title="Tài chính"
            meta="12 tháng gần nhất"
          />
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-12 lg:items-stretch">
            <MonthlyFinancialTable className="lg:col-span-7" />
            <BankTransferBreakdownCard className="lg:col-span-5" />
            <ProjectProfitabilityCard className="lg:col-span-12" />
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading kicker="Con người" title="Nhân sự" meta={monthLabel} />
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-12 lg:items-stretch">
            <SalaryDistributionChart className="lg:col-span-7" />
            <TopPaidEmployeesCard month={monthParam} className="lg:col-span-5" />
            <RecentEmployeesCard
              employees={employees.employees}
              isLoading={employees.isLoading}
              isLoadingMore={employees.isLoadingMore}
              totalEmployees={employees.totalEmployees}
              weeks={employees.weeks}
              onEmployeeClick={(employee) => actions.handleEmployeeClick?.(employee)}
              onLoadMore={employees.loadMore}
              className="lg:col-span-12"
            />
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <ConsoleSectionHeading
            kicker="Sức khỏe"
            title="Chấm công và hạn mức"
            meta={monthLabel}
          />
          <CheckInHealthStrip month={monthParam} />
        </section>
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
