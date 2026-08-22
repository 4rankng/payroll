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
import { EmptyState } from "@/components/shared/EmptyState";

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
          <Skeleton key={i} className="h-24 w-full rounded-xl" />
        ))}
      </div>
    );
  }

  const rows = query.data ?? [];

  if (rows.length === 0) {
    return (
      <div className="rounded-md border border-dashed px-4">
        <EmptyState title="Chưa có giao dịch chuyển tiền nào" size="sm" className="py-4" />
      </div>
    );
  }

  return (
    <>
      <div className="divide-y rounded-xl border bg-card">
        {rows.map((r) => (
          <div
            key={r.txn_id}
            className="flex min-h-24 cursor-pointer items-start gap-3 p-3 transition-colors hover:bg-muted/40"
            onClick={() => setSelected(r)}
          >
            <div className="flex-1 min-w-0">
              <div className="flex flex-col gap-1 min-[420px]:flex-row min-[420px]:items-baseline min-[420px]:justify-between min-[420px]:gap-3">
                <p className="break-words font-medium leading-snug">{r.recipient_name}</p>
                <p className="shrink-0 text-sm font-semibold">
                  {formatAmount(r.requested_amount)} ₫
                </p>
              </div>
              <div className="mt-1 flex flex-col gap-1 text-xs text-muted-foreground min-[420px]:flex-row min-[420px]:items-center min-[420px]:justify-between min-[420px]:gap-3">
                <span className="break-words">
                  {bankLabel(r.recipient_bank, banks)} · {r.recipient_account_no}
                </span>
                <span className="shrink-0">{formatTimestamp(r.created_at)}</span>
              </div>
              {r.description && (
                <p className="mt-1 break-words text-xs text-muted-foreground">{r.description}</p>
              )}
              <div className="mt-2 flex flex-col gap-2 min-[420px]:flex-row min-[420px]:items-center min-[420px]:justify-between min-[420px]:gap-3">
                <span className={`inline-flex min-h-7 w-fit items-center rounded-full px-2 py-1 text-xs ${statusBadgeClass(r.status)}`}>
                  {statusLabel(r.status)}
                  {(r.status === "pending" || r.status === "authorised") && (
                    <Loader2 className="ml-1 h-3 w-3 animate-spin" />
                  )}
                </span>
                {isRealErrorCode(r.error_code) && (
                  <span className="break-words text-xs text-destructive" title={r.error_message ?? undefined}>
                    {r.error_message || "Thất bại"}
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      <Dialog open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
        <DialogContent className="max-h-[92dvh] w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] overflow-y-auto sm:max-w-lg">
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
