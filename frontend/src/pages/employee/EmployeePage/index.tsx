import { useState, useMemo, useCallback } from "react";
import { Calendar } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/sonner";
import { authManager } from "@/lib/auth";
import { formatCurrency, formatNumber } from "@/utils/formatters";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
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
import { EmployeeMonthNavigator } from "@/components/employees/EmployeeMonthNavigator";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { groupTimesheetsByDay } from "@/utils/employeePortal/timesheetGrouping";
import { useInfiniteScroll } from "@/hooks/use-infinite-scroll";
import { getDayPaymentStatus } from "@/utils/employeePortal/paymentStatus";
import {
  createRegularEmployeeHomeModel,
  hasEmployeeBankInfo,
  type EmployeeNudge,
  type EmployeeQuickAction,
} from "@/utils/employeePortal/mobileHome";

const EmployeePage = () => {
  const navigate = useNavigate();

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
    authManager.removeToken();
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
      behavior: "smooth",
      block: "start",
    });
  }, []);

  if (profileLoading || summaryLoading || timesheetsLoading) {
    return (
      <div className="employee-mobile-page min-h-[100dvh]" style={{ backgroundImage: "url('/employee-bg.avif')", backgroundSize: "cover", backgroundPosition: "center top", paddingBottom: 'env(safe-area-inset-bottom)' }}>
        <div className="sticky top-0 z-10 border-b border-white/30 bg-employee">
          <div
            className="max-w-2xl mx-auto px-4 pb-4 flex items-center justify-between"
            style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.875rem)" }}
          >
            <div className="space-y-1.5">
              <Skeleton className="h-3 w-24 bg-sky-200/60" />
              <Skeleton className="h-5 w-36 bg-sky-200/60" />
            </div>
            <div className="flex gap-2">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10 w-10 rounded-full bg-sky-200/60" />)}
            </div>
          </div>
        </div>
        <div className="max-w-2xl mx-auto p-4 space-y-4">
          <div className="rounded-2xl border border-white/60 bg-white/75 px-4 py-3 shadow-sm backdrop-blur">
            <p className="text-xs font-semibold uppercase tracking-[0.08em] text-sky-700">Đang tải hồ sơ</p>
            <p className="mt-1 text-sm text-slate-500">Chuẩn bị bảng công và thông tin thanh toán của bạn.</p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-20 w-full rounded-xl bg-card/50" />)}
          </div>
          <Skeleton className="h-12 w-full rounded-xl bg-card/50" />
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32 w-full rounded-xl bg-card/50" />)}
        </div>
      </div>
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

        {/* Timesheets Section */}
        <section id="employee-timesheets" className="scroll-mt-4">
          <EmployeeMonthNavigator month={month} className="mb-3" />
          <div className="mb-3 flex items-center gap-2 rounded-2xl border border-slate-200 bg-white px-3 py-2.5 shadow-sm">
            <Calendar className="h-4 w-4 text-sky-600" />
            <h2 className="employee-type-hero-title text-slate-900">Bảng công</h2>
            {totalRecords > 0 && (
              <Badge className="employee-type-pill border-0 bg-sky-100 px-2 py-1 text-sky-700 hover:bg-sky-100">
                {totalRecords}
              </Badge>
            )}
          </div>

          {/* Timesheet cards */}
          {timesheetsLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32 w-full rounded-xl bg-card/50" />)}
            </div>
          ) : groupedDays.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-slate-200 bg-white/80 py-8 text-center">
              <div className="w-12 h-12 bg-sky-100/80 rounded-full flex items-center justify-center mx-auto mb-3">
                <Calendar className="w-6 h-6 text-sky-400" />
              </div>
              <p className="employee-type-card-title text-foreground">Chưa có bảng công</p>
              <p className="employee-type-body-sm mt-1 text-slate-400">Dữ liệu sẽ hiển thị tại đây</p>
            </div>
          ) : (
            <div className="space-y-2.5">
              {groupedDays.map((day) => {
                const status = getDayPaymentStatus(day.totalAmount, day.totalPaidAmount, bulkTransferPercentage);
                const isPaid = status === "full";

                return (
                  <div key={day.date} className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
                    {/* Card header */}
                    <div className="flex items-center justify-between px-3 py-3">
                      <div className="flex items-center gap-2">
                        <div className={`w-2.5 h-2.5 rounded-full shrink-0 ${isPaid ? "bg-emerald-500" : "bg-amber-400"}`} />
                        <span className="employee-type-row-amount capitalize text-slate-800">
                          {format(new Date(day.date), "EEEE, dd/MM", { locale: vi })}
                        </span>
                      </div>
                      <span className={`employee-type-pill rounded-full px-2.5 py-1 ${
                        isPaid ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"
                      }`}>
                        {isPaid ? "Đã trả" : "Chưa trả"}
                      </span>
                    </div>

                    {/* Stats row */}
                    <div className="grid grid-cols-3 divide-x divide-slate-100 border-t border-slate-100">                      {[
                        { label: "Giờ công", value: `${day.totalHours % 1 === 0 ? day.totalHours : formatNumber(day.totalHours, 1)}`, unit: "h", color: "text-foreground" },
                        { label: "Tổng lương", value: formatCurrency(day.totalAmount), unit: null, color: "text-foreground" },
                        { label: "Đã nhận", value: formatCurrency(day.totalPaidAmount), unit: null, color: isPaid ? "text-emerald-600" : "text-slate-400" },
                      ].map(({ label, value, unit, color }) => (
                        <div key={label} className="min-w-0 px-2 py-2.5 text-center">
                          <p className="employee-type-label mb-1 text-slate-400">{label}</p>
                          <p className={`employee-type-row-amount whitespace-nowrap tabular-nums ${color}`}>
                            {value}{unit && <span className="type-caption ml-0.5 font-normal text-slate-400">{unit}</span>}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {isFetchingNextPage && (
            <div className="flex items-center justify-center py-4 gap-2 text-muted-foreground text-sm">
              <div className="animate-spin rounded-full h-4 w-4 border-2 border-border border-t-sky-600" />
              Đang tải thêm...
            </div>
          )}
          <div ref={observerRef} className="h-3" />
        </section>

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
