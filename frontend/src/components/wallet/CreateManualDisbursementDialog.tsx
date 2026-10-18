import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowRightLeft } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
} from "@/components/ui/dialog";
import {
  useInitiateManualDisbursement,
  useManualDisbursementBanks,
  useManualDisbursementStatus,
} from "@/hooks/api/useManualDisbursement";
import { useDisbursementFeeSchedules } from "@/hooks/api/useDisbursementFeeSchedules";
import { ConfirmManualDisbursementDialog } from "@/pages/admin/ManualDisbursementPage/ConfirmManualDisbursementDialog";
import { ManualDisbursementForm } from "@/pages/admin/ManualDisbursementPage/ManualDisbursementForm";
import { TransactionStatusPanel } from "@/pages/admin/ManualDisbursementPage/TransactionStatusPanel";
import {
  type BankOption,
  type DisbursementRecipientPrefill,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  initialFormState,
  validateForm,
} from "@/pages/admin/ManualDisbursementPage/helpers";

interface CreateManualDisbursementDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  /**
   * Optional provider-confirmed recipient handed off from the account-lookup
   * dialog. When set, the form opens pre-filled with this account and marked
   * verified, so the admin only enters the amount and description. Cleared via
   * `onOpenChange(false)` by the owner.
   */
  initialRecipient?: DisbursementRecipientPrefill | null;
}

export default function CreateManualDisbursementDialog({
  open,
  onOpenChange,
  onSuccess,
  initialRecipient = null,
}: CreateManualDisbursementDialogProps) {
  const queryClient = useQueryClient();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [form, setForm] = useState<ManualDisbursementFormState>(initialFormState);
  const [verified, setVerified] = useState<VerifiedAccount | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [activeTxnId, setActiveTxnId] = useState<string | null>(null);

  const initiateMutation = useInitiateManualDisbursement();
  const statusQuery = useManualDisbursementStatus(activeTxnId);
  const banksQuery = useManualDisbursementBanks();
  const feeQuery = useDisbursementFeeSchedules();

  const banks: BankOption[] = useMemo(
    () => (banksQuery.data ?? []).map((b) => ({ code: b.bank_code, label: b.bank_name, swiftCode: b.swift_code })),
    [banksQuery.data],
  );

  const activeFee = useMemo(() => {
    return feeQuery.data?.find((e) => e.isCurrentlyActive)?.feeVnd ?? null;
  }, [feeQuery.data]);

  const handleSubmit = () => {
    if (Object.keys(validateForm(form)).length > 0) return;
    setConfirmOpen(true);
  };

  const handleConfirm = async () => {
    try {
      const res = await initiateMutation.mutateAsync({
        amount: Number(form.amount),
        description: form.description.trim(),
        bank_code: form.bankCode,
        account_no: form.accountNo,
        account_name: form.accountName.trim(),
        account_type: form.accountType,
      });
      setConfirmOpen(false);
      const txnId = res.data?.txn_id;
      if (txnId) setActiveTxnId(txnId);
      onSuccess();
    } catch {
      // Error toast handled by hook's onError
    }
  };

  const reset = () => {
    setForm(initialFormState);
    setVerified(null);
    setActiveTxnId(null);
    queryClient.removeQueries({ queryKey: ["manual-disbursement", "detail"] });
    scrollRef.current?.scrollTo(0, 0);
  };

  // Reset form when dialog closes
  useEffect(() => {
    if (!open) {
      const timer = setTimeout(() => {
        setForm(initialFormState);
        setVerified(null);
        setActiveTxnId(null);
      }, 200);
      return () => clearTimeout(timer);
    }
  }, [open]);

  // Seed the form from a lookup-confirmed recipient when the dialog opens.
  // The account fields and the verified marker are both carried over, so the
  // admin lands on a pre-verified recipient and only fills in amount +
  // description. `verifiedAt` is stamped now so the normal 5-minute verification
  // TTL applies from the moment of handoff.
  useEffect(() => {
    if (!open || !initialRecipient) return;
    setForm({
      ...initialFormState,
      bankCode: initialRecipient.bankCode,
      accountNo: initialRecipient.accountNo,
      accountName: initialRecipient.accountName,
    });
    setVerified({
      bankCode: initialRecipient.bankCode,
      accountNo: initialRecipient.accountNo,
      verifiedName: initialRecipient.accountName,
      verifiedAt: Date.now(),
    });
    setActiveTxnId(null);
  }, [open, initialRecipient]);

  const inProgressRow = statusQuery.data ?? null;

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className="gap-0 overflow-hidden sm:max-w-[580px]"
          contentPadding="none"
          hideCloseButton
        >
          <DialogNavyHeader
            title={
              <span className="flex items-center gap-2">
                <ArrowRightLeft className="h-4 w-4 text-white/70" />
                Chuyển tiền
              </span>
            }
            description="Khởi tạo giao dịch chuyển tiền thủ công"
          />

          <div ref={scrollRef} className="max-h-[calc(92dvh-5rem)] overflow-y-auto px-4 py-4 sm:px-5">
            {inProgressRow ? (
              <TransactionStatusPanel row={inProgressRow} onReset={reset} />
            ) : (
              <ManualDisbursementForm
                banks={banks}
                banksLoading={banksQuery.isLoading}
                state={form}
                onChange={setForm}
                onSubmit={handleSubmit}
                verified={verified}
                onVerified={setVerified}
                disabled={initiateMutation.isPending}
              />
            )}
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmManualDisbursementDialog
        banks={banks}
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirm}
        state={form}
        verified={verified}
        fee={activeFee}
        pending={initiateMutation.isPending}
      />
    </>
  );
}
