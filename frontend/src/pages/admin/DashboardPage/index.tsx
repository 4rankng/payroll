import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { format, startOfMonth } from 'date-fns';
import {
  Activity,
  AlertTriangle,
  ArrowRightLeft,
  BarChart3,
  Building2,
  Clock,
  FolderKanban,
  ShieldCheck,
  TrendingUp,
  Trophy,
  UserPlus,
  Users,
  type LucideIcon,
} from 'lucide-react';

import { DashboardLoadingSkeleton } from '@/components/ui/loading-states';
import { DashboardHeader } from '@/components/admin-dashboard/DashboardHeader';
import { DashboardSectionHeader } from '@/components/admin-dashboard/DashboardSectionHeader';
import { RecentEmployeesCard } from '@/components/admin-dashboard/RecentEmployeesCard';
import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { ActivityUsersSheet } from '@/components/admin-dashboard/ActivityUsersSheet';
import { ProjectProfitabilityCard } from '@/components/admin-dashboard/ProjectProfitabilityCard';
import { SalaryDistributionChart } from '@/components/admin-dashboard/SalaryDistributionChart';
import { MonthlyFinancialTable } from '@/components/admin-dashboard/MonthlyFinancialTable';
import { TopPaidEmployeesCard } from '@/components/admin-dashboard/TopPaidEmployeesCard';
import { BankTransferBreakdownCard, ProjectSelector } from '@/components/admin-dashboard/BankTransferBreakdownCard';
import { CheckInHealthStrip } from '@/components/admin-dashboard/CheckInHealthStrip';
import { useDashboardData } from '@/hooks/admin-dashboard/useDashboardData';
import { useDashboardActions } from '@/hooks/admin-dashboard/useDashboardActions';
import { useDashboardStats } from '@/hooks/admin-dashboard/useDashboardStats';
import { useTimesheets } from '@/hooks/api/useTimesheets';
import { useBankUsageAllProjects, useEmployeeActivityStats } from '@/hooks/api/useDashboard';
import { useDashboardNavigation } from '@/hooks/useDashboardNavigation';
import { cn } from '@/lib/utils';
import { formatCompactCurrency as formatVND } from '@/utils/formatters';

interface AttentionActionProps {
  title: string;
  detail: string;
  value: string;
  icon: LucideIcon;
  tone: 'warning' | 'primary' | 'neutral';
  onClick?: () => void;
}

interface DashboardLedgerPanelProps {
  eyebrow: string;
  title: string;
  subtitle: string;
  icon: LucideIcon;
  children: ReactNode;
  actions?: ReactNode;
  className?: string;
}

const TONE_CLASSES: Record<AttentionActionProps['tone'], string> = {
  warning:
    'border-amber-200 bg-amber-50/80 text-amber-900 hover:border-amber-300 hover:bg-amber-50',
  primary:
    'border-primary/15 bg-primary/[0.045] text-foreground hover:border-primary/25 hover:bg-primary/[0.065]',
  neutral:
    'border-border/70 bg-background text-foreground hover:border-border hover:bg-muted/20',
};

function DashboardLedgerPanel({
  eyebrow,
  title,
  subtitle,
  icon,
  children,
  actions,
  className,
}: DashboardLedgerPanelProps) {
  return (
    <section
      className={cn(
        'admin-dashboard-panel rounded-[28px] border border-border/70 bg-white shadow-sm',
        className,
      )}
    >
      <div className="border-b border-border/60 px-4 py-4 sm:px-5">
        <DashboardSectionHeader
          eyebrow={eyebrow}
          title={title}
          subtitle={subtitle}
          icon={icon}
          actions={actions}
        />
      </div>
      <div className="p-4 sm:p-5">{children}</div>
    </section>
  );
}

function AttentionAction({ title, detail, value, icon: Icon, tone, onClick }: AttentionActionProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      className={cn(
        'admin-dashboard-attention-item flex min-h-11 w-full flex-col justify-between rounded-3xl border px-4 py-3 text-left transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        'motion-reduce:transition-none',
        TONE_CLASSES[tone],
        !onClick && 'cursor-default',
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            {title}
          </p>
          <p className="mt-2 text-xl font-semibold tracking-tight text-foreground sm:text-2xl">{value}</p>
        </div>
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-current/10 bg-white/70">
          <Icon className="h-4 w-4" />
        </span>
      </div>
      <p className="mt-3 text-xs leading-relaxed text-muted-foreground">{detail}</p>
    </button>
  );
}

const AdminDashboard = () => {
  const [selectedMonth, setSelectedMonth] = useState<string>(format(startOfMonth(new Date()), 'yyyy-MM'));
  const [activitySchedule, setActivitySchedule] = useState<'weekly' | 'monthly' | 'flexible' | null>(null);
  const [bankProjectId, setBankProjectId] = useState<number | null>(null);

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

  const { data, loading, employees, refetch } = useDashboardData(monthParam);
  const actions = useDashboardActions();
  const { data: pendingData } = useTimesheets({ status: 'pending_approval', page: 1, pageSize: 20 });
  const { data: activityData } = useEmployeeActivityStats(monthParam);
  const dashboardNav = useDashboardNavigation();

  const pendingApprovals = pendingData?.pagination?.totalRecords || 0;

  const handleRefresh = useCallback(() => {
    refetch();
    actions.handleRefresh?.();
  }, [actions, refetch]);

  const {
    employeeStats,
    salaryStats,
    profitStats,
    activityStats,
  } = useDashboardStats({
    dashboardSummary: data.dashboardSummary,
    totalPendingApprovals: pendingApprovals,
    dashboardNav,
    activityStats: activityData ?? null,
    onActivityClick: setActivitySchedule,
  });

  const buildInlineItems = useCallback(
    (items: Array<{ label: string; value: string | number; unit?: string; onClick?: () => void }>): InlineStatItem[] =>
      items.map((item) => ({
        label: item.label,
        value: item.value,
        unit: item.unit,
        onClick: item.onClick,
      })),
    [],
  );

  const statGroups = useMemo(
    () =>
      [
        {
          key: 'salary',
          eyebrow: 'Kỳ lương',
          title: 'Sổ chi trả',
          subtitle: 'Khoản đang chờ, đã trả và hàng đợi duyệt công.',
          icon: Clock,
          items: buildInlineItems(salaryStats),
        },
        {
          key: 'profit',
          eyebrow: 'Dòng tiền',
          title: 'Tài chính',
          subtitle: 'Tiền ứng và lợi nhuận theo kỳ đang xem.',
          icon: TrendingUp,
          items: buildInlineItems(profitStats),
        },
        {
          key: 'people',
          eyebrow: 'Nhân sự',
          title: 'Biên chế công trường',
          subtitle: 'Tổng quân số, đang làm và nhân viên mới.',
          icon: Users,
          items: buildInlineItems(employeeStats),
        },
        {
          key: 'activity',
          eyebrow: monthLabel === 'Tất cả' ? 'Đăng nhập' : `Đăng nhập ${monthLabel}`,
          title: 'Hoạt động tài khoản',
          subtitle: 'Nhấn để xem nhân viên hoạt động theo hình thức trả lương.',
          icon: Activity,
          items: buildInlineItems(activityStats),
        },
      ].filter((group) => group.items.length > 0),
    [activityStats, buildInlineItems, employeeStats, monthLabel, profitStats, salaryStats],
  );

  const attentionItems = useMemo(() => {
    if (!data.dashboardSummary) {
      return [];
    }

    return [
      {
        title: 'Bảng công chờ duyệt',
        detail:
          pendingApprovals > 0
            ? 'Xử lý các bảng công này trước khi khóa sổ lương kỳ hiện tại.'
            : 'Không có bảng công đang chờ xử lý trong kỳ hiện tại.',
        value: pendingApprovals.toLocaleString('vi-VN'),
        icon: AlertTriangle,
        tone: pendingApprovals > 0 ? 'warning' : 'neutral',
        onClick: pendingApprovals > 0 ? dashboardNav.navigateToPendingApprovals : undefined,
      },
      {
        title: 'Lương chưa giải ngân',
        detail: `Đã trả ${formatVND(data.dashboardSummary.paid_salary_this_month)} trong kỳ ${monthLabel}.`,
        value: formatVND(data.dashboardSummary.pending_salary_this_month),
        icon: ArrowRightLeft,
        tone: data.dashboardSummary.pending_salary_this_month > 0 ? 'primary' : 'neutral',
        onClick: dashboardNav.navigateToSalaryLedger,
      },
      {
        title: 'Nhân viên đang làm',
        detail: `${data.dashboardSummary.employees_hired_this_month.toLocaleString('vi-VN')} nhân viên mới trong ${monthLabel}.`,
        value: `${data.dashboardSummary.total_working_employees.toLocaleString('vi-VN')} / ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')}`,
        icon: Users,
        tone: 'neutral',
        onClick: dashboardNav.navigateToActiveEmployees,
      },
    ] satisfies AttentionActionProps[];
  }, [dashboardNav, data.dashboardSummary, monthLabel, pendingApprovals]);

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

        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(320px,1fr)]">
          <DashboardLedgerPanel
            eyebrow="Sổ lương công trường"
            title="Ngoại lệ và việc cần khóa sổ"
            subtitle="Ưu tiên các điểm nghẽn ảnh hưởng trực tiếp đến bảng công, giải ngân và quân số đang làm."
            icon={AlertTriangle}
            actions={
              <button
                type="button"
                onClick={handleRefresh}
                className="min-h-11 rounded-2xl border border-border/70 px-3 text-xs font-semibold text-foreground transition-colors hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 motion-reduce:transition-none"
              >
                Làm mới số liệu
              </button>
            }
          >
            <div className="grid gap-3 md:grid-cols-3">
              {attentionItems.map((item) => (
                <AttentionAction key={item.title} {...item} />
              ))}
            </div>
          </DashboardLedgerPanel>

          <section className="admin-dashboard-kpis grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
            {data.dashboardSummary && (
              <>
                <KpiHeroCard
                  label="NV trả lương tuần này"
                  value={data.dashboardSummary.total_working_employees}
                  icon={Users}
                  color="blue"
                  sublabel={`Tổng: ${data.dashboardSummary.total_employees.toLocaleString('vi-VN')} NV`}
                  onClick={dashboardNav.navigateToActiveEmployees}
                  className="motion-reduce:transition-none"
                />
                <KpiHeroCard
                  label="Lợi nhuận tháng này"
                  value={data.dashboardSummary.total_profit_this_month}
                  formattedValue={formatVND(data.dashboardSummary.total_profit_this_month)}
                  icon={TrendingUp}
                  color="emerald"
                  sublabel={`Tổng: ${formatVND(data.dashboardSummary.total_profit)}`}
                  className="motion-reduce:transition-none"
                />
                <KpiHeroCard
                  label={`Nhân viên mới (${monthLabel})`}
                  value={data.dashboardSummary.employees_hired_this_month}
                  icon={UserPlus}
                  color="amber"
                  sublabel="Theo kỳ đang xem"
                  onClick={dashboardNav.navigateToNewEmployees}
                  className="motion-reduce:transition-none"
                />
              </>
            )}
          </section>
        </div>

        <div className="grid gap-4 md:grid-cols-2 2xl:grid-cols-4">
          {statGroups.map((group) => (
            <DashboardLedgerPanel
              key={group.key}
              eyebrow={group.eyebrow}
              title={group.title}
              subtitle={group.subtitle}
              icon={group.icon}
              className="admin-dashboard-ledger-surface"
            >
              <InlineStatStrip
                items={group.items}
                variant="subtle"
                direction="vertical"
                className="admin-dashboard-ledger-strip"
              />
            </DashboardLedgerPanel>
          ))}
        </div>

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,1fr)]">
          <div className="space-y-6">
            <section className="admin-dashboard-finance-group space-y-3">
              <DashboardSectionHeader
                eyebrow="Sổ dòng tiền"
                title="Lịch sử tài chính"
                subtitle="Sổ chi phí, doanh thu và lợi nhuận tích lũy 12 tháng gần nhất."
                icon={TrendingUp}
              />
              <MonthlyFinancialTable />
            </section>

            <section className="admin-dashboard-finance-group space-y-3">
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

          <section className="admin-dashboard-finance-group space-y-3">
            <DashboardSectionHeader
              eyebrow="Công trường"
              title="Lợi nhuận dự án"
              subtitle="Theo dõi đà lợi nhuận và dự án đang kéo kết quả kỳ lương."
              icon={FolderKanban}
            />
            <ProjectProfitabilityCard />
          </section>
        </div>

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(360px,1fr)]">
          <section className="admin-dashboard-workforce-group space-y-3">
            <DashboardSectionHeader
              eyebrow="Phân tích chi trả"
              title="Phân bổ lương"
              subtitle="Các khoảng lương điển hình để kiểm tra độ lệch trước khi giải ngân."
              icon={BarChart3}
            />
            <SalaryDistributionChart />
          </section>

          <div className="space-y-6">
            <section className="admin-dashboard-workforce-group space-y-3">
              <DashboardSectionHeader
                eyebrow="Nhân sự"
                title={`Chi trả theo nhân viên — ${monthLabel}`}
                subtitle="Nhân viên nhận lương cao nhất trong kỳ đang xem."
                icon={Trophy}
              />
              <TopPaidEmployeesCard month={monthParam} />
            </section>

            <section className="admin-dashboard-workforce-group space-y-3">
              <DashboardSectionHeader
                eyebrow="Nhân sự"
                title="Nhân viên mới nhất"
                subtitle="Các hồ sơ vừa được thêm để đối chiếu biên chế công trường."
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
        </div>

        <section className="admin-dashboard-health-group space-y-3">
          <DashboardSectionHeader
            eyebrow="Sức khỏe vận hành"
            title="Tự chấm công và hạn mức ứng lương"
            subtitle="Đặt kiểm soát vận hành ở cuối sổ để rà soát trước khi chốt kỳ."
            icon={ShieldCheck}
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
