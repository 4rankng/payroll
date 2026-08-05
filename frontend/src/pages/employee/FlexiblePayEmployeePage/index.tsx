import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { AlertCircle, RefreshCw } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { useAuth } from "@/contexts/AuthContext";
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
import { EmployeeMobileShell } from "@/components/employees/EmployeeMobileShell";
import { EmployeeCheckInCard } from "@/components/employees/EmployeeCheckInCard";
import { EmployeeAttendanceHistoryCard } from "@/components/employees/EmployeeAttendanceHistoryCard";
import { EmployeeMonthNavigator } from "@/components/employees/EmployeeMonthNavigator";
import { useEmployeeMonth } from "@/hooks/useEmployeeMonth";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { AdvancePaymentRequestForm } from "@/components/advance-payment/AdvancePaymentRequestForm";
import { AdvancePaymentHistoryCard } from "@/components/advance-payment/AdvancePaymentHistoryCard";
import { AdvancePaymentConfirmSheet } from "@/components/advance-payment/AdvancePaymentConfirmSheet";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import {
  formatPayrollMonthRange,
  getInitialEmployeeAdvanceMonth,
  isPastAdvancePaymentPeriod,
} from "@/utils/advancePaymentHelpers";
import { getEmployeeAccountHolder, hasEmployeeBankInfo } from "@/utils/employeePortal/mobileHome";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";

interface AdvanceRequestConfirmation {
  amount: number;
  forMonth: string;
  submittedAt: string;
}

const EMPTY_HISTORY: AdvancePaymentHistoryItem[] = [];

function scrollToEmployeeSection(sectionId: string) {
  const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  document.getElementById(sectionId)?.scrollIntoView({
    behavior: prefersReducedMotion ? "auto" : "smooth",
    block: "start",
  });
}

const FlexiblePayEmployeePage = () => {
  const navigate = useNavigate();
  const { logout } = useAuth();
  const [passwordSheetOpen, setPasswordSheetOpen] = useState(false);
  const [notificationSheetOpen, setNotificationSheetOpen] = useState(false);
  const [confirmSheetOpen, setConfirmSheetOpen] = useState(false);
  const [latestAmount, setLatestAmount] = useState<number>(0);
  const [latestMonth, setLatestMonth] = useState<string>("");
  const [formKey, setFormKey] = useState(0);
  const [requestConfirmation, setRequestConfirmation] = useState<AdvanceRequestConfirmation | null>(null);
  const [confirmationDataReady, setConfirmationDataReady] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const initialMonthResolvedRef = useRef(false);

  const { data: profile, isLoading: profileLoading } = useEmployeeProfile();
  const { data: unreadNotifications } = useUnreadNotifications();
  // URL-backed month drives the viewed quota and attendance period. Request
  // history remains all-time so employees always see their latest activity.
  const month = useEmployeeMonth();
  const {
    hasExplicitMonth,
    setValue: setMonth,
    value: selectedMonth,
  } = month;
  // Check-in-enabled employees use the dedicated /me/check-in-advance flow
  // (Admin-configured advanceable cap, calendar-month window); others use the admin-upload flow.
  const isCheckInEnabled = Boolean(profile?.check_in_enabled);
  const isCheckIn = isCheckInEnabled;
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
  const history = historyResponse?.data ?? EMPTY_HISTORY;
  const historyTotal = historyResponse?.pagination?.totalRecords ?? history.length;

  // A non-check-in employee can request against the previous payroll month
  // through the cutoff. The API's forMonth is authoritative; only use it for
  // the initial view so explicit URL and navigator selections stay intact.
  useEffect(() => {
    if (profileLoading || infoLoading || !info || initialMonthResolvedRef.current) return;

    const initialMonth = getInitialEmployeeAdvanceMonth(
      info?.forMonth,
      isCheckIn,
      hasExplicitMonth,
    );
    initialMonthResolvedRef.current = true;

    if (initialMonth && selectedMonth !== initialMonth) {
      setMonth(initialMonth);
    }
  }, [
    info,
    infoLoading,
    isCheckIn,
    hasExplicitMonth,
    profileLoading,
    selectedMonth,
    setMonth,
  ]);

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
    scrollToEmployeeSection("employee-bank");
  }, []);

  const handleAdvanceRequestAction = useCallback(() => {
    scrollToEmployeeSection("employee-advance-request");
  }, []);

  const handleConfirmSubmit = useCallback(async () => {
    try {
      await requestMutation.mutateAsync({ amount: latestAmount, forMonth: latestMonth });
      setConfirmSheetOpen(false);
      setRequestConfirmation({
        amount: latestAmount,
        forMonth: latestMonth,
        submittedAt: new Date().toISOString(),
      });
      setConfirmationDataReady(false);
      setFormKey((k) => k + 1);
      setLatestAmount(0);
      setLatestMonth("");
      setServerFeeDetails(null);
      void Promise.all([refetchInfo(), refetchHistory()]).finally(() => setConfirmationDataReady(true));
    } catch {
      /* handled */
    }
  }, [latestAmount, latestMonth, requestMutation, refetchHistory, refetchInfo]);

  useEffect(() => {
    if (!requestConfirmation) return;

    if (month.value !== requestConfirmation.forMonth) {
      setRequestConfirmation(null);
      setConfirmationDataReady(false);
      return;
    }

    const hasMatchingPendingRequest = history.some(
      (item) => item.status === "PENDING" && item.forMonth === requestConfirmation.forMonth
    );

    // Keep the acknowledgement visible until fresh request data confirms that
    // the employee can make another request for this payroll month.
    if (confirmationDataReady && !hasMatchingPendingRequest && info?.canRequest) {
      setRequestConfirmation(null);
      setConfirmationDataReady(false);
    }
  }, [confirmationDataReady, history, info?.canRequest, month.value, requestConfirmation]);

  const handleCancelRequest = useCallback(
    async (id: number) => {
      await cancelMutation.mutateAsync(id);
    },
    [cancelMutation]
  );

  const handleLogout = () => {
    logout();
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

  const isInitialLoading = profileLoading || infoLoading;

  if (isInitialLoading) {
    return (
      <EmployeeMobileShell chrome="skeleton" contentClassName="max-w-lg space-y-5">
        <Skeleton className="h-14 w-full rounded-xl" />
        <Skeleton className="h-48 w-full rounded-2xl" />
        <div className="space-y-2.5">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-32 w-full rounded-xl" />
        </div>
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
      hasActionToolbar={isCheckInEnabled}
      contentClassName="max-w-6xl space-y-4 sm:space-y-5 lg:space-y-6"
    >
      <EmployeeMonthNavigator month={month} className="lg:min-h-[68px]" />

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-[minmax(0,1.12fr)_minmax(320px,0.88fr)] lg:items-start lg:gap-6">
        {infoError ? (
          <section
            id="employee-advance-request"
            className="scroll-mt-24 rounded-2xl border border-[var(--employee-border)] bg-white px-4 py-5 text-center lg:col-start-1 lg:row-start-1"
            role="alert"
          >
            <AlertCircle className="mx-auto h-5 w-5 text-[var(--employee-error)]" aria-hidden="true" />
            <h2 className="employee-type-card-title mt-2 text-[var(--employee-text)]">Chưa tải được hạn mức ứng lương</h2>
            <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Kiểm tra kết nối rồi thử lại.</p>
            <button
              type="button"
              onClick={() => { void refetchInfo(); }}
              className="employee-type-action mt-3 inline-flex min-h-11 items-center gap-2 rounded-[10px] border border-[var(--employee-border-strong)] px-4 text-[#344054] transition-colors duration-200 hover:bg-[var(--employee-surface-muted)] active:bg-[var(--employee-accent-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Tải lại
            </button>
          </section>
        ) : info ? (
          <section id="employee-advance-request" className="scroll-mt-24 lg:col-start-1 lg:row-start-1">
            <AdvancePaymentRequestForm
              key={`${formKey}-${month.value}`}
              info={info}
              viewMonth={month.value}
              isPastMonth={isPastAdvancePaymentPeriod(month.value, info.forMonth)}
              isSelfCheckInFlow={isCheckIn}
              history={history}
              feeDetails={feeDetails}
              hasBankDestination={hasEmployeeBankInfo(profile)}
              onSubmit={handleRequestSubmit}
              onAmountChange={handleAmountChange}
              onBankAction={handleBankAction}
              isPending={requestMutation.isPending}
              requestConfirmation={requestConfirmation}
              className="overflow-hidden p-4 sm:p-5"
            />
          </section>
        ) : null}

        <section
          id="employee-history"
          className={
            isCheckInEnabled
              ? "scroll-mt-24 lg:col-start-2 lg:row-start-1"
              : "scroll-mt-24 lg:col-span-2 lg:row-start-2"
          }
        >
          <AdvancePaymentHistoryCard
            history={history}
            isLoading={historyLoading}
            isError={historyError}
            onRetry={() => { void refetchHistory(); }}
            onCancel={handleCancelRequest}
            cancellingRequestId={cancelMutation.isPending ? cancelMutation.variables : undefined}
            totalCount={historyTotal}
          />
        </section>

        <section
          id="employee-bank"
          className={isCheckInEnabled
            ? "scroll-mt-24 lg:col-start-2 lg:row-start-2"
            : "scroll-mt-24 lg:col-start-2 lg:row-start-1 lg:self-stretch"}
        >
          <EmployeeBankInfoCard profile={profile!} />
        </section>

        {isCheckInEnabled && (
          <section id="employee-check-in" className="scroll-mt-24 lg:col-start-1 lg:row-start-2">
            <EmployeeCheckInCard
              checkInTarget={profile.check_in_target}
              shiftStart={profile.shift_start}
              shiftEnd={profile.shift_end}
              checkInWindowStart={profile.check_in_window_start}
              checkInWindowEnd={profile.check_in_window_end}
              checkOutWindowStart={profile.check_out_window_start}
              checkOutWindowEnd={profile.check_out_window_end}
              scheduleWindows={profile.schedule_windows}
              activeScheduleWindow={profile.active_schedule_window}
              onAdvanceRequest={handleAdvanceRequestAction}
            />
          </section>
        )}

        {isCheckInEnabled && (
          <EmployeeAttendanceHistoryCard
            fromDate={month.fromDate}
            toDate={month.toDate}
            monthLabel={month.shortLabel}
            className="overflow-hidden rounded-2xl border border-[var(--employee-border)] bg-white lg:col-start-1 lg:row-start-3"
          />
        )}
      </div>

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
