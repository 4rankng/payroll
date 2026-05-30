import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Clock, AlertCircle } from "lucide-react";
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
  useAdvancePaymentHistory,
  useCalculateFee,
  useCancelAdvancePaymentRequest,
} from "@/hooks/api/useAdvancePayments";
import { useUnreadNotifications } from "@/hooks/api/useNotifications";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import { EmployeeBankInfoCard } from "@/components/employees/EmployeeBankInfoCard";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";
import { EmployeeCheckInCard } from "@/components/employees/EmployeeCheckInCard";
import { EmployeeAttendanceHistoryCard } from "@/components/employees/EmployeeAttendanceHistoryCard";
import { ChangePasswordSheet } from "@/components/employees/ChangePasswordSheet";
import { AdvancePaymentLimitCard } from "@/components/advance-payment/AdvancePaymentLimitCard";
import { AdvancePaymentRequestForm } from "@/components/advance-payment/AdvancePaymentRequestForm";
import { AdvancePaymentHistoryCard } from "@/components/advance-payment/AdvancePaymentHistoryCard";
import { AdvancePaymentConfirmSheet } from "@/components/advance-payment/AdvancePaymentConfirmSheet";
import { NotificationSheet } from "@/components/notifications/NotificationSheet";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";

const cardShadow = { boxShadow: "0 1px 6px rgba(0,0,0,0.08)" } as const;

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
  const {
    data: infoResponse,
    isLoading: infoLoading,
    refetch: refetchInfo,
  } = useAdvancePaymentInfo();
  const { data: historyResponse, isLoading: historyLoading } =
    useAdvancePaymentHistory({ page: 1, pageSize: 10 });

  const calculateFeeMutation = useCalculateFee();
  const requestMutation = useRequestAdvancePayment();
  const cancelMutation = useCancelAdvancePaymentRequest();
  const updatePasswordMutation = useUpdateEmployeePassword();

  const info = infoResponse?.data;
  const history = historyResponse?.data || [];

  // Debounced server-side fee calculation
  const [serverFeeDetails, setServerFeeDetails] = useState<{
    fee: number;
    netAmount: number;
  } | null>(null);

  const handleAmountChange = useCallback(
    (amount: number) => {
      setLatestAmount(amount);
      if (amount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) {
        setServerFeeDetails(null);
        return;
      }
      if (debounceRef.current) clearTimeout(debounceRef.current);
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

  if (profileLoading || infoLoading) {
    return (
      <div
        className="min-h-[100dvh]"
        style={{
          backgroundImage: "url('/employee-bg.avif')",
          backgroundSize: "cover",
          backgroundPosition: "center top",
          backgroundAttachment: "fixed",
          paddingTop: "env(safe-area-inset-top)",
          paddingBottom: "env(safe-area-inset-bottom)",
        }}
      >
        <div
          className="px-4 py-4 flex items-center justify-between"
          style={{ background: EMPLOYEE_BRAND_COLOR }}
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
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-2xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div
      className="min-h-[100dvh]"
      style={{
        backgroundImage: "url('/employee-bg.avif')",
        backgroundSize: "cover",
        backgroundPosition: "center top",
        backgroundAttachment: "fixed",
        paddingTop: "env(safe-area-inset-top)",
        paddingBottom: "env(safe-area-inset-bottom)",
      }}
    >
      <EmployeePortalHeader
        employeeName={profile?.fullname}
        unreadCount={unreadNotifications?.count}
        onNotificationClick={() => setNotificationSheetOpen(true)}
        onChangePassword={() => setPasswordSheetOpen(true)}
        onLogout={handleLogout}
      />

      <div className="max-w-2xl mx-auto p-4 space-y-3 pb-24">
        <EmployeeBankInfoCard
          profile={profile!}
          className="bg-white rounded-2xl overflow-hidden"
          style={cardShadow}
        />

        {profile?.check_in_enabled && (
          <>
            <EmployeeCheckInCard
              className="bg-white rounded-2xl overflow-hidden"
              style={cardShadow}
            />

            <EmployeeAttendanceHistoryCard
              className="bg-white rounded-2xl overflow-hidden"
              style={cardShadow}
            />
          </>
        )}

        {info && <AdvancePaymentLimitCard info={info} style={cardShadow} />}

        {info && !info.canRequest && info.canRequestReason && (
          <div
            className="bg-amber-50 border border-amber-200 rounded-2xl p-4 flex items-start gap-3"
            style={cardShadow}
          >
            <div className="shrink-0 mt-0.5">
              <div className="w-8 h-8 rounded-full bg-amber-100 flex items-center justify-center">
                <Clock className="h-4 w-4 text-amber-600" />
              </div>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-amber-800 mb-0.5">
                Tạm khóa yêu cầu ứng lương
              </p>
              <p className="text-xs text-amber-700 leading-relaxed">
                {info.canRequestReason}
              </p>
            </div>
            <AlertCircle className="h-4 w-4 text-amber-500 shrink-0 mt-0.5" />
          </div>
        )}

        {info?.canRequest && (
          <AdvancePaymentRequestForm
            key={formKey}
            info={info}
            feeDetails={feeDetails}
            onSubmit={handleRequestSubmit}
            onAmountChange={handleAmountChange}
            isPending={requestMutation.isPending}
            style={cardShadow}
          />
        )}

        <AdvancePaymentHistoryCard
          history={history}
          isLoading={historyLoading}
          onCancel={handleCancelRequest}
          style={cardShadow}
        />
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
        onConfirm={handleConfirmSubmit}
        isPending={requestMutation.isPending}
      />

      <NotificationSheet
        isOpen={notificationSheetOpen}
        onClose={() => setNotificationSheetOpen(false)}
      />
    </div>
  );
};

export default FlexiblePayEmployeePage;
