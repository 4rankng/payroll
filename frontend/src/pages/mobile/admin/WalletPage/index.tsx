import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { ArrowRightLeft, SearchCheck, Upload, Wallet as WalletIcon } from 'lucide-react';
import WalletTransactionsList from '@/components/wallet/WalletTransactionsList';
import CreateManualDisbursementDialog from '@/components/wallet/CreateManualDisbursementDialog';
import { EmployeeAccountLookupDialog } from '@/components/wallet/EmployeeAccountLookupDialog';
import { BulkTransferBatchList } from '@/components/wallet/BulkTransferBatchList';
import { BulkTransferUploadDialog } from '@/components/wallet/BulkTransferUploadDialog';
import { WalletBalanceCard } from '@/components/disbursement/WalletBalanceCard';

export default function WalletPageMobile() {
  const queryClient = useQueryClient();
  const [disbursementOpen, setDisbursementOpen] = useState(false);
  const [employeeAccountLookupOpen, setEmployeeAccountLookupOpen] = useState(false);
  const [bulkTransferDialogOpen, setBulkTransferDialogOpen] = useState(false);
  const [selectedBulkTransferBatchId, setSelectedBulkTransferBatchId] =
    useState<number | null>(null);

  const invalidateAll = () => { queryClient.invalidateQueries({ queryKey: ['wallet'] }); };

  return (
    <div className="min-h-dvh bg-neutral text-neutral-content">
      {/* Dark hero zone */}
      <div className="px-4 pt-[var(--mobile-header-top-padding,calc(env(safe-area-inset-top)+1rem))] pb-6">
        <div className="relative z-10 mb-3 flex items-center gap-2 pt-1">
          <WalletIcon className="h-5 w-5 shrink-0 text-neutral-content/60" />
          <h1 className="truncate text-[17px] font-semibold text-neutral-content tracking-tight">Ví điện tử</h1>
        </div>

        {/* Treasury dark panel — owns balance, sync, as-of meta, the
            pending-out companion on its rail, and mismatch reconciliation.
            Zone ownership: pending out lives ONLY here on this page. */}
        <WalletBalanceCard className="rounded-2xl" />

        <div className="mt-3 grid grid-cols-3 gap-1.5">
          <button
            type="button"
            onClick={() => setEmployeeAccountLookupOpen(true)}
            className="ct-btn ct-btn-outline h-auto min-h-11 flex-col gap-1 rounded-xl px-1 py-2 border-base-100/20 bg-base-100/5 text-[11px] font-semibold normal-case text-neutral-content shadow-none"
          >
            <SearchCheck className="h-4 w-4" />
            Tra cứu tài khoản
          </button>
          <button
            type="button"
            onClick={() => setBulkTransferDialogOpen(true)}
            className="ct-btn ct-btn-outline h-auto min-h-11 flex-col gap-1 rounded-xl px-1 py-2 border-base-100/20 bg-base-100/5 text-[11px] font-semibold normal-case text-neutral-content shadow-none"
          >
            <Upload className="h-4 w-4" />
            Tải file
          </button>
          <button
            type="button"
            onClick={() => setDisbursementOpen(true)}
            className="ct-btn ct-btn-primary h-auto min-h-11 flex-col gap-1 rounded-xl px-1 py-2 text-[11px] font-semibold normal-case shadow-none"
          >
            <ArrowRightLeft className="h-4 w-4" />
            Chuyển tiền
          </button>
        </div>
      </div>

      {/* Light transaction panel */}
      <div className="ct-card rounded-t-3xl -mt-4 min-h-[60dvh] bg-base-100 shadow-lg">
        <div className="ct-card-body space-y-4 px-4 pt-5 pb-[calc(5rem+env(safe-area-inset-bottom))]">
          <WalletTransactionsList />
          <BulkTransferBatchList
            selectedBatchId={selectedBulkTransferBatchId}
            onSelectedBatchIdChange={setSelectedBulkTransferBatchId}
          />
        </div>
      </div>

      <CreateManualDisbursementDialog
        open={disbursementOpen}
        onOpenChange={setDisbursementOpen}
        onSuccess={invalidateAll}
      />
      <EmployeeAccountLookupDialog
        open={employeeAccountLookupOpen}
        onOpenChange={setEmployeeAccountLookupOpen}
      />
      <BulkTransferUploadDialog
        open={bulkTransferDialogOpen}
        onOpenChange={setBulkTransferDialogOpen}
        onViewProgress={setSelectedBulkTransferBatchId}
      />
    </div>
  );
}
