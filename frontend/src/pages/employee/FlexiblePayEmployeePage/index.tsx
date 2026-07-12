import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { AlertCircle, RefreshCw } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { authManager } from "@/lib/auth";
import {
  useEmployeeProfile,
  useUpdateEmployeePassword,
} from "@/hooks/api/useEmployeePortal";
import {
  useAdvancePaymentInfo,
  useRequestAdvancePayment,
  useCheckInAdvanceInfo,
  useRequestCheckInAdvance,
  useAdvancePaymentHistory,
  useCalculateFee,
  useCancelAdvancePaymentRequest,
} from "@/hooks/api/useAdvancePayments";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import { EmployeeBankInfoCard } from "@/components/employees/EmployeeBankInfoCard";
import { EmployeeMobileShell, employeeCardShadow } from "@/components/employees/EmployeeMobileShell";
import { EmployeeCheckInCard } from "@/components/employees/EmployeeCheckInCard";
import { EmployeeAttendanceHistoryCard } from "@/components/employees/EmployeeAttendanceHistoryCard";
import { EmployeeMonthNavigator } from "@/components/employees/EmployeeMonthNavigator";
import { useEmployeeMonth } from "@/hooks/useEmployeeMonth";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { AdvancePaymentRequestForm } from "@/components/advance-payment/AdvancePaymentRequestForm";
import { AdvancePaymentHistoryCard } from "@/components/advance-payment/AdvancePaymentHistoryCard";
import { AdvancePaymentConfirmSheet } from "@/components/advance-payment/AdvancePaymentConfirmSheet";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { formatPayrollMonthRange } from "@/utils/advancePaymentHelpers";
import { getEmployeeAccountHolder, hasEmployeeBankInfo } from "@/utils/employeePortal/mobileHome";

const FlexiblePayEmployeePage = () => {
  const navigate = useNavigate();
  const [passwordSheetOpen, setPasswordSheetOpen] = useState(false);
  const [notificationSheetOpen, setNotificationSheetOpen] = useState(false);
  const [confirmSheetOpen, setConfirmSheetOpen] = useState(false);
  const [latestAmount, setLatestAmount] = useState<number>(0);
  const [latestMonth, setLatestMonth] = useState<string>("");
  const [formKey, setFormKey] = useState(0);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data: profile, isLoading: profileLoading } = useEmployeeProfile();
  const { data: unreadNotifications } = useUnreadNotifications();
  // URL-backed month drives the viewed quota and attendance period. Request
  // history remains all-time so employees always see their latest activity.
  const month = useEmployeeMonth();
  // Check-in-enabled employees use the dedicated /me/check-in-advance flow
  // (70% advanceable cap, calendar-month window); others use the admin-upload flow.
  const isCheckIn = !!profile?.check_in_enabled;
  const regularInfoQuery = useAdvancePaymentInfo({ enabled: !isCheckIn });
  const checkInInfoQuery = useCheckInAdvanceInfo({ enabled: isCheckIn });
  const {
    data: infoResponse,
    isLoading: infoLoading,
    isError: infoError,
    refetch: refetchInfo,
  } = isCheckIn
    ? checkInInfoQuery
    : regularInfoQuery;
  const {
    data: historyResponse,
    isLoading: historyLoading,
    isError: historyError,
    refetch: refetchHistory,
  } = useAdvancePaymentHistory({ page: 1, pageSize: 100 });

  const calculateFeeMutation = useCalculateFee();
  const regularRequestMutation = useRequestAdvancePayment();
  const checkInRequestMutation = useRequestCheckInAdvance();
  const requestMutation = isCheckIn ? checkInRequestMutation : regularRequestMutation;
  const cancelMutation = useCancelAdvancePaymentRequest();
  const updatePasswordMutation = useUpdateEmployeePassword();

  const info = infoResponse?.data;
  const history = useMemo(() => historyResponse?.data ?? [], [historyResponse?.data]);
  const historyTotal = historyResponse?.pagination?.totalRecords ?? history.length;

  // Debounced server-side fee calculation
  const [serverFeeDetails, setServerFeeDetails] = useState<{
    fee: number;
    netAmount: number;
  } | null>(null);

  const handleAmountChange = useCallback(
    (amount: number) => {
      setLatestAmount(amount);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      if (amount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) {
        setServerFeeDetails(null);
        return;
      }
      debounceRef.current = setTimeout(() => {
        calculateFeeMutation.mutate(
          { amount },
          {
            onSuccess: (r) => setServerFeeDetails(r),
            onError: () => setServerFeeDetails(null),
          }
        );
      }, 300);
    },
    [calculateFeeMutation]
  );

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  const feeDetails = serverFeeDetails;

  const handleRequestSubmit = useCallback((data: { amount: number; forMonth: string }) => {
    setLatestAmount(data.amount);
    setLatestMonth(data.forMonth);
    setConfirmSheetOpen(true);
  }, []);

  const handleBankAction = useCallback(() => {
    const section = document.getElementById("employee-bank");
    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    section?.scrollIntoView({
      behavior: prefersReducedMotion ? "auto" : "smooth",
      block: "start",
    });
  }, []);

  const handleAdvanceRequestAction = useCallback(() => {
    const section = document.getElementById("employee-advance-request");
    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    section?.scrollIntoView({
      behavior: prefersReducedMotion ? "auto" : "smooth",
      block: "start",
    });
  }, []);

  const handleConfirmSubmit = useCallback(async () => {
    try {
      await requestMutation.mutateAsync({ amount: latestAmount, forMonth: latestMonth });
      setConfirmSheetOpen(false);
      setFormKey((k) => k + 1);
      setLatestAmount(0);
      setLatestMonth("");
      setServerFeeDetails(null);
      refetchInfo();
    } catch {
      /* handled */
    }
  }, [latestAmount, latestMonth, requestMutation, refetchInfo]);

  const handleCancelRequest = useCallback(
    (id: number) => {
      cancelMutation.mutate(id);
    },
    [cancelMutation]
  );

  const handleLogout = () => {
    authManager.removeToken();
    localStorage.removeItem("userRole");
    toast({ title: "Đăng xuất thành công", description: "Hẹn gặp lại bạn!" });
    navigate("/login");
  };

  const handleChangePassword = async (data: {
    currentPassword: string;
    newPassword: string;
  }) => {
    await updatePasswordMutation.mutateAsync({
      current_password: data.currentPassword,
      new_password: data.newPassword,
    });
    setPasswordSheetOpen(false);
  };

  if (profileLoading || infoLoading) {
    return (
      <div className="employee-mobile-page min-h-[100dvh] bg-[var(--employee-page)]" style={{ paddingBottom: "env(safe-area-inset-bottom)" }}>
        <div
          className="flex items-center justify-between border-b border-[#E4E7EC] bg-white px-4 pb-3"
          style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}
        >
          <div className="space-y-2">
            <Skeleton className="h-7 w-44" />
            <Skeleton className="h-4 w-28" />
          </div>
          <div className="flex gap-2">
            {[1, 2].map((i) => (
              <Skeleton key={i} className="h-11 w-11 rounded-xl" />
            ))}
          </div>
        </div>
        <div className="mx-auto max-w-lg space-y-5 p-4">
          <Skeleton className="h-14 w-full rounded-xl" />
          <Skeleton className="h-48 w-full rounded-2xl" />
          <div className="space-y-2.5">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-32 w-full rounded-xl" />
          </div>
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
        contentClassName={
          profile?.check_in_enabled
            ? "max-w-lg space-y-6 pb-[calc(5.5rem+env(safe-area-inset-bottom))]"
            : "max-w-lg space-y-6"
        }
      >
        <EmployeeMonthNavigator month={month} />

        {infoError ? (
          <section id="employee-advance-request" className="scroll-mt-24 rounded-2xl border border-[var(--employee-border)] bg-white px-4 py-5 text-center shadow-[var(--employee-shadow)]" role="alert">
            <AlertCircle className="mx-auto h-5 w-5 text-[var(--employee-error)]" aria-hidden="true" />
            <h2 className="employee-type-card-title mt-2 text-[var(--employee-text)]">Chưa tải được hạn mức ứng lương</h2>
            <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Kiểm tra kết nối rồi thử lại.</p>
            <button
              type="button"
              onClick={() => refetchInfo()}
              className="employee-type-action mt-3 inline-flex min-h-11 items-center gap-2 rounded-[10px] border border-[var(--employee-border-strong)] px-4 text-[#344054] transition-transform duration-200 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Tải lại
            </button>
          </section>
        ) : info ? (
          <section id="employee-advance-request" className="scroll-mt-24">
            <AdvancePaymentRequestForm
              key={`${formKey}-${month.value}`}
              info={info}
              viewMonth={month.value}
              isPastMonth={!month.isCurrentMonth}
              history={history}
              feeDetails={feeDetails}
              hasBankDestination={hasEmployeeBankInfo(profile)}
              onSubmit={handleRequestSubmit}
              onAmountChange={handleAmountChange}
              onBankAction={handleBankAction}
              isPending={requestMutation.isPending}
              className="overflow-hidden rounded-2xl border border-[var(--employee-border)] bg-white p-4 shadow-[var(--employee-shadow)]"
            />
          </section>
        ) : null}

        <section id="employee-history" className="scroll-mt-24">
          <AdvancePaymentHistoryCard
            history={history}
            isLoading={historyLoading}
            isError={historyError}
            onRetry={() => refetchHistory()}
            onCancel={handleCancelRequest}
            totalCount={historyTotal}
          />
        </section>

        <section id="employee-bank" className="scroll-mt-24">
          <EmployeeBankInfoCard profile={profile!} />
        </section>

        {profile?.check_in_enabled && (
          <section id="employee-check-in" className="scroll-mt-24">
            <EmployeeCheckInCard
              checkInTarget={profile.check_in_target}
              shiftStart={profile.shift_start}
              shiftEnd={profile.shift_end}
              checkInWindowStart={profile.check_in_window_start}
              checkInWindowEnd={profile.check_in_window_end}
              onAdvanceRequest={handleAdvanceRequestAction}
            />
          </section>
        )}

        {profile?.check_in_enabled && (
          <EmployeeAttendanceHistoryCard
            fromDate={month.fromDate}
            toDate={month.toDate}
            monthLabel={month.shortLabel}
            className="overflow-hidden rounded-2xl border border-[var(--employee-border)] bg-white"
            style={employeeCardShadow}
          />
        )}

      <ChangePasswordSheet
        open={passwordSheetOpen}
        onOpenChange={setPasswordSheetOpen}
        onSubmit={handleChangePassword}
        isPending={updatePasswordMutation.isPending}
      />

      <AdvancePaymentConfirmSheet
        open={confirmSheetOpen}
        onOpenChange={setConfirmSheetOpen}
        amount={latestAmount}
        feeDetails={feeDetails}
        bankAccountNumber={profile?.bank_account_number}
        bankName={profile?.bank?.branch_name}
        bankAccountName={getEmployeeAccountHolder(profile)}
        payrollPeriodLabel={formatPayrollMonthRange(latestMonth)}
        onConfirm={handleConfirmSubmit}
        isPending={requestMutation.isPending}
      />

      <NotificationSheet
        isOpen={notificationSheetOpen}
        onClose={() => setNotificationSheetOpen(false)}
      />
    </EmployeeMobileShell>
  );
};

export default FlexiblePayEmployeePage;
