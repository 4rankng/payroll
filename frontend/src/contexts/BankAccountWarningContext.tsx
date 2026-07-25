import React, { createContext, useCallback, useContext, useState } from "react";
import { AlertTriangle, ArrowRight, Check, X } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { getBankAccountNameMismatch } from "@/utils/bank-account-warning";

/**
 * BankAccountWarningContext surfaces a click-to-dismiss modal dialog when a
 * saved employee has an invalid bank account. The save itself succeeds
 * (HTTP 200); this dialog forces the admin/partner to acknowledge the
 * problem so payments don't silently fail later.
 *
 * Unlike a toast (which auto-dismisses and is easily missed), this is a
 * centered modal that requires an explicit close action.
 */

interface BankAccountWarning {
  accountName?: string;
  reason: string;
}

interface BankAccountWarningContextValue {
  showBankAccountWarning: (warning: BankAccountWarning) => void;
}

const BankAccountWarningContext = createContext<BankAccountWarningContextValue | null>(null);

export function BankAccountWarningProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const [warning, setWarning] = useState<BankAccountWarning | null>(null);
  const nameMismatch = warning
    ? getBankAccountNameMismatch(warning.reason, warning.accountName)
    : null;

  const showBankAccountWarning = useCallback((w: BankAccountWarning) => {
    setWarning(w);
    setOpen(true);
  }, []);

  return (
    <BankAccountWarningContext.Provider value={{ showBankAccountWarning }}>
      {children}
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent
          className="max-w-lg border-0"
          overlayClassName="bg-slate-950/85 backdrop-blur-sm"
        >
          <AlertDialogHeader className="flex-row items-start gap-3 space-y-0 pb-4 text-left sm:px-6">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-rose-50 text-rose-600 ring-8 ring-rose-50/60">
              <AlertTriangle className="h-5 w-5" aria-hidden="true" />
            </div>
            <div className="min-w-0 pt-0.5">
              <AlertDialogTitle className="text-xl">
                {nameMismatch
                  ? "Tên chủ tài khoản không khớp"
                  : "Tài khoản ngân hàng không hợp lệ"}
              </AlertDialogTitle>
              <AlertDialogDescription className="sr-only">
                {nameMismatch
                  ? "So sánh tên đã nhập với tên do ngân hàng cung cấp."
                  : warning?.reason ?? "Thông tin tài khoản cần được kiểm tra lại."}
              </AlertDialogDescription>
            </div>
          </AlertDialogHeader>

          {nameMismatch ? (
            <div className="grid gap-3 border-y border-border bg-muted/35 px-5 py-5 sm:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] sm:items-center sm:gap-4 sm:px-6">
              <div className="min-w-0">
                <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-rose-700">
                  <X className="h-4 w-4" aria-hidden="true" />
                  Tên đã nhập
                </div>
                <p className="break-words text-base font-semibold text-foreground">
                  {nameMismatch.enteredName}
                </p>
              </div>

              <ArrowRight
                className="h-5 w-5 rotate-90 text-muted-foreground sm:rotate-0"
                aria-hidden="true"
              />

              <div className="min-w-0">
                <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-emerald-700">
                  <Check className="h-4 w-4" aria-hidden="true" />
                  Ngân hàng ghi
                </div>
                <p className="break-words text-base font-semibold text-foreground">
                  {nameMismatch.bankName}
                </p>
              </div>
            </div>
          ) : (
            <p className="border-y border-border bg-muted/35 px-5 py-4 text-sm font-medium text-foreground sm:px-6">
              {warning?.reason ?? "Thông tin tài khoản cần được kiểm tra lại."}
            </p>
          )}

          <div className="px-5 py-4 sm:px-6">
            <AlertDialogAction
              className="w-full"
              onClick={() => setOpen(false)}
            >
              Đóng
            </AlertDialogAction>
          </div>
        </AlertDialogContent>
      </AlertDialog>
    </BankAccountWarningContext.Provider>
  );
}

export function useBankAccountWarning(): BankAccountWarningContextValue {
  const ctx = useContext(BankAccountWarningContext);
  if (!ctx) {
    // Graceful no-op if the provider isn't mounted (e.g. in isolated tests).
    return { showBankAccountWarning: () => {} };
  }
  return ctx;
}
