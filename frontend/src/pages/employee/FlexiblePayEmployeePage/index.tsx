import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
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
import { EmployeeWalletHero } from "@/components/employees/EmployeeWalletHero";
import { EmployeeCheckInCard } from "@/components/employees/EmployeeCheckInCard";
import { EmployeeAttendanceHistoryCard } from "@/components/employees/EmployeeAttendanceHistoryCard";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { AdvancePaymentRequestForm } from "@/components/advance-payment/AdvancePaymentRequestForm";
import { AdvancePaymentHistoryCard } from "@/components/advance-payment/AdvancePaymentHistoryCard";
import { AdvancePaymentConfirmSheet } from "@/components/advance-payment/AdvancePaymentConfirmSheet";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { formatAdvancePeriodDisplay } from "@/utils/advancePaymentHelpers";
import {
  createFlexibleEmployeeHomeModel,
  getEmployeeAccountHolder,
  hasEmployeeBankInfo,
  type EmployeeNudge,
  type EmployeeQuickAction,
} from "@/utils/employeePortal/mobileHome";

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
  // Check-in-enabled employees use the dedicated /me/check-in-advance flow
  // (70% advanceable cap, calendar-month window); others use the admin-upload flow.
  const isCheckIn = !!profile?.check_in_enabled;
  const regularInfoQuery = useAdvancePaymentInfo({ enabled: !isCheckIn });
  const checkInInfoQuery = useCheckInAdvanceInfo({ enabled: isCheckIn });
  const { data: infoResponse, isLoading: infoLoading, refetch: refetchInfo } = isCheckIn
    ? checkInInfoQuery
    : regularInfoQuery;
  const { data: historyResponse, isLoading: historyLoading } =
    useAdvancePaymentHistory({ page: 1, pageSize: 10 });

  const calculateFeeMutation = useCalculateFee();
  const regularRequestMutation = useRequestAdvancePayment();
  const checkInRequestMutation = useRequestCheckInAdvance();
  const requestMutation = isCheckIn ? checkInRequestMutation : regularRequestMutation;
  const cancelMutation = useCancelAdvancePaymentRequest();
  const updatePasswordMutation = useUpdateEmployeePassword();

  const info = infoResponse?.data;
  const history = useMemo(() => historyResponse?.data ?? [], [historyResponse?.data]);
  const homeModel = useMemo(
    () =>
      info
        ? createFlexibleEmployeeHomeModel({
            info,
            history,
            hasCheckIn: isCheckIn,
            hasBankInfo: hasEmployeeBankInfo(profile),
          })
        : null,
    [history, info, isCheckIn, profile]
  );

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

  if (profileLoading || infoLoading) {
    return (
      <div
        className="employee-mobile-page min-h-[100dvh]"
        style={{
          backgroundImage: "url('/employee-bg.avif')",
          backgroundSize: "cover",
          backgroundPosition: "center top",
          backgroundAttachment: "fixed",
          paddingBottom: "env(safe-area-inset-bottom)",
        }}
      >
        <div
          className="px-4 pb-4 flex items-center justify-between bg-employee"
          style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.875rem)" }}
        >
          <div className="space-y-2">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-28" />
          </div>
          <div className="flex gap-2">
            {[1, 2].map((i) => (
              <Skeleton key={i} className="h-10 w-10 rounded-full" />
            ))}
          </div>
        </div>
        <div className="p-4 space-y-3">
          <div className="rounded-2xl border border-white/60 bg-white/75 px-4 py-3 shadow-sm backdrop-blur">
            <p className="text-xs font-semibold uppercase tracking-[0.08em] text-emerald-700">Đang tải FlexPay</p>
            <p className="mt-1 text-sm text-slate-500">Kiểm tra hạn mức ứng lương và lịch sử gần đây.</p>
          </div>
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-2xl" />
          ))}
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
        contentClassName="pb-[calc(8rem+env(safe-area-inset-bottom))]"
      >
        {homeModel && (
          <EmployeeWalletHero model={homeModel} onAction={handleHomeAction} />
        )}

        {info?.canRequest && (
          <section id="employee-advance-request" className="scroll-mt-4">
            <AdvancePaymentRequestForm
              key={formKey}
              info={info}
              history={history}
              feeDetails={feeDetails}
              onSubmit={handleRequestSubmit}
              onAmountChange={handleAmountChange}
              isPending={requestMutation.isPending}
              style={employeeCardShadow}
            />
          </section>
        )}

        {profile?.check_in_enabled && (
          <>
            <section id="employee-check-in" className="scroll-mt-4">
              <EmployeeCheckInCard
                className="bg-white/95 rounded-2xl overflow-hidden ring-1 ring-white/80"
                checkInTarget={profile.check_in_target}
                checkInGeofenceRadiusMeters={profile.check_in_geofence_radius_meters}
                shiftStart={profile.shift_start}
                shiftEnd={profile.shift_end}
                checkInWindowStart={profile.check_in_window_start}
                checkInWindowEnd={profile.check_in_window_end}
                style={employeeCardShadow}
              />
            </section>

            <EmployeeAttendanceHistoryCard
              className="bg-white/95 rounded-2xl overflow-hidden ring-1 ring-white/80"
              style={employeeCardShadow}
            />
          </>
        )}

        <section id="employee-bank" className="scroll-mt-4">
          <EmployeeBankInfoCard
            profile={profile!}
            className="bg-white/95 rounded-2xl overflow-hidden ring-1 ring-white/80"
            style={employeeCardShadow}
          />
        </section>

        <section id="employee-history" className="scroll-mt-4">
          <AdvancePaymentHistoryCard
            history={history}
            isLoading={historyLoading}
            onCancel={handleCancelRequest}
            style={employeeCardShadow}
          />
        </section>

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
        payrollPeriodLabel={formatAdvancePeriodDisplay(latestMonth)}
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
