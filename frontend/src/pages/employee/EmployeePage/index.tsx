import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { useAuth } from "@/contexts/AuthContext";
import {
  useEmployeeProfile,
  useEmployeeSummary,
  useUpdateEmployeePassword,
  useEmployeeTimesheetsInfinite,
} from "@/hooks/api/useEmployeePortal";
import { useSettingByKey } from "@/hooks/api/useSettings";
import { useEmployeeMonth } from "@/hooks/useEmployeeMonth";
import type { EmployeeTimesheetFilters } from "@/types/api/auth.types";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { EmployeeBankInfoCard } from "@/components/employees/EmployeeBankInfoCard";
import { EmployeeMobileShell } from "@/components/employees/EmployeeMobileShell";
import { EmployeeWalletHero } from "@/components/employees/EmployeeWalletHero";
import { EmployeeTimesheetPanel } from "@/components/employees/EmployeeTimesheetPanel";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { groupTimesheetsByDay } from "@/utils/employeePortal/timesheetGrouping";
import { useInfiniteScroll } from "@/hooks/use-infinite-scroll";
import {
  createRegularEmployeeHomeModel,
  hasEmployeeBankInfo,
} from "@/utils/employeePortal/mobileHome";

const EmployeePage = () => {
  const navigate = useNavigate();
  const { logout } = useAuth();

  const month = useEmployeeMonth();
  const [passwordSheetOpen, setPasswordSheetOpen] = useState(false);
  const [notificationSheetOpen, setNotificationSheetOpen] = useState(false);

  const { data: profile, isLoading: profileLoading } = useEmployeeProfile();
  const { data: summaryData, isLoading: summaryLoading } = useEmployeeSummary(4);
  const summary = summaryData?.data;
  const { data: bulkTransferSetting } = useSettingByKey("bulk_transfer_payment_percentage", !!summary);
  const { data: unreadNotifications } = useUnreadNotifications();
  const bulkTransferPercentage = bulkTransferSetting?.value ? parseFloat(bulkTransferSetting.value) : 0;

  const baseFilters = useMemo<Omit<EmployeeTimesheetFilters, "page" | "pageSize">>(
    () => ({
      sortBy: "date",
      sortOrder: "desc",
      fromDate: month.fromDate,
      toDate: month.toDate,
    }),
    [month.fromDate, month.toDate]
  );

  const { data: infiniteData, isLoading: timesheetsLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useEmployeeTimesheetsInfinite(baseFilters, 50);

  const {
    groupedDays,
    monthlyTotalHours,
    monthlyTotalSalary,
    totalPaid,
    totalPayable,
    totalRecords,
  } = useMemo(() => {
    const timesheets = infiniteData?.pages.flatMap((page) => page.data) ?? [];
    const totals = timesheets.reduce(
      (summary, timesheet) => ({
        monthlyTotalSalary: summary.monthlyTotalSalary + timesheet.amount,
        monthlyTotalHours: summary.monthlyTotalHours + timesheet.hours_worked,
        totalPaid: summary.totalPaid + timesheet.paid_amount,
        totalPayable: summary.totalPayable + timesheet.amount * bulkTransferPercentage,
      }),
      {
        monthlyTotalSalary: 0,
        monthlyTotalHours: 0,
        totalPaid: 0,
        totalPayable: 0,
      }
    );

    return {
      groupedDays: groupTimesheetsByDay(timesheets),
      totalRecords: infiniteData?.pages?.[0]?.pagination?.totalRecords ?? 0,
      ...totals,
    };
  }, [bulkTransferPercentage, infiniteData]);
  const { observerRef } = useInfiniteScroll({
    hasMore: !!hasNextPage,
    isLoading: isFetchingNextPage || timesheetsLoading,
    onLoadMore: () => { fetchNextPage(); },
  });

  const selectedMonthLabel = useMemo(
    () => format(month.date, "MMMM yyyy", { locale: vi }),
    [month.date]
  );
  const homeModel = useMemo(
    () =>
      createRegularEmployeeHomeModel({
        monthLabel: selectedMonthLabel,
        monthlyTotalSalary,
        monthlyTotalHours,
        totalPayable,
        totalPaid,
        workDayCount: groupedDays.length,
        totalRecords,
        hasBankInfo: hasEmployeeBankInfo(profile),
        unreadCount: unreadNotifications?.count,
      }),
    [
      groupedDays.length,
      monthlyTotalHours,
      monthlyTotalSalary,
      profile,
      selectedMonthLabel,
      totalPaid,
      totalPayable,
      totalRecords,
      unreadNotifications?.count,
    ]
  );

  const updatePasswordMutation = useUpdateEmployeePassword();

  const handleLogout = () => {
    logout();
    localStorage.removeItem("userRole");
    toast({ title: "Đăng xuất thành công", description: "Hẹn gặp lại bạn!" });
    navigate("/login");
  };

  const handleChangePassword = async (data: { currentPassword: string; newPassword: string }) => {
    await updatePasswordMutation.mutateAsync({
      current_password: data.currentPassword,
      new_password: data.newPassword,
    });
    setPasswordSheetOpen(false);
  };

  const isInitialLoading = profileLoading || summaryLoading || timesheetsLoading;

  if (isInitialLoading) {
    return (
      <EmployeeMobileShell chrome="skeleton" contentClassName="max-w-lg space-y-4">
        <div className="employee-surface-card px-4 py-3">
          <p className="employee-type-label-caps text-[var(--employee-accent)]">Đang tải hồ sơ</p>
          <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Chuẩn bị bảng công và thông tin thanh toán của bạn.</p>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {[1, 2, 3, 4].map((index) => <Skeleton key={index} className="h-20 w-full rounded-xl" />)}
        </div>
        <Skeleton className="h-12 w-full rounded-xl" />
        {[1, 2, 3].map((index) => <Skeleton key={index} className="h-32 w-full rounded-xl" />)}
      </EmployeeMobileShell>
    );
  }

  return (
    <EmployeeMobileShell
      employeeName={profile?.fullname}
      unreadCount={unreadNotifications?.count}
      onNotificationClick={() => setNotificationSheetOpen(true)}
      onChangePassword={() => setPasswordSheetOpen(true)}
      onLogout={handleLogout}
    >
      <div className="grid gap-5 lg:grid-cols-[minmax(320px,0.8fr)_minmax(0,1.2fr)] lg:items-start lg:gap-6">
        <EmployeeWalletHero
          model={homeModel}
          className="order-1 lg:sticky lg:top-28 lg:col-start-1 lg:row-start-1"
        />

        <EmployeeTimesheetPanel
          month={month}
          days={groupedDays}
          totalRecords={totalRecords}
          bulkTransferPercentage={bulkTransferPercentage}
          isLoading={timesheetsLoading}
          isFetchingNextPage={isFetchingNextPage}
          observerRef={observerRef}
          className="order-2 lg:col-start-2 lg:row-span-2 lg:row-start-1"
        />

        <section id="employee-bank" className="order-3 scroll-mt-4 lg:col-start-1 lg:row-start-2">
          <EmployeeBankInfoCard profile={profile!} />
        </section>
      </div>

      <ChangePasswordSheet
        open={passwordSheetOpen}
        onOpenChange={setPasswordSheetOpen}
        onSubmit={handleChangePassword}
        isPending={updatePasswordMutation.isPending}
      />

      <NotificationSheet isOpen={notificationSheetOpen} onClose={() => setNotificationSheetOpen(false)} />
    </EmployeeMobileShell>
  );
};

export default EmployeePage;
