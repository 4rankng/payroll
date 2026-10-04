import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { formatCurrency } from "@/utils/formatters";

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
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="bottom"
        className="mx-auto flex h-auto max-h-[86dvh] w-full flex-col overflow-hidden rounded-t-[28px] border-t border-[var(--employee-border-strong)] bg-[var(--employee-surface)] p-0 shadow-none sm:max-w-lg"
        title="Xác nhận yêu cầu ứng lương"
        description="Kiểm tra số tiền thực nhận và tài khoản nhận tiền"
      >
        <div className="flex shrink-0 justify-center pt-3">
          <div className="h-1.5 w-12 rounded-full bg-[var(--employee-border-strong)]" />
        </div>
        <SheetHeader className="shrink-0 border-b border-[var(--employee-border-strong)] px-5 pb-3 pt-3 text-left">
          <SheetTitle className="employee-type-hero-title text-[var(--employee-text)]">
            Xác nhận yêu cầu ứng lương
          </SheetTitle>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto px-5">
          <section className="border-b border-[var(--employee-border)] py-4">
            <p className="employee-type-label-caps text-fg-tertiary">
              Số tiền thực nhận
            </p>
            <p className="employee-type-confirm-amount mt-1 break-words text-[var(--employee-accent)] tabular-nums">
              {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
            </p>

            <div className="employee-type-body mt-4 space-y-2.5">
              <div className="flex justify-between gap-4">
                <span className="text-fg-tertiary">Số tiền yêu cầu</span>
                <span className="font-semibold text-[var(--employee-text)] tabular-nums">
                  {formatCurrency(amount)}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-fg-tertiary">Phí giao dịch</span>
                <span className="font-semibold text-fg-error-secondary tabular-nums">
                  {feeDetails ? `−${formatCurrency(feeDetails.fee)}` : "Đang tính..."}
                </span>
              </div>
              {payrollPeriodLabel && (
                <div className="flex justify-between gap-4 border-t border-[var(--employee-border)] pt-3">
                  <span className="text-fg-tertiary">Trong kỳ lương</span>
                  <span className="text-right font-semibold text-[var(--employee-text)]">
                    {payrollPeriodLabel}
                  </span>
                </div>
              )}
            </div>
          </section>

          <section className="py-4">
            <p className="employee-type-label-caps text-fg-tertiary">
              Chuyển đến chủ tài khoản
            </p>
            <p className="employee-type-card-amount mt-1 break-words text-[var(--employee-accent)]">
              {bankAccountName || "Chưa cập nhật"}
            </p>
            <div className="employee-type-body mt-4 space-y-2.5">
              <div className="flex justify-between gap-4">
                <span className="text-fg-tertiary">Số tài khoản</span>
                <span className="break-all text-right font-semibold text-[var(--employee-text)] tabular-nums">
                  {bankAccountNumber || "Chưa cập nhật"}
                </span>
              </div>
              <div className="flex justify-between gap-4 border-t border-[var(--employee-border)] pt-3">
                <span className="text-fg-tertiary">Ngân hàng</span>
                <span className="max-w-[58%] text-right font-semibold text-[var(--employee-text)]">
                  {bankName || "Chưa cập nhật"}
                </span>
              </div>
            </div>
          </section>
        </div>

        <div
          className="shrink-0 border-t border-[var(--employee-border-strong)] bg-[var(--employee-surface)] px-5 py-3"
          style={{ paddingBottom: "calc(env(safe-area-inset-bottom, 0px) + 0.875rem)" }}
        >
          <div className="flex gap-3">
            <button
              onClick={() => onOpenChange(false)}
              className="employee-type-action min-h-14 flex-1 rounded-2xl border border-[var(--employee-border-strong)] py-3 text-fg-secondary transition-colors focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)] active:bg-[var(--employee-surface-muted)]"
            >
              Hủy
            </button>
            <button
              onClick={onConfirm}
              disabled={isPending || !feeDetails}
              className="employee-type-action min-h-14 flex-[1.4] rounded-2xl bg-[var(--employee-accent)] py-3 text-white shadow-none transition-colors focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)] active:bg-[var(--employee-accent-strong)] disabled:pointer-events-none disabled:opacity-50"
            >
              {isPending ? (
                <span className="inline-flex items-center gap-1.5">
                  <span className="animate-spin h-3.5 w-3.5 border-2 border-white/30 border-t-white rounded-full motion-reduce:animate-none" />
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
