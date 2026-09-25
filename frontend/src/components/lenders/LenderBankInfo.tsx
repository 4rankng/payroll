import { Building, CreditCard, User, AlertCircle } from "lucide-react";
import type { Lender } from "@/types/api/loan.types";
import { useBank } from "@/hooks/api/useBanks";

interface LenderBankInfoProps {
  lender: Lender;
}

export function LenderBankInfo({ lender }: LenderBankInfoProps) {
  const hasBankAccountDetails = !!(lender.bank_account_number || lender.bank_account_name);
  const hasBankId = !!lender.bank_id;

  // Load bank details if bank_id exists
  const { data: bank, isLoading: isBankLoading } = useBank(lender.bank_id || 0, {
    enabled: hasBankId && !lender.bank // Only fetch if we have bank_id but not the full bank object
  });

  const hasBankDetails = hasBankAccountDetails || hasBankId;

  if (!hasBankDetails) {
    return (
      <div className="flex items-center gap-3 p-3 bg-amber-50 border border-amber-200 rounded-xl">
        <AlertCircle className="w-5 h-5 text-amber-700 shrink-0" />
        <div className="flex-1">
          <p className="typography-body-medium text-amber-900">
            Chủ nợ chưa có thông tin ngân hàng
          </p>
          <p className="typography-body-small text-amber-700 mt-0.5">
            Vui lòng cập nhật thông tin ngân hàng để có thể thực hiện thanh toán
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border bg-card grid grid-cols-1 sm:grid-cols-3 divide-y sm:divide-y-0 sm:divide-x divide-border/50">
      <div className="flex flex-col gap-0.5 px-3 py-2.5">
        <span className="text-xs text-muted-foreground">Ngân hàng</span>
        <span className="text-sm font-semibold">
          {isBankLoading ? 'Đang tải...' : (bank?.branch_name || lender.bank?.branch_name || '-')}
        </span>
      </div>
      <div className="flex flex-col gap-0.5 px-3 py-2.5">
        <span className="text-xs text-muted-foreground">Số tài khoản</span>
        <span className="text-sm font-semibold font-mono">{lender.bank_account_number || '-'}</span>
      </div>
      <div className="flex flex-col gap-0.5 px-3 py-2.5">
        <span className="text-xs text-muted-foreground">Chủ tài khoản</span>
        <span className="text-sm font-semibold">{lender.bank_account_name || '-'}</span>
      </div>
    </div>
  );
}
