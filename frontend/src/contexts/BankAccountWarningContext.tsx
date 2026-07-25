import React, { createContext, useCallback, useContext, useState } from "react";
import { AlertTriangle } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

/**
 * BankAccountWarningContext surfaces a click-to-dismiss modal dialog when a
 * saved employee has an OnePay-confirmed invalid bank account. The save
 * itself succeeds (HTTP 200); this dialog forces the admin/partner to
 * acknowledge the problem so payments don't silently fail later.
 *
 * Unlike a toast (which auto-dismisses and is easily missed), this is a
 * centered modal that requires an explicit "Đã hiểu" click.
 */

interface BankAccountWarning {
  employeeName?: string;
  reason: string;
}

interface BankAccountWarningContextValue {
  showBankAccountWarning: (warning: BankAccountWarning) => void;
}

const BankAccountWarningContext = createContext<BankAccountWarningContextValue | null>(null);

export function BankAccountWarningProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const [warning, setWarning] = useState<BankAccountWarning | null>(null);

  const showBankAccountWarning = useCallback((w: BankAccountWarning) => {
    setWarning(w);
    setOpen(true);
  }, []);

  return (
    <BankAccountWarningContext.Provider value={{ showBankAccountWarning }}>
      {children}
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader className="items-center text-center sm:text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-amber-100">
              <AlertTriangle className="h-6 w-6 text-amber-600" aria-hidden="true" />
            </div>
            <AlertDialogTitle className="text-lg font-semibold text-foreground">
              Tài khoản ngân hàng không hợp lệ
            </AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-muted-foreground text-center">
              Đã lưu thông tin nhân viên
              {warning?.employeeName ? ` (${warning.employeeName})` : ""}, nhưng
              OnePay xác nhận tài khoản ngân hàng không hợp lệ. Bạn cần kiểm tra
              lại để tránh lỗi khi thanh toán.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3">
            <p className="text-xs font-medium text-amber-900">Lý do</p>
            <p className="mt-1 text-sm text-amber-800">
              {warning?.reason ?? "Không xác định"}
            </p>
          </div>

          <div className="mt-5 flex justify-center">
            <AlertDialogAction
              className="min-w-[120px]"
              onClick={() => setOpen(false)}
            >
              Đã hiểu
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
