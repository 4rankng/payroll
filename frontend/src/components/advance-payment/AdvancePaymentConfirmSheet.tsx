import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { formatCurrency } from "@/utils/formatters";
import { maskBankAccountNumber } from "@/utils/employeePortal/mobileHome";

interface AdvancePaymentConfirmSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  amount: number;
  feeDetails: { fee: number; netAmount: number } | null;
  bankAccountNumber?: string;
  bankName?: string;
  bankAccountName?: string;
  payrollPeriodLabel?: string;
  onConfirm: () => void;
  isPending: boolean;
}

export function AdvancePaymentConfirmSheet({
  open,
  onOpenChange,
  amount,
  feeDetails,
  bankAccountNumber,
  bankName,
  bankAccountName,
  payrollPeriodLabel,
  onConfirm,
  isPending,
}: AdvancePaymentConfirmSheetProps) {
  const maskedBankAccountNumber = maskBankAccountNumber(bankAccountNumber);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="bottom"
        className="flex h-[92dvh] max-h-[92dvh] flex-col overflow-hidden rounded-t-[28px] border-t border-white/70 bg-white p-0 shadow-[0_-24px_80px_-36px_rgba(15,23,42,0.65)]"
        title="Xác nhận yêu cầu ứng lương"
        description="Kiểm tra số tiền thực nhận và tài khoản nhận tiền"
      >
        <div className="flex shrink-0 justify-center pt-3">
          <div className="h-1.5 w-12 rounded-full bg-slate-300" />
        </div>
        <SheetHeader className="shrink-0 border-b border-slate-200 px-5 pb-4 pt-4 text-left">
          <SheetTitle className="text-[22px] font-extrabold leading-7 tracking-normal text-slate-950">
            Xác nhận yêu cầu ứng lương
          </SheetTitle>
          <p className="text-[14px] font-medium leading-6 text-slate-500">
            Kiểm tra kỹ thông tin trước khi gửi giao dịch.
          </p>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-5">
          <section className="border-b border-slate-100 py-5">
            <p className="text-[13px] font-bold uppercase leading-5 text-slate-500">
              Số tiền thực nhận
            </p>
            <p className="mt-1 break-words text-[36px] font-extrabold leading-none tracking-normal text-employee tabular-nums">
              {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
            </p>

            <div className="mt-5 space-y-3 text-[16px] leading-6">
              <div className="flex justify-between gap-4">
                <span className="text-slate-500">Số tiền yêu cầu</span>
                <span className="font-bold text-slate-900 tabular-nums">
                  {formatCurrency(amount)}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-slate-500">Phí giao dịch</span>
                <span className="font-bold text-red-500 tabular-nums">
                  {feeDetails ? `−${formatCurrency(feeDetails.fee)}` : "Đang tính..."}
                </span>
              </div>
              {payrollPeriodLabel && (
                <div className="flex justify-between gap-4 border-t border-slate-100 pt-3">
                  <span className="text-slate-500">Trong kỳ lương</span>
                  <span className="text-right font-bold text-slate-900">
                    {payrollPeriodLabel}
                  </span>
                </div>
              )}
            </div>
          </section>

          <section className="py-5">
            <p className="text-[13px] font-bold uppercase leading-5 text-slate-500">
              Chuyển đến chủ tài khoản
            </p>
            <p className="mt-1 break-words text-[27px] font-extrabold leading-8 tracking-normal text-employee">
              {bankAccountName || "Chưa cập nhật"}
            </p>
            <div className="mt-5 space-y-3 text-[16px] leading-6">
              <div className="flex justify-between gap-4">
                <span className="text-slate-500">Số tài khoản</span>
                <span className="break-all text-right font-bold text-slate-900 tabular-nums">
                  {maskedBankAccountNumber || "Chưa cập nhật"}
                </span>
              </div>
              <div className="flex justify-between gap-4 border-t border-slate-100 pt-3">
                <span className="text-slate-500">Ngân hàng</span>
                <span className="max-w-[58%] text-right font-bold text-slate-900">
                  {bankName || "Chưa cập nhật"}
                </span>
              </div>
            </div>
          </section>
        </div>

        <div
          className="shrink-0 border-t border-slate-200 bg-white px-5 py-3"
          style={{ paddingBottom: "calc(env(safe-area-inset-bottom, 0px) + 0.875rem)" }}
        >
          <div className="flex gap-3">
            <button
              onClick={() => onOpenChange(false)}
              className="min-h-14 flex-1 rounded-2xl border border-slate-200 py-3 text-[16px] font-bold text-slate-600 transition-transform active:scale-[0.97]"
            >
              Hủy
            </button>
            <button
              onClick={onConfirm}
              disabled={isPending || !feeDetails}
              className="min-h-14 flex-[1.4] rounded-2xl bg-employee py-3 text-[16px] font-extrabold text-white shadow-[0_14px_30px_-18px_rgba(0,177,79,0.9)] transition-transform active:scale-[0.97] disabled:opacity-50"
            >
              {isPending ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="animate-spin h-3.5 w-3.5 border-2 border-white/30 border-t-white rounded-full" />
                  Đang gửi...
                </span>
              ) : (
                "Xác nhận giao dịch"
              )}
            </button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
