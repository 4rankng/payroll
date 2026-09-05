import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowRightLeft,
  SearchCheck,
  Upload,
  Wallet as WalletIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import WalletTransactionsList from "@/components/wallet/WalletTransactionsList";
import CreateManualDisbursementDialog from "@/components/wallet/CreateManualDisbursementDialog";
import { EmployeeAccountLookupDialog } from "@/components/wallet/EmployeeAccountLookupDialog";
import { BulkTransferBatchList } from "@/components/wallet/BulkTransferBatchList";
import { BulkTransferUploadDialog } from "@/components/wallet/BulkTransferUploadDialog";
import { WalletBalanceCard } from "@/components/disbursement/WalletBalanceCard";

/* ------------------------------------------------------------------ */
/*  Main page                                                          */
/* ------------------------------------------------------------------ */
export default function WalletPage() {
  const queryClient = useQueryClient();
  const [disbursementOpen, setDisbursementOpen] = useState(false);
  const [employeeAccountLookupOpen, setEmployeeAccountLookupOpen] = useState(false);
  const [bulkTransferDialogOpen, setBulkTransferDialogOpen] = useState(false);
  const [selectedBulkTransferBatchId, setSelectedBulkTransferBatchId] =
    useState<number | null>(null);

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
  };

  return (
    <div className="min-h-full bg-background">
      <div className="max-w-[1280px] mx-auto px-4 md:px-6 py-5 md:py-6 space-y-5">

        {/* Page header */}
        <header className="flex items-start justify-between gap-3 flex-wrap">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary/10 border border-primary/20">
              <WalletIcon className="h-4 w-4 text-primary" />
            </div>
            <div>
              <h1 className="text-base font-semibold text-foreground leading-tight tracking-tight">
                Quản lý ví
              </h1>
              <p className="text-xs text-muted-foreground mt-0.5">
                Theo dõi số dư, chuyển khoản và đối soát giao dịch
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setEmployeeAccountLookupOpen(true)}
              className="h-9 gap-1.5"
            >
              <SearchCheck className="size-3.5" />
              Tra cứu tài khoản
            </Button>
            <Button
              size="sm"
              onClick={() => setBulkTransferDialogOpen(true)}
              className="gap-1.5 h-9"
            >
              <Upload className="h-3.5 w-3.5" />
              Tải File
            </Button>
            <Button
              size="sm"
              onClick={() => setDisbursementOpen(true)}
              className="gap-1.5 h-9"
            >
              <ArrowRightLeft className="h-3.5 w-3.5" />
              Chuyển tiền
            </Button>
          </div>
        </header>

        {/* Hero balance — the treasury dark panel owns sync, the as-of meta
            line, the pending-out companion on its rail, and mismatch
            reconciliation. Zone ownership: pending out lives ONLY here. */}
        <WalletBalanceCard className="rounded-2xl" />

        {/* Transactions */}
        <WalletTransactionsList />

        {/* Wallet Bulk Transfer Pipeline — Stage 2 history */}
        <BulkTransferBatchList
          selectedBatchId={selectedBulkTransferBatchId}
          onSelectedBatchIdChange={setSelectedBulkTransferBatchId}
        />
      </div>

      {/* Dialogs */}
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
