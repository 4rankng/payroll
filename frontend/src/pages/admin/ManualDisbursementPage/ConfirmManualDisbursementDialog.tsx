import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";

import {
  type BankOption,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  bankLabel,
  formatAmount,
} from "./helpers";

interface Props {
  banks: BankOption[];
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  state: ManualDisbursementFormState;
  verified: VerifiedAccount | null;
  fee: number | null;
  pending: boolean;
}

export function ConfirmManualDisbursementDialog(props: Props) {
  const { banks, open, onClose, onConfirm, state, verified, fee, pending } = props;
  const [acknowledged, setAcknowledged] = useState(false);

  useEffect(() => {
    if (!open) setAcknowledged(false);
  }, [open]);

  const amount = Number(state.amount) || 0;
  const total = fee !== null ? amount + fee : null;
  const recipientName = verified?.verifiedName || state.accountName;

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-h-[92dvh] w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Xác nhận chuyển tiền</DialogTitle>
          <DialogDescription>
            Vui lòng kiểm tra kỹ thông tin bên dưới. Giao dịch sẽ chuyển tiền
            thật và không thể hoàn lại trực tiếp từ hệ thống này.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 rounded-lg border bg-muted/30 p-4">
          <div className="text-center">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              Người nhận
            </p>
            <p className="mt-1 break-words text-2xl font-semibold leading-tight">
              {recipientName}
            </p>
          </div>
          <div className="grid grid-cols-1 gap-3 text-sm min-[420px]:grid-cols-2">
            <div>
              <p className="text-xs text-muted-foreground">Ngân hàng</p>
              <p className="break-words font-medium">{bankLabel(state.bankCode, banks)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Số tài khoản</p>
              <p className="break-all font-mono">{state.accountNo}</p>
            </div>
          </div>
          <div className="text-center border-t pt-3">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              Số tiền chuyển
            </p>
            <p className="mt-1 text-3xl font-bold leading-tight">
              {formatAmount(amount)} <span className="text-lg font-sans">₫</span>
            </p>
            {fee !== null && (
              <p className="mt-1 text-xs text-muted-foreground">
                Phí: {formatAmount(fee)} ₫
                {total !== null && <> · Tổng: {formatAmount(total)} ₫</>}
              </p>
            )}
          </div>
          {state.description && (
            <div className="text-sm border-t pt-3">
              <p className="text-xs text-muted-foreground">Nội dung</p>
              <p className="mt-0.5">{state.description}</p>
            </div>
          )}
        </div>

        <label className="flex items-start gap-2 cursor-pointer rounded-md border p-3">
          <Checkbox
            id="md-confirm-ack"
            checked={acknowledged}
            onCheckedChange={(v) => setAcknowledged(v === true)}
            disabled={pending}
            className="mt-0.5"
          />
          <span className="text-sm leading-snug select-none">
            Tôi xác nhận thông tin trên là chính xác và muốn thực hiện giao dịch.
          </span>
        </label>

        <DialogFooter className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={pending}
            autoFocus
            className="min-h-11 w-full sm:w-auto"
          >
            Hủy
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={onConfirm}
            disabled={!acknowledged || pending}
            className="min-h-11 w-full sm:w-auto"
          >
            {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Xác nhận chuyển
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
