import { Copy } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { toast } from "@/components/ui/sonner";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

export function EmployeeBankInfoCard({
  profile,
  className,
  style,
}: EmployeeBankInfoCardProps) {
  if (!profile) return null;

  const accountOwner = profile.bank_account_name?.trim();
  const hasBankInfo = !!(
    profile.bank_account_number ||
    profile.bank ||
    accountOwner
  );

  const handleCopyAccountNumber = async (accountNumber: string) => {
    try {
      await navigator.clipboard.writeText(accountNumber);
      toast({ title: "Đã sao chép số tài khoản" });
    } catch {
      toast({
        title: "Chưa thể sao chép số tài khoản",
        description: "Vui lòng thử lại.",
        variant: "destructive",
      });
    }
  };

  return (
    <div
      className={className}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <h2 id="employee-bank-title" className="employee-type-section-title mb-2 px-0.5 text-[#101828]">
        Tài khoản nhận tiền
      </h2>

      <div className="overflow-hidden rounded-xl border border-[var(--employee-border)] bg-white shadow-[var(--employee-shadow)]" style={style}>
        {hasBankInfo ? (
          <dl className="divide-y divide-[var(--employee-border)] px-4">
            <div className="grid grid-cols-[92px_minmax(0,1fr)] items-start gap-3 py-3">
              <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Ngân hàng</dt>
              <dd className="employee-type-bank-value min-w-0 break-words text-right text-[var(--employee-text)]">
                {profile.bank?.branch_name || "—"}
              </dd>
            </div>

            <div className="grid grid-cols-[92px_minmax(0,1fr)] items-center gap-3 py-2">
              <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Số tài khoản</dt>
              <dd className="flex min-w-0 items-center justify-end gap-1">
                <span className="employee-type-bank-value min-w-0 break-all font-mono text-right text-[var(--employee-text)] tabular-nums">
                  {profile.bank_account_number || "—"}
                </span>
                {profile.bank_account_number && (
                  <button
                    type="button"
                    onClick={() => handleCopyAccountNumber(profile.bank_account_number!)}
                    className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[10px] text-[var(--employee-accent)] transition-transform duration-200 active:scale-[0.94] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
                    aria-label="Sao chép số tài khoản"
                    title="Sao chép số tài khoản"
                  >
                    <Copy className="h-4 w-4" aria-hidden="true" />
                  </button>
                )}
              </dd>
            </div>

            <div className="grid grid-cols-[92px_minmax(0,1fr)] items-start gap-3 py-3">
              <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Chủ tài khoản</dt>
              <dd className="employee-type-bank-value min-w-0 break-words text-right text-[var(--employee-text)]">
                {accountOwner || "—"}
              </dd>
            </div>
          </dl>
        ) : (
          <div className="px-4 py-4 text-center">
            <p className="employee-type-strong text-[#344054]">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body-sm mt-1 text-[#667085]">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
