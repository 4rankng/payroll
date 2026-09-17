import { useCallback, useMemo, useState } from "react";
import { SlideSheetTemplate } from "@/components/sheets/templates/SlideSheetTemplate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Landmark,
  Calendar,
  Timer,
  Banknote,
  Wallet,
  X,
  CheckCircle,
  Trash2,
  CreditCard,
  Check,
} from "lucide-react";
import { useLoan, useLoanSchedule, useDeleteLoan } from "@/hooks/api/useLoans";
import { DisburseLoanModal } from "@/components/modals/DisburseLoanModal";
import { RepayScheduleModal } from "@/components/modals/RepayScheduleModal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { MarkSchedulePaidDialog } from "@/components/dialogs/MarkSchedulePaidDialog";
import { formatDate } from "@/utils/formatters";
import type { CustomScheduleItem } from "@/types/api/loan.types";
import {
  formatVND,
  getVietnameseLoanStatus,
  getLoanStatusColor,
  getLoanTypeLabel,
  getLoanTypeBadgeColor,
  getScheduleItemStatusLabel,
  getScheduleItemStatusColor,
  getPaymentUrgencyColor,
  daysUntil,
} from "@/utils/loanHelpers";

interface LoanDetailsSheetProps {
  isOpen: boolean;
  onClose: () => void;
  loanId?: number | null;
}

export function LoanDetailsSheet({
  isOpen,
  onClose,
  loanId,
}: LoanDetailsSheetProps) {
  const id = typeof loanId === "number" ? loanId : 0;
  const [isDisburseModalOpen, setIsDisburseModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isRepayScheduleModalOpen, setIsRepayScheduleModalOpen] =
    useState(false);
  const [selectedSchedule, setSelectedSchedule] =
    useState<CustomScheduleItem | null>(null);
  const [isMarkPaidDialogOpen, setIsMarkPaidDialogOpen] = useState(false);

  // Hooks return plain entities (not wrapped ApiResponse) for detail endpoints
  const { data: loan, isLoading: isLoadingLoan } = useLoan(id, isOpen && !!id);
  const { data: schedule = [], isLoading: isLoadingSchedule } = useLoanSchedule(
    id,
    isOpen && !!id && loan?.loan_type === "bullet_loan",
  );
  const deleteLoan = useDeleteLoan();

  const isLoading = isLoadingLoan || (isOpen && !!id && isLoadingSchedule);
  const isAutoInterest = loan?.loan_type === "bullet_loan";
  const isCustomSchedule = loan?.loan_type === "custom_schedule";

  const fallbackSchedule = useMemo<CustomScheduleItem[]>(
    () =>
      schedule?.map((item, index) => ({
        id: item.period ?? index + 1,
        period: item.period ?? index + 1,
        due_date: item.due_date,
        amount: item.amount,
        // The legacy schedule endpoint does not expose a principal split.
        // Treat it as interest-only in the preview so we never promise a
        // lower outstanding principal than the server will persist.
        principal_amount: 0,
        interest_amount: item.amount,
        status: item.status,
        paid_at: null as string | null,
      })) ?? [],
    [schedule],
  );

  const handleDisburse = () => {
    setIsDisburseModalOpen(true);
  };

  const handleDisburseModalClose = () => {
    setIsDisburseModalOpen(false);
  };

  const handleDelete = () => {
    setIsDeleteModalOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!loan) return;
    try {
      await deleteLoan.mutateAsync(loan.id);
      setIsDeleteModalOpen(false);
      onClose();
    } catch (error) {
      // Error handling is done in the mutation
    }
  };

  const handleDeleteModalClose = () => {
    setIsDeleteModalOpen(false);
  };

  const handleRepaySchedule = () => {
    setIsRepayScheduleModalOpen(true);
  };

  const handleRepayScheduleModalClose = () => {
    setIsRepayScheduleModalOpen(false);
  };

  const handleScheduleClick = useCallback(
    (scheduleItem: CustomScheduleItem) => {
      if (scheduleItem.status === "pending") {
        setSelectedSchedule(scheduleItem);
        setIsMarkPaidDialogOpen(true);
      }
    },
    [],
  );

  const handleMarkPaidDialogClose = () => {
    setIsMarkPaidDialogOpen(false);
    setSelectedSchedule(null);
  };

  const header = useMemo(() => {
    if (!loan) {
      return (
        <div className="flex items-center gap-4 flex-1 min-w-0">
          <div className="h-12 w-12 rounded-xl bg-blue-100 flex items-center justify-center flex-shrink-0">
            <Landmark className="h-6 w-6 text-blue-600" />
          </div>
          <div className="space-y-1 flex-1 min-w-0">
            <h1 className="typography-headline-medium text-base sm:text-lg font-medium">
              Chi tiết khoản vay
            </h1>
          </div>
        </div>
      );
    }

    return (
      <div className="flex items-center gap-4 flex-1 min-w-0">
        <div className="h-12 w-12 rounded-xl bg-blue-100 flex items-center justify-center flex-shrink-0">
          <Landmark className="h-6 w-6 text-blue-600" />
        </div>
        <div className="space-y-1 flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="typography-headline-medium text-base sm:text-lg font-medium">
              {loan.loan_code}
            </h1>
            <Badge
              className={getLoanStatusColor(loan.status)}
              variant="secondary"
            >
              {getVietnameseLoanStatus(loan.status)}
            </Badge>
            <Badge
              className={getLoanTypeBadgeColor(loan.loan_type)}
              variant="secondary"
            >
              {getLoanTypeLabel(loan.loan_type)}
            </Badge>
          </div>
          <div className="typography-body-small break-words text-muted-foreground">
            Chủ nợ:{" "}
            <span className="font-medium text-foreground">
              {loan.lender.name}
            </span>
          </div>
        </div>
      </div>
    );
  }, [loan]);

  const scheduleList = useMemo(() => {
    if (!loan) {
      return null;
    }

    const schedulesToDisplay =
      loan.schedules && loan.schedules.length > 0
        ? loan.schedules
        : fallbackSchedule;

    if (schedulesToDisplay.length === 0) {
      return (
        <div className="text-muted-foreground typography-body-small">
          Chưa có dữ liệu lịch trả.
        </div>
      );
    }

    return (
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {schedulesToDisplay.map((item) => {
          const urgencyColor = getPaymentUrgencyColor(daysUntil(item.due_date));
          const isPending = item.status === "pending";
          return (
            <div
              key={item.id}
              onClick={() => isPending && handleScheduleClick(item)}
              className={`
                rounded-xl border bg-card p-3 sm:p-4 shadow-sm flex flex-col gap-3 relative overflow-hidden
                ${
                  isPending
                    ? "cursor-pointer hover:border-blue-400 transition-all duration-200 group"
                    : "cursor-default opacity-75"
                }
              `}
            >
              {isPending && (
                <div className="absolute inset-0 bg-blue-50/0 group-hover:bg-blue-50/50 transition-colors duration-200 pointer-events-none" />
              )}
              <div className="relative flex flex-col gap-2 min-[380px]:flex-row min-[380px]:items-start min-[380px]:justify-between">
                <div className="min-w-0 space-y-1">
                  <p className="text-xs uppercase text-muted-foreground">
                    Đến hạn
                  </p>
                  <p
                    className={`typography-body-medium font-semibold ${urgencyColor}`}
                  >
                    {formatDate(item.due_date)}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {isPending && (
                    <div className="opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                      <Check className="h-4 w-4 text-blue-600" />
                    </div>
                  )}
                  <Badge
                    className={getScheduleItemStatusColor(item.status)}
                    variant="secondary"
                  >
                    {getScheduleItemStatusLabel(item.status)}
                  </Badge>
                </div>
              </div>
              <div className="relative flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                <span className="text-sm text-muted-foreground">
                  Kỳ {item.period}
                </span>
                <span className="typography-body-medium break-words font-semibold">
                  {formatVND(item.amount)}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    );
  }, [loan, fallbackSchedule, handleScheduleClick]);

  const hasPendingSchedules = useMemo(() => {
    if (!isCustomSchedule || !loan?.schedules) return false;
    return loan.schedules.some((s) => s.status === "pending");
  }, [isCustomSchedule, loan]);

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={onClose}
        avatar={{ custom: header }}
        className="w-full sm:w-[620px] md:w-[780px] lg:w-[960px] xl:w-[1100px]"
        footer={
          // One adaptive row of equal-width actions (see TransactionDetailsSheet).
          <div className="flex w-full flex-wrap items-center gap-2">
            {loan && !loan.disbursement_date && (
              <Button type="button" variant="destructive" size="sm" onClick={handleDelete} className="min-h-11 flex-1 basis-0 gap-1.5 px-3">
                <Trash2 className="w-3.5 h-3.5" />Xóa
              </Button>
            )}
            {loan && !loan.disbursement_date && (
              <Button type="button" size="sm" onClick={handleDisburse} className="min-h-11 flex-1 basis-0 gap-1.5 px-3">
                <CheckCircle className="w-3.5 h-3.5" />Giải ngân
              </Button>
            )}
            {loan && loan.disbursement_date && isCustomSchedule && hasPendingSchedules && (
              <Button type="button" size="sm" onClick={handleRepaySchedule} className="min-h-11 flex-1 basis-0 gap-1.5 px-3">
                <CreditCard className="w-3.5 h-3.5" />Thanh toán theo lịch
              </Button>
            )}
            <Button type="button" variant="outline" size="sm" onClick={onClose} className="min-h-11 flex-1 basis-0 px-3">
              Đóng
            </Button>
          </div>
        }
      >
        {isLoading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-muted-foreground">Đang tải...</p>
          </div>
        )}

        {!isLoading && loan && (
          <div className="space-y-6 pt-1 pb-4">
            {/* Disbursement Status Alert */}
            {!loan.disbursement_date && (
              <div className="bg-orange-50 border border-orange-200 rounded-xl p-3 sm:p-4 flex items-start gap-3">
                <Banknote className="h-4 w-4 text-orange-600 flex-shrink-0 mt-0.5" />
                <p className="typography-body-small text-orange-800">
                  <span className="font-semibold">Chưa giải ngân:</span> Khoản
                  vay này chưa được giải ngân. Vui lòng nhấn nút "Giải ngân" bên
                  dưới để hoàn tất.
                </p>
              </div>
            )}

            {loan.disbursement_date && (
              <div className="bg-green-50 border border-green-200 rounded-xl p-3 sm:p-4 flex items-start gap-3">
                <CheckCircle className="h-4 w-4 text-green-600 flex-shrink-0 mt-0.5" />
                <p className="typography-body-small text-green-800">
                  <span className="font-semibold">Đã giải ngân:</span>{" "}
                  {formatDate(loan.disbursement_date)}
                </p>
              </div>
            )}

            {/* Amounts & dates */}
            <div className="space-y-3">
              <div className="typography-label-medium text-muted-foreground uppercase font-semibold">
                Tổng quan
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3">
                <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                    <Banknote className="h-4 w-4" aria-hidden="true" />
                    <span>Tổng vay</span>
                  </div>
                  <div className="typography-body-medium font-semibold text-foreground">
                    {formatVND(loan.principal_amount)}
                  </div>
                </div>
                <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                    <Wallet className="h-4 w-4" aria-hidden="true" />
                    <span>Dư nợ</span>
                  </div>
                  <div className="typography-body-medium font-semibold text-orange-600">
                    {formatVND(loan.outstanding_principal)}
                  </div>
                </div>
                <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                    <Calendar className="h-4 w-4" aria-hidden="true" />
                    <span>Ngày giải ngân</span>
                  </div>
                  <div className="typography-body-medium font-semibold text-foreground">
                    {loan.disbursement_date
                      ? formatDate(loan.disbursement_date)
                      : "Chưa giải ngân"}
                  </div>
                </div>
                <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                    <Timer className="h-4 w-4" aria-hidden="true" />
                    <span>Kỳ thanh toán tới</span>
                  </div>
                  <div
                    className={`typography-body-medium font-semibold ${loan.next_payment_date ? getPaymentUrgencyColor(daysUntil(loan.next_payment_date)) : "text-muted-foreground"}`}
                  >
                    {loan.next_payment_date
                      ? formatDate(loan.next_payment_date)
                      : "Chưa có lịch"}
                  </div>
                </div>
                <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                    <CreditCard className="h-4 w-4" aria-hidden="true" />
                    <span>Số tiền kỳ tới</span>
                  </div>
                  <div className="typography-body-medium font-semibold text-foreground">
                    {loan.next_payment_amount
                      ? formatVND(loan.next_payment_amount)
                      : "-"}
                  </div>
                </div>
                {isAutoInterest && (
                  <div className="rounded-xl border bg-card p-4 shadow-sm space-y-2">
                    <div className="flex items-center gap-2 text-muted-foreground text-xs uppercase">
                      <Timer className="h-4 w-4" aria-hidden="true" />
                      <span>Lãi hàng tháng</span>
                    </div>
                    <div className="typography-body-medium font-semibold text-foreground">
                      {formatVND(loan.monthly_interest ?? 0)}
                    </div>
                  </div>
                )}
              </div>
            </div>

            <Separator />

            {/* Schedule */}
            <div className="space-y-3">
              <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                <div className="typography-label-medium text-muted-foreground uppercase font-semibold">
                  Lịch trả
                </div>
                {loan.next_payment_date && (
                  <div className="break-words text-xs text-muted-foreground">
                    Kỳ tới: {formatDate(loan.next_payment_date)}
                  </div>
                )}
              </div>
              {scheduleList}
            </div>
          </div>
        )}
      </SlideSheetTemplate>

      {loan && (
        <>
          <DisburseLoanModal
            isOpen={isDisburseModalOpen}
            onClose={handleDisburseModalClose}
            loan={loan}
          />
          <RepayScheduleModal
            isOpen={isRepayScheduleModalOpen}
            onClose={handleRepayScheduleModalClose}
            loan={loan}
          />
          <ConfirmDialog
            open={isDeleteModalOpen}
            onOpenChange={handleDeleteModalClose}
            onConfirm={handleDeleteConfirm}
            title="Xóa khoản vay"
            description={
              <>
                Bạn có chắc chắn muốn xóa khoản vay{" "}
                <strong>{loan.loan_code}</strong>?
                <br />
                <br />
                Hành động này không thể hoàn thành.
              </>
            }
            confirmText="Xóa"
            cancelText="Hủy bỏ"
            confirmVariant="destructive"
            loading={deleteLoan.isPending}
          />
          <MarkSchedulePaidDialog
            isOpen={isMarkPaidDialogOpen}
            onClose={handleMarkPaidDialogClose}
            loan={loan}
            schedule={selectedSchedule}
          />
        </>
      )}
    </>
  );
}

export default LoanDetailsSheet;
