import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { formatCurrency } from "@/utils/formatters";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";

interface AdvancePaymentConfirmSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  amount: number;
  feeDetails: { fee: number; netAmount: number } | null;
  bankAccountNumber?: string;
  onConfirm: () => void;
  isPending: boolean;
}

export function AdvancePaymentConfirmSheet({
  open,
  onOpenChange,
  amount,
  feeDetails,
  bankAccountNumber,
  onConfirm,
  isPending,
}: AdvancePaymentConfirmSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="bottom"
        className="h-auto max-h-[70vh] rounded-t-3xl px-5"
      >
        <SheetHeader className="pb-3 border-b border-gray-100">
          <SheetTitle className="text-base font-bold text-gray-900">
            Xác nhận yêu cầu ứng lương
          </SheetTitle>
        </SheetHeader>
        <div className="space-y-3 mt-3" style={{ paddingBottom: "calc(env(safe-area-inset-bottom) + 16px)" }}>
          <div className="bg-gray-50 rounded-xl px-4 py-3 space-y-2 border border-gray-100">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Số tiền yêu cầu</span>
              <span className="font-semibold text-gray-700">
                {formatCurrency(amount)}
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">Phí giao dịch</span>
              <span className="font-semibold text-red-500">
                −{feeDetails ? formatCurrency(feeDetails.fee) : "…"}
              </span>
            </div>
            <div className="flex justify-between pt-2 border-t border-gray-200">
              <span className="font-semibold text-gray-700">Thực nhận</span>
              <span
                className="text-lg font-bold"
                style={{ color: EMPLOYEE_BRAND_COLOR }}
              >
                {feeDetails ? formatCurrency(feeDetails.netAmount) : "…"}
              </span>
            </div>
          </div>
          {bankAccountNumber && (
            <p className="text-xs text-gray-400 text-center">
              Nhận về tài khoản {bankAccountNumber}
            </p>
          )}
          <div className="flex gap-3 pt-1">
            <button
              onClick={() => onOpenChange(false)}
              className="flex-1 py-2.5 rounded-xl text-sm font-semibold border border-gray-200 text-gray-600 active:scale-[0.97] transition-transform"
            >
              Hủy
            </button>
            <button
              onClick={onConfirm}
              disabled={isPending}
              className="flex-1 py-2.5 rounded-xl text-sm font-semibold text-white disabled:opacity-50 active:scale-[0.97] transition-transform"
              style={{ background: EMPLOYEE_BRAND_COLOR }}
            >
              {isPending ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="animate-spin h-3.5 w-3.5 border-2 border-white/30 border-t-white rounded-full" />
                  Đang gửi...
                </span>
              ) : (
                "Xác nhận"
              )}
            </button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
