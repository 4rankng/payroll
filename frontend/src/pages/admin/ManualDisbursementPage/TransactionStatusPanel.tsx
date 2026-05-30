import { useMemo } from "react";
import { CheckCircle2, Loader2, XCircle, Undo2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { ManualDisbursementResponse } from "@/types/api/manual-disbursement.types";
import { isTerminalStatus } from "@/types/api/manual-disbursement.types";
import { useManualDisbursementBanks } from "@/hooks/api/useManualDisbursement";

import {
  type BankOption,
  bankLabel,
  formatAmount,
  formatTimestamp,
  isProviderSuccess,
  isRealErrorCode,
  statusBadgeClass,
  statusLabel,
} from "./helpers";

interface Props {
  row: ManualDisbursementResponse;
  onReset: () => void;
  hideResetButton?: boolean;
}

export function TransactionStatusPanel({ row, onReset, hideResetButton = false }: Props) {
  const terminal = isTerminalStatus(row.status);
  const providerOk = isProviderSuccess(row);
  const effectivelyDone = terminal || providerOk || isRealErrorCode(row.error_code);
  const Icon = providerOk ? CheckCircle2 : iconForStatus(row.status);
  const accent = providerOk ? "bg-success/15 text-success" : accentForStatus(row.status);
  const banksQuery = useManualDisbursementBanks();
  const banks: BankOption[] = useMemo(
    () => (banksQuery.data ?? []).map((b) => ({ code: b.bank_code, label: b.bank_name, swiftCode: b.swift_code })),
    [banksQuery.data],
  );

  return (
    <div className="rounded-lg border bg-card p-5 space-y-4">
      <div className="flex items-start gap-4">
        <div className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl ${accent}`}>
          <Icon className={`h-6 w-6 ${row.status === "pending" || (row.status === "authorised" && !providerOk) ? "animate-spin" : ""}`} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{providerOk ? "Thành công" : statusLabel(row.status)}</h2>
            <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs ${providerOk ? "bg-success/15 text-success border border-success/30" : statusBadgeClass(row.status)}`}>
              {providerOk ? "Thành công" : statusLabel(row.status)}
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            {row.invoice_no
              ? <>Mã giao dịch: <span className="font-mono">{row.invoice_no}</span></>
              : <>Mã yêu cầu: <span className="font-mono text-xs">{row.request_id}</span></>}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 rounded-md bg-muted/30 p-3 text-sm">
        <div>
          <p className="text-xs text-muted-foreground">Người nhận</p>
          <p className="font-medium truncate">{row.recipient_name}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Ngân hàng</p>
          <p className="font-medium truncate">{bankLabel(row.recipient_bank, banks)}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Số tài khoản</p>
          <p className="font-mono">{row.recipient_account_no}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Số tiền</p>
          <p className="font-semibold">{formatAmount(row.requested_amount)} đ</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Phí giao dịch</p>
          <p>{formatAmount(row.fee)} đ</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Khởi tạo lúc</p>
          <p>{formatTimestamp(row.created_at)}</p>
        </div>
        {row.settled_at && (
          <div className="col-span-2">
            <p className="text-xs text-muted-foreground">Hoàn tất lúc</p>
            <p>{formatTimestamp(row.settled_at)}</p>
          </div>
        )}
      </div>

      {isRealErrorCode(row.error_code) ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm">
          <p className="text-xs">
            Mã lỗi: <span className="font-mono">{row.error_code}</span>
          </p>
          {row.error_message && <p className="mt-1">{row.error_message}</p>}
        </div>
      ) : null}

      {!terminal && !providerOk && (
        <p className="text-xs text-muted-foreground">
          Hệ thống đang theo dõi giao dịch. Trạng thái sẽ tự cập nhật khi
          ngân hàng phản hồi. Bạn cũng sẽ nhận được thông báo đẩy.
        </p>
      )}

      {!hideResetButton && (
        <div className="flex justify-end">
          <Button variant="outline" onClick={onReset}>
            Chuyển tiền mới
          </Button>
        </div>
      )}
    </div>
  );
}

function iconForStatus(s: ManualDisbursementResponse["status"]) {
  switch (s) {
    case "completed":
      return CheckCircle2;
    case "failed":
      return XCircle;
    case "pending":
    case "authorised":
    default:
      return Loader2;
  }
}

function accentForStatus(s: ManualDisbursementResponse["status"]): string {
  switch (s) {
    case "completed":
      return "bg-success/15 text-success";
    case "failed":
      return "bg-destructive/15 text-destructive";
    case "pending":
    case "authorised":
    default:
      return "bg-primary/10 text-primary";
  }
}
