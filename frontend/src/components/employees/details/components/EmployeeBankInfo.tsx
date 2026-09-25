import { Building, CreditCard, User, AlertCircle } from "lucide-react";
import type { EmployeeDetailsProps } from "../types";

export function EmployeeBankInfo({ employee }: EmployeeDetailsProps) {
  const hasBankDetails = employee.bank_account_number || employee.bank;

  if (!hasBankDetails) {
    return (
      <div className="flex items-start gap-3 p-3 bg-amber-50 border border-amber-200 rounded-xl">
        <AlertCircle className="w-4 h-4 text-amber-700 shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-amber-900">Chưa có thông tin ngân hàng</p>
          <p className="text-xs text-amber-700 mt-0.5">Cập nhật để thực hiện thanh toán</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-muted/30 rounded-xl px-3 py-1">
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-x-4">
        <div className="flex items-baseline justify-between gap-2 py-1.5 border-b sm:border-b-0 sm:border-r border-border/50">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1">
            <Building className="w-3 h-3" />Ngân hàng
          </span>
          <span className="text-xs font-medium text-right">{employee.bank?.branch_name || '-'}</span>
        </div>
        <div className="flex items-baseline justify-between gap-2 py-1.5 border-b sm:border-b-0 sm:border-r border-border/50 sm:px-3">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1">
            <CreditCard className="w-3 h-3" />Số TK
          </span>
          <span className="text-xs font-mono font-medium text-right">{employee.bank_account_number || '-'}</span>
        </div>
        <div className="flex items-baseline justify-between gap-2 py-1.5 sm:px-3">
          <span className="text-xs text-muted-foreground shrink-0 flex items-center gap-1">
            <User className="w-3 h-3" />Chủ TK
          </span>
          <span className="text-xs font-medium text-right">{employee.bank_account_name || '-'}</span>
        </div>
      </div>
    </div>
  );
}
