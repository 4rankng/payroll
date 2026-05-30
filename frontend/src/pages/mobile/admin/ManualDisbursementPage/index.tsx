import { useMemo, useState } from "react";
import { ChevronDown, ChevronUp, Send } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useDisbursementFeeSchedules } from "@/hooks/api/useDisbursementFeeSchedules";
import {
  useInitiateManualDisbursement,
  useManualDisbursementBanks,
} from "@/hooks/api/useManualDisbursement";

import { ConfirmManualDisbursementDialog } from "@/pages/admin/ManualDisbursementPage/ConfirmManualDisbursementDialog";
import { ManualDisbursementForm } from "@/pages/admin/ManualDisbursementPage/ManualDisbursementForm";
import { RecentTransfers } from "@/pages/admin/ManualDisbursementPage/RecentTransfers";
import {
  type BankOption,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  initialFormState,
  validateForm,
} from "@/pages/admin/ManualDisbursementPage/helpers";

const ManualDisbursementPageMobile = () => {
  const [form, setForm] = useState<ManualDisbursementFormState>(initialFormState);
  const [verified, setVerified] = useState<VerifiedAccount | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [recentExpanded, setRecentExpanded] = useState(false);

  const initiateMutation = useInitiateManualDisbursement();
  const banksQuery = useManualDisbursementBanks();

  const banks: BankOption[] = useMemo(
    () => (banksQuery.data ?? []).map((b) => ({ code: b.bank_code, label: `${b.bank_name} (${b.bank_code || b.swift_code})`, swiftCode: b.swift_code })),
    [banksQuery.data],
  );

  const feeQuery = useDisbursementFeeSchedules();
  const activeFee = useMemo(
    () => feeQuery.data?.find((e) => e.isCurrentlyActive)?.feeVnd ?? null,
    [feeQuery.data],
  );

  const handleSubmit = () => {
    if (Object.keys(validateForm(form)).length > 0) return;
    setConfirmOpen(true);
  };

  const handleConfirm = async () => {
    try {
      await initiateMutation.mutateAsync({
        amount: Number(form.amount),
        description: form.description.trim(),
        bank_code: form.bankCode,
        account_no: form.accountNo,
        account_name: form.accountName.trim(),
        account_type: form.accountType,
      });
      setConfirmOpen(false);
      reset();
    } catch {
      // Hook surfaces toast on error; modal stays open for retry.
    }
  };

  const reset = () => {
    setForm(initialFormState);
    setVerified(null);
  };

  return (
    <div className="px-3 pb-24 pt-3 space-y-4">
      <div className="flex items-start gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10">
          <Send className="h-4 w-4 text-primary" />
        </div>
        <div className="flex-1">
          <h1 className="text-base font-semibold tracking-tight">Chuyển tiền</h1>
          <p className="text-xs text-muted-foreground">
            Khởi tạo giao dịch chuyển tiền
          </p>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-3">
        <ManualDisbursementForm
          banks={banks}
          state={form}
          onChange={setForm}
          onSubmit={handleSubmit}
          verified={verified}
          onVerified={setVerified}
          disabled={initiateMutation.isPending}
        />
      </div>

      <div className="space-y-2">
        <Button
          type="button"
          variant="ghost"
          className="w-full justify-between"
          onClick={() => setRecentExpanded((v) => !v)}
        >
          <span>Xem giao dịch gần đây</span>
          {recentExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        </Button>
        {recentExpanded && <RecentTransfers limit={20} />}
      </div>

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
    </div>
  );
};

export default ManualDisbursementPageMobile;
