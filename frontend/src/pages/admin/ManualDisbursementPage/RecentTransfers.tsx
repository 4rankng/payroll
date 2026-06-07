import { useMemo, useState } from "react";
import { Loader2 } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  useManualDisbursementBanks,
  useManualDisbursementList,
} from "@/hooks/api/useManualDisbursement";
import type { ManualDisbursementResponse } from "@/types/api/manual-disbursement.types";

import {
  type BankOption,
  bankLabel,
  formatAmount,
  formatTimestamp,
  isRealErrorCode,
  statusBadgeClass,
  statusLabel,
} from "./helpers";
import { TransactionStatusPanel } from "./TransactionStatusPanel";

interface Props {
  limit?: number;
}

export function RecentTransfers({ limit = 20 }: Props) {
  const query = useManualDisbursementList(limit);
  const banksQuery = useManualDisbursementBanks();
  const [selected, setSelected] = useState<ManualDisbursementResponse | null>(null);

  const banks: BankOption[] = useMemo(
    () => (banksQuery.data ?? []).map((b) => ({ code: b.bank_code, label: b.bank_name, swiftCode: b.swift_code })),
    [banksQuery.data],
  );

  if (query.isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-14 w-full" />
        ))}
      </div>
    );
  }

  const rows = query.data ?? [];

  if (rows.length === 0) {
    return (
      <div className="rounded-md border border-dashed p-6 text-center text-sm text-muted-foreground">
        Chưa có giao dịch chuyển tiền nào.
      </div>
    );
  }

  return (
    <>
      <div className="divide-y rounded-xl border bg-card">
        {rows.map((r) => (
          <div
            key={r.txn_id}
            className="flex items-start gap-3 p-3 cursor-pointer hover:bg-muted/40 transition-colors"
            onClick={() => setSelected(r)}
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-baseline justify-between gap-3">
                <p className="font-medium truncate">{r.recipient_name}</p>
                <p className="text-sm font-semibold whitespace-nowrap">
                  {formatAmount(r.requested_amount)} đ
                </p>
              </div>
              <div className="flex items-center justify-between gap-3 mt-1 text-xs text-muted-foreground">
                <span className="truncate">
                  {bankLabel(r.recipient_bank, banks)} · {r.recipient_account_no}
                </span>
                <span className="whitespace-nowrap">{formatTimestamp(r.created_at)}</span>
              </div>
              {r.description && (
                <p className="mt-1 text-xs text-muted-foreground truncate">{r.description}</p>
              )}
              <div className="mt-2 flex items-center justify-between gap-3">
                <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs ${statusBadgeClass(r.status)}`}>
                  {statusLabel(r.status)}
                  {(r.status === "pending" || r.status === "authorised") && (
                    <Loader2 className="ml-1 h-3 w-3 animate-spin" />
                  )}
                </span>
                {isRealErrorCode(r.error_code) && (
                  <span className="text-xs text-destructive truncate" title={r.error_message ?? undefined}>
                    {r.error_message || "Thất bại"}
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      <Dialog open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
        <DialogContent className="w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Chi tiết giao dịch</DialogTitle>
          </DialogHeader>
          {selected && (
            <TransactionStatusPanel
              row={selected}
              onReset={() => setSelected(null)}
              hideResetButton
            />
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
