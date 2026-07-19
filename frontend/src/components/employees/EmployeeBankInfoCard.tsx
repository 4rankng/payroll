import { Building2, Copy, CreditCard, UserRound } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { toast } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";

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
      className={cn("ct-card employee-surface-card overflow-hidden bg-base-100", className)}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <div className="border-b border-[var(--employee-accent-border)] bg-[var(--employee-summary-wash)] px-4 py-3.5">
        <p className="employee-type-label-caps text-[var(--employee-accent)]">Thông tin chi trả</p>
        <h2 id="employee-bank-title" className="employee-type-section-title mt-0.5 text-[var(--employee-text)]">
          Tài khoản nhận tiền
        </h2>
      </div>

      <div className="m-2 overflow-hidden rounded-[var(--employee-radius-card)] border border-base-300 bg-base-100" style={style}>
        {hasBankInfo ? (
          <dl className="divide-y divide-[var(--employee-border)]">
            <div className="grid grid-cols-[32px_minmax(0,1fr)] items-center gap-3 px-3.5 py-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-[10px] bg-slate-50 text-slate-500">
                <Building2 className="h-4 w-4" aria-hidden="true" />
              </span>
              <div className="min-w-0">
                <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Ngân hàng</dt>
                <dd className="employee-type-bank-value mt-0.5 break-words text-[var(--employee-text)]">
                  {profile.bank?.branch_name || "—"}
                </dd>
              </div>
            </div>

            <div className="grid grid-cols-[32px_minmax(0,1fr)] items-center gap-3 px-3.5 py-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-[10px] bg-employee/10 text-employee">
                <CreditCard className="h-4 w-4" aria-hidden="true" />
              </span>
              <div className="min-w-0">
                <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Số tài khoản</dt>
                <dd className="mt-0.5 flex min-w-0 items-center gap-2">
                  <span className="employee-type-bank-value min-w-0 flex-1 break-all font-mono text-[var(--employee-text)] tabular-nums">
                    {profile.bank_account_number || "—"}
                  </span>
                  {profile.bank_account_number && (
                    <button
                      type="button"
                      onClick={() => handleCopyAccountNumber(profile.bank_account_number!)}
                      className="ct-btn ct-btn-ghost ct-btn-circle employee-icon-button shrink-0 border border-primary/20 bg-primary/10 text-primary"
                      aria-label="Sao chép số tài khoản"
                      title="Sao chép số tài khoản"
                    >
                      <Copy className="h-4 w-4" aria-hidden="true" />
                    </button>
                  )}
                </dd>
              </div>
            </div>

            <div className="grid grid-cols-[32px_minmax(0,1fr)] items-center gap-3 px-3.5 py-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-[10px] bg-slate-50 text-slate-500">
                <UserRound className="h-4 w-4" aria-hidden="true" />
              </span>
              <div className="min-w-0">
                <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">Chủ tài khoản</dt>
                <dd className="employee-type-bank-value mt-0.5 break-words text-[var(--employee-text)]">
                  {accountOwner || "—"}
                </dd>
              </div>
            </div>
          </dl>
        ) : (
          <div className="px-4 py-7 text-center">
            <span className="mx-auto flex h-11 w-11 items-center justify-center rounded-[14px] bg-slate-50 text-slate-400">
              <Building2 className="h-5 w-5" aria-hidden="true" />
            </span>
            <p className="employee-type-strong mt-3 text-[var(--employee-text)]">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
