import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
// UU PRO restyle (W13c): lucide → verified @untitledui/icons; daisyUI
// ct-btn/ct-card/base-100/neutral chrome replaced with equivalent utility
// classes on the UU bridge (dark hero keeps the brand-section surface).
import {
  SearchMd,
  SwitchHorizontal02,
  Upload01,
  Wallet01,
} from '@untitledui/icons';
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
    <div className="min-h-dvh bg-brand-section_subtle text-fg-white">
      {/* Dark hero zone */}
      <div className="px-4 pt-[var(--mobile-header-top-padding,calc(env(safe-area-inset-top)+1rem))] pb-6">
        <div className="relative z-10 mb-3 flex items-center gap-2 pt-1">
          <Wallet01 className="h-5 w-5 shrink-0 text-white/60" aria-hidden="true" />
          <h1 className="truncate text-[17px] font-semibold text-fg-white tracking-tight">Ví điện tử</h1>
        </div>

        {/* Treasury dark panel — owns balance, sync, as-of meta, the
            pending-out companion on its rail, and mismatch reconciliation.
            Zone ownership: pending out lives ONLY here on this page. */}
        <WalletBalanceCard className="rounded-2xl" />

        <div className="mt-3 grid grid-cols-3 gap-1.5">
          <button
            type="button"
            onClick={() => setEmployeeAccountLookupOpen(true)}
            className="inline-flex h-auto min-h-11 flex-col items-center justify-center gap-1 rounded-xl border border-white/20 bg-white/5 px-1 py-2 text-xs font-semibold text-fg-white shadow-none"
          >
            <SearchMd className="h-4 w-4" aria-hidden="true" />
            Tra cứu tài khoản
          </button>
          <button
            type="button"
            onClick={() => setBulkTransferDialogOpen(true)}
            className="inline-flex h-auto min-h-11 flex-col items-center justify-center gap-1 rounded-xl border border-white/20 bg-white/5 px-1 py-2 text-xs font-semibold text-fg-white shadow-none"
          >
            <Upload01 className="h-4 w-4" aria-hidden="true" />
            Tải file
          </button>
          <button
            type="button"
            onClick={() => setDisbursementOpen(true)}
            className="inline-flex h-auto min-h-11 flex-col items-center justify-center gap-1 rounded-xl bg-primary text-primary-foreground px-1 py-2 text-xs font-semibold shadow-none"
          >
            <SwitchHorizontal02 className="h-4 w-4" aria-hidden="true" />
            Chuyển tiền
          </button>
        </div>
      </div>

      {/* Light transaction panel */}
      <div className="rounded-t-3xl -mt-4 min-h-[60dvh] bg-white shadow-lg">
        <div className="space-y-4 px-4 pt-5 pb-[calc(5rem+env(safe-area-inset-bottom))]">
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
