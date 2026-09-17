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
import { useAdvanceFeePreview } from "@/hooks/advance-payment/useAdvanceFeePreview";
import { EmployeeBankInfoCard } from "@/components/employees/EmployeeBankInfoCard";
import { EmployeeMobileShell } from "@/components/employees/EmployeeMobileShell";
import { EmployeeCanopy } from "@/components/employees/EmployeeCanopy";
import { EmployeeCheckInCard } from "@/components/employees/EmployeeCheckInCard";
import { EmployeeAttendanceHistoryCard } from "@/components/employees/EmployeeAttendanceHistoryCard";
import { EmployeeMonthNavigator } from "@/components/employees/EmployeeMonthNavigator";
import { useEmployeeMonth } from "@/hooks/useEmployeeMonth";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { AdvancePaymentRequestForm } from "@/components/advance-payment/AdvancePaymentRequestForm";
import { EmployeeAdBanner } from "@/components/employees/EmployeeAdBanner";
import { AdvancePaymentHistoryCard } from "@/components/advance-payment/AdvancePaymentHistoryCard";
import { AdvancePaymentConfirmSheet } from "@/components/advance-payment/AdvancePaymentConfirmSheet";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import {
  formatPayrollMonthRange,
  getInitialEmployeeAdvanceMonth,
  getInitialHybridAdvanceMonth,
  isPastAdvancePaymentPeriod,
  isPriorMonthRequestable,
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
  // (Admin-configured advanceable cap, calendar-month window) for the CURRENT
  // month. During the days 1-8 tail they may ALSO request the previous payroll
  // month through the regular admin-upload flow, so both surfaces stay live.
  const isCheckInEnabled = Boolean(profile?.check_in_enabled);
  const isPendingCheckIn = Boolean(profile?.pending_check_in_enabled);
  // Which surface owns the selected month: current calendar month → checkin
  // flow; previous month during the tail → regular flow.
  const currentCalMonth = useMemo(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  }, []);
  const prevCalMonth = useMemo(() => {
    const now = new Date();
    const prev = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    return `${prev.getFullYear()}-${String(prev.getMonth() + 1).padStart(2, "0")}`;
  }, []);
  // Hybrid: checkin-enabled AND viewing the prior month while its tail is open.
  const isCheckIn = isCheckInEnabled && selectedMonth === currentCalMonth;
  const regularInfoQuery = useAdvancePaymentInfo({ enabled: !isCheckInEnabled || !isCheckIn });
  const checkInInfoQuery = useCheckInAdvanceInfo({ enabled: isCheckInEnabled });
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
  // Month decides the endpoint: current month → checkin flow, prior month
  // during the tail → regular flow. Both mutations share the same payload.
  const requestMutation = isCheckIn ? checkInRequestMutation : regularRequestMutation;
  // After a submit, refetch whichever surface owns the requested month.
  const refetchOwnedInfo = isCheckIn ? checkInInfoQuery.refetch : regularInfoQuery.refetch;
  const cancelMutation = useCancelAdvancePaymentRequest();
  const updatePasswordMutation = useUpdateEmployeePassword();

  const info = infoResponse?.data;
  const history = historyResponse?.data ?? EMPTY_HISTORY;
  const historyTotal = historyResponse?.pagination?.totalRecords ?? history.length;

  // A non-check-in employee can request against the previous payroll month
  // through the cutoff. The API's forMonth is authoritative; only use it for
  // the initial view so explicit URL and navigator selections stay intact.
  // Hybrid (checkin-enabled) employees open on the prior month when its tail
  // is open with unused quota, otherwise on the current checkin month.
  useEffect(() => {
    if (profileLoading || !profile || infoLoading || !info || initialMonthResolvedRef.current) return;

    let initialMonth: string | undefined;
    if (isCheckInEnabled) {
      initialMonth = getInitialHybridAdvanceMonth(new Date(), currentCalMonth, prevCalMonth, regularInfoQuery.data?.data);
    } else {
      initialMonth = getInitialEmployeeAdvanceMonth(info?.forMonth, false, hasExplicitMonth);
    }
    initialMonthResolvedRef.current = true;

    if (initialMonth && selectedMonth !== initialMonth) {
      setMonth(initialMonth);
    }
  }, [
    info,
    infoLoading,
    isCheckInEnabled,
    hasExplicitMonth,
    profileLoading,
    profile,
    regularInfoQuery.data,
    currentCalMonth,
    prevCalMonth,
    selectedMonth,
    setMonth,
  ]);

  const { feeDetails, feeError, calculate: calculateFee, retry: retryFee } =
    useAdvanceFeePreview(calculateFeeMutation.mutateAsync);

  const handleAmountChange = useCallback(
    (amount: number) => {
      setLatestAmount(amount);
      calculateFee(amount);
    },
    [calculateFee]
  );

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
      calculateFee(0);
      void Promise.all([refetchInfo(), refetchOwnedInfo(), refetchHistory()]).finally(() => setConfirmationDataReady(true));
    } catch {
      /* handled */
    }
  }, [calculateFee, latestAmount, latestMonth, requestMutation, refetchHistory, refetchInfo, refetchOwnedInfo]);

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
      <EmployeeMobileShell chrome="skeleton" contentClassName="max-w-lg space-y-4">
        <Skeleton className="h-14 w-full rounded-2xl border border-slate-200/60 bg-white" />
        <Skeleton className="h-64 w-full rounded-2xl border border-slate-200/60 bg-white" />
        <div className="space-y-3">
          <Skeleton className="h-6 w-40 rounded-xl" />
          <Skeleton className="h-36 w-full rounded-2xl border border-slate-200/60 bg-white" />
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
      hasActionToolbar={isCheckInEnabled || isPendingCheckIn}
      contentClassName="max-w-6xl space-y-4 sm:space-y-5"
      canopy={
        <EmployeeCanopy
          employeeName={profile?.fullname}
          unreadCount={unreadNotifications?.count}
          onNotificationClick={() => setNotificationSheetOpen(true)}
          onChangePassword={() => setPasswordSheetOpen(true)}
          onLogout={handleLogout}
          periodSlot={<EmployeeMonthNavigator month={month} variant="canopy" />}
        />
      }
    >
      {/* Ad slot: sibling ABOVE the pinned grid — grid children have explicit
          lg:col-start/row-start placement and must not gain a sibling inside. */}
      <EmployeeAdBanner />

      <div className="grid gap-4 sm:gap-5 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:items-start lg:gap-6">
        {infoError ? (
          <section
            id="employee-advance-request"
            className="scroll-mt-24 rounded-2xl border border-slate-200/60 bg-white px-5 py-6 text-center shadow-[0_1px_3px_rgba(0,0,0,0.04),0_4px_16px_rgba(0,0,0,0.03)] lg:col-start-1 lg:row-start-1"
            role="alert"
          >
            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-red-50 text-red-400">
              <AlertCircle className="h-6 w-6" aria-hidden="true" />
            </div>
            <h2 className="mt-3 text-[1rem] font-semibold text-slate-700">Chưa tải được hạn mức ứng lương</h2>
            <p className="mt-1 text-[0.8125rem] text-slate-400">Kiểm tra kết nối rồi thử lại.</p>
            <button
              type="button"
              onClick={() => { void refetchInfo(); }}
              className="mt-4 inline-flex h-11 items-center gap-2 rounded-xl border border-slate-200 bg-white px-5 text-[0.875rem] font-semibold text-slate-600 transition-all duration-150 hover:border-emerald-300 hover:bg-emerald-50 hover:text-emerald-700 active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
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
              isPastMonth={
                isCheckInEnabled && selectedMonth === prevCalMonth
                  ? !isPriorMonthRequestable(new Date(), prevCalMonth, regularInfoQuery.data?.data)
                  : isPastAdvancePaymentPeriod(month.value, info.forMonth)
              }
              isSelfCheckInFlow={isCheckIn}
              history={history}
              feeDetails={feeDetails}
              feeError={feeError}
              onRetryFee={retryFee}
              hasBankDestination={hasEmployeeBankInfo(profile)}
              onSubmit={handleRequestSubmit}
              onAmountChange={handleAmountChange}
              onBankAction={handleBankAction}
              isPending={requestMutation.isPending}
              requestConfirmation={requestConfirmation}
            />
          </section>
        ) : null}

        <section
          id="employee-history"
          className={
            isCheckInEnabled || isPendingCheckIn
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
          className={isCheckInEnabled || isPendingCheckIn
            ? "scroll-mt-24 lg:col-start-2 lg:row-start-2"
            : "scroll-mt-24 lg:col-start-2 lg:row-start-1 lg:self-stretch"}
        >
          <EmployeeBankInfoCard profile={profile!} />
        </section>

        {(isCheckInEnabled || isPendingCheckIn) && (
          <section id="employee-check-in" className="scroll-mt-24 lg:col-start-1 lg:row-start-2">
            <EmployeeCheckInCard
              isPendingActivation={isPendingCheckIn && !isCheckInEnabled}
              pendingEffectiveFrom={profile.check_in_effective_from}
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

        {(isCheckInEnabled || isPendingCheckIn) && (
          <EmployeeAttendanceHistoryCard
            fromDate={month.fromDate}
            toDate={month.toDate}
            monthLabel={month.shortLabel}
            advancePercentage={info?.advancePercentage}
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
