import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
// UU PRO restyle (W13c): lucide → verified @untitledui/icons.
import {
  SearchMd,
  SwitchHorizontal02,
  Upload01,
  Wallet01,
} from "@untitledui/icons";

import { Button } from "@/components/ui/button";
import {
  AdminPageCanvas,
} from "@/components/shared/AdminPageFrame";
import { PageHeader } from "@/components/shared/PageHeader";
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
    <AdminPageCanvas>
      {/* Page header — slim bar: title left, actions right */}
      <PageHeader
        icon={Wallet01}
        title="Quản lý ví"
      >
        <Button
          variant="outline"
          size="sm"
          onClick={() => setEmployeeAccountLookupOpen(true)}
          className="h-9 gap-1.5"
        >
          <SearchMd className="size-3.5" aria-hidden="true" />
          Tra cứu
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setBulkTransferDialogOpen(true)}
          className="gap-1.5 h-9"
        >
          <Upload01 className="h-3.5 w-3.5" aria-hidden="true" />
          Chuyển lô
        </Button>
        <Button
          size="sm"
          onClick={() => setDisbursementOpen(true)}
          className="gap-1.5 h-9"
        >
          <SwitchHorizontal02 className="h-3.5 w-3.5" aria-hidden="true" />
          Chuyển tiền
        </Button>
      </PageHeader>

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
    </AdminPageCanvas>
  );
}
