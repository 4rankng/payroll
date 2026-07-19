import { useState, useMemo, useCallback } from "react";
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
  type EmployeeNudge,
  type EmployeeQuickAction,
} from "@/utils/employeePortal/mobileHome";

const EmployeePage = () => {
  const navigate = useNavigate();
  const { logout } = useAuth();

  const month = useEmployeeMonth();
  const statusFilter = "all";
  const [passwordSheetOpen, setPasswordSheetOpen] = useState(false);
  const [notificationSheetOpen, setNotificationSheetOpen] = useState(false);

  const { data: profile, isLoading: profileLoading } = useEmployeeProfile();
  const { data: summaryData, isLoading: summaryLoading } = useEmployeeSummary(4);
  const summary = summaryData?.data;
  const { data: bulkTransferSetting } = useSettingByKey("bulk_transfer_payment_percentage", !!summary);
  const { data: unreadNotifications } = useUnreadNotifications();

  const baseFilters = useMemo<Omit<EmployeeTimesheetFilters, "page" | "pageSize">>(() => {
    const f: Omit<EmployeeTimesheetFilters, "page" | "pageSize"> = {
      sortBy: "date",
      sortOrder: "desc",
      fromDate: month.fromDate,
      toDate: month.toDate,
    };
    if (statusFilter && statusFilter !== "all") {
      f.payment_status = statusFilter as "paid" | "unpaid";
    }
    return f;
  }, [month.fromDate, month.toDate, statusFilter]);

  const { data: infiniteData, isLoading: timesheetsLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useEmployeeTimesheetsInfinite(baseFilters, 50);

  const timesheets = useMemo(() => infiniteData?.pages.flatMap((p) => p.data) ?? [], [infiniteData]);
  const totalRecords = useMemo(() => infiniteData?.pages?.[0]?.pagination?.totalRecords ?? 0, [infiniteData]);
  const { observerRef } = useInfiniteScroll({
    hasMore: !!hasNextPage,
    isLoading: isFetchingNextPage || timesheetsLoading,
    onLoadMore: () => { fetchNextPage(); },
  });

  const bulkTransferPercentage = bulkTransferSetting?.value ? parseFloat(bulkTransferSetting.value) : 0;

  const monthlyTotalSalary = useMemo(() => timesheets.reduce((s, e) => s + e.amount, 0), [timesheets]);
  const monthlyTotalHours = useMemo(() => timesheets.reduce((s, e) => s + e.hours_worked, 0), [timesheets]);
  const totalPayable = useMemo(() => timesheets.reduce((s, e) => s + e.amount * bulkTransferPercentage, 0), [timesheets, bulkTransferPercentage]);
  const totalPaid = useMemo(() => timesheets.reduce((s, e) => s + e.paid_amount, 0), [timesheets]);
  const groupedDays = useMemo(() => groupTimesheetsByDay(timesheets), [timesheets]);
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

  const handleHomeAction = useCallback((action: EmployeeQuickAction | EmployeeNudge) => {
    if (action.intent === "notifications") {
      setNotificationSheetOpen(true);
      return;
    }
    if (!action.targetId) return;
    document.getElementById(action.targetId)?.scrollIntoView({
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
      block: "start",
    });
  }, []);

  if (profileLoading || summaryLoading || timesheetsLoading) {
    return (
      <EmployeeMobileShell chrome="skeleton" contentClassName="max-w-lg space-y-4">
          <div className="employee-surface-card px-4 py-3">
            <p className="employee-type-label-caps text-[var(--employee-accent)]">Đang tải hồ sơ</p>
            <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Chuẩn bị bảng công và thông tin thanh toán của bạn.</p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-20 w-full rounded-xl" />)}
          </div>
          <Skeleton className="h-12 w-full rounded-xl" />
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32 w-full rounded-xl" />)}
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
        <EmployeeWalletHero model={homeModel} onAction={handleHomeAction} />

        <EmployeeTimesheetPanel
          month={month}
          days={groupedDays}
          totalRecords={totalRecords}
          bulkTransferPercentage={bulkTransferPercentage}
          isLoading={timesheetsLoading}
          isFetchingNextPage={isFetchingNextPage}
          observerRef={observerRef}
        />

        <section id="employee-bank" className="scroll-mt-4">
          <EmployeeBankInfoCard
            profile={profile!}
            className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
          />
        </section>

      {/* Password Sheet */}
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
