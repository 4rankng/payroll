import { useMemo, useState, type ReactNode } from 'react';
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
  TrendingUp,
  UserPlus,
  Users,
  type LucideIcon,
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
import { formatCompactCurrency as formatVND } from '@/utils/formatters';

interface DashboardDisclosureProps {
  value: string;
  eyebrow: string;
  title: string;
  summary: string;
  icon: LucideIcon;
  children: ReactNode;
}

function DashboardDisclosureSection({
  value,
  eyebrow,
  title,
  summary,
  icon: Icon,
  children,
}: DashboardDisclosureProps) {
  return (
    <AccordionItem
      value={value}
      className="admin-dashboard-mobile-disclosure overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white shadow-sm"
    >
      <AccordionTrigger className="min-h-11 px-4 py-4 no-underline hover:no-underline">
        <div className="flex min-w-0 items-start gap-3 pr-2 text-left">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-primary/10 bg-primary/5">
            <Icon className="h-4 w-4 text-primary/70" />
          </span>
          <span className="min-w-0">
            <span className="block text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              {eyebrow}
            </span>
            <span className="mt-1 block text-sm font-semibold text-foreground">{title}</span>
            <span className="mt-1 block text-xs leading-relaxed text-muted-foreground">{summary}</span>
          </span>
        </div>
      </AccordionTrigger>
      <AccordionContent className="ct-card-body gap-4 px-4 pb-4 pt-0">
        {children}
      </AccordionContent>
    </AccordionItem>
  );
}

const AdminDashboardMobile = () => {
  const navigate = useNavigate();
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);

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

  const priorityRows = useMemo<MobileTaskRow[]>(
    () => [
      {
        title: 'Duyệt bảng công',
        description: pendingApprovals > 0 ? 'Xem các bảng công đang chờ xác nhận' : 'Không có bảng công cần xử lý',
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
    ],
    [bankData, dashboardNav, navigate, pendingApprovals],
  );

  if (loading) {
    return (
      <MobilePageShell className="admin-dashboard-page-mobile">
        <DashboardLoadingSkeleton />
      </MobilePageShell>
    );
  }

  const dashboardSummary = data.dashboardSummary;

  return (
    <MobilePageShell className="admin-dashboard-page-mobile space-y-4 overflow-x-hidden">
      <MobilePageHeader
        title="Tổng quan"
        subtitle={format(new Date(), 'EEEE, dd/MM', { locale: vi })}
        icon={BarChart3}
        sticky={false}
        bordered={false}
        actions={
          pendingApprovals > 0 ? (
            <button
              type="button"
              onClick={dashboardNav.navigateToPendingApprovals}
              className="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border border-amber-200 bg-amber-50 px-2.5 text-xs font-semibold text-amber-700 shadow-xs ring-1 ring-inset ring-amber-200/70 transition-colors hover:bg-amber-100 hover:border-amber-300 hover:ring-amber-300/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400 focus-visible:ring-offset-2 active:scale-[0.98] normal-case motion-reduce:transition-none"
              aria-label={`${pendingApprovals} bảng công chờ duyệt`}
            >
              <AlertCircle className="h-3 w-3 shrink-0 text-amber-500" strokeWidth={2.5} aria-hidden="true" />
              <span className="tabular-nums">{pendingApprovals}</span>
              <span>chờ duyệt</span>
            </button>
          ) : undefined
        }
      />

      <div className="admin-dashboard-mobile-monthbar overflow-hidden rounded-2xl border border-[hsl(var(--surface-border))] bg-white">
        <div className="flex items-center gap-1.5 overflow-x-auto px-2 py-2">
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
          <div
            className={cn(
              'flex min-h-11 items-center gap-1.5 rounded-full bg-muted/60 px-3 text-xs font-semibold text-foreground',
              selectedMonth === 'all' && 'opacity-50',
            )}
          >
            <CalendarIcon className="h-3 w-3 text-muted-foreground" />
            {selectedMonth === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
          </div>
          <button
            onClick={() => setSelectedMonth(format(addMonths(selectedDate, 1), 'yyyy-MM'))}
            className="ct-btn ct-btn-ghost ct-btn-sm ct-btn-square h-11 w-11 min-h-11 rounded-full border-0 shadow-none"
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
          subtitle="Theo dõi bảng công, ví trả lương và dữ liệu nhân sự trước khi khóa sổ."
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

      <Accordion
        type="multiple"
        defaultValue={['salary-workforce']}
        className="space-y-3 motion-reduce:[&_[data-state=open]]:animate-none motion-reduce:[&_[data-state=closed]]:animate-none"
      >
        <DashboardDisclosureSection
          value="salary-workforce"
          eyebrow="Sổ vận hành"
          title="Lương và nhân sự"
          summary="Giữ phần lõi ở một khối gọn: bảng công, quân số và người mới."
          icon={Users}
        >
          <div className="space-y-3">
            {salaryStats.length > 0 && <GroupedStatCard title="Bảng công" icon={Clock} stats={salaryStats} />}
            {employeeStats.length > 0 && <GroupedStatCard title="Nhân viên" icon={Users} stats={employeeStats} />}
            {activityStats.length > 0 && (
              <GroupedStatCard title={`Hoạt động (${monthLabel})`} icon={Activity} stats={activityStats} />
            )}
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
          summary="Mở khi cần soi sâu phân bổ lương, lãi lỗ và người nhận lương cao."
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
          summary="Theo dõi phần phụ trợ và kiểm tra vận hành khi cần xác minh sâu hơn."
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
