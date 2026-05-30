import { useMemo, useState } from "react";
import { Send } from "lucide-react";
import { PageHeader } from "@/components/shared/PageHeader";
import { WalletBalanceCard } from "@/components/disbursement/WalletBalanceCard";
import { useDisbursementFeeSchedules } from "@/hooks/api/useDisbursementFeeSchedules";
import {
  useInitiateManualDisbursement,
  useManualDisbursementBanks,
} from "@/hooks/api/useManualDisbursement";

import { ConfirmManualDisbursementDialog } from "./ConfirmManualDisbursementDialog";
import { ManualDisbursementForm } from "./ManualDisbursementForm";
import { RecentTransfers } from "./RecentTransfers";
import {
  type BankOption,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  initialFormState,
  validateForm,
} from "./helpers";

const ManualDisbursementPage = () => {
  const [form, setForm] = useState<ManualDisbursementFormState>(initialFormState);
  const [verified, setVerified] = useState<VerifiedAccount | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const initiateMutation = useInitiateManualDisbursement();
  const banksQuery = useManualDisbursementBanks();

  const banks: BankOption[] = useMemo(() => {
    const seen = new Set<string>();
    return (banksQuery.data ?? []).filter((b) => {
      const key = b.bank_code || b.swift_code;
      if (!key || seen.has(key)) return false;
      seen.add(key);
      return true;
    }).map((b) => ({ code: b.bank_code, label: b.bank_name, swiftCode: b.swift_code }));
  }, [banksQuery.data]);

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
      // Error toast already fired by the hook's onError; keep modal open
      // so the user can retry without re-entering the form.
    }
  };

  const reset = () => {
    setForm(initialFormState);
    setVerified(null);
  };

  return (
    <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
      <PageHeader
        icon={Send}
        title="Chuyển tiền"
        description="Khởi tạo giao dịch chuyển tiền thủ công. Mọi giao dịch đều được ghi nhận và thông báo cho người khởi tạo."
      />

      <div className="grid gap-5 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-5">
          <div className="rounded-lg border bg-card p-5">
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
          </div>
        </div>

        <aside className="lg:col-span-1 space-y-3">
          <WalletBalanceCard />
          <h2 className="text-sm font-semibold text-muted-foreground">
            Giao dịch gần đây
          </h2>
          <RecentTransfers limit={20} />
        </aside>
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

export default ManualDisbursementPage;
