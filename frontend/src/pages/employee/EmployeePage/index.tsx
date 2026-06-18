import { useState, useMemo } from "react";
import {
  Calendar,
  DollarSign,
  Clock,
  CreditCard,
  Wallet,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/sonner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import type { EmployeeTimesheetFilters } from "@/types/api/auth.types";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { EmployeeBankInfoCard } from "@/components/employees/EmployeeBankInfoCard";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { groupTimesheetsByDay } from "@/utils/employeePortal/timesheetGrouping";
import { useInfiniteScroll } from "@/hooks/use-infinite-scroll";
import { getDayPaymentStatus } from "@/utils/employeePortal/paymentStatus";

const EmployeePage = () => {
  const navigate = useNavigate();

  const monthOptions = useMemo(() => {
    const options = [];
    const now = new Date();
    for (let i = 0; i < 6; i++) {
      const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
      const value = format(date, "yyyy-MM");
      const label = format(date, "MMMM yyyy", { locale: vi });
      options.push({ value, label });
    }
    return options;
  }, []);

  const [monthFilter, setMonthFilter] = useState<string>(monthOptions[0]?.value || "");
  const statusFilter = "all";
  const [passwordSheetOpen, setPasswordSheetOpen] = useState(false);
  const [notificationSheetOpen, setNotificationSheetOpen] = useState(false);

  const { data: profile, isLoading: profileLoading } = useEmployeeProfile();
  const { data: summaryData, isLoading: summaryLoading } = useEmployeeSummary(4);
  const summary = summaryData?.data;
  const { data: bulkTransferSetting } = useSettingByKey("bulk_transfer_payment_percentage", !!summary);
  const { data: unreadNotifications } = useUnreadNotifications();

  const baseFilters = useMemo<Omit<EmployeeTimesheetFilters, "page" | "pageSize">>(() => {
    const f: Omit<EmployeeTimesheetFilters, "page" | "pageSize"> = { sortBy: "date", sortOrder: "desc" };
    if (monthFilter) {
      const [year, month] = monthFilter.split("-");
      const lastDay = new Date(parseInt(year), parseInt(month), 0).getDate();
      f.fromDate = `${monthFilter}-01`;
      f.toDate = `${monthFilter}-${lastDay.toString().padStart(2, "0")}`;
    }
    if (statusFilter && statusFilter !== "all") {
      f.payment_status = statusFilter as "paid" | "unpaid";
    }
    return f;
  }, [monthFilter, statusFilter]);

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

  if (profileLoading || summaryLoading) {
    return (
      <div className="min-h-[100dvh]" style={{ backgroundImage: "url('/employee-bg.avif')", backgroundSize: "cover", backgroundPosition: "center top", paddingTop: 'env(safe-area-inset-top)', paddingBottom: 'env(safe-area-inset-bottom)' }}>
        <div className="sticky top-0 z-10 border-b border-white/30" style={{ background: "rgba(255,255,255,0.72)", backdropFilter: "blur(20px)" }}>
          <div className="max-w-2xl mx-auto px-4 py-3.5 flex items-center justify-between">
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
    <div
      className="min-h-[100dvh]"
      style={{ paddingTop: 'env(safe-area-inset-top)', paddingBottom: 'env(safe-area-inset-bottom)', backgroundImage: "url('/employee-bg.avif')", backgroundSize: "cover", backgroundPosition: "center top", backgroundAttachment: "fixed" }}
    >
      {/* Header */}
      <EmployeePortalHeader
        employeeName={profile?.fullname}
        unreadCount={unreadNotifications?.count}
        onNotificationClick={() => setNotificationSheetOpen(true)}
        onChangePassword={() => setPasswordSheetOpen(true)}
        onLogout={handleLogout}
      />

      {/* Main Content */}
      <div className="max-w-2xl mx-auto p-4 space-y-4 pb-20">

        {/* Summary Stats */}
        <div className="grid grid-cols-2 gap-3">
          {(summaryLoading || timesheetsLoading) ? (
            [1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-20 rounded-xl bg-card/50" />)
          ) : (
            <>
              {[
                { icon: DollarSign, iconText: "text-sky-600",     watermark: "text-sky-500/15",     label: "Tổng lương", value: formatCurrency(monthlyTotalSalary), sub: format(new Date(monthFilter + "-01"), "MMMM yyyy", { locale: vi }) },
                { icon: Clock,      iconText: "text-emerald-600", watermark: "text-emerald-500/15", label: "Tổng công",  value: `${monthlyTotalHours % 1 === 0 ? monthlyTotalHours : formatNumber(monthlyTotalHours, 1)} giờ`, sub: `${groupedDays.length} ngày làm việc` },
                { icon: CreditCard, iconText: "text-violet-600",  watermark: "text-violet-500/15",  label: "Hạn mức trả", value: formatCurrency(totalPayable), sub: null },
                { icon: Wallet,     iconText: "text-amber-600",   watermark: "text-amber-500/15",   label: "Đã nhận",     value: formatCurrency(totalPaid), sub: null },
              ].map(({ icon: Icon, iconText, watermark, label, value, sub }) => (
                <div key={label} className="group relative rounded-xl p-4 overflow-hidden transition-colors glass-card">
                  {/* Watermark — large faint icon decoration */}
                  <Icon
                    className={`absolute right-3 top-1/2 -translate-y-1/2 h-14 w-14 pointer-events-none transition-transform duration-300 group-hover:scale-105 ${watermark}`}
                    strokeWidth={1.5}
                  />
                  <div className="relative pr-12">
                    <div className="flex items-center gap-1.5 mb-2">
                      <Icon className={`h-3.5 w-3.5 shrink-0 ${iconText}`} strokeWidth={2.2} />
                      <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight truncate">{label}</span>
                    </div>
                    <p className="text-base font-bold text-slate-800 tabular-nums truncate leading-tight">{value}</p>
                    {sub && <p className="text-xs text-slate-400 mt-1 truncate">{sub}</p>}
                  </div>
                </div>
              ))}
            </>
          )}
        </div>

        {/* Bank Account Information */}
        <EmployeeBankInfoCard
          profile={profile!}
          className="rounded-xl overflow-hidden glass-card"
        />

        {/* Timesheets Section */}
        <div>
          <div
            className="flex items-center justify-between gap-3 mb-3 px-4 py-3 rounded-xl"
            style={{
              background: "rgba(255,255,255,0.82)",
              backdropFilter: "blur(16px) saturate(1.4)",
              WebkitBackdropFilter: "blur(16px) saturate(1.4)",
              boxShadow: "0 1px 4px rgba(0,0,0,0.08)",
            }}
          >
            <div className="flex items-center gap-2">
              <Calendar className="w-5 h-5 text-sky-600" />
              <h2 className="text-lg font-bold text-slate-800 uppercase tracking-wide">BẢNG CÔNG</h2>
              {totalRecords > 0 && (
                <Badge className="text-xs font-semibold px-2 py-0.5 bg-sky-100 text-sky-700 border-0 hover:bg-sky-100">
                  {totalRecords}
                </Badge>
              )}
            </div>
            <Select value={monthFilter} onValueChange={setMonthFilter}>
              <SelectTrigger className="h-9 text-sm font-medium w-auto inline-flex rounded-lg border-0 bg-white/60 hover:bg-white/80 transition-colors">
                <div className="flex items-center gap-1.5">
                  <Calendar className="h-3.5 w-3.5 text-slate-400 shrink-0" />
                  <SelectValue />
                </div>
              </SelectTrigger>
              <SelectContent className="rounded-xl border-white/60 bg-white/90 backdrop-blur-xl shadow-lg overflow-hidden">
                {monthOptions.map((o) => (
                  <SelectItem key={o.value} value={o.value} className="text-sm py-2.5 rounded-lg focus:bg-sky-50/80 focus:text-sky-700">{o.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Timesheet cards */}
          {timesheetsLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32 w-full rounded-xl bg-card/50" />)}
            </div>
          ) : groupedDays.length === 0 ? (
            <div className="rounded-xl border border-dashed border-white/60 py-12 text-center" style={{ background: "rgba(255,255,255,0.50)" }}>
              <div className="w-12 h-12 bg-sky-100/80 rounded-full flex items-center justify-center mx-auto mb-3">
                <Calendar className="w-6 h-6 text-sky-400" />
              </div>
              <p className="text-sm font-semibold text-foreground">Chưa có bảng công</p>
              <p className="text-xs text-slate-400 mt-1">Dữ liệu sẽ hiển thị tại đây</p>
            </div>
          ) : (
            <div className="space-y-2.5">
              {groupedDays.map((day) => {
                const status = getDayPaymentStatus(day.totalAmount, day.totalPaidAmount, bulkTransferPercentage);
                const isPaid = status === "full";

                return (
                  <div key={day.date} className="rounded-xl overflow-hidden glass-card">
                    {/* Card header */}
                    <div className="flex items-center justify-between px-4 pt-4 pb-3">
                      <div className="flex items-center gap-2">
                        <div className={`w-2.5 h-2.5 rounded-full shrink-0 ${isPaid ? "bg-emerald-500" : "bg-amber-400"}`} />
                        <span className="text-sm font-bold text-slate-800 capitalize">
                          {format(new Date(day.date), "EEEE, dd/MM", { locale: vi })}
                        </span>
                      </div>
                      <span className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                        isPaid ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"
                      }`}>
                        {isPaid ? "Đã trả" : "Chưa trả"}
                      </span>
                    </div>

                    {/* Stats row */}
                    <div className="grid grid-cols-3 divide-x divide-sky-100/60 border-t border-border">                      {[
                        { label: "Giờ công", value: `${day.totalHours % 1 === 0 ? day.totalHours : formatNumber(day.totalHours, 1)}`, unit: "h", color: "text-foreground" },
                        { label: "Tổng lương", value: formatCurrency(day.totalAmount), unit: null, color: "text-foreground" },
                        { label: "Đã nhận", value: formatCurrency(day.totalPaidAmount), unit: null, color: isPaid ? "text-emerald-600" : "text-slate-400" },
                      ].map(({ label, value, unit, color }) => (
                        <div key={label} className="px-3 py-3 text-center">
                          <p className="text-xs text-slate-400 font-medium mb-1">{label}</p>
                          <p className={`text-sm font-bold tabular-nums truncate ${color}`}>
                            {value}{unit && <span className="text-xs font-normal text-slate-400 ml-0.5">{unit}</span>}
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
          <div ref={observerRef} className="h-4" />
        </div>
      </div>

      {/* Password Sheet */}
      <ChangePasswordSheet
        open={passwordSheetOpen}
        onOpenChange={setPasswordSheetOpen}
        onSubmit={handleChangePassword}
        isPending={updatePasswordMutation.isPending}
      />

      <NotificationSheet isOpen={notificationSheetOpen} onClose={() => setNotificationSheetOpen(false)} />
    </div>
  );
};

export default EmployeePage;
