import { useMemo, useRef, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useQueryClient } from '@tanstack/react-query';
import { useDisbursementFeeSchedules } from '@/hooks/api/useDisbursementFeeSchedules';
import {
  useInitiateManualDisbursement,
  useManualDisbursementBanks,
  useManualDisbursementStatus,
} from '@/hooks/api/useManualDisbursement';
import { ManualDisbursementForm } from '@/pages/admin/ManualDisbursementPage/ManualDisbursementForm';
import { ConfirmManualDisbursementDialog } from '@/pages/admin/ManualDisbursementPage/ConfirmManualDisbursementDialog';
import { TransactionStatusPanel } from '@/pages/admin/ManualDisbursementPage/TransactionStatusPanel';
import {
  type BankOption,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  initialFormState,
  validateForm,
} from '@/pages/admin/ManualDisbursementPage/helpers';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SendMoneyDialog({ open, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [form, setForm] = useState<ManualDisbursementFormState>(initialFormState);
  const [verified, setVerified] = useState<VerifiedAccount | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [activeTxnId, setActiveTxnId] = useState<string | null>(null);

  const initiateMutation = useInitiateManualDisbursement();
  const statusQuery = useManualDisbursementStatus(activeTxnId);
  const banksQuery = useManualDisbursementBanks();

  const banks: BankOption[] = useMemo(
    () => (banksQuery.data ?? []).map((b) => ({ code: b.bank_code, label: b.bank_name, swiftCode: b.swift_code })),
    [banksQuery.data],
  );

  const feeQuery = useDisbursementFeeSchedules();
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
    } catch {
      // Error toast handled by hook's onError
    }
  };

  const reset = () => {
    setForm(initialFormState);
    setVerified(null);
    setActiveTxnId(null);
    queryClient.removeQueries({ queryKey: ['manual-disbursement', 'detail'] });
    scrollRef.current?.scrollTo(0, 0);
  };

  const inProgressRow = statusQuery.data ?? null;

  return (
    <>
      <Dialog open={open && !confirmOpen} onOpenChange={(v) => { if (!v) { reset(); } onOpenChange(v); }}>
        <DialogContent className="w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] sm:max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Chuyển tiền</DialogTitle>
          </DialogHeader>
          <div ref={scrollRef} className="overflow-y-auto">
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
