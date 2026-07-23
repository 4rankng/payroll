import { useCallback, useMemo, useState } from 'react';
import { format, startOfMonth } from 'date-fns';
import {
  AlertTriangle,
  ArrowRightLeft,
  BarChart3,
  Building2,
  FolderKanban,
  ShieldCheck,
  TrendingUp,
  Trophy,
  Users,
  WalletCards,
} from 'lucide-react';

import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { DashboardHeader } from '@/components/admin-dashboard/DashboardHeader';
import { DashboardSectionHeader } from '@/components/admin-dashboard/DashboardSectionHeader';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { ActivityUsersSheet } from '@/components/admin-dashboard/ActivityUsersSheet';
import { ProjectProfitabilityCard } from '@/components/admin-dashboard/ProjectProfitabilityCard';
import { SalaryDistributionChart } from '@/components/admin-dashboard/SalaryDistributionChart';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';
import {
  DashboardActivityPanel,
  DashboardAreaHeader,
  DashboardMetricStrip,
  DashboardPriorityList,
  type DashboardMetricItem,
  type DashboardPriorityItem,
} from '@/components/admin-dashboard/DashboardOverview';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useBankUsageAllProjects, useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { formatFullCurrency as formatVND } from '@/utils/formatters';

const AdminDashboard = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [activitySchedule, setActivitySchedule] = useState<'weekly' | 'monthly' | 'flexible' | null>(null);
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);

  const { data: bankData } = useBankUsageAllProjects();

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
  const dashboardNav = useDashboardNavigation();

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
      activityStats.map((item) => ({
        label: typeof item.label === 'string' ? item.label : 'Hoạt động',
        value: item.value,
        onClick: item.onClick,
      })),
    [activityStats],
  );

  const metricItems = useMemo<DashboardMetricItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    return [
      {
        label: 'Đã trả kỳ này',
        value: formatVND(data.dashboardSummary.paid_salary_this_month),
        context: `Tổng đã trả ${formatVND(data.dashboardSummary.total_paid_salary)}`,
        icon: WalletCards,
        tone: 'primary',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
      {
        label: 'Chờ giải ngân',
        value: formatVND(data.dashboardSummary.pending_salary_this_month),
        context: hasPendingError
          ? 'Không thể tải hàng đợi bảng công'
          : pendingApprovals === null
            ? 'Đang tải hàng đợi bảng công'
            : `${pendingApprovals.toLocaleString('vi-VN')} bảng công chờ duyệt trên toàn hệ thống`,
        icon: ArrowRightLeft,
        tone: data.dashboardSummary.pending_salary_this_month > 0 ? 'warning' : 'neutral',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
      {
        label: 'Lợi nhuận kỳ này',
        value: formatVND(data.dashboardSummary.total_profit_this_month),
        context: `Lũy kế ${formatVND(data.dashboardSummary.total_profit)}`,
        icon: TrendingUp,
        tone: 'success',
      },
      {
        label: 'Nhân sự đang làm',
        value: data.dashboardSummary.total_working_employees.toLocaleString('vi-VN'),
        context: `${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} nhân viên · ${data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN')} mới`,
        icon: Users,
        tone: 'neutral',
        onClick: dashboardNav.navigateToActiveEmployees,
      },
    ];
  }, [dashboardNav, data.dashboardSummary, hasPendingError, pendingApprovals]);

  const priorityItems = useMemo<DashboardPriorityItem[]>(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    return [
      {
        title: 'Bảng công chờ duyệt',
        detail:
          hasPendingError
            ? 'Không thể tải hàng đợi toàn hệ thống. Làm mới số liệu để thử lại.'
            : pendingApprovals === null
              ? 'Đang tải hàng đợi bảng công trên toàn hệ thống.'
              : pendingApprovals > 0
                ? 'Xử lý các bảng công này trước khi khóa sổ lương.'
                : 'Không có bảng công đang chờ xử lý trên toàn hệ thống.',
        value: pendingApprovals === null ? '--' : pendingApprovals.toLocaleString('vi-VN'),
        statusLabel: hasPendingError
          ? 'Không thể tải'
          : pendingApprovals === null
            ? 'Đang tải'
            : pendingApprovals > 0
              ? 'Cần duyệt'
              : 'Đã xử lý',
        icon: AlertTriangle,
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
        title: 'Lương chờ giải ngân',
        detail: `Đã trả ${formatVND(data.dashboardSummary.paid_salary_this_month)} trong kỳ ${monthLabel}.`,
        value: formatVND(data.dashboardSummary.pending_salary_this_month),
        statusLabel:
          data.dashboardSummary.pending_salary_this_month > 0 ? 'Cần theo dõi' : 'Đã giải ngân',
        icon: ArrowRightLeft,
        tone: data.dashboardSummary.pending_salary_this_month > 0 ? 'primary' : 'success',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
    ];
  }, [dashboardNav, data.dashboardSummary, hasPendingError, monthLabel, pendingApprovals]);

  if (loading) {
    return (
      <div className="admin-dashboard-page p-4 lg:p-6">
        <DashboardLoadingSkeleton />
      </div>
    );
  }

  return (
    <div className="admin-dashboard-page min-h-full bg-transparent">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-6 overflow-x-hidden p-4 lg:p-6">
        <div className="motion-reduce:animate-none motion-reduce:opacity-100">
          <DashboardHeader value={selectedMonth} onChange={handleMonthChange} />
        </div>

        <DashboardMetricStrip items={metricItems} />

        <div
          data-slot="admin-dashboard-bento"
          className="admin-dashboard-bento grid grid-cols-1 gap-4 xl:grid-cols-12 xl:items-start"
        >
          <div className="min-w-0 xl:col-span-7">
            <DashboardPriorityList
              items={priorityItems}
              isRefreshing={isDashboardRefreshing}
              onRefresh={handleRefresh}
            />
          </div>

          <div className="min-w-0 xl:col-span-5">
            <DashboardActivityPanel
              monthLabel={monthLabel}
              totalActive={activityData?.active_total}
              items={activityItems}
              isLoading={isActivityLoading && !activityData}
              hasError={hasActivityError}
            />
          </div>

          <DashboardAreaHeader
            eyebrow="Tài chính"
            title="Dòng tiền và hiệu quả"
            description="Rà soát lịch sử tài chính, cơ cấu ngân hàng và lợi nhuận theo từng dự án."
            className="mt-2 xl:col-span-12"
          />

          <div
            data-slot="dashboard-finance-bento"
            className="grid min-w-0 gap-4 xl:col-span-12 xl:grid-cols-12 xl:items-start"
          >
            <div className="min-w-0 space-y-4 xl:col-span-7">
              <section data-slot="dashboard-financial-history" className="admin-dashboard-finance-group min-w-0 space-y-3">
                <DashboardSectionHeader
                  eyebrow="Sổ dòng tiền"
                  title="Lịch sử tài chính"
                  subtitle="Sổ chi phí, doanh thu và lợi nhuận tích lũy 12 tháng gần nhất."
                  icon={TrendingUp}
                />
                <MonthlyFinancialTable />
              </section>

              <section data-slot="dashboard-bank-distribution" className="admin-dashboard-finance-group min-w-0 space-y-3">
                <DashboardSectionHeader
                  eyebrow="Chi trả"
                  title="Ngân hàng nhận lương"
                  subtitle="Phân bổ nhân sự theo ngân hàng để rà soát dữ liệu nhận tiền."
                  icon={Building2}
                  actions={
                    bankData && bankData.projects.length > 0 ? (
                      <ProjectSelector
                        projects={bankData.projects}
                        selectedId={bankProjectId}
                        onSelect={setBankProjectId}
                      />
                    ) : undefined
                  }
                />
                <BankTransferBreakdownCard selectedProjectId={bankProjectId} />
              </section>
            </div>

            <section
              data-slot="dashboard-project-profitability"
              className="admin-dashboard-finance-group min-w-0 space-y-3 xl:col-span-5"
            >
              <DashboardSectionHeader
                eyebrow="Dự án"
                title="Lợi nhuận dự án"
                subtitle="Theo dõi đà lợi nhuận và dự án đang kéo kết quả kỳ lương."
                icon={FolderKanban}
              />
              <ProjectProfitabilityCard />
            </section>
          </div>

          <DashboardAreaHeader
            eyebrow="Nhân sự"
            title="Chi trả và biến động nhân sự"
            description="Phát hiện phân bổ lương bất thường, đối chiếu nhóm nhận cao và hồ sơ vừa được thêm."
            className="mt-2 xl:col-span-12"
          />

          <div
            data-slot="dashboard-workforce-bento"
            className="grid min-w-0 gap-4 xl:col-span-12 xl:grid-cols-12 xl:items-start"
          >
            <section
              data-slot="dashboard-salary-distribution"
              className="admin-dashboard-workforce-group min-w-0 space-y-3 xl:col-span-7 xl:col-start-1 xl:row-start-1"
            >
              <DashboardSectionHeader
                eyebrow="Phân tích chi trả"
                title="Phân bổ lương"
                subtitle="Các khoảng lương điển hình để kiểm tra độ lệch trước khi giải ngân."
                icon={BarChart3}
              />
              <SalaryDistributionChart />
            </section>

            <section
              data-slot="dashboard-top-paid"
              className="admin-dashboard-workforce-group min-w-0 space-y-3 xl:col-span-5 xl:col-start-8 xl:row-span-2 xl:row-start-1"
            >
              <DashboardSectionHeader
                eyebrow="Nhân sự"
                title={`Chi trả theo nhân viên — ${monthLabel}`}
                subtitle="Nhân viên nhận lương cao nhất trong kỳ đang xem."
                icon={Trophy}
              />
              <TopPaidEmployeesCard month={monthParam} />
            </section>

            <section
              data-slot="dashboard-recent-employees"
              className="admin-dashboard-workforce-group min-w-0 space-y-3 xl:col-span-7 xl:col-start-1 xl:row-start-2"
            >
              <DashboardSectionHeader
                eyebrow="Nhân sự"
                title="Nhân viên mới nhất"
                subtitle="Các hồ sơ vừa được thêm để đối chiếu biên chế dự án."
                icon={Users}
              />
              <RecentEmployeesCard
                employees={employees.employees}
                isLoading={employees.isLoading}
                isLoadingMore={employees.isLoadingMore}
                totalEmployees={employees.totalEmployees}
                weeks={employees.weeks}
                onEmployeeClick={(employee) => actions.handleEmployeeClick?.(employee)}
                onLoadMore={employees.loadMore}
              />
            </section>
          </div>

          <section
            data-slot="dashboard-operational-health"
            className="admin-dashboard-health-group min-w-0 space-y-3 xl:col-span-12"
          >
            <DashboardSectionHeader
              eyebrow="Sức khỏe vận hành"
              title="Tự chấm công và hạn mức ứng lương"
              subtitle="Đặt kiểm soát vận hành ở cuối sổ để rà soát trước khi chốt kỳ."
              icon={ShieldCheck}
            />
            <CheckInHealthStrip month={monthParam} />
          </section>
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
